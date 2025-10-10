// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// MessageStream represents a bidirectional message stream
type MessageStream interface {
	// Send sends a message to the stream
	Send(*types.Message) error

	// Recv receives a message from the stream
	Recv() (*types.Message, error)

	// Close closes the stream
	Close() error
}
