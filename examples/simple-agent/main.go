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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
)

func main() {
	// Get OpenAI API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create OpenAI provider
	provider := llm.OpenAI(&llm.OpenAIConfig{
		APIKey: apiKey,
		Model:  "gpt-3.5-turbo",
	})

	// Create A2A config with server URL
	a2aConfig := &config.A2AConfig{
		Enabled:   true,
		Version:   "0.2.2",
		ServerURL: "http://localhost:8080/",
		Timeout:   30,
	}

	// Build the chatbot agent
	chatbot, err := builder.NewAgent("simple-chatbot").
		WithLLM(provider).
		WithProtocol(protocol.ProtocolA2A).
		WithA2AConfig(a2aConfig).
		OnMessage(handleMessage(provider)).
		BeforeStart(func(ctx context.Context) error {
			log.Println("🤖 Simple Chatbot Agent starting...")
			log.Println("📡 Listening on http://localhost:8080")
			log.Println("💬 Ready to receive messages via A2A protocol")
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			log.Println("👋 Simple Chatbot Agent stopped")
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

// handleMessage creates a message handler that uses LLM to generate responses
func handleMessage(provider llm.Provider) agent.MessageHandler {
	return func(ctx context.Context, msg agent.MessageContext) error {
		// Get message text
		userText := msg.Text()
		if userText == "" {
			return fmt.Errorf("empty message received")
		}

		log.Printf("📨 Received message: %s", userText)

		// Create LLM completion request
		request := &llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: "You are a helpful AI assistant. Provide concise and friendly responses."},
				{Role: llm.RoleUser, Content: userText},
			},
			Temperature: 0.7,
		}

		// Get response from LLM
		response, err := provider.Complete(ctx, request)
		if err != nil {
			log.Printf("❌ LLM error: %v", err)
			return fmt.Errorf("failed to get LLM response: %w", err)
		}

		// Extract response text
		responseText := response.Content
		log.Printf("💬 Response: %s", responseText)

		// Reply to the message
		return msg.Reply(responseText)
	}
}
