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

// Package builder provides a fluent API for constructing AI agents.
//
// The builder pattern enables progressive complexity, allowing developers
// to start simple and add features as needed. This design is inspired by
// Cosmos SDK's app builder pattern.
//
// # Design Philosophy
//
// The builder follows three core principles:
//
//  1. Progressive Disclosure: Simple things are simple, complex things are possible
//  2. Sensible Defaults: Zero-config works out of the box
//  3. Validation: Catch configuration errors at build time, not runtime
//
// # Basic Usage
//
// The simplest agent requires only a name:
//
//	agent := builder.NewAgent("my-agent").MustBuild()
//	agent.Start(":8080")
//
// This creates an agent with:
//   - A2A protocol (standard agent communication)
//   - Memory storage (in-process, non-persistent)
//   - Echo handler (reflects messages back)
//   - No LLM provider (pure message routing)
//
// # Progressive Complexity
//
// Add an LLM provider for AI capabilities:
//
//	agent := builder.NewAgent("chatbot").
//	    WithLLM(llm.OpenAI()).
//	    OnMessage(func(ctx context.Context, msg *types.Message) error {
//	        response, _ := msg.LLM().Complete(ctx, msg.Text())
//	        return msg.Reply(response)
//	    }).
//	    MustBuild()
//
// Add persistent storage:
//
//	agent := builder.NewAgent("chatbot").
//	    WithLLM(llm.OpenAI()).
//	    WithStorage(storage.Redis(redisClient)).
//	    OnMessage(handleMessage).
//	    MustBuild()
//
// Use SAGE protocol for blockchain-secured communication:
//
//	agent := builder.NewAgent("secure-agent").
//	    WithProtocol(protocol.ProtocolSAGE).
//	    WithSAGEConfig(&config.SAGEConfig{
//	        DID:             "did:sage:ethereum:0x...",
//	        Network:         "ethereum",
//	        RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/...",
//	        ContractAddress: "0x...",
//	    }).
//	    WithLLM(llm.OpenAI()).
//	    OnMessage(handleMessage).
//	    MustBuild()
//
// # Lifecycle Hooks
//
// Add initialization and cleanup logic:
//
//	agent := builder.NewAgent("monitored-agent").
//	    WithLLM(llm.OpenAI()).
//	    BeforeStart(func(ctx context.Context) error {
//	        log.Println("Agent starting, warming up caches...")
//	        return warmupCaches(ctx)
//	    }).
//	    AfterStop(func(ctx context.Context) error {
//	        log.Println("Agent stopping, flushing buffers...")
//	        return flushBuffers(ctx)
//	    }).
//	    OnMessage(handleMessage).
//	    MustBuild()
//
// # Error Handling
//
// Use Build() for error handling, MustBuild() for simplicity:
//
//	// With error handling
//	agent, err := builder.NewAgent("my-agent").Build()
//	if err != nil {
//	    log.Fatalf("Failed to build agent: %v", err)
//	}
//
//	// Panic on error (simpler for examples)
//	agent := builder.NewAgent("my-agent").MustBuild()
//
// # Validation
//
// The builder validates configuration at build time:
//
//	// This will fail at Build():
//	agent, err := builder.NewAgent("").Build()  // Empty name
//	// err: "agent name cannot be empty"
//
//	// This will fail at Build():
//	agent, err := builder.NewAgent("secure").
//	    WithProtocol(protocol.ProtocolSAGE).  // SAGE mode
//	    Build()  // But no SAGEConfig!
//	// err: "SAGE mode requires SAGEConfig"
//
// # Protocol Modes
//
// Three protocol modes are available:
//
//  1. ProtocolA2A (default): Standard A2A agent-to-agent communication
//  2. ProtocolSAGE: Blockchain-secured SAGE protocol
//  3. ProtocolAuto: Auto-detect from message metadata
//
// Example with auto-detection:
//
//	agent := builder.NewAgent("hybrid").
//	    WithProtocol(protocol.ProtocolAuto).  // Auto-detect
//	    WithSAGEConfig(sageConfig).           // Optional SAGE config
//	    OnMessage(handleMessage).
//	    MustBuild()
//
//	// Agent now handles both:
//	// - A2A messages (no security metadata)
//	// - SAGE messages (with security metadata)
//
// # Storage Backends
//
// Three storage backends are available:
//
//  1. Memory (default): In-process, non-persistent
//  2. Redis: Production-ready with persistence
//  3. PostgreSQL: Enterprise-grade ACID storage
//
// Example with Redis:
//
//	redisClient := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//
//	agent := builder.NewAgent("prod-agent").
//	    WithStorage(storage.Redis(redisClient)).
//	    WithLLM(llm.OpenAI()).
//	    OnMessage(handleMessage).
//	    MustBuild()
//
// # LLM Providers
//
// Multiple LLM providers are supported:
//
//  1. OpenAI (GPT-3.5, GPT-4, GPT-4o)
//  2. Anthropic (Claude 3 Sonnet, Opus, Haiku)
//  3. Gemini (Gemini Pro, Ultra, Flash)
//
// Example with environment-based configuration:
//
//	// Uses OPENAI_API_KEY from environment
//	agent := builder.NewAgent("chatbot").
//	    WithLLM(llm.OpenAI()).
//	    OnMessage(handleMessage).
//	    MustBuild()
//
// Example with explicit configuration:
//
//	agent := builder.NewAgent("chatbot").
//	    WithLLM(llm.OpenAI(llm.Config{
//	        APIKey: "sk-...",
//	        Model:  "gpt-4",
//	    })).
//	    OnMessage(handleMessage).
//	    MustBuild()
//
// # Comparison with Cosmos SDK
//
// SAGE ADK's builder is inspired by Cosmos SDK's baseapp:
//
//	// Cosmos SDK (blockchain app)
//	app := baseapp.NewBaseApp(name, logger, db)
//	app.MountStores(keyMain, keyAccount)
//	app.SetAnteHandler(auth.NewAnteHandler(accountKeeper))
//
//	// SAGE ADK (AI agent)
//	agent := builder.NewAgent(name).
//	    WithStorage(storage.Redis(redisClient)).
//	    WithLLM(llm.OpenAI()).
//	    OnMessage(handleMessage).
//	    Build()
//
// Both provide:
//   - Fluent API for progressive complexity
//   - Sensible defaults for quick prototyping
//   - Validation at build time
//   - Modular architecture for extensibility
//
// # Integration Examples
//
// See the examples/ directory for complete working examples:
//
//   - examples/simple-chatbot/     - Basic LLM chat (5 lines)
//   - examples/mcp-agent/          - Agent with tool access (MCP)
//   - examples/secure-agent/       - SAGE-secured agent
//   - examples/orchestrator/       - Multi-agent system
//
// # Package Dependencies
//
// The builder package depends on:
//   - core/agent: Agent runtime
//   - core/protocol: Protocol selection
//   - adapters/llm: LLM providers
//   - storage: Storage backends
//   - config: Configuration types
//   - pkg/types: Core types
//   - pkg/errors: Error handling
package builder
