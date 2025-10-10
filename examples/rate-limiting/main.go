// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

/*
Rate Limiting Example

This example demonstrates how to use rate limiting in SAGE ADK agents.

Features:
  - Token bucket rate limiting
  - Sliding window rate limiting
  - Per-user rate limiting
  - Global rate limiting
  - Custom rate limit handlers

Run:
	go run main.go
*/
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage-adk/ratelimit"
)

func main() {
	fmt.Println("=== SAGE ADK Rate Limiting Example ===")

	// Run examples
	tokenBucketExample()
	slidingWindowExample()
	middlewareExample()
	burstHandlingExample()
	multiUserExample()
}

// tokenBucketExample demonstrates token bucket rate limiting
func tokenBucketExample() {
	fmt.Println("1. Token Bucket Example")
	fmt.Println("   Allows smooth rate limiting with burst support")
	fmt.Println()

	// Create token bucket limiter
	// Rate: 5 requests per second
	// Capacity: 10 (allows bursts up to 10)
	limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Rate:     5.0,
		Capacity: 10,
	})
	defer limiter.Close()

	// Simulate requests
	fmt.Println("   Sending 15 requests (burst):")
	allowed := 0
	denied := 0

	for i := 0; i < 15; i++ {
		if limiter.Allow("user-123") {
			allowed++
			fmt.Printf("   ✓ Request %d: ALLOWED\n", i+1)
		} else {
			denied++
			fmt.Printf("   ✗ Request %d: DENIED (rate limit exceeded)\n", i+1)
		}
	}

	fmt.Printf("\n   Summary: %d allowed, %d denied\n", allowed, denied)
	fmt.Printf("   Stats: %+v\n", limiter.Stats())
	fmt.Println()
}

// slidingWindowExample demonstrates sliding window rate limiting
func slidingWindowExample() {
	fmt.Println("2. Sliding Window Example")
	fmt.Println("   Provides precise rate limiting over time windows")
	fmt.Println()

	// Create sliding window limiter
	// Limit: 5 requests per second
	limiter := ratelimit.NewSlidingWindow(ratelimit.SlidingWindowConfig{
		Limit:  5,
		Window: time.Second,
	})
	defer limiter.Close()

	// First batch
	fmt.Println("   First batch (5 requests):")
	for i := 0; i < 5; i++ {
		if limiter.Allow("user-456") {
			fmt.Printf("   ✓ Request %d: ALLOWED\n", i+1)
		} else {
			fmt.Printf("   ✗ Request %d: DENIED\n", i+1)
		}
	}

	// Try one more (should be denied)
	fmt.Println("\n   Extra request:")
	if limiter.Allow("user-456") {
		fmt.Println("   ✓ ALLOWED")
	} else {
		fmt.Println("   ✗ DENIED (window limit reached)")
	}

	// Wait for window to expire
	fmt.Println("\n   Waiting 1 second for window to expire...")
	time.Sleep(1100 * time.Millisecond)

	// Second batch (should work)
	fmt.Println("\n   Second batch (5 requests):")
	for i := 0; i < 5; i++ {
		if limiter.Allow("user-456") {
			fmt.Printf("   ✓ Request %d: ALLOWED\n", i+1)
		}
	}

	fmt.Printf("\n   Stats: %+v\n", limiter.Stats())
	fmt.Println()
}

