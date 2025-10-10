// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	tests := []struct {
		name     string
		config   TokenBucketConfig
		requests int
		sleep    time.Duration
		wantPass int
	}{
		{
			name: "under limit",
			config: TokenBucketConfig{
				Rate:     10.0,
				Capacity: 10,
			},
			requests: 5,
			sleep:    0,
			wantPass: 5,
		},
		{
			name: "at limit",
			config: TokenBucketConfig{
				Rate:     10.0,
				Capacity: 10,
			},
			requests: 10,
			sleep:    0,
			wantPass: 10,
		},
		{
			name: "over limit",
			config: TokenBucketConfig{
				Rate:     10.0,
				Capacity: 10,
			},
			requests: 15,
			sleep:    0,
			wantPass: 10,
		},
		{
			name: "refill after sleep",
			config: TokenBucketConfig{
				Rate:     10.0, // 10 tokens per second
				Capacity: 10,
			},
			requests: 10,
			sleep:    200 * time.Millisecond, // Should refill ~2 tokens
			wantPass: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewTokenBucket(tt.config)
			defer limiter.Close()

			passed := 0
			halfRequests := tt.requests / 2

			// First batch
			for i := 0; i < halfRequests; i++ {
				if limiter.Allow("test-key") {
					passed++
				}
			}

			// Sleep to allow refill
			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			// Second batch
			for i := 0; i < tt.requests-halfRequests; i++ {
				if limiter.Allow("test-key") {
					passed++
				}
			}

			if passed < tt.wantPass-2 || passed > tt.wantPass+2 {
				t.Errorf("Allow() passed %d requests, want ~%d", passed, tt.wantPass)
			}
		})
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     10.0,
		Capacity: 100,
	})
	defer limiter.Close()

	// Should allow batch of 50
	if !limiter.AllowN("test-key", 50) {
		t.Error("AllowN(50) should be allowed")
	}

	// Should allow another batch of 50
	if !limiter.AllowN("test-key", 50) {
		t.Error("AllowN(50) should be allowed")
	}

	// Should deny batch of 10 (over capacity)
	if limiter.AllowN("test-key", 10) {
		t.Error("AllowN(10) should be denied")
	}
}

func TestTokenBucket_Wait(t *testing.T) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     100.0, // Fast rate for testing
		Capacity: 5,
	})
	defer limiter.Close()

	// Use up all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow("test-key")
	}

	// Wait should eventually succeed
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := limiter.Wait(ctx, "test-key")
	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
}

func TestTokenBucket_Reserve(t *testing.T) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     10.0,
		Capacity: 10,
	})
	defer limiter.Close()

	// First reserve should be immediate
	wait := limiter.Reserve("test-key")
	if wait != 0 {
		t.Errorf("Reserve() wait = %v, want 0", wait)
	}

	// Use up all tokens
	for i := 0; i < 9; i++ {
		limiter.Allow("test-key")
	}

	// Reserve should return wait time
	wait = limiter.Reserve("test-key")
	if wait == 0 {
		t.Error("Reserve() should return non-zero wait time")
	}
}

func TestTokenBucket_Stats(t *testing.T) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     10.0,
		Capacity: 5,
		Config: Config{
			EnableMetrics: true,
		},
	})
	defer limiter.Close()

	// Allow some requests
	for i := 0; i < 3; i++ {
		limiter.Allow("test-key")
	}

	// Deny some requests
	for i := 0; i < 5; i++ {
		limiter.Allow("test-key")
	}

	stats := limiter.Stats()

	if stats.Allowed != 5 { // 3 + 2 from capacity of 5
		t.Errorf("Stats.Allowed = %d, want 5", stats.Allowed)
	}

	if stats.Denied != 3 {
		t.Errorf("Stats.Denied = %d, want 3", stats.Denied)
	}
}

func TestTokenBucket_Reset(t *testing.T) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     10.0,
		Capacity: 5,
	})
	defer limiter.Close()

	// Use up all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow("test-key")
	}

	// Should be denied
	if limiter.Allow("test-key") {
		t.Error("Allow() should be denied")
	}

	// Reset
	limiter.Reset("test-key")

	// Should be allowed after reset
	if !limiter.Allow("test-key") {
		t.Error("Allow() should be allowed after reset")
	}
}

func TestTokenBucket_MultipleKeys(t *testing.T) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     10.0,
		Capacity: 5,
	})
	defer limiter.Close()

	// Use up tokens for key1
	for i := 0; i < 5; i++ {
		limiter.Allow("key1")
	}

	// key1 should be denied
	if limiter.Allow("key1") {
		t.Error("key1 should be denied")
	}

	// key2 should still be allowed
	if !limiter.Allow("key2") {
		t.Error("key2 should be allowed")
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     1000000.0, // Very high rate to minimize denials
		Capacity: 1000000,
	})
	defer limiter.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow("bench-key")
		}
	})
}

func BenchmarkTokenBucket_AllowMultipleKeys(b *testing.B) {
	limiter := NewTokenBucket(TokenBucketConfig{
		Rate:     1000000.0,
		Capacity: 1000000,
	})
	defer limiter.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key-" + string(rune(i%10))
			limiter.Allow(key)
			i++
		}
	})
}
