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

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Agent represents an AI agent instance.
type Agent interface {
	// Name returns the agent name.
	Name() string

	// Description returns the agent description.
	Description() string

	// Card returns the agent card with metadata.
	Card() *types.AgentCard

	// Config returns the agent configuration.
	Config() *config.Config

	// Process processes a message and returns a response.
	Process(ctx context.Context, msg *types.Message) (*types.Message, error)
}

// Builder constructs agents with fluent API.
type Builder interface {
	// WithName sets the agent name.
	WithName(name string) Builder

	// WithDescription sets the agent description.
	WithDescription(desc string) Builder

	// WithVersion sets the agent version.
	WithVersion(version string) Builder

	// OnMessage sets the message handler.
	OnMessage(handler MessageHandler) Builder

	// Build validates configuration and constructs the agent.
	Build() (Agent, error)
}

// MessageContext provides context and helpers for message processing.
type MessageContext interface {
	// Text returns the message text content.
	Text() string

	// Parts returns all message parts.
	Parts() []types.Part

	// ContextID returns the conversation context ID.
	ContextID() string

	// MessageID returns the message ID.
	MessageID() string

	// Reply sends a text response.
	Reply(text string) error

	// ReplyWithParts sends a response with multiple parts.
	ReplyWithParts(parts []types.Part) error
}

// MessageHandler processes incoming messages.
type MessageHandler func(ctx context.Context, msg MessageContext) error
