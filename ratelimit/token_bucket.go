// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// TokenBucketConfig holds token bucket configuration
type TokenBucketConfig struct {
	// Rate is the number of tokens added per second
	Rate float64

	// Capacity is the maximum number of tokens in the bucket
	Capacity int

	// Config holds common configuration
	Config
}

// DefaultTokenBucketConfig returns default token bucket configuration
func DefaultTokenBucketConfig() TokenBucketConfig {
	return TokenBucketConfig{
		Rate:     10.0,
		Capacity: 100,
		Config:   DefaultConfig(),
	}
}

// TokenBucket implements token bucket rate limiting algorithm
type TokenBucket struct {
	config  TokenBucketConfig
	buckets sync.Map
	stats   Stats
	done    chan struct{}
}

// bucket represents a token bucket for a specific key
type bucket struct {
	tokens       float64
	lastRefill   time.Time
	mu           sync.Mutex
}

// NewTokenBucket creates a new token bucket limiter
func NewTokenBucket(config TokenBucketConfig) *TokenBucket {
	if config.Rate <= 0 {
		config = DefaultTokenBucketConfig()
	}

	tb := &TokenBucket{
		config: config,
		done:   make(chan struct{}),
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go tb.cleanup()
	}

	return tb
}

// Allow checks if a request is allowed
func (tb *TokenBucket) Allow(key string) bool {
	return tb.AllowN(key, 1)
}

// AllowN checks if N requests are allowed
func (tb *TokenBucket) AllowN(key string, n int) bool {
	if n <= 0 {
		return true
	}

	b := tb.getBucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.config.Rate

	b.tokens += tokensToAdd
	if b.tokens > float64(tb.config.Capacity) {
		b.tokens = float64(tb.config.Capacity)
	}
	b.lastRefill = now

	// Check if enough tokens
	if b.tokens >= float64(n) {
		b.tokens -= float64(n)
		if tb.config.EnableMetrics {
			atomic.AddInt64(&tb.stats.Allowed, 1)
		}
		return true
	}

	if tb.config.EnableMetrics {
		atomic.AddInt64(&tb.stats.Denied, 1)
	}
	return false
}

// Wait blocks until a request is allowed
func (tb *TokenBucket) Wait(ctx context.Context, key string) error {
	for {
		if tb.Allow(key) {
			return nil
		}

		// Calculate wait time
		waitTime := time.Duration(1000.0/tb.config.Rate) * time.Millisecond

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue loop
		}
	}
}

// Reserve reserves a token and returns wait duration
func (tb *TokenBucket) Reserve(key string) time.Duration {
	b := tb.getBucket(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	tokensToAdd := elapsed * tb.config.Rate

	b.tokens += tokensToAdd
	if b.tokens > float64(tb.config.Capacity) {
		b.tokens = float64(tb.config.Capacity)
	}
	b.lastRefill = now

	// Check if token available
	if b.tokens >= 1 {
		b.tokens -= 1
		return 0
	}

	// Calculate wait time
	tokensNeeded := 1.0 - b.tokens
	waitSeconds := tokensNeeded / tb.config.Rate
	return time.Duration(waitSeconds * float64(time.Second))
}

// Stats returns limiter statistics
func (tb *TokenBucket) Stats() Stats {
	stats := Stats{
		Allowed: atomic.LoadInt64(&tb.stats.Allowed),
		Denied:  atomic.LoadInt64(&tb.stats.Denied),
	}

	// Count current keys
	count := 0
	tb.buckets.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	stats.CurrentKeys = count

	return stats
}

// Reset resets the limiter for a specific key
func (tb *TokenBucket) Reset(key string) {
	tb.buckets.Delete(key)
}

// Close closes the limiter
func (tb *TokenBucket) Close() error {
	close(tb.done)
	return nil
}

// getBucket gets or creates a bucket for a key
func (tb *TokenBucket) getBucket(key string) *bucket {
	if v, ok := tb.buckets.Load(key); ok {
		return v.(*bucket)
	}

	b := &bucket{
		tokens:     float64(tb.config.Capacity),
		lastRefill: time.Now(),
	}

	actual, _ := tb.buckets.LoadOrStore(key, b)
	return actual.(*bucket)
}

// cleanup periodically removes inactive buckets
func (tb *TokenBucket) cleanup() {
	ticker := time.NewTicker(tb.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tb.done:
			return
		case <-ticker.C:
			tb.performCleanup()
		}
	}
}

// performCleanup removes buckets that haven't been used recently
func (tb *TokenBucket) performCleanup() {
	now := time.Now()
	threshold := tb.config.CleanupInterval * 2

	keysToDelete := make([]string, 0)

	tb.buckets.Range(func(key, value interface{}) bool {
		b := value.(*bucket)
		b.mu.Lock()
		if now.Sub(b.lastRefill) > threshold {
			keysToDelete = append(keysToDelete, key.(string))
		}
		b.mu.Unlock()
		return true
	})

	for _, key := range keysToDelete {
		tb.buckets.Delete(key)
	}
}
