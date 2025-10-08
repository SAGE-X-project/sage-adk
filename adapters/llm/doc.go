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

// Package llm provides LLM (Large Language Model) provider abstraction.
//
// This package allows AI agents to interact with multiple LLM providers
// (OpenAI, Anthropic, Gemini) through a unified interface.
//
// # Phase 1 Implementation
//
// The current implementation (Phase 1) focuses on core abstractions and
// testing infrastructure. Actual LLM provider implementations will be
// added in future phases.
//
// Implemented:
//   - Provider interface definition
//   - Request/Response types
//   - Registry for provider management
//   - Mock provider for testing
//
// Not Implemented (Future):
//   - OpenAI provider
//   - Anthropic provider
//   - Gemini provider
//   - Streaming support
//   - Advanced features (function calling, vision, etc.)
//
// # Provider Interface
//
// All LLM providers implement the Provider interface:
//
//	type Provider interface {
//	    Name() string
//	    Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
//	    Stream(ctx context.Context, req *CompletionRequest, fn StreamFunc) error
//	    SupportsStreaming() bool
//	}
//
// # Basic Usage
//
//	// Create a mock provider
//	provider := llm.NewMockProvider("test", []string{
//	    "Hello, how can I help you?",
//	    "Sure, I can help with that.",
//	})
//
//	// Create a completion request
//	req := &llm.CompletionRequest{
//	    Model: "gpt-4",
//	    Messages: []llm.Message{
//	        {Role: llm.RoleUser, Content: "Hello"},
//	    },
//	    MaxTokens: 1000,
//	    Temperature: 0.7,
//	}
//
//	// Get completion
//	resp, err := provider.Complete(context.Background(), req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(resp.Content)
//
// # Using Registry
//
//	// Create registry
//	registry := llm.NewRegistry()
//
//	// Register providers
//	registry.Register("mock", mockProvider)
//	registry.Register("openai", openaiProvider)
//
//	// Set default
//	registry.SetDefault(openaiProvider)
//
//	// Get provider by name
//	provider, err := registry.Get("openai")
//
//	// Use default provider
//	defaultProvider := registry.Default()
//
// # Integration with Agent
//
//	agent, _ := agent.NewAgent("llm-agent").
//	    OnMessage(func(ctx context.Context, msg MessageContext) error {
//	        // Get LLM provider
//	        provider := getLLMProvider()
//
//	        // Create request
//	        req := &llm.CompletionRequest{
//	            Model: "gpt-4",
//	            Messages: []llm.Message{
//	                {Role: llm.RoleUser, Content: msg.Text()},
//	            },
//	            MaxTokens: 1000,
//	        }
//
//	        // Get completion
//	        resp, err := provider.Complete(ctx, req)
//	        if err != nil {
//	            return err
//	        }
//
//	        // Reply with LLM response
//	        return msg.Reply(resp.Content)
//	    }).
//	    Build()
//
// # Message Roles
//
// Three message roles are supported:
//
//   - RoleUser: Message from the user
//   - RoleAssistant: Message from the AI assistant
//   - RoleSystem: System message (instructions)
//
// Example conversation:
//
//	messages := []llm.Message{
//	    {Role: llm.RoleSystem, Content: "You are a helpful assistant."},
//	    {Role: llm.RoleUser, Content: "What is 2+2?"},
//	    {Role: llm.RoleAssistant, Content: "2+2 equals 4."},
//	    {Role: llm.RoleUser, Content: "What about 3+3?"},
//	}
//
// # Design Principles
//
// Based on AI agent development research:
//
//   - Provider Agnostic: One interface, multiple providers
//   - Progressive Disclosure: Simple for basic use, powerful for advanced
//   - Type Safety: Strong typing for requests and responses
//   - Error Handling: Consistent error handling across providers
//
// # Mock Provider
//
// The mock provider is useful for testing:
//
//	provider := llm.NewMockProvider("test", []string{
//	    "First response",
//	    "Second response",
//	    "Third response",
//	})
//
//	// Each call returns the next response in order
//	resp1, _ := provider.Complete(ctx, req)  // "First response"
//	resp2, _ := provider.Complete(ctx, req)  // "Second response"
//	resp3, _ := provider.Complete(ctx, req)  // "Third response"
//
// # Future Enhancements
//
// Phase 2: Provider Implementations
//   - OpenAI provider (GPT-4, GPT-3.5-turbo)
//   - Anthropic provider (Claude 3.5 Sonnet, Claude 3 Opus)
//   - Gemini provider (Gemini Pro, Gemini Ultra)
//
// Phase 3: Advanced Features
//   - Streaming support
//   - Function calling
//   - Vision (multimodal)
//   - Embeddings
//   - Token counting
//   - Cost estimation
//
// Phase 4: Optimizations
//   - Response caching
//   - Retry logic with backoff
//   - Client-side rate limiting
//   - Request batching
package llm
