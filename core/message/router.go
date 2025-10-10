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

package message

import (
	"context"
	"sync"

	"github.com/sage-x-project/sage-adk/core/middleware"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// Router routes messages to appropriate handlers through middleware chain.
// It manages protocol adapters and middleware for message processing.
type Router struct {
	// Protocol adapters indexed by name
	adapters map[string]protocol.ProtocolAdapter

	// Protocol mode (auto, a2a, sage)
	mode protocol.ProtocolMode

	// Middleware chain
	chain *middleware.Chain

	// Default handler for processing messages
	handler middleware.Handler

	// Mutex for thread-safe access
	mu sync.RWMutex
}

// NewRouter creates a new message router.
func NewRouter(mode protocol.ProtocolMode) *Router {
	return &Router{
		adapters: make(map[string]protocol.ProtocolAdapter),
		mode:     mode,
		chain:    middleware.NewChain(),
	}
}

// RegisterAdapter registers a protocol adapter.
func (r *Router) RegisterAdapter(adapter protocol.ProtocolAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if adapter == nil {
		return errors.ErrInvalidInput.WithMessage("adapter is nil")
	}

	name := adapter.Name()
	if name == "" {
		return errors.ErrInvalidInput.WithMessage("adapter name is empty")
	}

	r.adapters[name] = adapter
	return nil
}

// GetAdapter returns a protocol adapter by name.
func (r *Router) GetAdapter(name string) (protocol.ProtocolAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[name]
	if !ok {
		return nil, errors.ErrNotFound.
			WithMessage("adapter not found").
			WithDetail("name", name)
	}

	return adapter, nil
}

// UseMiddleware adds middleware to the chain.
func (r *Router) UseMiddleware(mw middleware.Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chain.Use(mw)
}

// SetHandler sets the default message handler.
func (r *Router) SetHandler(handler middleware.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handler = handler
}

// Route routes a message through the middleware chain to the handler.
// It automatically selects the appropriate protocol adapter based on the mode.
func (r *Router) Route(ctx context.Context, msg *types.Message) (*types.Message, error) {
	r.mu.RLock()
	handler := r.handler
	chain := r.chain
	mode := r.mode
	r.mu.RUnlock()

	if handler == nil {
		return nil, errors.ErrOperationFailed.WithMessage("no handler configured")
	}

	if msg == nil {
		return nil, errors.ErrInvalidInput.WithMessage("message is nil")
	}

	// Select protocol adapter based on mode
	adapter, err := r.selectAdapter(msg, mode)
	if err != nil {
		return nil, err
	}

	// Add adapter to context for middleware/handler access
	ctx = context.WithValue(ctx, adapterContextKey, adapter)

	// Execute through middleware chain
	return chain.Execute(ctx, msg, handler)
}

// Send sends a message using the appropriate protocol adapter.
func (r *Router) Send(ctx context.Context, msg *types.Message) error {
	r.mu.RLock()
	mode := r.mode
	r.mu.RUnlock()

	if msg == nil {
		return errors.ErrInvalidInput.WithMessage("message is nil")
	}

	// Select protocol adapter
	adapter, err := r.selectAdapter(msg, mode)
	if err != nil {
		return err
	}

	// Send message
	return adapter.SendMessage(ctx, msg)
}

// Receive receives a message using the specified protocol adapter.
func (r *Router) Receive(ctx context.Context, adapterName string) (*types.Message, error) {
	adapter, err := r.GetAdapter(adapterName)
	if err != nil {
		return nil, err
	}

	return adapter.ReceiveMessage(ctx)
}

// Verify verifies a message using the appropriate protocol adapter.
func (r *Router) Verify(ctx context.Context, msg *types.Message) error {
	r.mu.RLock()
	mode := r.mode
	r.mu.RUnlock()

	if msg == nil {
		return errors.ErrInvalidInput.WithMessage("message is nil")
	}

	// Select protocol adapter
	adapter, err := r.selectAdapter(msg, mode)
	if err != nil {
		return err
	}

	// Verify message
	return adapter.Verify(ctx, msg)
}

// selectAdapter selects the appropriate protocol adapter based on mode and message.
func (r *Router) selectAdapter(msg *types.Message, mode protocol.ProtocolMode) (protocol.ProtocolAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var adapterName string

	switch mode {
	case protocol.ProtocolAuto:
		// Auto-detect protocol from message
		detectedMode := protocol.DetectProtocol(msg)
		if detectedMode == protocol.ProtocolSAGE {
			adapterName = "sage"
		} else {
			adapterName = "a2a"
		}

	case protocol.ProtocolA2A:
		adapterName = "a2a"

	case protocol.ProtocolSAGE:
		adapterName = "sage"

	default:
		return nil, errors.ErrInvalidValue.
			WithMessage("unknown protocol mode").
			WithDetail("mode", mode.String())
	}

	adapter, ok := r.adapters[adapterName]
	if !ok {
		return nil, errors.ErrNotFound.
			WithMessage("protocol adapter not registered").
			WithDetail("adapter", adapterName)
	}

	return adapter, nil
}

// GetProtocolMode returns the current protocol mode.
func (r *Router) GetProtocolMode() protocol.ProtocolMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

// SetProtocolMode sets the protocol mode.
func (r *Router) SetProtocolMode(mode protocol.ProtocolMode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mode = mode
}

// Context keys
type contextKeyType string

const (
	adapterContextKey contextKeyType = "protocol_adapter"
)

// AdapterFromContext retrieves the protocol adapter from context.
func AdapterFromContext(ctx context.Context) (protocol.ProtocolAdapter, bool) {
	adapter, ok := ctx.Value(adapterContextKey).(protocol.ProtocolAdapter)
	return adapter, ok
}
