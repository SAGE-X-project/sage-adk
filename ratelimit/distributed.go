// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedConfig holds distributed rate limiter configuration
type DistributedConfig struct {
	// RedisClient is the Redis client
	RedisClient *redis.Client

	// KeyPrefix is the prefix for Redis keys
	KeyPrefix string

	// Limit is the maximum number of requests allowed
	Limit int

	// Window is the time window duration
	Window time.Duration

	// Algorithm is the rate limiting algorithm
	Algorithm Algorithm

	// Config holds common configuration
	Config
}

// DefaultDistributedConfig returns default distributed configuration
func DefaultDistributedConfig() DistributedConfig {
	return DistributedConfig{
		KeyPrefix: "ratelimit:",
		Limit:     100,
		Window:    time.Minute,
		Algorithm: AlgorithmSlidingWindow,
		Config:    DefaultConfig(),
	}
}

// Distributed implements distributed rate limiting using Redis
type Distributed struct {
	config DistributedConfig
	stats  Stats
}

// NewDistributed creates a new distributed rate limiter
func NewDistributed(config DistributedConfig) (*Distributed, error) {
	if config.RedisClient == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	if config.Limit <= 0 {
		config = DefaultDistributedConfig()
	}

	return &Distributed{
		config: config,
	}, nil
}

// Allow checks if a request is allowed
func (d *Distributed) Allow(key string) bool {
	return d.AllowN(key, 1)
}

// AllowN checks if N requests are allowed
func (d *Distributed) AllowN(key string, n int) bool {
	if n <= 0 {
		return true
	}

	ctx := context.Background()
	redisKey := d.config.KeyPrefix + key

	switch d.config.Algorithm {
	case AlgorithmSlidingWindow:
		return d.allowSlidingWindow(ctx, redisKey, n)
	case AlgorithmFixedWindow:
		return d.allowFixedWindow(ctx, redisKey, n)
	default:
		return d.allowSlidingWindow(ctx, redisKey, n)
	}
}

// allowSlidingWindow implements sliding window using Redis sorted set
func (d *Distributed) allowSlidingWindow(ctx context.Context, key string, n int) bool {
	now := time.Now()
	windowStart := now.Add(-d.config.Window)

	pipe := d.config.RedisClient.Pipeline()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixNano(), 10))

	// Count current entries
	countCmd := pipe.ZCard(ctx, key)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		if d.config.EnableMetrics {
			atomic.AddInt64(&d.stats.Denied, int64(n))
		}
		return false
	}

	currentCount := countCmd.Val()

	// Check if under limit
	if int(currentCount)+n <= d.config.Limit {
		// Add new entries
		pipe2 := d.config.RedisClient.Pipeline()
		for i := 0; i < n; i++ {
			timestamp := now.Add(time.Duration(i) * time.Nanosecond)
			pipe2.ZAdd(ctx, key, redis.Z{
				Score:  float64(timestamp.UnixNano()),
				Member: fmt.Sprintf("%d-%d", timestamp.UnixNano(), i),
			})
		}

		// Set expiration
		pipe2.Expire(ctx, key, d.config.Window*2)

		_, err := pipe2.Exec(ctx)
		if err != nil {
			if d.config.EnableMetrics {
				atomic.AddInt64(&d.stats.Denied, int64(n))
			}
			return false
		}

		if d.config.EnableMetrics {
			atomic.AddInt64(&d.stats.Allowed, int64(n))
		}
		return true
	}

	if d.config.EnableMetrics {
		atomic.AddInt64(&d.stats.Denied, int64(n))
	}
	return false
}

// allowFixedWindow implements fixed window using Redis counter
func (d *Distributed) allowFixedWindow(ctx context.Context, key string, n int) bool {
	now := time.Now()
	windowKey := fmt.Sprintf("%s:%d", key, now.Unix()/int64(d.config.Window.Seconds()))

	pipe := d.config.RedisClient.Pipeline()

	// Increment counter
	incrCmd := pipe.IncrBy(ctx, windowKey, int64(n))

	// Set expiration on first increment
	pipe.Expire(ctx, windowKey, d.config.Window*2)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		if d.config.EnableMetrics {
			atomic.AddInt64(&d.stats.Denied, int64(n))
		}
		return false
	}

	newCount := incrCmd.Val()

	// Check if under limit
	if int(newCount) <= d.config.Limit {
		if d.config.EnableMetrics {
			atomic.AddInt64(&d.stats.Allowed, int64(n))
		}
		return true
	}

	// Over limit, decrement back
	d.config.RedisClient.DecrBy(ctx, windowKey, int64(n))

	if d.config.EnableMetrics {
		atomic.AddInt64(&d.stats.Denied, int64(n))
	}
	return false
}

// Wait blocks until a request is allowed
func (d *Distributed) Wait(ctx context.Context, key string) error {
	for {
		if d.Allow(key) {
			return nil
		}

		// Wait for a short period
		waitTime := d.config.Window / time.Duration(d.config.Limit)
		if waitTime < 10*time.Millisecond {
			waitTime = 10 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop
		}
	}
}

// Reserve reserves a request and returns wait duration
func (d *Distributed) Reserve(key string) time.Duration {
	if d.Allow(key) {
		return 0
	}

	// Estimate wait time
	return d.config.Window / time.Duration(d.config.Limit)
}

// Stats returns limiter statistics
func (d *Distributed) Stats() Stats {
	return Stats{
		Allowed: atomic.LoadInt64(&d.stats.Allowed),
		Denied:  atomic.LoadInt64(&d.stats.Denied),
	}
}

// Reset resets the limiter for a specific key
func (d *Distributed) Reset(key string) {
	ctx := context.Background()
	redisKey := d.config.KeyPrefix + key
	d.config.RedisClient.Del(ctx, redisKey)
}

// Close closes the limiter
func (d *Distributed) Close() error {
	return nil
}
