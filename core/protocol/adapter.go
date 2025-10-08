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

package protocol

import (
	"context"
	"fmt"

	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// ProtocolMode represents the protocol selection mode.
type ProtocolMode int

const (
	// ProtocolAuto automatically detects the protocol from message metadata.
	ProtocolAuto ProtocolMode = iota

	// ProtocolA2A uses the Agent-to-Agent protocol exclusively.
	ProtocolA2A

	// ProtocolSAGE uses the Secure Agent Guarantee Engine protocol exclusively.
	ProtocolSAGE
)

// String returns the string representation of the protocol mode.
func (m ProtocolMode) String() string {
	switch m {
	case ProtocolAuto:
		return "auto"
	case ProtocolA2A:
		return "a2a"
	case ProtocolSAGE:
		return "sage"
	default:
		return "unknown"
	}
}

// StreamFunc is a callback function for streaming protocol responses.
// It receives chunks of the response as they arrive.
type StreamFunc func(chunk string) error

// ProtocolAdapter defines the interface for protocol adapters.
// Each protocol (A2A, SAGE, etc.) must implement this interface.
type ProtocolAdapter interface {
	// Name returns the adapter name (e.g., "a2a", "sage").
	Name() string

	// SendMessage sends a message using this protocol.
	SendMessage(ctx context.Context, msg *types.Message) error

	// ReceiveMessage receives a message using this protocol.
	ReceiveMessage(ctx context.Context) (*types.Message, error)

	// Verify verifies a message according to this protocol's security requirements.
	Verify(ctx context.Context, msg *types.Message) error

	// SupportsStreaming returns true if this protocol supports streaming responses.
	SupportsStreaming() bool

	// Stream sends a message and streams the response through the callback.
	// Returns an error if streaming is not supported.
	Stream(ctx context.Context, fn StreamFunc) error
}

// DetectProtocol automatically detects the protocol from a message.
// It checks the message's security metadata to determine the protocol.
func DetectProtocol(msg *types.Message) ProtocolMode {
	if msg.Security != nil && msg.Security.Mode == types.ProtocolModeSAGE {
		return ProtocolSAGE
	}
	return ProtocolA2A
}

// MockAdapter is a mock implementation of ProtocolAdapter for testing.
type MockAdapter struct {
	AdapterName      string
	SentMessages     []*types.Message
	ReceivedMessages []*types.Message
	Streaming        bool
}

// NewMockAdapter creates a new mock adapter.
func NewMockAdapter(name string) *MockAdapter {
	return &MockAdapter{
		AdapterName:      name,
		SentMessages:     make([]*types.Message, 0),
		ReceivedMessages: make([]*types.Message, 0),
		Streaming:        false,
	}
}

// Name returns the adapter name.
func (m *MockAdapter) Name() string {
	return m.AdapterName
}

// SendMessage records the sent message.
func (m *MockAdapter) SendMessage(ctx context.Context, msg *types.Message) error {
	m.SentMessages = append(m.SentMessages, msg)
	return nil
}

// ReceiveMessage returns the next message from the queue.
func (m *MockAdapter) ReceiveMessage(ctx context.Context) (*types.Message, error) {
	if len(m.ReceivedMessages) == 0 {
		return nil, errors.ErrNotFound.WithMessage("no messages available")
	}

	msg := m.ReceivedMessages[0]
	m.ReceivedMessages = m.ReceivedMessages[1:]
	return msg, nil
}

// Verify always returns nil for mock.
func (m *MockAdapter) Verify(ctx context.Context, msg *types.Message) error {
	return nil
}

// SupportsStreaming returns the streaming capability.
func (m *MockAdapter) SupportsStreaming() bool {
	return m.Streaming
}

// Stream calls the callback with a test chunk if streaming is supported.
func (m *MockAdapter) Stream(ctx context.Context, fn StreamFunc) error {
	if !m.Streaming {
		return fmt.Errorf("streaming not supported")
	}

	return fn("test chunk")
}
