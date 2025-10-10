// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

/*
Package ratelimit provides advanced rate limiting strategies for SAGE ADK.

This package implements multiple rate limiting algorithms optimized for
different use cases:

  - Token Bucket: Smooth rate limiting with burst support
  - Sliding Window: Precise rate limiting with time-based windows
  - Distributed: Rate limiting across multiple instances using Redis

Features:
  - Multiple algorithm support
  - Distributed rate limiting
  - Per-key rate limiting
  - Burst handling
  - Thread-safe implementations
  - Metrics collection

Example:

	import "github.com/sage-x-project/sage-adk/ratelimit"

	// Token bucket limiter
	limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
	    Rate:     100,  // 100 requests per second
	    Capacity: 200,  // Allow bursts up to 200
	})

	// Check if request is allowed
	if limiter.Allow("user-123") {
	    // Process request
	}

	// Sliding window limiter
	limiter := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
	    Limit:  1000,           // 1000 requests
	    Window: time.Minute,    // Per minute
	})
*/
package ratelimit

import (
	"context"
	"time"
)

// Limiter defines the interface for rate limiters
type Limiter interface {
	// Allow checks if a request is allowed for the given key
	Allow(key string) bool

	// AllowN checks if N requests are allowed for the given key
	AllowN(key string, n int) bool

	// Wait blocks until a request is allowed
	Wait(ctx context.Context, key string) error

	// Reserve reserves a request and returns time until available
	Reserve(key string) time.Duration

	// Stats returns limiter statistics
	Stats() Stats

	// Reset resets the limiter for a specific key
	Reset(key string)

	// Close closes the limiter and releases resources
	Close() error
}

// Stats holds rate limiter statistics
type Stats struct {
	// Allowed is the number of allowed requests
	Allowed int64

	// Denied is the number of denied requests
	Denied int64

	// CurrentKeys is the number of active keys
	CurrentKeys int

	// TotalKeys is the total number of keys seen
	TotalKeys int64
}

// Config holds common rate limiter configuration
type Config struct {
	// CleanupInterval is how often to clean up expired entries
	CleanupInterval time.Duration

	// EnableMetrics enables metrics collection
	EnableMetrics bool

	// MaxKeys is the maximum number of keys to track (0 = unlimited)
	MaxKeys int
}

// DefaultConfig returns default rate limiter configuration
func DefaultConfig() Config {
	return Config{
		CleanupInterval: 1 * time.Minute,
		EnableMetrics:   true,
		MaxKeys:         10000,
	}
}

// Algorithm specifies the rate limiting algorithm
type Algorithm string

const (
	// AlgorithmTokenBucket uses token bucket algorithm
	AlgorithmTokenBucket Algorithm = "token_bucket"

	// AlgorithmSlidingWindow uses sliding window counter
	AlgorithmSlidingWindow Algorithm = "sliding_window"

	// AlgorithmFixedWindow uses fixed window counter
	AlgorithmFixedWindow Algorithm = "fixed_window"

	// AlgorithmLeakyBucket uses leaky bucket algorithm
	AlgorithmLeakyBucket Algorithm = "leaky_bucket"
)

// KeyFunc generates a rate limit key from request context
type KeyFunc func(ctx context.Context) string

// DefaultKeyFunc returns a default key (single global limit)
func DefaultKeyFunc(ctx context.Context) string {
	return "default"
}

// IPKeyFunc returns the IP address as key
func IPKeyFunc(ctx context.Context) string {
	// This would extract IP from context
	return "0.0.0.0"
}

// UserKeyFunc returns the user ID as key
func UserKeyFunc(ctx context.Context) string {
	// This would extract user ID from context
	return "anonymous"
}
