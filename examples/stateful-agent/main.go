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
	"syscall"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/core/state"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Global state manager
var stateManager state.Manager

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

	// Create state manager
	stateConfig := &state.Config{
		DefaultTTL:        24 * time.Hour,
		MaxMessages:       50,
		CleanupInterval:   1 * time.Hour,
		EnableAutoCleanup: true,
	}
	stateManager = state.NewMemoryManager(stateConfig)
	defer func() {
		if closer, ok := stateManager.(*state.MemoryManager); ok {
			closer.Close()
		}
	}()

	// Create A2A config
	a2aConfig := &config.A2AConfig{
		Enabled:   true,
		Version:   "0.2.2",
		ServerURL: "http://localhost:8080/",
		Timeout:   30,
	}

	// Build the agent
	chatbot, err := builder.NewAgent("stateful-agent").
		WithLLM(provider).
		WithProtocol(protocol.ProtocolA2A).
		WithA2AConfig(a2aConfig).
		OnMessage(handleMessageWithState(provider)).
		BeforeStart(func(ctx context.Context) error {
			log.Println("Stateful Agent starting...")
			log.Println("Features:")
			log.Println("  ✓ Conversation history")
			log.Println("  ✓ Session management")
			log.Println("  ✓ Context-aware responses")
			log.Println("  ✓ User preferences")
			log.Printf("  ✓ Max %d messages per session\n", stateConfig.MaxMessages)
			log.Printf("  ✓ Session TTL: %v\n", stateConfig.DefaultTTL)
			log.Println("Listening on http://localhost:8080")
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			log.Println("Stateful Agent stopped")
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

// handleMessageWithState creates a handler that maintains conversation state.
func handleMessageWithState(provider llm.Provider) agent.MessageHandler {
	return func(ctx context.Context, msg agent.MessageContext) error {
		userText := msg.Text()
		if userText == "" {
			return fmt.Errorf("empty message received")
		}

		// Extract or create session ID
		sessionID := extractSessionID(msg)
		log.Printf("📨 [Session: %s] Received: %s", sessionID, userText)

		// Get or create session state
		session, err := getOrCreateSession(ctx, sessionID)
		if err != nil {
			log.Printf("❌ Failed to get/create session: %v", err)
			return msg.Reply("Sorry, I couldn't access the session state.")
		}

		// Create user message
		userMsg := &types.Message{
			MessageID: types.GenerateMessageID(),
			Role:      types.MessageRoleUser,
			Parts: []types.Part{
				&types.TextPart{
					Kind: "text",
					Text: userText,
				},
			},
		}

		// Add to conversation history
		if err := stateManager.AddMessage(ctx, sessionID, userMsg); err != nil {
			log.Printf("⚠️  Failed to add message to history: %v", err)
		}

		// Check for special commands
		if handled, response := handleSpecialCommands(ctx, sessionID, userText); handled {
			return msg.Reply(response)
		}

		// Build conversation history for LLM
		messages, err := stateManager.GetMessages(ctx, sessionID, 10)
		if err != nil {
			log.Printf("⚠️  Failed to get message history: %v", err)
			messages = []*types.Message{userMsg}
		}

		// Convert to LLM messages
		llmMessages := make([]llm.Message, 0, len(messages)+1)

		// System message with context
		systemMsg := buildSystemMessage(ctx, sessionID, session)
		llmMessages = append(llmMessages, llm.Message{
			Role:    llm.RoleSystem,
			Content: systemMsg,
		})

		// Add conversation history
		for _, m := range messages {
			var content string
			for _, part := range m.Parts {
				if textPart, ok := part.(*types.TextPart); ok {
					content = textPart.Text
					break
				}
			}

			var role llm.MessageRole
			if m.Role == types.MessageRoleUser {
				role = llm.RoleUser
			} else {
				role = llm.RoleAssistant
			}

			llmMessages = append(llmMessages, llm.Message{
				Role:    role,
				Content: content,
			})
		}

		// Get LLM response
		request := &llm.CompletionRequest{
			Messages:    llmMessages,
			Temperature: 0.7,
		}

		response, err := provider.Complete(ctx, request)
		if err != nil {
			log.Printf("❌ LLM error: %v", err)
			return fmt.Errorf("failed to get LLM response: %w", err)
		}

		// Create assistant message
		assistantMsg := &types.Message{
			MessageID: types.GenerateMessageID(),
			Role:      types.MessageRoleAgent,
			Parts: []types.Part{
				&types.TextPart{
					Kind: "text",
					Text: response.Content,
				},
			},
		}

		// Add to conversation history
		if err := stateManager.AddMessage(ctx, sessionID, assistantMsg); err != nil {
			log.Printf("⚠️  Failed to add response to history: %v", err)
		}

		// Update session interaction count
		count, _ := stateManager.GetVariable(ctx, sessionID, "interaction_count")
		if count == nil {
			count = 0
		}
		stateManager.SetVariable(ctx, sessionID, "interaction_count", count.(int)+1)

		log.Printf("💬 [Session: %s] Response: %s", sessionID, response.Content)
		return msg.Reply(response.Content)
	}
}

// extractSessionID extracts or generates a session ID from the message context.
func extractSessionID(msg agent.MessageContext) string {
	// In a real implementation, extract from message metadata or headers
	// For demo, use a simple session ID
	return "demo-session-001"
}

// getOrCreateSession gets an existing session or creates a new one.
func getOrCreateSession(ctx context.Context, sessionID string) (*state.State, error) {
	// Try to get existing session
	session, err := stateManager.Get(ctx, sessionID)
	if err == nil {
		return session, nil
	}

	// Create new session
	if err == state.ErrStateNotFound {
		session = &state.State{
			SessionID: sessionID,
			AgentID:   "stateful-agent",
			Metadata: map[string]interface{}{
				"created": time.Now().Format(time.RFC3339),
			},
			Variables: map[string]interface{}{
				"interaction_count": 0,
			},
		}

		if err := stateManager.Create(ctx, session); err != nil {
			return nil, err
		}

		log.Printf("✨ Created new session: %s", sessionID)
		return session, nil
	}

	return nil, err
}

// buildSystemMessage builds a system message with context from the session.
func buildSystemMessage(ctx context.Context, sessionID string, session *state.State) string {
	systemMsg := "You are a helpful AI assistant with memory of the conversation. "

	// Add user name if set
	if userName, err := stateManager.GetVariable(ctx, sessionID, "user_name"); err == nil {
		systemMsg += fmt.Sprintf("The user's name is %s. ", userName)
	}

	// Add interaction count
	if count, err := stateManager.GetVariable(ctx, sessionID, "interaction_count"); err == nil {
		systemMsg += fmt.Sprintf("This is interaction #%d in this conversation. ", count.(int)+1)
	}

	// Add preferences if set
	if pref, err := stateManager.GetVariable(ctx, sessionID, "preferences"); err == nil {
		systemMsg += fmt.Sprintf("User preferences: %v. ", pref)
	}

	systemMsg += "Be conversational and remember the context of our discussion."

	return systemMsg
}

// handleSpecialCommands handles special commands like /clear, /stats, /name.
func handleSpecialCommands(ctx context.Context, sessionID string, text string) (bool, string) {
	switch text {
	case "/clear":
		if err := stateManager.Clear(ctx, sessionID); err != nil {
			return true, fmt.Sprintf("Failed to clear history: %v", err)
		}
		log.Printf("🗑️  [Session: %s] Cleared conversation history", sessionID)
		return true, "Conversation history cleared. Let's start fresh!"

	case "/stats":
		messages, _ := stateManager.GetMessages(ctx, sessionID, 0)
		count, _ := stateManager.GetVariable(ctx, sessionID, "interaction_count")
		if count == nil {
			count = 0
		}

		session, _ := stateManager.Get(ctx, sessionID)
		var created string
		if session != nil {
			created = session.CreatedAt.Format("2006-01-02 15:04:05")
		}

		stats := fmt.Sprintf("📊 Session Statistics:\n"+
			"  Session ID: %s\n"+
			"  Created: %s\n"+
			"  Messages: %d\n"+
			"  Interactions: %d",
			sessionID, created, len(messages), count)

		return true, stats

	case "/help":
		help := "Available commands:\n" +
			"  /clear - Clear conversation history\n" +
			"  /stats - Show session statistics\n" +
			"  /name <name> - Set your name\n" +
			"  /help - Show this help message"
		return true, help

	default:
		// Check for /name command
		if len(text) > 6 && text[:6] == "/name " {
			userName := text[6:]
			if err := stateManager.SetVariable(ctx, sessionID, "user_name", userName); err != nil {
				return true, fmt.Sprintf("Failed to set name: %v", err)
			}
			log.Printf("👤 [Session: %s] User name set to: %s", sessionID, userName)
			return true, fmt.Sprintf("Nice to meet you, %s! I'll remember your name.", userName)
		}
	}

	return false, ""
}