// middlewareExample demonstrates rate limiting middleware
func middlewareExample() {
	fmt.Println("3. Middleware Example")
	fmt.Println("   Using rate limiting as middleware")
	fmt.Println()

	// Create middleware
	middleware := ratelimit.NewTokenBucketMiddleware(
		ratelimit.TokenBucketConfig{
			Rate:     3.0,
			Capacity: 5,
		},
		ratelimit.PerUserKeyFunc,
	)

	// Create handler
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return types.NewMessage(
			"assistant",
			[]types.Part{types.NewTextPart("Response: " + msg.Parts[0].(*types.TextPart).Text)},
		), nil
	}

	// Wrap with middleware
	rateLimitedHandler := middleware(handler)

	// Send requests
	ctx := context.Background()
	userID := "user-789"

	fmt.Println("   Sending 8 requests:")
	for i := 0; i < 8; i++ {
		msg := types.NewMessage(
			types.MessageRoleUser,
			[]types.Part{types.NewTextPart(fmt.Sprintf("Request %d", i+1))},
		)
		msg.Metadata = map[string]interface{}{
			"user_id": userID,
		}

		resp, err := rateLimitedHandler(ctx, msg)
		if err != nil {
			fmt.Printf("   ✗ Request %d: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✓ Request %d: %s\n", i+1, resp.Parts[0].(*types.TextPart).Text)
		}
	}
	fmt.Println()
}

// burstHandlingExample demonstrates burst handling
func burstHandlingExample() {
	fmt.Println("4. Burst Handling Example")
	fmt.Println("   Token bucket handles bursts gracefully")
	fmt.Println()

	limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Rate:     2.0,  // 2 tokens per second
		Capacity: 10,   // Can handle burst of 10
	})
	defer limiter.Close()

	// Large burst
	fmt.Println("   Large burst (15 requests):")
	for i := 0; i < 15; i++ {
		if limiter.Allow("burst-test") {
			fmt.Printf("   ✓ ")
		} else {
			fmt.Printf("   ✗ ")
		}
	}
	fmt.Println()

	// Wait for refill
	fmt.Println("\n   Waiting 2 seconds for token refill...")
	time.Sleep(2 * time.Second)

	// Should allow ~4 more requests (2 tokens/sec * 2 sec)
	fmt.Println("\n   After refill (5 requests):")
	for i := 0; i < 5; i++ {
		if limiter.Allow("burst-test") {
			fmt.Printf("   ✓ Request %d: ALLOWED\n", i+1)
		} else {
			fmt.Printf("   ✗ Request %d: DENIED\n", i+1)
		}
	}
	fmt.Println()
}

// multiUserExample demonstrates per-user rate limiting
func multiUserExample() {
	fmt.Println("5. Multi-User Example")
	fmt.Println("   Each user has independent rate limits")
	fmt.Println()

	limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Rate:     5.0,
		Capacity: 5,
	})
	defer limiter.Close()

	users := []string{"alice", "bob", "charlie"}

	// Each user sends 7 requests
	for _, user := range users {
		fmt.Printf("   User: %s\n", user)
		allowed := 0
		for i := 0; i < 7; i++ {
			if limiter.Allow(user) {
				allowed++
			}
		}
		fmt.Printf("   → %d/7 requests allowed\n\n", allowed)
	}

	// Show stats
	stats := limiter.Stats()
	fmt.Printf("   Total Stats: %d allowed, %d denied\n", stats.Allowed, stats.Denied)
	fmt.Printf("   Active users: %d\n", stats.CurrentKeys)
	fmt.Println()
}

// waitExample demonstrates Wait functionality
func waitExample() {
	fmt.Println("6. Wait Example")
	fmt.Println("   Blocking until rate limit allows request")
	fmt.Println()

	limiter := ratelimit.NewTokenBucket(ratelimit.TokenBucketConfig{
		Rate:     10.0,
		Capacity: 5,
	})
	defer limiter.Close()

	// Use up all tokens
	for i := 0; i < 5; i++ {
		limiter.Allow("wait-test")
	}

	fmt.Println("   All tokens used, waiting for refill...")

	start := time.Now()
	ctx := context.Background()
	if err := limiter.Wait(ctx, "wait-test"); err != nil {
		log.Printf("   Error: %v\n", err)
	} else {
		elapsed := time.Since(start)
		fmt.Printf("   ✓ Request allowed after %v\n", elapsed)
	}
	fmt.Println()
}
