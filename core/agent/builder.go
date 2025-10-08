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
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// builder implements the Builder interface.
type builder struct {
	agent *agentImpl
}

// NewAgent creates a new agent builder with the specified name.
func NewAgent(name string) Builder {
	return &builder{
		agent: &agentImpl{
			name:    name,
			version: "0.1.0",
			config:  config.DefaultConfig(),
		},
	}
}

// WithName sets the agent name.
func (b *builder) WithName(name string) Builder {
	b.agent.name = name
	return b
}

// WithDescription sets the agent description.
func (b *builder) WithDescription(desc string) Builder {
	b.agent.description = desc
	return b
}

// WithVersion sets the agent version.
func (b *builder) WithVersion(version string) Builder {
	b.agent.version = version
	return b
}

// OnMessage sets the message handler.
func (b *builder) OnMessage(handler MessageHandler) Builder {
	b.agent.messageHandler = handler
	return b
}

// Build validates configuration and constructs the agent.
func (b *builder) Build() (Agent, error) {
	// Validation
	if b.agent.name == "" {
		return nil, errors.ErrMissingField.WithDetail("field", "name")
	}

	if b.agent.messageHandler == nil {
		return nil, errors.ErrMissingField.WithDetail("field", "message_handler")
	}

	// Update config with agent info
	b.agent.config.Agent.Name = b.agent.name
	b.agent.config.Agent.Description = b.agent.description
	b.agent.config.Agent.Version = b.agent.version

	// Generate agent card
	b.agent.card = types.NewAgentCard(
		b.agent.name,
		b.agent.description,
		b.agent.version,
	)

	return b.agent, nil
}
