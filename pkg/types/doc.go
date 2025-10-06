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

// Package types provides core type definitions for SAGE ADK.
//
// This package defines the fundamental types used throughout the ADK framework,
// supporting both A2A (Agent-to-Agent) and SAGE (Secure Agent Guarantee Engine)
// protocols. The type system is designed to be:
//
//   - Protocol Agnostic: Core types work with both A2A and SAGE
//   - Extensible: Easy to add new protocols or features
//   - Type Safe: Strong typing with minimal interface{} usage
//   - Conversion Friendly: Easy conversion between A2A and ADK types
//   - Validation Ready: Built-in validation support
//
// # Type Categories
//
// The package organizes types into several categories:
//
//   - Message Types: Message, Part (TextPart, FilePart, DataPart), MessageRole
//   - Task Types: Task, TaskStatus, TaskState, Artifact
//   - Agent Types: Agent, AgentCapability
//   - Security Types: SecurityMetadata, SignatureData, ProtocolMode
//   - Context Types: Context
//
// # Protocol Support
//
// Messages can be created for either protocol:
//
//	// A2A message
//	msg := &types.Message{
//	    MessageID: "msg-123",
//	    Role: types.MessageRoleUser,
//	    Parts: []types.Part{
//	        &types.TextPart{Text: "Hello"},
//	    },
//	}
//
//	// SAGE message with security
//	msg := &types.Message{
//	    MessageID: "msg-456",
//	    Role: types.MessageRoleAgent,
//	    Parts: []types.Part{
//	        &types.TextPart{Text: "Response"},
//	    },
//	    Security: &types.SecurityMetadata{
//	        Mode: types.ProtocolModeSAGE,
//	        AgentDID: "did:sage:eth:0x123",
//	        Nonce: "nonce-789",
//	    },
//	}
//
// # Validation
//
// All major types implement validation:
//
//	if err := msg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Conversion
//
// Convert between A2A and ADK types:
//
//	// A2A to ADK
//	adkMsg, err := types.FromA2AMessage(a2aMsg)
//
//	// ADK to A2A
//	a2aMsg, err := adkMsg.ToA2AMessage()
package types
