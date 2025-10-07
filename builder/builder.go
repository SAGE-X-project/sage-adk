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

package builder

import (
	"context"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/storage"
)

// Builder provides a fluent API for constructing AI agents.
//
// The builder pattern allows for progressive complexity:
//   - Simple: agent := NewAgent("my-agent").Build()
//   - Medium: agent := NewAgent("my-agent").WithLLM(llm.OpenAI()).Build()
//   - Advanced: Full configuration with all options
//
// Inspired by Cosmos SDK's app builder pattern.
type Builder struct {
	// Agent configuration
	name   string
	config *config.Config

	// Protocol configuration
	protocolMode protocol.ProtocolMode
	a2aConfig    *config.A2AConfig
	sageConfig   *config.SAGEConfig

	// LLM provider
	llmProvider llm.Provider

	// Storage backend
	storageBackend storage.Storage

	// Message handler
	messageHandler agent.MessageHandler

	// Lifecycle hooks
	beforeStart func(context.Context) error
	afterStop   func(context.Context) error

	// Validation state
	validated bool
	errors    []error
}

// NewAgent creates a new agent builder with the given name.
//
// This is the entry point for building an agent using the fluent API.
//
// Example:
//
//	agent := NewAgent("chatbot").
//	    WithLLM(llm.OpenAI()).
//	    Build()
func NewAgent(name string) *Builder {
	return &Builder{
		name:         name,
		config:       config.NewConfig(),
		protocolMode: protocol.ProtocolA2A, // Default to A2A
	}
}

// WithLLM sets the LLM provider for the agent.
//
// Example:
//
//	builder.WithLLM(llm.OpenAI())
//	builder.WithLLM(llm.Anthropic())
//	builder.WithLLM(llm.Gemini())
func (b *Builder) WithLLM(provider llm.Provider) *Builder {
	b.llmProvider = provider
	return b
}

// WithProtocol sets the protocol mode for agent communication.
//
// Available modes:
//   - ProtocolA2A: Standard A2A protocol (default)
//   - ProtocolSAGE: Blockchain-secured SAGE protocol
//   - ProtocolAuto: Auto-detect from message metadata
//
// Example:
//
//	builder.WithProtocol(protocol.ProtocolA2A)
//	builder.WithProtocol(protocol.ProtocolSAGE)
//	builder.WithProtocol(protocol.ProtocolAuto)
func (b *Builder) WithProtocol(mode protocol.ProtocolMode) *Builder {
	b.protocolMode = mode
	return b
}

// WithA2AConfig sets custom A2A protocol configuration.
//
// If not called, sensible defaults are used.
//
// Example:
//
//	builder.WithA2AConfig(&config.A2AConfig{
//	    ServerURL: "http://agent-b:8080",
//	    Timeout:   30,
//	})
func (b *Builder) WithA2AConfig(cfg *config.A2AConfig) *Builder {
	b.a2aConfig = cfg
	return b
}

// WithSAGEConfig sets custom SAGE protocol configuration.
//
// Required when using ProtocolSAGE mode.
//
// Example:
//
//	builder.WithSAGEConfig(&config.SAGEConfig{
//	    DID:             "did:sage:ethereum:0x...",
//	    Network:         "ethereum",
//	    RPCEndpoint:     "https://eth-mainnet.g.alchemy.com/v2/...",
//	    ContractAddress: "0x...",
//	})
func (b *Builder) WithSAGEConfig(cfg *config.SAGEConfig) *Builder {
	b.sageConfig = cfg
	return b
}

// WithStorage sets the storage backend for the agent.
//
// Available backends:
//   - storage.Memory() - In-memory storage (default)
//   - storage.Redis() - Redis storage
//   - storage.Postgres() - PostgreSQL storage
//
// Example:
//
//	builder.WithStorage(storage.Memory())
//	builder.WithStorage(storage.Redis(redisClient))
func (b *Builder) WithStorage(backend storage.Storage) *Builder {
	b.storageBackend = backend
	return b
}

