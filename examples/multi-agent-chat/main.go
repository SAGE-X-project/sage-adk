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

/*
Multi-Agent Chat Example

This example demonstrates a multi-agent chat system where multiple AI agents
collaborate to answer user questions. The system includes:

- Coordinator Agent: Routes questions to appropriate specialist agents
- Math Agent: Handles mathematical questions
- Code Agent: Answers programming questions
- General Agent: Handles general knowledge questions

Architecture:
┌──────────┐      ┌─────────────┐      ┌──────────────┐
│  Client  │─────>│ Coordinator │─────>│ Specialist   │
└──────────┘      │   Agent     │      │   Agents     │
                  └─────────────┘      └──────────────┘
                         │                    │
                         └────────────────────┘
                          Collaboration Flow
*/
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

	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/client"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage-adk/storage"
)

// Agent endpoints
const (
	coordinatorPort = ":8090"
	mathPort        = ":8091"
	codePort        = ":8092"
	generalPort     = ":8093"
)

func main() {
	ctx := context.Background()

	log.Println("🚀 Starting Multi-Agent Chat System...")

	// Create shared storage for agent communication
	store := storage.NewMemoryStorage()

	// Start all agents
	coordinator := startCoordinatorAgent(ctx, store)
	mathAgent := startMathAgent(ctx)
	codeAgent := startCodeAgent(ctx)
	generalAgent := startGeneralAgent(ctx)

	// Wait for all agents to start
	time.Sleep(2 * time.Second)

	log.Println("✅ All agents started successfully")
	log.Println("\n📋 Available Agents:")
	log.Println("  - Coordinator (port 8090): Routes questions to specialists")
	log.Println("  - Math Agent (port 8091): Answers mathematical questions")
	log.Println("  - Code Agent (port 8092): Answers programming questions")
	log.Println("  - General Agent (port 8093): Handles general knowledge")

	// Run interactive demo
	if len(os.Args) > 1 && os.Args[1] == "--demo" {
		runDemo(ctx)
	} else {
		log.Println("\n💡 Tip: Run with --demo flag to see an interactive demonstration")
		log.Println("   Press Ctrl+C to stop all agents")
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 Shutting down agents...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	coordinator.Stop(shutdownCtx)
	mathAgent.Stop(shutdownCtx)
	codeAgent.Stop(shutdownCtx)
	generalAgent.Stop(shutdownCtx)

	log.Println("✅ All agents stopped")
}

// startCoordinatorAgent starts the coordinator that routes questions
func startCoordinatorAgent(ctx context.Context, store storage.Storage) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		question := msg.Text()
		log.Printf("📨 [Coordinator] Received: %s", question)

		// Simple routing logic based on keywords
		var targetURL string
		var agentName string

		questionLower := strings.ToLower(question)
		switch {
		case strings.Contains(questionLower, "math") ||
			strings.Contains(questionLower, "calculate") ||
			strings.Contains(questionLower, "number"):
			targetURL = "http://localhost" + mathPort
			agentName = "Math Agent"

		case strings.Contains(questionLower, "code") ||
			strings.Contains(questionLower, "program") ||
			strings.Contains(questionLower, "function"):
			targetURL = "http://localhost" + codePort
			agentName = "Code Agent"

		default:
			targetURL = "http://localhost" + generalPort
			agentName = "General Agent"
		}

		log.Printf("🔀 [Coordinator] Routing to %s at %s", agentName, targetURL)

		// Forward to specialist agent
		c, err := client.NewClient(targetURL, client.WithProtocol(protocol.ProtocolA2A))
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", agentName, err)
		}
		defer c.Close()

		forwardMsg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart(question),
		})

		response, err := c.SendMessage(ctx, forwardMsg)
		if err != nil {
			return fmt.Errorf("failed to get response from %s: %w", agentName, err)
		}

		// Extract text from response
		var responseText string
		for _, part := range response.Parts {
			if textPart, ok := part.(*types.TextPart); ok {
				responseText = textPart.Text
				break
			}
		}

		// Send back to user
		finalResponse := fmt.Sprintf("[Routed to %s]\n%s", agentName, responseText)
		return msg.Reply(finalResponse)
	}

	b := builder.NewAgent("coordinator").
		WithDescription("Routes questions to specialist agents").
		WithVersion("1.0.0").
		WithStorage(store).
		OnMessage(handler)

	agentInstance, err := b.Build()
	if err != nil {
		log.Fatalf("Failed to create coordinator: %v", err)
	}

	impl := agentInstance.(*agent.AgentImpl)
	go impl.Start(coordinatorPort)

	return impl
}

