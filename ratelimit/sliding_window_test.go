// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestSlidingWindow_Allow(t *testing.T) {
	tests := []struct {
		name     string
		config   SlidingWindowConfig
		requests int
		sleep    time.Duration
		wantPass int
	}{
		{
			name: "under limit",
			config: SlidingWindowConfig{
				Limit:  10,
				Window: time.Second,
			},
			requests: 5,
			sleep:    0,
			wantPass: 5,
		},
		{
			name: "at limit",
			config: SlidingWindowConfig{
				Limit:  10,
				Window: time.Second,
			},
			requests: 10,
			sleep:    0,
			wantPass: 10,
		},
		{
			name: "over limit",
			config: SlidingWindowConfig{
				Limit:  10,
				Window: time.Second,
			},
			requests: 15,
			sleep:    0,
			wantPass: 10,
		},
		{
			name: "window expiration",
			config: SlidingWindowConfig{
				Limit:  5,
				Window: 200 * time.Millisecond,
			},
			requests: 10,
			sleep:    250 * time.Millisecond, // Wait for window to expire
			wantPass: 10,                      // First 5 + next 5 after expiration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewSlidingWindow(tt.config)
			defer limiter.Close()

			passed := 0
			halfRequests := tt.requests / 2

			// First batch
			for i := 0; i < halfRequests; i++ {
				if limiter.Allow("test-key") {
					passed++
				}
			}

			// Sleep to allow window expiration
			if tt.sleep > 0 {
				time.Sleep(tt.sleep)
			}

			// Second batch
			for i := 0; i < tt.requests-halfRequests; i++ {
				if limiter.Allow("test-key") {
					passed++
				}
			}

			if passed != tt.wantPass {
				t.Errorf("Allow() passed %d requests, want %d", passed, tt.wantPass)
			}
		})
	}
}

func TestSlidingWindow_AllowN(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  100,
		Window: time.Second,
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

	// Should deny batch of 10 (over limit)
	if limiter.AllowN("test-key", 10) {
		t.Error("AllowN(10) should be denied")
	}
}

func TestSlidingWindow_Wait(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  5,
		Window: 100 * time.Millisecond,
	})
	defer limiter.Close()

	// Use up all requests
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

func TestSlidingWindow_Reserve(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  10,
		Window: time.Second,
	})
	defer limiter.Close()

	// First reserve should be immediate
	wait := limiter.Reserve("test-key")
	if wait != 0 {
		t.Errorf("Reserve() wait = %v, want 0", wait)
	}

	// Use up all slots
	for i := 0; i < 9; i++ {
		limiter.Allow("test-key")
	}

	// Reserve should return wait time
	wait = limiter.Reserve("test-key")
	if wait == 0 {
		t.Error("Reserve() should return non-zero wait time")
	}
}

func TestSlidingWindow_Stats(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  5,
		Window: time.Second,
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

	if stats.Allowed != 5 {
		t.Errorf("Stats.Allowed = %d, want 5", stats.Allowed)
	}

	if stats.Denied != 3 {
		t.Errorf("Stats.Denied = %d, want 3", stats.Denied)
	}
}

func TestSlidingWindow_Reset(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  5,
		Window: time.Second,
	})
	defer limiter.Close()

	// Use up all slots
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

func TestSlidingWindow_MultipleKeys(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  5,
		Window: time.Second,
	})
	defer limiter.Close()

	// Use up slots for key1
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

func TestSlidingWindow_Precision(t *testing.T) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  3,
		Window: 100 * time.Millisecond,
	})
	defer limiter.Close()

	// Allow 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th should be denied
	if limiter.Allow("test-key") {
		t.Error("4th request should be denied")
	}

	// Wait for window to expire
	time.Sleep(110 * time.Millisecond)

	// Should allow 3 more requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("Request %d after window should be allowed", i+1)
		}
	}
}

func BenchmarkSlidingWindow_Allow(b *testing.B) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  1000000,
		Window: time.Hour, // Large window to minimize denials
	})
	defer limiter.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow("bench-key")
		}
	})
}

func BenchmarkSlidingWindow_AllowMultipleKeys(b *testing.B) {
	limiter := NewSlidingWindow(SlidingWindowConfig{
		Limit:  1000000,
		Window: time.Hour,
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
