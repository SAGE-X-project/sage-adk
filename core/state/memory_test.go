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

package state

import (
	"context"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewMemoryManager(t *testing.T) {
	manager := NewMemoryManager(nil)
	if manager == nil {
		t.Fatal("NewMemoryManager returned nil")
	}
	defer manager.Close()

	if manager.Count() != 0 {
		t.Errorf("Count() = %d, want 0", manager.Count())
	}
}

func TestMemoryManager_Create(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if manager.Count() != 1 {
		t.Errorf("Count() = %d, want 1", manager.Count())
	}
}

func TestMemoryManager_CreateDuplicate(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Try to create again
	err = manager.Create(context.Background(), state)
	if err != ErrStateExists {
		t.Errorf("Create() error = %v, want ErrStateExists", err)
	}
}

func TestMemoryManager_CreateInvalidSessionID(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "", // Empty
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != ErrInvalidSessionID {
		t.Errorf("Create() error = %v, want ErrInvalidSessionID", err)
	}
}

func TestMemoryManager_CreateInvalidAgentID(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "", // Empty
	}

	err := manager.Create(context.Background(), state)
	if err != ErrInvalidAgentID {
		t.Errorf("Create() error = %v, want ErrInvalidAgentID", err)
	}
}

func TestMemoryManager_Get(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := manager.Get(context.Background(), "session-001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.SessionID != state.SessionID {
		t.Errorf("SessionID = %s, want %s", retrieved.SessionID, state.SessionID)
	}
	if retrieved.AgentID != state.AgentID {
		t.Errorf("AgentID = %s, want %s", retrieved.AgentID, state.AgentID)
	}
}

func TestMemoryManager_GetNotFound(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	_, err := manager.Get(context.Background(), "nonexistent")
	if err != ErrStateNotFound {
		t.Errorf("Get() error = %v, want ErrStateNotFound", err)
	}
}

func TestMemoryManager_Update(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	state.ContextID = "context-001"
	err = manager.Update(context.Background(), state)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Retrieve and verify
	retrieved, err := manager.Get(context.Background(), "session-001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.ContextID != "context-001" {
		t.Errorf("ContextID = %s, want context-001", retrieved.ContextID)
	}
}

func TestMemoryManager_UpdateNotFound(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Update(context.Background(), state)
	if err != ErrStateNotFound {
		t.Errorf("Update() error = %v, want ErrStateNotFound", err)
	}
}

func TestMemoryManager_Delete(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = manager.Delete(context.Background(), "session-001")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if manager.Count() != 0 {
		t.Errorf("Count() = %d, want 0", manager.Count())
	}
}

func TestMemoryManager_DeleteNotFound(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	err := manager.Delete(context.Background(), "nonexistent")
	if err != ErrStateNotFound {
		t.Errorf("Delete() error = %v, want ErrStateNotFound", err)
	}
}

