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

//go:build examples
// +build examples

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/middleware"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func main() {
	// Get OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create LLM provider
	provider := llm.OpenAI(&llm.OpenAIConfig{
		APIKey: apiKey,
		Model:  "gpt-3.5-turbo",
	})

	// Create custom content filter middleware
	profanityFilter := middleware.ContentFilter(func(content string) (bool, string) {
		badWords := []string{"spam", "badword", "prohibited"}
		contentLower := strings.ToLower(content)
		for _, word := range badWords {
			if strings.Contains(contentLower, word) {
				return false, fmt.Sprintf("contains prohibited word: %s", word)
			}
		}
		return true, ""
	})

	// Create custom logging middleware
	customLogger := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			log.Printf("🔵 [Custom] Processing message: %s", msg.MessageID)

			// Add custom context
			ctx = middleware.ContextWithMetadata(ctx, map[string]interface{}{
				"custom_processor": "middleware-agent",
				"timestamp":        time.Now().Unix(),
			})

			resp, err := next(ctx, msg)

			if err != nil {
				log.Printf("🔴 [Custom] Processing failed: %v", err)
			} else {
				log.Printf("🟢 [Custom] Processing completed successfully")
			}

			return resp, err
		}
	}

	// Create A2A config
	a2aConfig := &config.A2AConfig{
		Enabled:   true,
		Version:   "0.2.2",
		ServerURL: "http://localhost:8080/",
		Timeout:   30,
	}

	// Build middleware chain
	mwChain := middleware.NewChain(
		middleware.Recovery(),                          // Recover from panics
		middleware.Logger(log.Default()),               // Log requests
		customLogger,                                   // Custom logging
		middleware.RequestID(),                         // Add request ID
		middleware.Timer(),                             // Track execution time
		middleware.Validator(),                         // Validate messages
		profanityFilter,                                // Filter content
		middleware.RateLimiter(middleware.RateLimiterConfig{
			MaxRequests: 10,
			Window:      1 * time.Minute,
		}),                                             // Rate limiting
		middleware.Timeout(30 * time.Second),          // Request timeout
		middleware.Metadata(map[string]interface{}{    // Add metadata
			"service": "middleware-agent",
			"version": "1.0.0",
		}),
	)

	// Build the agent
	chatbot, err := builder.NewAgent("middleware-agent").
		WithLLM(provider).
		WithProtocol(protocol.ProtocolA2A).
		WithA2AConfig(a2aConfig).
		OnMessage(handleMessageWithMiddleware(provider, mwChain)).
		BeforeStart(func(ctx context.Context) error {
			log.Println("Middleware Agent starting...")
			log.Printf("Middleware chain has %d middleware", mwChain.Len())
			log.Println("Features enabled:")
			log.Println("  ✓ Panic recovery")
			log.Println("  ✓ Request logging")
			log.Println("  ✓ Request ID tracking")
			log.Println("  ✓ Execution timing")
			log.Println("  ✓ Message validation")
			log.Println("  ✓ Content filtering")
			log.Println("  ✓ Rate limiting (10 req/min)")
			log.Println("  ✓ Request timeout (30s)")
			log.Println("  ✓ Custom metadata")
			log.Println("Listening on http://localhost:8080")
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			log.Println("Middleware Agent stopped")
			return nil
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to build agent: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent
	go func() {
		if err := chatbot.Start(":8080"); err != nil {
			log.Fatalf("Failed to start agent: %v", err)
		}
	}()

	// Wait for shutdown
	<-sigChan
	log.Println("\n📥 Shutdown signal received, stopping agent...")

	ctx := context.Background()
	if err := chatbot.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop agent: %v", err)
	}
}

// handleMessageWithMiddleware creates a handler that uses middleware chain.
func handleMessageWithMiddleware(provider llm.Provider, mwChain *middleware.Chain) agent.MessageHandler {
	return func(ctx context.Context, msg agent.MessageContext) error {
		// Convert to types.Message
		typesMsg := &types.Message{
			MessageID: types.GenerateMessageID(),
			Role:      types.MessageRoleUser,
			Parts: []types.Part{
				&types.TextPart{
					Kind: "text",
					Text: msg.Text(),
				},
			},
		}

		// Create actual handler
		handler := func(ctx context.Context, m *types.Message) (*types.Message, error) {
			// Extract text from message
			var text string
			for _, part := range m.Parts {
				if textPart, ok := part.(*types.TextPart); ok {
					text = textPart.Text
					break
				}
			}

			// Get LLM response
			request := &llm.CompletionRequest{
				Messages: []llm.Message{
					{
						Role:    llm.RoleSystem,
						Content: "You are a helpful assistant. Respond concisely and professionally.",
					},
					{
						Role:    llm.RoleUser,
						Content: text,
					},
				},
				Temperature: 0.7,
			}

			response, err := provider.Complete(ctx, request)
			if err != nil {
				return nil, fmt.Errorf("LLM error: %w", err)
			}

			// Create response message
			return &types.Message{
				MessageID: types.GenerateMessageID(),
				Role:      types.MessageRoleAgent,
				Parts: []types.Part{
					&types.TextPart{
						Kind: "text",
						Text: response.Content,
					},
				},
			}, nil
		}

		// Execute with middleware chain
		respMsg, err := mwChain.Execute(ctx, typesMsg, handler)
		if err != nil {
			log.Printf("❌ Error: %v", err)
			return msg.Reply(fmt.Sprintf("Error: %v", err))
		}

		// Extract response text
		var responseText string
		for _, part := range respMsg.Parts {
			if textPart, ok := part.(*types.TextPart); ok {
				responseText = textPart.Text
				break
			}
		}

		// Log metadata
		if respMsg.Metadata != nil {
			if processingTime, ok := respMsg.Metadata["processing_time_ms"]; ok {
				log.Printf("⏱️  Processing time: %v ms", processingTime)
			}
		}

		return msg.Reply(responseText)
	}
}