// OnMessage sets the message handler for the agent.
//
// This is the core business logic that processes incoming messages.
//
// Example:
//
//	builder.OnMessage(func(ctx context.Context, msg *types.Message) error {
//	    // Process message
//	    return msg.Reply("Hello!")
//	})
func (b *Builder) OnMessage(handler agent.MessageHandler) *Builder {
	b.messageHandler = handler
	return b
}

// BeforeStart sets a hook that runs before the agent starts.
//
// Useful for initialization tasks like warming up caches,
// establishing connections, etc.
//
// Example:
//
//	builder.BeforeStart(func(ctx context.Context) error {
//	    log.Println("Agent starting...")
//	    return nil
//	})
func (b *Builder) BeforeStart(hook func(context.Context) error) *Builder {
	b.beforeStart = hook
	return b
}

// AfterStop sets a hook that runs after the agent stops.
//
// Useful for cleanup tasks like closing connections,
// flushing buffers, etc.
//
// Example:
//
//	builder.AfterStop(func(ctx context.Context) error {
//	    log.Println("Agent stopped")
//	    return nil
//	})
func (b *Builder) AfterStop(hook func(context.Context) error) *Builder {
	b.afterStop = hook
	return b
}

// WithConfig sets custom global configuration.
//
// This overrides all default configuration.
//
// Example:
//
//	builder.WithConfig(customConfig)
func (b *Builder) WithConfig(cfg *config.Config) *Builder {
	b.config = cfg
	return b
}

// Build constructs the agent with the configured options.
//
// This method validates the configuration and returns an error
// if the configuration is invalid.
//
// Example:
//
//	agent, err := builder.Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
func (b *Builder) Build() (*agent.AgentImpl, error) {
	// Apply defaults
	if err := b.applyDefaults(); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := b.validate(); err != nil {
		return nil, err
	}

	// Build agent
	return b.buildAgent()
}

// MustBuild is like Build but panics on error.
//
// Useful for simple examples where error handling is verbose.
//
// Example:
//
//	agent := NewAgent("chatbot").WithLLM(llm.OpenAI()).MustBuild()
func (b *Builder) MustBuild() *agent.AgentImpl {
	ag, err := b.Build()
	if err != nil {
		panic(err)
	}
	return ag
}

// applyDefaults applies sensible defaults for unset options.
func (b *Builder) applyDefaults() error {
	// Default storage: Memory
	if b.storageBackend == nil {
		b.storageBackend = storage.NewMemoryStorage()
	}

	// Default A2A config
	if b.a2aConfig == nil {
		b.a2aConfig = &config.A2AConfig{
			Enabled:   true,
			Version:   "0.2.2",
			ServerURL: "", // Will be set by runtime
			Timeout:   30,
		}
	}

	// Default SAGE config (disabled unless explicitly set)
	if b.sageConfig == nil && b.protocolMode == protocol.ProtocolSAGE {
		return errors.ErrInvalidInput.WithMessage("SAGE mode requires SAGEConfig")
	}

	// Default message handler (echo)
	if b.messageHandler == nil {
		b.messageHandler = func(ctx context.Context, msg agent.MessageContext) error {
			// Default: Echo back the message
			return nil
		}
	}

	return nil
}

// validate checks that the builder configuration is valid.
func (b *Builder) validate() error {
	if b.validated {
		return nil
	}

	v := &validator{builder: b}
	v.validateName()
	v.validateProtocol()
	v.validateLLM()
	v.validateStorage()
	v.validateHandler()

	b.validated = true

	if len(v.errors) > 0 {
		return errors.ErrInvalidInput.WithMessage("builder validation failed").
			WithDetail("errors", v.errors)
	}

	return nil
}

// buildAgent constructs the actual agent instance.
func (b *Builder) buildAgent() (*agent.AgentImpl, error) {
	// Create agent options
	opts := &agent.Options{
		Name:           b.name,
		Config:         b.config,
		ProtocolMode:   b.protocolMode,
		A2AConfig:      b.a2aConfig,
		SAGEConfig:     b.sageConfig,
		LLMProvider:    b.llmProvider,
		Storage:        b.storageBackend,
		MessageHandler: b.messageHandler,
		BeforeStart:    b.beforeStart,
		AfterStop:      b.afterStop,
	}

	// Create agent using agent package
	return agent.NewAgentWithOptions(opts)
}
