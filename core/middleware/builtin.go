// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package middleware

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Logger creates a logging middleware that logs request/response information.
func Logger(logger *log.Logger) Middleware {
	if logger == nil {
		logger = log.Default()
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			start := time.Now()

			// Log incoming message
			logger.Printf("[Middleware] Incoming message ID: %s, Role: %s",
				msg.MessageID, msg.Role)

			// Call next handler
			resp, err := next(ctx, msg)

			// Log response
			duration := time.Since(start)
			if err != nil {
				logger.Printf("[Middleware] Request failed (ID: %s) - Duration: %v - Error: %v",
					msg.MessageID, duration, err)
			} else {
				logger.Printf("[Middleware] Request completed (ID: %s) - Duration: %v",
					msg.MessageID, duration)
			}

			return resp, err
		}
	}
}

// RequestID creates a middleware that adds a request ID to the context.
func RequestID() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Use message ID as request ID if available
			requestID := msg.MessageID
			if requestID == "" {
				requestID = types.GenerateMessageID()
			}

			// Add to context
			ctx = ContextWithRequestID(ctx, requestID)

			return next(ctx, msg)
		}
	}
}

// Timer creates a middleware that tracks execution time.
func Timer() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			start := time.Now()
			ctx = context.WithValue(ctx, StartTimeKey, start)

			resp, err := next(ctx, msg)

			duration := time.Since(start)

			// Add timing metadata to response if successful
			if err == nil && resp != nil {
				if resp.Metadata == nil {
					resp.Metadata = make(map[string]interface{})
				}
				resp.Metadata["processing_time_ms"] = duration.Milliseconds()
			}

			return resp, err
		}
	}
}

// Recovery creates a middleware that recovers from panics.
func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (resp *types.Message, err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[Middleware] Panic recovered: %v", r)
					err = fmt.Errorf("panic recovered: %v", r)
					resp = nil
				}
			}()

			return next(ctx, msg)
		}
	}
}

// Validator creates a middleware that validates messages.
func Validator() Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Validate message
			if msg == nil {
				return nil, fmt.Errorf("message is nil")
			}

			if msg.MessageID == "" {
				return nil, fmt.Errorf("message ID is empty")
			}

			if msg.Role == "" {
				return nil, fmt.Errorf("message role is empty")
			}

			if len(msg.Parts) == 0 {
				return nil, fmt.Errorf("message has no parts")
			}

			return next(ctx, msg)
		}
	}
}

// RateLimiter creates a simple rate limiting middleware.
type RateLimiterConfig struct {
	MaxRequests int
	Window      time.Duration
}

// RateLimiter creates a rate limiting middleware.
func RateLimiter(config RateLimiterConfig) Middleware {
	requests := make(map[string][]time.Time)

	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Use message ID or context ID as key (simplified for demo)
			key := "default"
			if msg.MessageID != "" {
				key = msg.MessageID
			}

			now := time.Now()

			// Clean old requests
			var recentRequests []time.Time
			for _, t := range requests[key] {
				if now.Sub(t) < config.Window {
					recentRequests = append(recentRequests, t)
				}
			}

			// Check rate limit
			if len(recentRequests) >= config.MaxRequests {
				return nil, fmt.Errorf("rate limit exceeded: max %d requests per %v",
					config.MaxRequests, config.Window)
			}

			// Add current request
			recentRequests = append(recentRequests, now)
			requests[key] = recentRequests

			return next(ctx, msg)
		}
	}
}

// Metadata creates a middleware that adds metadata to messages.
func Metadata(metadata map[string]interface{}) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Add metadata to context
			ctx = ContextWithMetadata(ctx, metadata)

			// Call next handler
			resp, err := next(ctx, msg)

			// Add metadata to response
			if err == nil && resp != nil {
				if resp.Metadata == nil {
					resp.Metadata = make(map[string]interface{})
				}
				for k, v := range metadata {
					resp.Metadata[k] = v
				}
			}

			return resp, err
		}
	}
}

// Timeout creates a middleware that adds a timeout to request processing.
func Timeout(duration time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(ctx, duration)
			defer cancel()

			// Channel to receive result
			type result struct {
				resp *types.Message
				err  error
			}
			resultChan := make(chan result, 1)

			// Execute handler in goroutine
			go func() {
				resp, err := next(ctx, msg)
				resultChan <- result{resp, err}
			}()

			// Wait for result or timeout
			select {
			case res := <-resultChan:
				return res.resp, res.err
			case <-ctx.Done():
				return nil, fmt.Errorf("request timeout after %v", duration)
			}
		}
	}
}

// ContentFilter creates a middleware that filters message content.
type ContentFilterFunc func(content string) (allowed bool, reason string)

// ContentFilter creates a content filtering middleware.
func ContentFilter(filterFunc ContentFilterFunc) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Check each text part
			for _, part := range msg.Parts {
				if textPart, ok := part.(*types.TextPart); ok {
					allowed, reason := filterFunc(textPart.Text)
					if !allowed {
						return nil, fmt.Errorf("content blocked: %s", reason)
					}
				}
			}

			return next(ctx, msg)
		}
	}
}
