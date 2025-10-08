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
	"github.com/sage-x-project/sage-adk/core/tools"
)

// Global tool registry
var toolRegistry *tools.Registry

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

	// Create and setup tool registry
	toolRegistry = tools.NewRegistry()
	if err := tools.RegisterBuiltinTools(toolRegistry); err != nil {
		log.Fatalf("Failed to register builtin tools: %v", err)
	}

	// Create A2A config
	a2aConfig := &config.A2AConfig{
		Enabled:   true,
		Version:   "0.2.2",
		ServerURL: "http://localhost:8080/",
		Timeout:   30,
	}

	// Build the agent
	chatbot, err := builder.NewAgent("tools-agent").
		WithLLM(provider).
		WithProtocol(protocol.ProtocolA2A).
		WithA2AConfig(a2aConfig).
		OnMessage(handleMessageWithTools(provider)).
		BeforeStart(func(ctx context.Context) error {
			log.Println("Tools Agent starting...")
			log.Printf("Registered %d tools:", toolRegistry.Count())
			for _, tool := range toolRegistry.List() {
				log.Printf("  - %s: %s", tool.Name(), tool.Description())
			}
			log.Println("Listening on http://localhost:8080")
			return nil
		}).
		AfterStop(func(ctx context.Context) error {
			log.Println("Tools Agent stopped")
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

// handleMessageWithTools creates a handler that can execute tools based on user requests.
func handleMessageWithTools(provider llm.Provider) agent.MessageHandler {
	return func(ctx context.Context, msg agent.MessageContext) error {
		userText := msg.Text()
		if userText == "" {
			return fmt.Errorf("empty message received")
		}

		log.Printf("📨 Received: %s", userText)

		// Check if the message is a tool invocation request
		// In a real implementation, you'd use LLM function calling
		// For this example, we parse simple commands
		if strings.HasPrefix(userText, "/tool ") {
			return handleToolCommand(ctx, msg, userText)
		}

		// Regular LLM response with tool awareness
		systemPrompt := buildSystemPrompt()

		request := &llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: llm.RoleSystem, Content: systemPrompt},
				{Role: llm.RoleUser, Content: userText},
			},
			Temperature: 0.7,
		}

		response, err := provider.Complete(ctx, request)
		if err != nil {
			log.Printf("❌ LLM error: %v", err)
			return fmt.Errorf("failed to get LLM response: %w", err)
		}

		log.Printf("💬 Response: %s", response.Content)
		return msg.Reply(response.Content)
	}
}

// handleToolCommand handles direct tool invocation commands.
// Format: /tool <tool_name> <json_params>
func handleToolCommand(ctx context.Context, msg agent.MessageContext, command string) error {
	parts := strings.SplitN(command, " ", 3)
	if len(parts) < 2 {
		return msg.Reply("Usage: /tool <tool_name> [json_params]")
	}

	toolName := parts[1]

	// Check if tool exists
	if !toolRegistry.Has(toolName) {
		availableTools := make([]string, 0)
		for _, t := range toolRegistry.List() {
			availableTools = append(availableTools, t.Name())
		}
		return msg.Reply(fmt.Sprintf("Tool '%s' not found. Available tools: %s",
			toolName, strings.Join(availableTools, ", ")))
	}

	// Parse parameters
	params := make(map[string]interface{})
	if len(parts) == 3 {
		// Simple key=value parsing for demo
		// In production, use JSON parsing
		paramStr := parts[2]
		for _, pair := range strings.Split(paramStr, ",") {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) == 2 {
				params[kv[0]] = kv[1]
			}
		}
	}

	log.Printf("🔧 Executing tool: %s with params: %v", toolName, params)

	// Execute tool
	result, err := toolRegistry.Execute(ctx, toolName, params)
	if err != nil {
		log.Printf("❌ Tool execution error: %v", err)
		return msg.Reply(fmt.Sprintf("Tool execution failed: %v", err))
	}

	// Format response
	var response string
	if result.Success {
		response = fmt.Sprintf("✅ Tool '%s' executed successfully\nOutput: %v", toolName, result.Output)
		if result.Metadata != nil && len(result.Metadata) > 0 {
			response += fmt.Sprintf("\nMetadata: %v", result.Metadata)
		}
	} else {
		response = fmt.Sprintf("❌ Tool '%s' failed\nError: %s", toolName, result.Error)
	}

	log.Printf("📤 Tool result: %s", response)
	return msg.Reply(response)
}

// buildSystemPrompt creates a system prompt that describes available tools.
func buildSystemPrompt() string {
	var prompt strings.Builder

	prompt.WriteString("You are a helpful AI assistant with access to the following tools:\n\n")

	for _, tool := range toolRegistry.List() {
		prompt.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	prompt.WriteString("\nWhen users ask you to perform calculations or check the time, ")
	prompt.WriteString("let them know they can use the tools directly with commands like:\n")
	prompt.WriteString("/tool calculator operation=add,a=5,b=3\n")
	prompt.WriteString("/tool current_time format=Human\n")

	return prompt.String()
}
