// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package ratelimit

import (
	"context"
	"fmt"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Handler is the message handler function type
type Handler func(ctx context.Context, msg *types.Message) (*types.Message, error)

// Middleware is the middleware function type
type Middleware func(Handler) Handler

// MiddlewareConfig holds middleware configuration
type MiddlewareConfig struct {
	// Limiter is the rate limiter to use
	Limiter Limiter

	// KeyFunc generates the rate limit key from context/message
	KeyFunc func(ctx context.Context, msg *types.Message) string

	// OnRateLimitExceeded is called when rate limit is exceeded
	OnRateLimitExceeded func(ctx context.Context, msg *types.Message, key string) (*types.Message, error)
}

// DefaultMiddlewareConfig returns default middleware configuration
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		KeyFunc: func(ctx context.Context, msg *types.Message) string {
			// Use message ID or context ID as key
			if msg.MessageID != "" {
				return msg.MessageID
			}
			if msg.ContextID != nil {
				return *msg.ContextID
			}
			return "default"
		},
		OnRateLimitExceeded: func(ctx context.Context, msg *types.Message, key string) (*types.Message, error) {
			return nil, fmt.Errorf("rate limit exceeded for key: %s", key)
		},
	}
}

// NewMiddleware creates a new rate limiting middleware
func NewMiddleware(config MiddlewareConfig) Middleware {
	if config.KeyFunc == nil {
		config = DefaultMiddlewareConfig()
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Generate rate limit key
			key := config.KeyFunc(ctx, msg)

			// Check rate limit
			if !config.Limiter.Allow(key) {
				if config.OnRateLimitExceeded != nil {
					return config.OnRateLimitExceeded(ctx, msg, key)
				}
				return nil, fmt.Errorf("rate limit exceeded")
			}

			// Process request
			return next(ctx, msg)
		}
	}
}

// NewTokenBucketMiddleware creates a token bucket rate limiting middleware
func NewTokenBucketMiddleware(config TokenBucketConfig, keyFunc func(context.Context, *types.Message) string) Middleware {
	limiter := NewTokenBucket(config)

	middlewareConfig := DefaultMiddlewareConfig()
	middlewareConfig.Limiter = limiter
	if keyFunc != nil {
		middlewareConfig.KeyFunc = keyFunc
	}

	return NewMiddleware(middlewareConfig)
}

// NewSlidingWindowMiddleware creates a sliding window rate limiting middleware
func NewSlidingWindowMiddleware(config SlidingWindowConfig, keyFunc func(context.Context, *types.Message) string) Middleware {
	limiter := NewSlidingWindow(config)

	middlewareConfig := DefaultMiddlewareConfig()
	middlewareConfig.Limiter = limiter
	if keyFunc != nil {
		middlewareConfig.KeyFunc = keyFunc
	}

	return NewMiddleware(middlewareConfig)
}

// NewDistributedMiddleware creates a distributed rate limiting middleware
func NewDistributedMiddleware(config DistributedConfig, keyFunc func(context.Context, *types.Message) string) (Middleware, error) {
	limiter, err := NewDistributed(config)
	if err != nil {
		return nil, err
	}

	middlewareConfig := DefaultMiddlewareConfig()
	middlewareConfig.Limiter = limiter
	if keyFunc != nil {
		middlewareConfig.KeyFunc = keyFunc
	}

	return NewMiddleware(middlewareConfig), nil
}

// PerUserKeyFunc generates a key based on user ID from metadata
func PerUserKeyFunc(ctx context.Context, msg *types.Message) string {
	if msg.Metadata != nil {
		if userID, ok := msg.Metadata["user_id"].(string); ok {
			return fmt.Sprintf("user:%s", userID)
		}
	}
	return "anonymous"
}

// PerContextKeyFunc generates a key based on context ID
func PerContextKeyFunc(ctx context.Context, msg *types.Message) string {
	if msg.ContextID != nil {
		return fmt.Sprintf("context:%s", *msg.ContextID)
	}
	return "no-context"
}

// GlobalKeyFunc generates a global key (single rate limit for all)
func GlobalKeyFunc(ctx context.Context, msg *types.Message) string {
	return "global"
}
