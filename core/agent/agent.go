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

// agentImpl implements the Agent interface.
type agentImpl struct {
	// Identity
	name        string
	description string
	version     string
	card        *types.AgentCard

	// Configuration
	config *config.Config

	// Message handling
	messageHandler MessageHandler
}

// Name returns the agent name.
func (a *agentImpl) Name() string {
	return a.name
}

// Description returns the agent description.
func (a *agentImpl) Description() string {
	return a.description
}

// Card returns the agent card.
func (a *agentImpl) Card() *types.AgentCard {
	return a.card
}

// Config returns the agent configuration.
func (a *agentImpl) Config() *config.Config {
	return a.config
}

// Process processes a message and returns a response.
func (a *agentImpl) Process(ctx context.Context, msg *types.Message) (*types.Message, error) {
	// Create message context
	msgCtx := &messageContext{
		agent:   a,
		message: msg,
		ctx:     ctx,
	}

	// Execute handler
	err := a.messageHandler(ctx, msgCtx)
	if err != nil {
		return nil, err
	}

	// Return the response that was sent via msgCtx
	return msgCtx.response, nil
}
