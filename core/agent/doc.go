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

// Package agent provides the core agent abstraction for SAGE ADK.
//
// The agent package implements a fluent builder API for creating AI agents
// with minimal boilerplate while supporting advanced features like protocol
// switching, LLM integration, state management, and middleware.
//
// # Quick Start
//
// Simple echo agent (5 lines):
//
//	agent.NewAgent("echo").
//	    OnMessage(func(ctx context.Context, msg MessageContext) error {
//	        return msg.Reply(msg.Text())
//	    }).
//	    Start(":8080")
//
// LLM chat agent (10 lines):
//
//	agent.NewAgent("chat").
//	    WithLLM(llm.OpenAI()).
//	    OnMessage(func(ctx context.Context, msg MessageContext) error {
//	        response, _ := msg.LLM().Generate(ctx, msg.Text())
//	        return msg.Reply(response)
//	    }).
//	    Start(":8080")
//
// # Architecture
//
// The agent package uses a builder pattern for configuration:
//
//	builder := agent.NewAgent("my-agent")        // Create builder
//	builder.WithLLM(llm.OpenAI())                // Configure LLM
//	builder.OnMessage(handleMessage)             // Set handler
//	agent, err := builder.Build()                // Validate and build
//
// # Message Handling
//
// Messages are processed through a MessageContext that provides:
//   - Content access: Text(), Parts(), ContextID()
//   - Response helpers: Reply(), ReplyWithParts(), Stream()
//   - LLM access: LLM()
//   - State access: State(), History()
//   - Tool access: CallTool()
//
// Example handler:
//
//	func handleMessage(ctx context.Context, msg agent.MessageContext) error {
//	    // Get message text
//	    text := msg.Text()
//
//	    // Access state
//	    state := msg.State()
//	    count, _ := state.Get("count")
//
//	    // Generate response
//	    response, err := msg.LLM().Generate(ctx, text)
//	    if err != nil {
//	        return err
//	    }
//
//	    // Send reply
//	    return msg.Reply(response)
//	}
//
// # Protocol Support
//
// The agent automatically detects and handles both A2A and SAGE protocols:
//
//	agent.NewAgent("hybrid").
//	    WithProtocol(agent.ProtocolAuto).  // Auto-detect
//	    WithSAGE(sage.Optional()).         // Enable SAGE if present
//	    Build()
//
// # State Management
//
// Agents support conversation state and history:
//
//	func handleStateful(ctx context.Context, msg agent.MessageContext) error {
//	    // Access conversation history
//	    history := msg.History()
//
//	    // Access session state
//	    state := msg.State()
//	    userPref, _ := state.Get("preference")
//
//	    // Use in LLM call
//	    response, _ := msg.LLM().GenerateWithHistory(ctx, msg.Text(), history)
//	    return msg.Reply(response)
//	}
//
// # Middleware
//
// Agents support middleware for cross-cutting concerns:
//
//	agent.NewAgent("production").
//	    Use(
//	        middleware.Logging(logger),
//	        middleware.RateLimit(100, time.Minute),
//	        middleware.Authentication(verify),
//	    ).
//	    Build()
//
// # Resilience
//
// Built-in retry and circuit breaker support:
//
//	agent.NewAgent("resilient").
//	    WithRetry(retry.Exponential(3, time.Second)).
//	    WithTimeout(30 * time.Second).
//	    WithCircuitBreaker(breaker.Default()).
//	    Build()
//
// # Testing
//
// Agents are designed for easy testing with mock implementations:
//
//	func TestAgent(t *testing.T) {
//	    mockLLM := &MockLLM{response: "test"}
//
//	    agent := agent.NewAgent("test").
//	        WithLLM(mockLLM).
//	        OnMessage(handler).
//	        Build()
//
//	    msg := types.NewMessage(...)
//	    response, err := agent.Process(context.Background(), msg)
//	    assert.NoError(t, err)
//	}
package agent
