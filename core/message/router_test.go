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
	"testing"

	"github.com/sage-x-project/sage-adk/core/middleware"
	"github.com/sage-x-project/sage-adk/core/protocol"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter(protocol.ProtocolAuto)
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}

	if router.GetProtocolMode() != protocol.ProtocolAuto {
		t.Errorf("GetProtocolMode() = %v, want %v", router.GetProtocolMode(), protocol.ProtocolAuto)
	}
}

func TestRouter_RegisterAdapter(t *testing.T) {
	router := NewRouter(protocol.ProtocolAuto)

	// Test successful registration
	adapter := protocol.NewMockAdapter("a2a")
	err := router.RegisterAdapter(adapter)
	if err != nil {
		t.Errorf("RegisterAdapter() error = %v, want nil", err)
	}

	// Test nil adapter
	err = router.RegisterAdapter(nil)
	if err == nil {
		t.Error("RegisterAdapter(nil) should return error")
	}
}

func TestRouter_GetAdapter(t *testing.T) {
	router := NewRouter(protocol.ProtocolAuto)
	adapter := protocol.NewMockAdapter("a2a")
	router.RegisterAdapter(adapter)

	// Test successful retrieval
	retrieved, err := router.GetAdapter("a2a")
	if err != nil {
		t.Errorf("GetAdapter() error = %v, want nil", err)
	}
	if retrieved != adapter {
		t.Error("GetAdapter() returned wrong adapter")
	}

	// Test not found
	_, err = router.GetAdapter("nonexistent")
	if err == nil {
		t.Error("GetAdapter() should return error for nonexistent adapter")
	}
}

func TestRouter_UseMiddleware(t *testing.T) {
	router := NewRouter(protocol.ProtocolAuto)

	// Add middleware
	called := false
	mw := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
			called = true
			return next(ctx, msg)
		}
	}

	router.UseMiddleware(mw)

	// Set handler
	router.SetHandler(func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		return msg, nil
	})

	// Register adapter
	adapter := protocol.NewMockAdapter("a2a")
	router.RegisterAdapter(adapter)

	// Route message
	msg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("test"),
	})
	_, err := router.Route(context.Background(), msg)
	if err != nil {
		t.Errorf("Route() error = %v, want nil", err)
	}

	if !called {
		t.Error("Middleware was not called")
	}
}

func TestRouter_Route(t *testing.T) {
	tests := []struct {
		name          string
		mode          protocol.ProtocolMode
		message       *types.Message
		adapters      []string
		expectError   bool
		expectedRoute string
	}{
		{
			name: "A2A mode routes to a2a adapter",
			mode: protocol.ProtocolA2A,
			message: types.NewMessage(types.MessageRoleUser, []types.Part{
				types.NewTextPart("test"),
			}),
			adapters:      []string{"a2a"},
			expectError:   false,
			expectedRoute: "a2a",
		},
		{
			name: "SAGE mode routes to sage adapter",
			mode: protocol.ProtocolSAGE,
			message: types.NewMessage(types.MessageRoleUser, []types.Part{
				types.NewTextPart("test"),
			}),
			adapters:      []string{"sage"},
			expectError:   false,
			expectedRoute: "sage",
		},
		{
			name: "Auto mode detects SAGE from message",
			mode: protocol.ProtocolAuto,
			message: &types.Message{
				MessageID: "test-1",
				Role:      types.MessageRoleUser,
				Parts: []types.Part{
					types.NewTextPart("test"),
				},
				Security: &types.SecurityMetadata{
					Mode: types.ProtocolModeSAGE,
				},
			},
			adapters:      []string{"a2a", "sage"},
			expectError:   false,
			expectedRoute: "sage",
		},
		{
			name: "Auto mode defaults to A2A",
			mode: protocol.ProtocolAuto,
			message: types.NewMessage(types.MessageRoleUser, []types.Part{
				types.NewTextPart("test"),
			}),
			adapters:      []string{"a2a", "sage"},
			expectError:   false,
			expectedRoute: "a2a",
		},
		{
			name: "Missing adapter returns error",
			mode: protocol.ProtocolA2A,
			message: types.NewMessage(types.MessageRoleUser, []types.Part{
				types.NewTextPart("test"),
			}),
			adapters:    []string{},
			expectError: true,
		},
		{
			name:        "Nil message returns error",
			mode:        protocol.ProtocolA2A,
			message:     nil,
			adapters:    []string{"a2a"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.mode)

			// Register adapters
			for _, name := range tt.adapters {
				adapter := protocol.NewMockAdapter(name)
				router.RegisterAdapter(adapter)
			}

			// Set handler
			var routedAdapter string
			router.SetHandler(func(ctx context.Context, msg *types.Message) (*types.Message, error) {
				if adapter, ok := AdapterFromContext(ctx); ok {
					routedAdapter = adapter.Name()
				}
				return msg, nil
			})

			// Route message
			_, err := router.Route(context.Background(), tt.message)

			if tt.expectError {
				if err == nil {
					t.Error("Route() should return error")
				}
				return
			}

			if err != nil {
				t.Errorf("Route() error = %v, want nil", err)
				return
			}

			if tt.expectedRoute != "" && routedAdapter != tt.expectedRoute {
				t.Errorf("Routed to %s, want %s", routedAdapter, tt.expectedRoute)
			}
		})
	}
}