// startMathAgent starts the math specialist agent
func startMathAgent(ctx context.Context) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		question := msg.Text()
		log.Printf("🔢 [Math Agent] Processing: %s", question)

		// Simulate math processing
		response := fmt.Sprintf("Math analysis: I can help with mathematical problems!\n"+
			"Your question: '%s'\n"+
			"(In a real implementation, this would use an LLM or math engine)", question)

		return msg.Reply(response)
	}

	return createSpecialistAgent("math-agent", "Mathematics specialist", handler, mathPort)
}

// startCodeAgent starts the coding specialist agent
func startCodeAgent(ctx context.Context) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		question := msg.Text()
		log.Printf("💻 [Code Agent] Processing: %s", question)

		response := fmt.Sprintf("Code analysis: I can help with programming questions!\n"+
			"Your question: '%s'\n"+
			"(In a real implementation, this would use an LLM trained on code)", question)

		return msg.Reply(response)
	}

	return createSpecialistAgent("code-agent", "Programming specialist", handler, codePort)
}

// startGeneralAgent starts the general knowledge agent
func startGeneralAgent(ctx context.Context) *agent.AgentImpl {
	handler := func(ctx context.Context, msg agent.MessageContext) error {
		question := msg.Text()
		log.Printf("🌍 [General Agent] Processing: %s", question)

		response := fmt.Sprintf("General knowledge: I can help with various topics!\n"+
			"Your question: '%s'\n"+
			"(In a real implementation, this would use a general-purpose LLM)", question)

		return msg.Reply(response)
	}

	return createSpecialistAgent("general-agent", "General knowledge specialist", handler, generalPort)
}

// createSpecialistAgent is a helper to create specialist agents
func createSpecialistAgent(name, description string, handler agent.MessageHandler, port string) *agent.AgentImpl {
	b := builder.NewAgent(name).
		WithDescription(description).
		WithVersion("1.0.0").
		OnMessage(handler)

	agentInstance, err := b.Build()
	if err != nil {
		log.Fatalf("Failed to create %s: %v", name, err)
	}

	impl := agentInstance.(*agent.AgentImpl)
	go impl.Start(port)

	return impl
}

// runDemo runs an interactive demonstration
func runDemo(ctx context.Context) {
	log.Println("\n🎬 Starting interactive demo...")
	time.Sleep(1 * time.Second)

	// Create client to coordinator
	c, err := client.NewClient(
		"http://localhost"+coordinatorPort,
		client.WithProtocol(protocol.ProtocolA2A),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	// Test questions
	questions := []string{
		"What is 123 times 456?",
		"How do I write a function in Go?",
		"What is the capital of France?",
		"Calculate the square root of 144",
		"Explain how to use pointers in C",
		"Who wrote Hamlet?",
	}

	for i, question := range questions {
		log.Printf("\n📝 Question %d: %s", i+1, question)

		msg := types.NewMessage(types.MessageRoleUser, []types.Part{
			types.NewTextPart(question),
		})

		response, err := c.SendMessage(ctx, msg)
		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		// Extract response text
		for _, part := range response.Parts {
			if textPart, ok := part.(*types.TextPart); ok {
				log.Printf("💬 Response:\n%s\n", textPart.Text)
			}
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("\n✅ Demo completed!")
	log.Println("   Agents will continue running. Press Ctrl+C to stop.")
}