func TestMemoryManager_List(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	// Create multiple states
	for i := 0; i < 5; i++ {
		state := &State{
			SessionID: "session-" + string(rune('0'+i)),
			AgentID:   "agent-001",
		}
		if err := manager.Create(context.Background(), state); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List all
	states, err := manager.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(states) != 5 {
		t.Errorf("List() returned %d states, want 5", len(states))
	}
}

func TestMemoryManager_ListWithFilter(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	// Create states with different agent IDs
	for i := 0; i < 3; i++ {
		state := &State{
			SessionID: "session-a-" + string(rune('0'+i)),
			AgentID:   "agent-001",
		}
		if err := manager.Create(context.Background(), state); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		state := &State{
			SessionID: "session-b-" + string(rune('0'+i)),
			AgentID:   "agent-002",
		}
		if err := manager.Create(context.Background(), state); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Filter by agent ID
	filter := &Filter{
		AgentID: "agent-001",
	}

	states, err := manager.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(states) != 3 {
		t.Errorf("List() returned %d states, want 3", len(states))
	}
}

func TestMemoryManager_ListWithPagination(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	// Create 10 states
	for i := 0; i < 10; i++ {
		state := &State{
			SessionID: "session-" + string(rune('0'+i)),
			AgentID:   "agent-001",
		}
		if err := manager.Create(context.Background(), state); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get first 5
	filter := &Filter{
		Limit: 5,
	}

	states, err := manager.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(states) != 5 {
		t.Errorf("List() returned %d states, want 5", len(states))
	}
}

func TestMemoryManager_AddMessage(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	message := &types.Message{
		MessageID: "msg-001",
		Role:      types.MessageRoleUser,
		Parts:     []types.Part{&types.TextPart{Kind: "text", Text: "Hello"}},
	}

	err = manager.AddMessage(context.Background(), "session-001", message)
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	// Verify
	retrieved, err := manager.Get(context.Background(), "session-001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(retrieved.Messages) != 1 {
		t.Errorf("Message count = %d, want 1", len(retrieved.Messages))
	}
}

func TestMemoryManager_AddMessageMaxLimit(t *testing.T) {
	config := DefaultConfig()
	config.MaxMessages = 3
	manager := NewMemoryManager(config)
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add 5 messages (max is 3)
	for i := 0; i < 5; i++ {
		message := &types.Message{
			MessageID: "msg-" + string(rune('0'+i)),
			Role:      types.MessageRoleUser,
			Parts:     []types.Part{&types.TextPart{Kind: "text", Text: "Hello"}},
		}
		err = manager.AddMessage(context.Background(), "session-001", message)
		if err != nil {
			t.Fatalf("AddMessage() error = %v", err)
		}
	}

	// Should only have last 3 messages
	messages, err := manager.GetMessages(context.Background(), "session-001", 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Message count = %d, want 3", len(messages))
	}

	// First message should be msg-2 (oldest was removed)
	if messages[0].MessageID != "msg-2" {
		t.Errorf("First message ID = %s, want msg-2", messages[0].MessageID)
	}
}

func TestMemoryManager_GetMessages(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add 5 messages
	for i := 0; i < 5; i++ {
		message := &types.Message{
			MessageID: "msg-" + string(rune('0'+i)),
			Role:      types.MessageRoleUser,
			Parts:     []types.Part{&types.TextPart{Kind: "text", Text: "Hello"}},
		}
		err = manager.AddMessage(context.Background(), "session-001", message)
		if err != nil {
			t.Fatalf("AddMessage() error = %v", err)
		}
	}

	// Get last 3 messages
	messages, err := manager.GetMessages(context.Background(), "session-001", 3)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("Message count = %d, want 3", len(messages))
	}

	// Should be msg-2, msg-3, msg-4
	if messages[0].MessageID != "msg-2" {
		t.Errorf("First message ID = %s, want msg-2", messages[0].MessageID)
	}
}

func TestMemoryManager_SetVariable(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = manager.SetVariable(context.Background(), "session-001", "user_name", "Alice")
	if err != nil {
		t.Fatalf("SetVariable() error = %v", err)
	}

	// Verify
	value, err := manager.GetVariable(context.Background(), "session-001", "user_name")
	if err != nil {
		t.Fatalf("GetVariable() error = %v", err)
	}

	if value != "Alice" {
		t.Errorf("Variable value = %v, want Alice", value)
	}
}

func TestMemoryManager_GetVariableNotFound(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = manager.GetVariable(context.Background(), "session-001", "nonexistent")
	if err != ErrVariableNotFound {
		t.Errorf("GetVariable() error = %v, want ErrVariableNotFound", err)
	}
}

func TestMemoryManager_Clear(t *testing.T) {
	manager := NewMemoryManager(DefaultConfig())
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add messages
	for i := 0; i < 3; i++ {
		message := &types.Message{
			MessageID: "msg-" + string(rune('0'+i)),
			Role:      types.MessageRoleUser,
			Parts:     []types.Part{&types.TextPart{Kind: "text", Text: "Hello"}},
		}
		err = manager.AddMessage(context.Background(), "session-001", message)
		if err != nil {
			t.Fatalf("AddMessage() error = %v", err)
		}
	}

	// Set variable
	err = manager.SetVariable(context.Background(), "session-001", "test_var", "value")
	if err != nil {
		t.Fatalf("SetVariable() error = %v", err)
	}

	// Clear messages
	err = manager.Clear(context.Background(), "session-001")
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify messages cleared
	messages, err := manager.GetMessages(context.Background(), "session-001", 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("Message count = %d, want 0", len(messages))
	}

	// Verify variables preserved
	value, err := manager.GetVariable(context.Background(), "session-001", "test_var")
	if err != nil {
		t.Errorf("GetVariable() error = %v", err)
	}
	if value != "value" {
		t.Errorf("Variable value = %v, want value", value)
	}
}

func TestMemoryManager_Cleanup(t *testing.T) {
	config := DefaultConfig()
	config.DefaultTTL = 100 * time.Millisecond
	manager := NewMemoryManager(config)
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	count, err := manager.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if count != 1 {
		t.Errorf("Cleanup() removed %d states, want 1", count)
	}

	if manager.Count() != 0 {
		t.Errorf("Count() = %d, want 0", manager.Count())
	}
}

func TestMemoryManager_ExpiredStateAccess(t *testing.T) {
	config := DefaultConfig()
	config.DefaultTTL = 50 * time.Millisecond
	manager := NewMemoryManager(config)
	defer manager.Close()

	state := &State{
		SessionID: "session-001",
		AgentID:   "agent-001",
	}

	err := manager.Create(context.Background(), state)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Try to access expired state
	_, err = manager.Get(context.Background(), "session-001")
	if err != ErrStateExpired {
		t.Errorf("Get() error = %v, want ErrStateExpired", err)
	}
}

func TestState_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"nil expiration", nil, false},
		{"expired", &past, true},
		{"not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &State{
				SessionID: "test",
				AgentID:   "test",
				ExpiresAt: tt.expiresAt,
			}

			if got := state.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_MessageCount(t *testing.T) {
	state := &State{
		SessionID: "test",
		AgentID:   "test",
		Messages: []*types.Message{
			{MessageID: "msg-1"},
			{MessageID: "msg-2"},
			{MessageID: "msg-3"},
		},
	}

	if count := state.MessageCount(); count != 3 {
		t.Errorf("MessageCount() = %d, want 3", count)
	}
}

func TestState_LastMessage(t *testing.T) {
	state := &State{
		SessionID: "test",
		AgentID:   "test",
		Messages: []*types.Message{
			{MessageID: "msg-1"},
			{MessageID: "msg-2"},
			{MessageID: "msg-3"},
		},
	}

	last := state.LastMessage()
	if last == nil {
		t.Fatal("LastMessage() returned nil")
	}

	if last.MessageID != "msg-3" {
		t.Errorf("LastMessage().MessageID = %s, want msg-3", last.MessageID)
	}
}

func TestState_LastMessageEmpty(t *testing.T) {
	state := &State{
		SessionID: "test",
		AgentID:   "test",
		Messages:  []*types.Message{},
	}

	last := state.LastMessage()
	if last != nil {
		t.Errorf("LastMessage() = %v, want nil", last)
	}
}

func TestState_TruncateMessages(t *testing.T) {
	state := &State{
		SessionID: "test",
		AgentID:   "test",
		Messages: []*types.Message{
			{MessageID: "msg-1"},
			{MessageID: "msg-2"},
			{MessageID: "msg-3"},
			{MessageID: "msg-4"},
			{MessageID: "msg-5"},
		},
	}

	state.TruncateMessages(3)

	if len(state.Messages) != 3 {
		t.Errorf("Message count = %d, want 3", len(state.Messages))
	}

	// Should keep last 3 messages
	if state.Messages[0].MessageID != "msg-3" {
		t.Errorf("First message ID = %s, want msg-3", state.Messages[0].MessageID)
	}
}
