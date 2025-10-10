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
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// BenchmarkMiddleware_Single benchmarks a single middleware
func BenchmarkMiddleware_Single(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	middleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Simple middleware that does nothing
			return next(ctx, msg)
		}
	}

	chain := NewChain(middleware)
	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chainedHandler := chain.Then(handler)
		if _, err := chainedHandler(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMiddleware_Chain benchmarks multiple middlewares in a chain
func BenchmarkMiddleware_Chain(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	// Create a simple middleware
	simpleMiddleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			return next(ctx, msg)
		}
	}

	chainLengths := []int{1, 5, 10, 20}
	for _, length := range chainLengths {
		b.Run(fmt.Sprintf("chain-%d", length), func(b *testing.B) {
			middlewares := make([]Middleware, length)
			for i := 0; i < length; i++ {
				middlewares[i] = simpleMiddleware
			}

			chain := NewChain(middlewares...)
			msg := &types.Message{
				MessageID: "test-id",
				Role:      types.MessageRoleUser,
				Parts: []types.Part{
					&types.TextPart{
						Kind: "text",
						Text: "test",
					},
				},
			}
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				chainedHandler := chain.Then(handler)
				if _, err := chainedHandler(ctx, msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMiddleware_Logging benchmarks logging middleware
func BenchmarkMiddleware_Logging(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	loggingMiddleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Simulate logging (but don't actually log to avoid I/O in benchmark)
			_ = msg.MessageID
			_ = msg.Parts
			return next(ctx, msg)
		}
	}

	chain := NewChain(loggingMiddleware)
	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test message content",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chainedHandler := chain.Then(handler)
		if _, err := chainedHandler(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMiddleware_Validation benchmarks validation middleware
func BenchmarkMiddleware_Validation(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	validationMiddleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Simple validation
			if msg == nil {
				return nil, fmt.Errorf("message is nil")
			}
			if msg.MessageID == "" {
				return nil, fmt.Errorf("message ID is empty")
			}
			return next(ctx, msg)
		}
	}

	chain := NewChain(validationMiddleware)
	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chainedHandler := chain.Then(handler)
		if _, err := chainedHandler(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMiddleware_Recovery benchmarks recovery middleware
func BenchmarkMiddleware_Recovery(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	recoveryMiddleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (result *types.Message, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			return next(ctx, msg)
		}
	}

	chain := NewChain(recoveryMiddleware)
	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chainedHandler := chain.Then(handler)
		if _, err := chainedHandler(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMiddleware_Context benchmarks context manipulation
func BenchmarkMiddleware_Context(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	contextMiddleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			// Add values to context
			ctx = context.WithValue(ctx, "key1", "value1")
			ctx = context.WithValue(ctx, "key2", "value2")
			ctx = context.WithValue(ctx, "key3", "value3")
			return next(ctx, msg)
		}
	}

	chain := NewChain(contextMiddleware)
	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chainedHandler := chain.Then(handler)
		if _, err := chainedHandler(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMiddleware_Parallel benchmarks parallel execution
func BenchmarkMiddleware_Parallel(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	middleware := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			return next(ctx, msg)
		}
	}

	chain := NewChain(middleware)
	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test",
			},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			chainedHandler := chain.Then(handler)
			if _, err := chainedHandler(ctx, msg); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMiddleware_ComplexChain benchmarks a realistic middleware chain
func BenchmarkMiddleware_ComplexChain(b *testing.B) {
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	}

	// Logging middleware
	loggingMW := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			_ = msg.MessageID
			return next(ctx, msg)
		}
	}

	// Validation middleware
	validationMW := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			if msg == nil || msg.MessageID == "" {
				return nil, fmt.Errorf("invalid message")
			}
			return next(ctx, msg)
		}
	}

	// Recovery middleware
	recoveryMW := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (result *types.Message, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
				}
			}()
			return next(ctx, msg)
		}
	}

	// Context middleware
	contextMW := func(next Handler) Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			ctx = context.WithValue(ctx, "request_id", msg.MessageID)
			return next(ctx, msg)
		}
	}

	chain := NewChain(loggingMW, validationMW, recoveryMW, contextMW)
	msg := &types.Message{
		MessageID: "test-id-123",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test message",
			},
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chainedHandler := chain.Then(handler)
		if _, err := chainedHandler(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}
