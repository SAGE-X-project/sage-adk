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

// Package protocol provides protocol abstraction and selection for SAGE ADK.
//
// This package implements a flexible protocol layer that supports multiple
// agent communication protocols through the adapter pattern. It provides
// automatic protocol detection and transparent switching between protocols.
//
// # Protocol Modes
//
// The protocol layer supports three modes:
//
//   - Auto: Automatically detect protocol from message metadata (default)
//   - A2A: Use Agent-to-Agent protocol exclusively
//   - SAGE: Use Secure Agent Guarantee Engine protocol exclusively
//
// # Protocol Detection
//
// When in Auto mode, the protocol is detected by examining the message's
// security metadata. If the message has SAGE security metadata, the SAGE
// protocol is used. Otherwise, the A2A protocol is used.
//
// Example:
//
//	selector := protocol.NewSelector()
//	selector.Register(protocol.ProtocolA2A, a2aAdapter)
//	selector.Register(protocol.ProtocolSAGE, sageAdapter)
//
//	// Auto-detect protocol
//	adapter := selector.Select(msg)
//	err := adapter.SendMessage(ctx, msg)
//
// # Fixed Protocol Mode
//
// You can fix the protocol mode to always use a specific protocol:
//
//	selector.SetMode(protocol.ProtocolSAGE)
//	adapter := selector.Select(msg) // Always returns SAGE adapter
//
// # Custom Adapters
//
// Implement the ProtocolAdapter interface to support new protocols:
//
//	type CustomAdapter struct{}
//
//	func (a *CustomAdapter) Name() string { return "custom" }
//	func (a *CustomAdapter) SendMessage(ctx context.Context, msg *types.Message) error {
//	    // Custom protocol implementation
//	    return nil
//	}
//	// ... implement other methods
//
//	selector.Register(ProtocolCustom, &CustomAdapter{})
package protocol
