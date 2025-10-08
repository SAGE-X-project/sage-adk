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
	"time"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

// State represents a conversation state or session.
type State struct {
	// SessionID is the unique identifier for this state/session.
	SessionID string `json:"sessionId"`

	// ContextID is the optional context identifier.
	ContextID string `json:"contextId,omitempty"`

	// AgentID is the agent identifier.
	AgentID string `json:"agentId"`

	// Messages is the conversation history.
	Messages []*types.Message `json:"messages"`

	// Metadata is optional state metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Variables is the state variables storage.
	Variables map[string]interface{} `json:"variables,omitempty"`

	// CreatedAt is the timestamp when the state was created.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is the timestamp when the state was last updated.
	UpdatedAt time.Time `json:"updatedAt"`

	// ExpiresAt is the optional expiration timestamp.
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// Manager is the interface for state management.
type Manager interface {
	// Get retrieves a state by session ID.
	Get(ctx context.Context, sessionID string) (*State, error)

	// Create creates a new state.
	Create(ctx context.Context, state *State) error

	// Update updates an existing state.
	Update(ctx context.Context, state *State) error

	// Delete deletes a state by session ID.
	Delete(ctx context.Context, sessionID string) error

	// List lists all states with optional filters.
	List(ctx context.Context, filter *Filter) ([]*State, error)

	// AddMessage adds a message to the conversation history.
	AddMessage(ctx context.Context, sessionID string, message *types.Message) error

	// GetMessages retrieves messages from a session.
	GetMessages(ctx context.Context, sessionID string, limit int) ([]*types.Message, error)

	// SetVariable sets a state variable.
	SetVariable(ctx context.Context, sessionID string, key string, value interface{}) error

	// GetVariable retrieves a state variable.
	GetVariable(ctx context.Context, sessionID string, key string) (interface{}, error)

	// Clear clears all messages from a session (keeps metadata and variables).
	Clear(ctx context.Context, sessionID string) error

	// Cleanup removes expired states.
	Cleanup(ctx context.Context) (int, error)
}

// Filter represents filtering options for listing states.
type Filter struct {
	// AgentID filters by agent ID.
	AgentID string

	// ContextID filters by context ID.
	ContextID string

	// Limit limits the number of results.
	Limit int

	// Offset is the pagination offset.
	Offset int

	// IncludeExpired includes expired states.
	IncludeExpired bool
}

// Config represents state manager configuration.
type Config struct {
	// DefaultTTL is the default time-to-live for states.
	DefaultTTL time.Duration

	// MaxMessages is the maximum number of messages to keep per session.
	MaxMessages int

	// CleanupInterval is the interval for cleaning up expired states.
	CleanupInterval time.Duration

	// EnableAutoCleanup enables automatic cleanup of expired states.
	EnableAutoCleanup bool
}

// DefaultConfig returns the default state manager configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultTTL:        24 * time.Hour,
		MaxMessages:       100,
		CleanupInterval:   1 * time.Hour,
		EnableAutoCleanup: true,
	}
}

// Validate validates the state.
func (s *State) Validate() error {
	if s.SessionID == "" {
		return ErrInvalidSessionID
	}
	if s.AgentID == "" {
		return ErrInvalidAgentID
	}
	return nil
}

// IsExpired checks if the state has expired.
func (s *State) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}

// MessageCount returns the number of messages in the state.
func (s *State) MessageCount() int {
	return len(s.Messages)
}

// LastMessage returns the last message in the conversation.
func (s *State) LastMessage() *types.Message {
	if len(s.Messages) == 0 {
		return nil
	}
	return s.Messages[len(s.Messages)-1]
}

// GetMessagesAfter returns messages after a specific timestamp.
func (s *State) GetMessagesAfter(after time.Time) []*types.Message {
	result := make([]*types.Message, 0)
	for _, msg := range s.Messages {
		if msg.Metadata != nil {
			if ts, ok := msg.Metadata["timestamp"]; ok {
				if timestamp, ok := ts.(time.Time); ok && timestamp.After(after) {
					result = append(result, msg)
				}
			}
		}
	}
	return result
}

// TruncateMessages truncates messages to keep only the last n messages.
func (s *State) TruncateMessages(n int) {
	if len(s.Messages) > n {
		s.Messages = s.Messages[len(s.Messages)-n:]
	}
}
