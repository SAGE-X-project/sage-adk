// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// SlidingWindowConfig holds sliding window configuration
type SlidingWindowConfig struct {
	// Limit is the maximum number of requests allowed
	Limit int

	// Window is the time window duration
	Window time.Duration

	// Config holds common configuration
	Config
}

// DefaultSlidingWindowConfig returns default sliding window configuration
func DefaultSlidingWindowConfig() SlidingWindowConfig {
	return SlidingWindowConfig{
		Limit:  100,
		Window: time.Minute,
		Config: DefaultConfig(),
	}
}

// SlidingWindow implements sliding window counter algorithm
type SlidingWindow struct {
	config  SlidingWindowConfig
	windows sync.Map
	stats   Stats
	done    chan struct{}
}

// window represents a sliding window for a specific key
type window struct {
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindow creates a new sliding window limiter
func NewSlidingWindow(config SlidingWindowConfig) *SlidingWindow {
	if config.Limit <= 0 {
		config = DefaultSlidingWindowConfig()
	}

	sw := &SlidingWindow{
		config: config,
		done:   make(chan struct{}),
	}

	// Start cleanup goroutine
	if config.CleanupInterval > 0 {
		go sw.cleanup()
	}

	return sw
}

// Allow checks if a request is allowed
func (sw *SlidingWindow) Allow(key string) bool {
	return sw.AllowN(key, 1)
}

// AllowN checks if N requests are allowed
func (sw *SlidingWindow) AllowN(key string, n int) bool {
	if n <= 0 {
		return true
	}

	w := sw.getWindow(key)
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.config.Window)

	// Remove expired requests
	validRequests := make([]time.Time, 0, len(w.requests))
	for _, t := range w.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	w.requests = validRequests

	// Check if under limit
	if len(w.requests)+n <= sw.config.Limit {
		// Add n requests
		for i := 0; i < n; i++ {
			w.requests = append(w.requests, now)
		}
		if sw.config.EnableMetrics {
			atomic.AddInt64(&sw.stats.Allowed, int64(n))
		}
		return true
	}

	if sw.config.EnableMetrics {
		atomic.AddInt64(&sw.stats.Denied, int64(n))
	}
	return false
}

// Wait blocks until a request is allowed
func (sw *SlidingWindow) Wait(ctx context.Context, key string) error {
	for {
		if sw.Allow(key) {
			return nil
		}

		// Wait for a short period
		waitTime := sw.config.Window / time.Duration(sw.config.Limit)
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
func (sw *SlidingWindow) Reserve(key string) time.Duration {
	w := sw.getWindow(key)
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.config.Window)

	// Remove expired requests
	validRequests := make([]time.Time, 0, len(w.requests))
	for _, t := range w.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	w.requests = validRequests

	// Check if under limit
	if len(w.requests) < sw.config.Limit {
		w.requests = append(w.requests, now)
		return 0
	}

	// Calculate wait time until oldest request expires
	oldestRequest := w.requests[0]
	waitUntil := oldestRequest.Add(sw.config.Window)
	return time.Until(waitUntil)
}

// Stats returns limiter statistics
func (sw *SlidingWindow) Stats() Stats {
	stats := Stats{
		Allowed: atomic.LoadInt64(&sw.stats.Allowed),
		Denied:  atomic.LoadInt64(&sw.stats.Denied),
	}

	// Count current keys
	count := 0
	sw.windows.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	stats.CurrentKeys = count

	return stats
}

// Reset resets the limiter for a specific key
func (sw *SlidingWindow) Reset(key string) {
	sw.windows.Delete(key)
}

// Close closes the limiter
func (sw *SlidingWindow) Close() error {
	close(sw.done)
	return nil
}

// getWindow gets or creates a window for a key
func (sw *SlidingWindow) getWindow(key string) *window {
	if v, ok := sw.windows.Load(key); ok {
		return v.(*window)
	}

	w := &window{
		requests: make([]time.Time, 0),
	}

	actual, _ := sw.windows.LoadOrStore(key, w)
	return actual.(*window)
}

// cleanup periodically removes inactive windows
func (sw *SlidingWindow) cleanup() {
	ticker := time.NewTicker(sw.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sw.done:
			return
		case <-ticker.C:
			sw.performCleanup()
		}
	}
}

// performCleanup removes windows that have no recent requests
func (sw *SlidingWindow) performCleanup() {
	now := time.Now()
	threshold := now.Add(-sw.config.Window * 2)

	keysToDelete := make([]string, 0)

	sw.windows.Range(func(key, value interface{}) bool {
		w := value.(*window)
		w.mu.Lock()

		// Check if all requests are expired
		hasRecentRequests := false
		for _, t := range w.requests {
			if t.After(threshold) {
				hasRecentRequests = true
				break
			}
		}

		if !hasRecentRequests && len(w.requests) > 0 {
			keysToDelete = append(keysToDelete, key.(string))
		}

		w.mu.Unlock()
		return true
	})

	for _, key := range keysToDelete {
		sw.windows.Delete(key)
	}
}
