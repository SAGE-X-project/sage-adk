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

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
)

func main() {
	// Get API key and provider from environment
	providerName := os.Getenv("LLM_PROVIDER")
	if providerName == "" {
		providerName = "openai" // Default
	}

	var provider llm.Provider
	var model string

	switch strings.ToLower(providerName) {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			log.Fatal("OPENAI_API_KEY environment variable is required")
		}
		model = os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-3.5-turbo"
		}
		provider = llm.OpenAI(&llm.OpenAIConfig{
			APIKey: apiKey,
			Model:  model,
		})

	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			log.Fatal("ANTHROPIC_API_KEY environment variable is required")
		}
		model = os.Getenv("ANTHROPIC_MODEL")
		if model == "" {
			model = "claude-3-sonnet-20240229"
		}
		provider = llm.Anthropic(&llm.AnthropicConfig{
			APIKey: apiKey,
			Model:  model,
		})

	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
		if apiKey == "" {
			log.Fatal("GEMINI_API_KEY or GOOGLE_API_KEY environment variable is required")
		}
		model = os.Getenv("GEMINI_MODEL")
		if model == "" {
			model = "gemini-pro"
		}
		provider = llm.Gemini(&llm.GeminiConfig{
			APIKey: apiKey,
			Model:  model,
		})

	default:
		log.Fatalf("Unsupported LLM provider: %s (supported: openai, anthropic, gemini)", providerName)
	}

	// Create A2A config
	a2aConfig := &config.A2AConfig{
		Enabled:   true,
		Version:   "0.2.2",
		ServerURL: "http://localhost:8080/",
		Timeout:   30,
	}

	// Build the streaming chatbot agent
	chatbot, err := builder.NewAgent("streaming-chatbot").
		WithLLM(provider).
		WithProtocol(protocol.ProtocolA2A).
		WithA2AConfig(a2aConfig).
		OnMessage(handleMessageWithStreaming(provider)).
		BeforeStart(func(ctx context.Context) error {
			log.Println("Streaming Chatbot Agent starting...")
			log.Printf("Provider: %s", providerName)
			log.Printf("Model: %s", model)
			log.Println("Listening on http://localhost:8080")
			log.Println("Responses will be streamed in real-time!")
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			log.Println("Streaming Chatbot Agent stopped")
			return nil
		}).
		Build()

	if err != nil {
		log.Fatalf("Failed to build agent: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent in goroutine
	go func() {
		if err := chatbot.Start(":8080"); err != nil {
			log.Fatalf("Failed to start agent: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\n📥 Shutdown signal received, stopping agent...")

	// Stop agent gracefully
	ctx := context.Background()
	if err := chatbot.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop agent: %v", err)
	}
}

// handleMessageWithStreaming creates a message handler that uses streaming for real-time responses.
func handleMessageWithStreaming(provider llm.Provider) agent.MessageHandler {
	return func(ctx context.Context, msg agent.MessageContext) error {
		// Get message text
		userText := msg.Text()
		if userText == "" {
			return fmt.Errorf("empty message received")
		}

		log.Printf("📨 Received message: %s", userText)

		// Create LLM request
		request := &llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: "You are a helpful AI assistant. Provide clear, concise, and engaging responses."},
				{Role: llm.RoleUser, Content: userText},
			},
			Temperature: 0.7,
		}

		// Collect response chunks for logging and final reply
		var responseBuilder strings.Builder
		chunkCount := 0

		log.Println("🔄 Streaming response...")

		// Stream the response
		err := provider.Stream(ctx, request, func(chunk string) error {
			chunkCount++
			responseBuilder.WriteString(chunk)

			// Log chunks (you could send these to a WebSocket connection instead)
			if chunkCount == 1 {
				log.Printf("💬 First chunk received: %q", chunk)
			}

			return nil
		})

		if err != nil {
			log.Printf("❌ Streaming error: %v", err)
			return fmt.Errorf("failed to stream LLM response: %w", err)
		}

		// Get full response
		fullResponse := responseBuilder.String()

		log.Printf("✅ Streaming complete - Total chunks: %d, Total length: %d characters",
			chunkCount, len(fullResponse))
		log.Printf("💬 Full response: %s", fullResponse)

		// Reply with the complete message
		// In a real streaming implementation, you would send chunks incrementally
		return msg.Reply(fullResponse)
	}
}
