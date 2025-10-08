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

package agent

import (
	"context"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage-adk/storage"
)

// ServerFactory is a function that creates a server instance.
//
// This allows the builder to inject server creation without creating
// an import cycle.
type ServerFactory func(config interface{}, handler MessageHandler) (interface{ Stop(context.Context) error }, error)

// Options contains all configuration options for creating an agent.
//
// This struct is used by the builder package to construct agents
// with the fluent API.
type Options struct {
	// Name is the agent name (required)
	Name string

	// Description is a human-readable description
	Description string

	// Version is the agent version
	Version string

	// Config is the global configuration
	Config *config.Config

	// Protocol configuration
	ProtocolMode protocol.ProtocolMode
	A2AConfig    *config.A2AConfig
	SAGEConfig   *config.SAGEConfig

	// LLM provider (optional, for AI capabilities)
	LLMProvider llm.Provider

	// Storage backend (required)
	Storage storage.Storage

	// Message handler (required)
	MessageHandler MessageHandler

	// Server factory (optional, protocol-specific)
	ServerFactory ServerFactory

	// Lifecycle hooks (optional)
	BeforeStart func(context.Context) error
	AfterStop   func(context.Context) error
}

// AgentImpl represents a complete AI agent with all capabilities.
//
// This struct extends agentImpl with runtime components like
// protocol adapters, LLM provider, storage, etc.
type AgentImpl struct {
	// Core agent
	*agentImpl

	// Protocol mode
	protocolMode protocol.ProtocolMode

	// Protocol adapters
	protocolSelector protocol.ProtocolSelector

	// LLM provider
	llmProvider llm.Provider

	// Storage backend
	storage storage.Storage

	// Lifecycle hooks
	beforeStart func(context.Context) error
	afterStop   func(context.Context) error

	// Runtime state
	a2aConfig  *config.A2AConfig
	sageConfig *config.SAGEConfig
	server     Server // HTTP server (A2A or custom)
}

// NewAgentWithOptions creates a new agent from options.
//
// This is the main constructor used by the builder package.
func NewAgentWithOptions(opts *Options) (*AgentImpl, error) {
	if opts == nil {
		return nil, errors.ErrInvalidInput.WithMessage("options cannot be nil")
	}

	// Validate required fields
	if opts.Name == "" {
		return nil, errors.ErrInvalidInput.WithMessage("agent name is required")
	}
	if opts.Storage == nil {
		return nil, errors.ErrInvalidInput.WithMessage("storage is required")
	}
	if opts.MessageHandler == nil {
		return nil, errors.ErrInvalidInput.WithMessage("message handler is required")
	}

	// Create agent card
	card := &types.AgentCard{
		Name:        opts.Name,
		Description: opts.Description,
		Version:     opts.Version,
	}

	// Create core agent
	impl := &agentImpl{
		name:           opts.Name,
		description:    opts.Description,
		version:        opts.Version,
		card:           card,
		config:         opts.Config,
		messageHandler: opts.MessageHandler,
	}

	// Create protocol selector
	selector := protocol.NewSelector()
	selector.SetMode(opts.ProtocolMode)

	// TODO: Register protocol adapters based on config
	// This will be implemented when we have working A2A and SAGE transports

	// Create full agent
	agent := &AgentImpl{
		agentImpl:        impl,
		protocolMode:     opts.ProtocolMode,
		protocolSelector: selector,
		llmProvider:      opts.LLMProvider,
		storage:          opts.Storage,
		beforeStart:      opts.BeforeStart,
		afterStop:        opts.AfterStop,
		a2aConfig:        opts.A2AConfig,
		sageConfig:       opts.SAGEConfig,
	}

	return agent, nil
}

// Server is an interface for agent servers that can be started and stopped.
type Server interface {
	Start(addr string) error
	Stop(ctx context.Context) error
}

// Start starts the agent server on the specified address.
//
// This is a blocking call that runs until the agent is stopped.
//
// Example:
//
//	agent.Start(":8080")  // Listen on port 8080
func (a *AgentImpl) Start(addr string) error {
	// Run BeforeStart hook
	if a.beforeStart != nil {
		ctx := context.Background()
		if err := a.beforeStart(ctx); err != nil {
			return errors.ErrOperationFailed.
				WithMessage("beforeStart hook failed").
				WithDetail("error", err.Error())
		}
	}

	// Check server is configured
	if a.server == nil {
		return errors.ErrInvalidInput.WithMessage("server not configured - use builder to create agent with protocol support")
	}

	// Start server (blocking)
	return a.server.Start(addr)
}

// Stop gracefully stops the agent.
func (a *AgentImpl) Stop(ctx context.Context) error {
	// Stop the server if it's running
	if a.server != nil {
		if err := a.server.Stop(ctx); err != nil {
			return errors.ErrOperationFailed.
				WithMessage("failed to stop server").
				WithDetail("error", err.Error())
		}
	}

	// Run AfterStop hook
	if a.afterStop != nil {
		if err := a.afterStop(ctx); err != nil {
			return errors.ErrOperationFailed.
				WithMessage("afterStop hook failed").
				WithDetail("error", err.Error())
		}
	}

	return nil
}

// LLMProvider returns the agent's LLM provider.
//
// Returns nil if no LLM provider is configured.
func (a *AgentImpl) LLMProvider() llm.Provider {
	return a.llmProvider
}

// Storage returns the agent's storage backend.
func (a *AgentImpl) Storage() storage.Storage {
	return a.storage
}

// ProtocolMode returns the agent's protocol mode.
func (a *AgentImpl) ProtocolMode() protocol.ProtocolMode {
	return a.protocolMode
}

// SetServer sets the agent's server instance.
//
// This is called by the builder to inject the protocol-specific server.
func (a *AgentImpl) SetServer(srv Server) error {
	if srv == nil {
		return errors.ErrInvalidInput.WithMessage("server cannot be nil")
	}
	a.server = srv
	return nil
}
