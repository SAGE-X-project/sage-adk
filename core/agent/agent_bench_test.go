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

package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// BenchmarkAgent_ProcessMessage benchmarks basic message processing
func BenchmarkAgent_ProcessMessage(b *testing.B) {
	handler := func(ctx context.Context, msg MessageContext) error {
		return msg.Reply("response")
	}

	agent := &agentImpl{
		name:           "bench-agent",
		description:    "Benchmark agent",
		messageHandler: handler,
	}

	msg := &types.Message{
		MessageID: "test-id",
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
		if _, err := agent.Process(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAgent_ProcessMessage_Parallel benchmarks parallel message processing
func BenchmarkAgent_ProcessMessage_Parallel(b *testing.B) {
	handler := func(ctx context.Context, msg MessageContext) error {
		return msg.Reply("response")
	}

	agent := &agentImpl{
		name:           "bench-agent",
		description:    "Benchmark agent",
		messageHandler: handler,
	}

	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test message",
			},
		},
	}

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			if _, err := agent.Process(ctx, msg); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkAgent_ProcessMessage_VaryingSize benchmarks messages of different sizes
func BenchmarkAgent_ProcessMessage_VaryingSize(b *testing.B) {
	handler := func(ctx context.Context, msg MessageContext) error {
		return msg.Reply("response")
	}

	agent := &agentImpl{
		name:           "bench-agent",
		description:    "Benchmark agent",
		messageHandler: handler,
	}

	sizes := []int{10, 100, 1000, 10000} // bytes
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			content := string(make([]byte, size))
			msg := &types.Message{
				MessageID: "test-id",
				Role:      types.MessageRoleUser,
				Parts: []types.Part{
					&types.TextPart{
						Kind: "text",
						Text: content,
					},
				},
			}
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := agent.Process(ctx, msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkAgent_ProcessMessage_WithMetadata benchmarks messages with metadata
func BenchmarkAgent_ProcessMessage_WithMetadata(b *testing.B) {
	handler := func(ctx context.Context, msg MessageContext) error {
		// Access message ID and context ID
		_ = msg.MessageID()
		_ = msg.ContextID()
		return msg.Reply("response")
	}

	agent := &agentImpl{
		name:           "bench-agent",
		description:    "Benchmark agent",
		messageHandler: handler,
	}

	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test message",
			},
		},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := agent.Process(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMessageContext_Operations benchmarks MessageContext operations
func BenchmarkMessageContext_Operations(b *testing.B) {
	b.Run("Text", func(b *testing.B) {
		mc := &messageContext{
			message: &types.Message{
				MessageID: "test-id",
				Role:      types.MessageRoleUser,
				Parts: []types.Part{
					&types.TextPart{
						Kind: "text",
						Text: "test message content",
					},
				},
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mc.Text()
		}
	})

	b.Run("MessageID", func(b *testing.B) {
		mc := &messageContext{
			message: &types.Message{
				MessageID: "test-id",
				Role:      types.MessageRoleUser,
				Parts:     []types.Part{},
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mc.MessageID()
		}
	})

	b.Run("Parts", func(b *testing.B) {
		mc := &messageContext{
			message: &types.Message{
				MessageID: "test-id",
				Role:      types.MessageRoleUser,
				Parts: []types.Part{
					&types.TextPart{
						Kind: "text",
						Text: "test",
					},
				},
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mc.Parts()
		}
	})

	b.Run("ContextID", func(b *testing.B) {
		contextID := "context-123"
		mc := &messageContext{
			message: &types.Message{
				MessageID: "test-id",
				Role:      types.MessageRoleUser,
				ContextID: &contextID,
				Parts:     []types.Part{},
			},
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = mc.ContextID()
		}
	})
}

// BenchmarkAgent_MessageHandling_ErrorPath benchmarks error handling
func BenchmarkAgent_MessageHandling_ErrorPath(b *testing.B) {
	handler := func(ctx context.Context, msg MessageContext) error {
		return fmt.Errorf("simulated error")
	}

	agent := &agentImpl{
		name:           "bench-agent",
		description:    "Benchmark agent",
		messageHandler: handler,
	}

	msg := &types.Message{
		MessageID: "test-id",
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
		_, err := agent.Process(ctx, msg)
		if err == nil {
			b.Fatal("expected error")
		}
	}
}

// BenchmarkAgent_ContextPropagation benchmarks context value propagation
func BenchmarkAgent_ContextPropagation(b *testing.B) {
	handler := func(ctx context.Context, msg MessageContext) error {
		// Access context values
		_ = ctx.Value("key1")
		_ = ctx.Value("key2")
		_ = ctx.Value("key3")
		return msg.Reply("response")
	}

	agent := &agentImpl{
		name:           "bench-agent",
		description:    "Benchmark agent",
		messageHandler: handler,
	}

	msg := &types.Message{
		MessageID: "test-id",
		Role:      types.MessageRoleUser,
		Parts: []types.Part{
			&types.TextPart{
				Kind: "text",
				Text: "test message",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		ctx = context.WithValue(ctx, "key1", "value1")
		ctx = context.WithValue(ctx, "key2", "value2")
		ctx = context.WithValue(ctx, "key3", "value3")

		if _, err := agent.Process(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}
