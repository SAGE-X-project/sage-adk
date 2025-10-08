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

package sage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// SessionManager manages active SAGE sessions.
type SessionManager struct {
	// Session storage
	sessions map[string]*Session // sessionID → Session
	didIndex map[string]string   // remoteDID → sessionID
	mu       sync.RWMutex

	// Configuration
	sessionTTL      time.Duration
	cleanupInterval time.Duration

	// Lifecycle
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewSessionManager creates a new session manager.
func NewSessionManager(sessionTTL, cleanupInterval time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		didIndex:        make(map[string]string),
		sessionTTL:      sessionTTL,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}

	// Start cleanup goroutine
	sm.wg.Add(1)
	go sm.cleanupLoop()

	return sm
}

// Create creates a new session.
func (sm *SessionManager) Create(localDID, remoteDID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, errors.ErrOperationFailed.
			WithMessage("failed to generate session ID").
			WithDetail("error", err.Error())
	}

	// Check if session already exists for this remote DID
	if existingID, exists := sm.didIndex[remoteDID]; exists {
		if existingSession, ok := sm.sessions[existingID]; ok && existingSession.IsActive() {
			// Reuse existing active session
			return existingSession, nil
		}
		// Clean up expired session
		delete(sm.sessions, existingID)
		delete(sm.didIndex, remoteDID)
	}

	// Create new session
	now := time.Now()
	session := &Session{
		ID:         sessionID,
		LocalDID:   localDID,
		RemoteDID:  remoteDID,
		Status:     SessionPending,
		CreatedAt:  now,
		ExpiresAt:  now.Add(sm.sessionTTL),
		LastActive: now,
		Metadata:   make(map[string]interface{}),
	}

	// Store session
	sm.sessions[sessionID] = session
	sm.didIndex[remoteDID] = sessionID

	return session, nil
}

// Get retrieves a session by ID.
func (sm *SessionManager) Get(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, errors.ErrNotFound.
			WithMessage("session not found").
			WithDetail("session_id", sessionID)
	}

	if session.IsExpired() {
		return nil, errors.ErrOperationFailed.
			WithMessage("session expired").
			WithDetail("session_id", sessionID)
	}

	return session, nil
}

// GetByDID retrieves a session by remote DID.
func (sm *SessionManager) GetByDID(remoteDID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionID, exists := sm.didIndex[remoteDID]
	if !exists {
		return nil, errors.ErrNotFound.
			WithMessage("no session found for DID").
			WithDetail("remote_did", remoteDID)
	}

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, errors.ErrNotFound.
			WithMessage("session not found").
			WithDetail("session_id", sessionID)
	}

	if session.IsExpired() {
		return nil, errors.ErrOperationFailed.
			WithMessage("session expired").
			WithDetail("session_id", sessionID)
	}

	return session, nil
}

// Update updates an existing session.
func (sm *SessionManager) Update(session *Session) error {
	if session == nil {
		return errors.ErrInvalidInput.WithMessage("session is nil")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	existing, exists := sm.sessions[session.ID]
	if !exists {
		return errors.ErrNotFound.
			WithMessage("session not found").
			WithDetail("session_id", session.ID)
	}

	// Update session
	*existing = *session
	existing.UpdateActivity()

	return nil
}

// Delete removes a session.
func (sm *SessionManager) Delete(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return errors.ErrNotFound.
			WithMessage("session not found").
			WithDetail("session_id", sessionID)
	}

	// Remove from indexes
	delete(sm.sessions, sessionID)
	delete(sm.didIndex, session.RemoteDID)

	return nil
}

// List returns all active sessions.
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		if !session.IsExpired() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		if session.IsActive() {
			count++
		}
	}

	return count
}

// Cleanup removes expired sessions.
func (sm *SessionManager) Cleanup() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	removed := 0
	now := time.Now()

	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) || session.Status == SessionExpired {
			delete(sm.sessions, sessionID)
			delete(sm.didIndex, session.RemoteDID)
			removed++
		}
	}

	return removed
}

// cleanupLoop periodically removes expired sessions.
func (sm *SessionManager) cleanupLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.Cleanup()
		case <-sm.stopChan:
			return
		}
	}
}

// Stop stops the session manager and cleanup goroutine.
func (sm *SessionManager) Stop() {
	close(sm.stopChan)
	sm.wg.Wait()
}

// generateSessionID generates a cryptographically random session ID.
func generateSessionID() (string, error) {
	bytes := make([]byte, 16) // 128 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "sess_" + hex.EncodeToString(bytes), nil
}

// GenerateNonce generates a cryptographically random nonce.
func GenerateNonce() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
