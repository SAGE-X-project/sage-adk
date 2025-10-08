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
	"sync"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// ProtocolSelector selects the appropriate protocol adapter for a message.
type ProtocolSelector interface {
	// Select returns the protocol adapter for the given message.
	// Returns nil if no adapter is registered for the selected protocol.
	Select(msg *types.Message) ProtocolAdapter

	// Register registers a protocol adapter for a protocol mode.
	Register(mode ProtocolMode, adapter ProtocolAdapter)

	// SetMode sets the protocol selection mode.
	SetMode(mode ProtocolMode)

	// GetMode returns the current protocol selection mode.
	GetMode() ProtocolMode
}

// selector implements ProtocolSelector.
type selector struct {
	mu       sync.RWMutex
	mode     ProtocolMode
	adapters map[ProtocolMode]ProtocolAdapter
}

// NewSelector creates a new protocol selector with auto-detection enabled.
func NewSelector() ProtocolSelector {
	return &selector{
		mode:     ProtocolAuto,
		adapters: make(map[ProtocolMode]ProtocolAdapter),
	}
}

// Select returns the appropriate protocol adapter for the message.
func (s *selector) Select(msg *types.Message) ProtocolAdapter {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var mode ProtocolMode

	// If mode is fixed (not Auto), use it directly
	if s.mode != ProtocolAuto {
		mode = s.mode
	} else {
		// Auto-detect protocol from message
		mode = DetectProtocol(msg)
	}

	return s.adapters[mode]
}

// Register registers a protocol adapter for a protocol mode.
func (s *selector) Register(mode ProtocolMode, adapter ProtocolAdapter) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.adapters[mode] = adapter
}

// SetMode sets the protocol selection mode.
func (s *selector) SetMode(mode ProtocolMode) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mode = mode
}

// GetMode returns the current protocol selection mode.
func (s *selector) GetMode() ProtocolMode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.mode
}
