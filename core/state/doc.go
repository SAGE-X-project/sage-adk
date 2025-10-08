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

// Package state provides conversation state and session management for agents.
//
// State management allows agents to maintain conversation history, context,
// and variables across multiple interactions. This is essential for:
//   - Multi-turn conversations
//   - Context-aware responses
//   - Session persistence
//   - Conversation history
//   - State variables and metadata
//
// Example:
//
//	// Create state manager
//	config := &state.Config{
//	    DefaultTTL:        24 * time.Hour,
//	    MaxMessages:       100,
//	    CleanupInterval:   1 * time.Hour,
//	    EnableAutoCleanup: true,
//	}
//	manager := state.NewMemoryManager(config)
//	defer manager.Close()
//
//	// Create new conversation state
//	session := &state.State{
//	    SessionID: "session-123",
//	    AgentID:   "my-agent",
//	}
//	err := manager.Create(ctx, session)
//
//	// Add messages to conversation
//	userMsg := &types.Message{
//	    MessageID: "msg-001",
//	    Role:      types.MessageRoleUser,
//	    Parts:     []types.Part{&types.TextPart{Text: "Hello"}},
//	}
//	manager.AddMessage(ctx, "session-123", userMsg)
//
//	// Retrieve conversation history
//	messages, err := manager.GetMessages(ctx, "session-123", 10)
//
//	// Store state variables
//	manager.SetVariable(ctx, "session-123", "user_name", "Alice")
//
//	// Retrieve state
//	state, err := manager.Get(ctx, "session-123")
//
// Custom State Manager:
//
//	// Implement the Manager interface for custom storage
//	type CustomManager struct {
//	    // Your storage implementation
//	}
//
//	func (m *CustomManager) Get(ctx context.Context, sessionID string) (*state.State, error) {
//	    // Retrieve from your storage
//	    return state, nil
//	}
//
//	func (m *CustomManager) Create(ctx context.Context, state *state.State) error {
//	    // Store in your storage
//	    return nil
//	}
//
//	// Implement other methods...
package state