func TestRouter_Send(t *testing.T) {
	router := NewRouter(protocol.ProtocolA2A)
	adapter := protocol.NewMockAdapter("a2a")
	router.RegisterAdapter(adapter)

	msg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("test"),
	})

	err := router.Send(context.Background(), msg)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	if len(adapter.SentMessages) != 1 {
		t.Errorf("Adapter received %d messages, want 1", len(adapter.SentMessages))
	}
}

func TestRouter_Receive(t *testing.T) {
	router := NewRouter(protocol.ProtocolA2A)
	adapter := protocol.NewMockAdapter("a2a")

	// Add message to adapter queue
	msg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("test"),
	})
	adapter.ReceivedMessages = append(adapter.ReceivedMessages, msg)

	router.RegisterAdapter(adapter)

	received, err := router.Receive(context.Background(), "a2a")
	if err != nil {
		t.Errorf("Receive() error = %v, want nil", err)
	}

	if received == nil {
		t.Fatal("Receive() returned nil message")
	}

	if received.MessageID != msg.MessageID {
		t.Errorf("Received wrong message")
	}
}

func TestRouter_Verify(t *testing.T) {
	router := NewRouter(protocol.ProtocolA2A)
	adapter := protocol.NewMockAdapter("a2a")
	router.RegisterAdapter(adapter)

	msg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("test"),
	})

	err := router.Verify(context.Background(), msg)
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
}

func TestRouter_SetProtocolMode(t *testing.T) {
	router := NewRouter(protocol.ProtocolAuto)

	router.SetProtocolMode(protocol.ProtocolSAGE)
	if router.GetProtocolMode() != protocol.ProtocolSAGE {
		t.Errorf("SetProtocolMode() did not update mode")
	}
}

func TestRouter_NoHandler(t *testing.T) {
	router := NewRouter(protocol.ProtocolA2A)
	adapter := protocol.NewMockAdapter("a2a")
	router.RegisterAdapter(adapter)

	msg := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("test"),
	})

	// Route without setting handler
	_, err := router.Route(context.Background(), msg)
	if err == nil {
		t.Error("Route() should return error when no handler is set")
	}
}

func TestAdapterFromContext(t *testing.T) {
	adapter := protocol.NewMockAdapter("test")
	ctx := context.WithValue(context.Background(), adapterContextKey, adapter)

	retrieved, ok := AdapterFromContext(ctx)
	if !ok {
		t.Error("AdapterFromContext() could not retrieve adapter")
	}

	if retrieved.Name() != "test" {
		t.Errorf("AdapterFromContext() returned wrong adapter")
	}

	// Test with empty context
	_, ok = AdapterFromContext(context.Background())
	if ok {
		t.Error("AdapterFromContext() should return false for empty context")
	}
}
