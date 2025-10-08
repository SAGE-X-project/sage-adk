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

package a2a

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"trpc.group/trpc-go/trpc-a2a-go/client"
	a2a "trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// Adapter implements the ProtocolAdapter interface for A2A protocol.
type Adapter struct {
	client *client.A2AClient
	config *config.A2AConfig
	mu     sync.RWMutex
}

// NewAdapter creates a new A2A protocol adapter.
func NewAdapter(cfg *config.A2AConfig) (*Adapter, error) {
	if cfg == nil {
		return nil, errors.ErrConfigurationError.WithMessage("A2A config is nil")
	}

	if cfg.ServerURL == "" {
		return nil, errors.ErrConfigurationError.WithDetail("field", "ServerURL")
	}

	// Create client options
	opts := []client.Option{}

	if cfg.Timeout > 0 {
		timeout := time.Duration(cfg.Timeout) * time.Second
		opts = append(opts, client.WithTimeout(timeout))
	}

	if cfg.UserAgent != "" {
		opts = append(opts, client.WithUserAgent(cfg.UserAgent))
	}

	// Create A2A client
	a2aClient, err := client.NewA2AClient(cfg.ServerURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create A2A client: %w", err)
	}

	return &Adapter{
		client: a2aClient,
		config: cfg,
	}, nil
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "a2a"
}

// SendMessage sends a message using the A2A protocol.
func (a *Adapter) SendMessage(ctx context.Context, msg *types.Message) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Convert sage-adk message to A2A message
	a2aMsg, err := toA2AMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to convert message: %w", err)
	}

	// Create send parameters
	params := a2a.SendMessageParams{
		Message: a2aMsg,
		RPCID:   a2a.GenerateRPCID(),
	}

	// Send message
	_, err = a.client.SendMessage(ctx, params)
	if err != nil {
		return convertError(err)
	}

	return nil
}

// ReceiveMessage is not supported by A2A (request-response model).
func (a *Adapter) ReceiveMessage(ctx context.Context) (*types.Message, error) {
	return nil, errors.ErrNotImplemented.WithMessage("A2A uses request-response model, not standalone receive")
}

// Verify verifies a message according to A2A protocol.
// A2A verification is handled by the sage-a2a-go client internally.
func (a *Adapter) Verify(ctx context.Context, msg *types.Message) error {
	// Verification handled by sage-a2a-go client
	return nil
}

// SupportsStreaming returns true as A2A supports streaming responses.
func (a *Adapter) SupportsStreaming() bool {
	return true
}

// Stream sends a message and streams the response through the callback.
func (a *Adapter) Stream(ctx context.Context, fn protocol.StreamFunc) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return errors.ErrNotImplemented.WithMessage("streaming not yet implemented")
}

// convertError converts sage-a2a-go errors to sage-adk errors.
func convertError(err error) error {
	if err == nil {
		return nil
	}

	// TODO: Map specific a2a errors to sage-adk errors
	// For now, wrap in ErrInternal
	return errors.ErrInternal.Wrap(err)
}
