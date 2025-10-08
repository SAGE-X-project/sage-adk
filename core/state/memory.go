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
	"sync"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// MemoryManager is an in-memory implementation of the Manager interface.
type MemoryManager struct {
	mu      sync.RWMutex
	states  map[string]*State
	config  *Config
	cleanup *time.Ticker
	done    chan struct{}
}

// NewMemoryManager creates a new in-memory state manager.
func NewMemoryManager(config *Config) *MemoryManager {
	if config == nil {
		config = DefaultConfig()
	}

	m := &MemoryManager{
		states: make(map[string]*State),
		config: config,
		done:   make(chan struct{}),
	}

	// Start automatic cleanup if enabled
	if config.EnableAutoCleanup && config.CleanupInterval > 0 {
		m.cleanup = time.NewTicker(config.CleanupInterval)
		go m.autoCleanup()
	}

	return m
}

// Get retrieves a state by session ID.
func (m *MemoryManager) Get(ctx context.Context, sessionID string) (*State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[sessionID]
	if !exists {
		return nil, ErrStateNotFound
	}

	// Check if expired
	if state.IsExpired() {
		return nil, ErrStateExpired
	}

	// Return a copy to prevent external modifications
	return m.copyState(state), nil
}

// Create creates a new state.
func (m *MemoryManager) Create(ctx context.Context, state *State) error {
	if err := state.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.states[state.SessionID]; exists {
		return ErrStateExists
	}

	// Set timestamps
	now := time.Now()
	state.CreatedAt = now
	state.UpdatedAt = now

	// Set expiration if not set
	if state.ExpiresAt == nil && m.config.DefaultTTL > 0 {
		expiresAt := now.Add(m.config.DefaultTTL)
		state.ExpiresAt = &expiresAt
	}

	// Initialize maps if nil
	if state.Metadata == nil {
		state.Metadata = make(map[string]interface{})
	}
	if state.Variables == nil {
		state.Variables = make(map[string]interface{})
	}
	if state.Messages == nil {
		state.Messages = make([]*types.Message, 0)
	}

	m.states[state.SessionID] = m.copyState(state)
	return nil
}

// Update updates an existing state.
func (m *MemoryManager) Update(ctx context.Context, state *State) error {
	if err := state.Validate(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.states[state.SessionID]
	if !exists {
		return ErrStateNotFound
	}

	// Check if expired
	if existing.IsExpired() {
		return ErrStateExpired
	}

	// Update timestamp
	state.UpdatedAt = time.Now()
	state.CreatedAt = existing.CreatedAt // Preserve creation time

	m.states[state.SessionID] = m.copyState(state)
	return nil
}

// Delete deletes a state by session ID.
func (m *MemoryManager) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.states[sessionID]; !exists {
		return ErrStateNotFound
	}

	delete(m.states, sessionID)
	return nil
}

// List lists all states with optional filters.
func (m *MemoryManager) List(ctx context.Context, filter *Filter) ([]*State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*State, 0)

	for _, state := range m.states {
		// Apply filters
		if filter != nil {
			if filter.AgentID != "" && state.AgentID != filter.AgentID {
				continue
			}
			if filter.ContextID != "" && state.ContextID != filter.ContextID {
				continue
			}
			if !filter.IncludeExpired && state.IsExpired() {
				continue
			}
		}

		result = append(result, m.copyState(state))
	}

	// Apply pagination
	if filter != nil {
		if filter.Offset > 0 && filter.Offset < len(result) {
			result = result[filter.Offset:]
		}
		if filter.Limit > 0 && filter.Limit < len(result) {
			result = result[:filter.Limit]
		}
	}

	return result, nil
}

// AddMessage adds a message to the conversation history.
func (m *MemoryManager) AddMessage(ctx context.Context, sessionID string, message *types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[sessionID]
	if !exists {
		return ErrStateNotFound
	}

	if state.IsExpired() {
		return ErrStateExpired
	}

	// Check max messages limit
	if m.config.MaxMessages > 0 && len(state.Messages) >= m.config.MaxMessages {
		// Remove oldest message
		state.Messages = state.Messages[1:]
	}

	// Add timestamp to message metadata
	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}
	message.Metadata["timestamp"] = time.Now()

	state.Messages = append(state.Messages, message)
	state.UpdatedAt = time.Now()

	return nil
}

// GetMessages retrieves messages from a session.
func (m *MemoryManager) GetMessages(ctx context.Context, sessionID string, limit int) ([]*types.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[sessionID]
	if !exists {
		return nil, ErrStateNotFound
	}

	if state.IsExpired() {
		return nil, ErrStateExpired
	}

	messages := state.Messages
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	// Return copies
	result := make([]*types.Message, len(messages))
	for i, msg := range messages {
		result[i] = msg // Messages are already immutable
	}

	return result, nil
}

// SetVariable sets a state variable.
func (m *MemoryManager) SetVariable(ctx context.Context, sessionID string, key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[sessionID]
	if !exists {
		return ErrStateNotFound
	}

	if state.IsExpired() {
		return ErrStateExpired
	}

	if state.Variables == nil {
		state.Variables = make(map[string]interface{})
	}

	state.Variables[key] = value
	state.UpdatedAt = time.Now()

	return nil
}

// GetVariable retrieves a state variable.
func (m *MemoryManager) GetVariable(ctx context.Context, sessionID string, key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[sessionID]
	if !exists {
		return nil, ErrStateNotFound
	}

	if state.IsExpired() {
		return nil, ErrStateExpired
	}

	value, exists := state.Variables[key]
	if !exists {
		return nil, ErrVariableNotFound
	}

	return value, nil
}

// Clear clears all messages from a session (keeps metadata and variables).
func (m *MemoryManager) Clear(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.states[sessionID]
	if !exists {
		return ErrStateNotFound
	}

	if state.IsExpired() {
		return ErrStateExpired
	}

	state.Messages = make([]*types.Message, 0)
	state.UpdatedAt = time.Now()

	return nil
}

// Cleanup removes expired states.
func (m *MemoryManager) Cleanup(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for sessionID, state := range m.states {
		if state.IsExpired() {
			delete(m.states, sessionID)
			count++
		}
	}

	return count, nil
}

// Close stops the automatic cleanup.
func (m *MemoryManager) Close() error {
	if m.cleanup != nil {
		m.cleanup.Stop()
	}
	close(m.done)
	return nil
}

// autoCleanup runs automatic cleanup periodically.
func (m *MemoryManager) autoCleanup() {
	for {
		select {
		case <-m.cleanup.C:
			m.Cleanup(context.Background())
		case <-m.done:
			return
		}
	}
}

// copyState creates a deep copy of a state.
func (m *MemoryManager) copyState(state *State) *State {
	copy := &State{
		SessionID: state.SessionID,
		ContextID: state.ContextID,
		AgentID:   state.AgentID,
		CreatedAt: state.CreatedAt,
		UpdatedAt: state.UpdatedAt,
		ExpiresAt: state.ExpiresAt,
	}

	// Copy messages
	copy.Messages = make([]*types.Message, 0, len(state.Messages))
	copy.Messages = append(copy.Messages, state.Messages...)

	// Copy metadata
	if state.Metadata != nil {
		copy.Metadata = make(map[string]interface{})
		for k, v := range state.Metadata {
			copy.Metadata[k] = v
		}
	}

	// Copy variables
	if state.Variables != nil {
		copy.Variables = make(map[string]interface{})
		for k, v := range state.Variables {
			copy.Variables[k] = v
		}
	}

	return copy
}

// Count returns the number of states.
func (m *MemoryManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.states)
}
