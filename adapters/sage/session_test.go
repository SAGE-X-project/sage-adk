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
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}

	if sm.sessions == nil {
		t.Error("sessions map is nil")
	}

	if sm.didIndex == nil {
		t.Error("didIndex map is nil")
	}

	sm.Stop()
}

func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	session, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if session.ID == "" {
		t.Error("session ID is empty")
	}

	if session.LocalDID != localDID {
		t.Errorf("LocalDID = %s, want %s", session.LocalDID, localDID)
	}

	if session.RemoteDID != remoteDID {
		t.Errorf("RemoteDID = %s, want %s", session.RemoteDID, remoteDID)
	}

	if session.Status != SessionPending {
		t.Errorf("Status = %v, want %v", session.Status, SessionPending)
	}

	if session.Metadata == nil {
		t.Error("Metadata map is nil")
	}
}

func TestSessionManager_Get(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	// Create session
	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	created, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get session
	retrieved, err := sm.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, created.ID)
	}
}

func TestSessionManager_GetByDID(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	created, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get by DID
	retrieved, err := sm.GetByDID(remoteDID)
	if err != nil {
		t.Fatalf("GetByDID failed: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %s, want %s", retrieved.ID, created.ID)
	}
}

func TestSessionManager_Update(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	session, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update session
	session.Status = SessionActive
	session.MessagesSent = 5

	err = sm.Update(session)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := sm.Get(session.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Status != SessionActive {
		t.Errorf("Status = %v, want %v", retrieved.Status, SessionActive)
	}

	if retrieved.MessagesSent != 5 {
		t.Errorf("MessagesSent = %d, want 5", retrieved.MessagesSent)
	}
}

func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	session, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete session
	err = sm.Delete(session.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = sm.Get(session.ID)
	if err == nil {
		t.Error("Get should fail after Delete")
	}

	// Verify DID index is cleaned up
	_, err = sm.GetByDID(remoteDID)
	if err == nil {
		t.Error("GetByDID should fail after Delete")
	}
}

func TestSessionManager_List(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	// Create multiple sessions
	_, err := sm.Create("did:sage:ethereum:0xAAA", "did:sage:ethereum:0xBBB")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = sm.Create("did:sage:ethereum:0xCCC", "did:sage:ethereum:0xDDD")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	sessions := sm.List()
	if len(sessions) != 2 {
		t.Errorf("List returned %d sessions, want 2", len(sessions))
	}
}

func TestSessionManager_Count(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	if count := sm.Count(); count != 0 {
		t.Errorf("Count = %d, want 0", count)
	}

	// Create sessions
	session1, _ := sm.Create("did:sage:ethereum:0xAAA", "did:sage:ethereum:0xBBB")
	session2, _ := sm.Create("did:sage:ethereum:0xCCC", "did:sage:ethereum:0xDDD")

	// Mark as active
	session1.Status = SessionActive
	session2.Status = SessionActive
	sm.Update(session1)
	sm.Update(session2)

	if count := sm.Count(); count != 2 {
		t.Errorf("Count = %d, want 2", count)
	}
}

func TestSessionManager_Cleanup(t *testing.T) {
	sm := NewSessionManager(100*time.Millisecond, time.Hour) // Short TTL for testing
	defer sm.Stop()

	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	session, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Manually set expiry to past
	session.ExpiresAt = time.Now().Add(-time.Second)
	sm.Update(session)

	// Run cleanup
	removed := sm.Cleanup()
	if removed != 1 {
		t.Errorf("Cleanup removed %d sessions, want 1", removed)
	}

	// Verify session is gone
	_, err = sm.Get(session.ID)
	if err == nil {
		t.Error("Get should fail after Cleanup")
	}
}

func TestSessionManager_ReuseActiveSession(t *testing.T) {
	sm := NewSessionManager(time.Hour, time.Minute)
	defer sm.Stop()

	localDID := "did:sage:ethereum:0xABC"
	remoteDID := "did:sage:ethereum:0xDEF"

	// Create first session
	session1, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Mark as active
	session1.Status = SessionActive
	sm.Update(session1)

	// Try to create another session with same remote DID
	session2, err := sm.Create(localDID, remoteDID)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should reuse existing session
	if session2.ID != session1.ID {
		t.Errorf("Create created new session, should reuse existing")
	}
}

func TestSession_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		session  *Session
		expected bool
	}{
		{
			name: "active and not expired",
			session: &Session{
				Status:    SessionActive,
				ExpiresAt: now.Add(time.Hour),
			},
			expected: true,
		},
		{
			name: "active but expired",
			session: &Session{
				Status:    SessionActive,
				ExpiresAt: now.Add(-time.Hour),
			},
			expected: false,
		},
		{
			name: "not active",
			session: &Session{
				Status:    SessionPending,
				ExpiresAt: now.Add(time.Hour),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsActive(); got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		session  *Session
		expected bool
	}{
		{
			name: "not expired",
			session: &Session{
				ExpiresAt: now.Add(time.Hour),
			},
			expected: false,
		},
		{
			name: "expired by time",
			session: &Session{
				ExpiresAt: now.Add(-time.Hour),
			},
			expected: true,
		},
		{
			name: "expired by status",
			session: &Session{
				Status:    SessionExpired,
				ExpiresAt: now.Add(time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGenerateSessionID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("generateSessionID failed: %v", err)
		}

		if id == "" {
			t.Error("generated ID is empty")
		}

		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true

		// Check format
		if len(id) < 10 {
			t.Errorf("ID too short: %s", id)
		}
	}
}

func TestGenerateNonce(t *testing.T) {
	// Generate multiple nonces
	nonces := make(map[string]bool)
	for i := 0; i < 100; i++ {
		nonce, err := GenerateNonce()
		if err != nil {
			t.Fatalf("GenerateNonce failed: %v", err)
		}

		if nonce == "" {
			t.Error("generated nonce is empty")
		}

		if nonces[nonce] {
			t.Errorf("duplicate nonce generated: %s", nonce)
		}
		nonces[nonce] = true

		// Check length (32 bytes = 64 hex chars)
		if len(nonce) != 64 {
			t.Errorf("nonce length = %d, want 64", len(nonce))
		}
	}
}

func TestSessionStatus_String(t *testing.T) {
	tests := []struct {
		status   SessionStatus
		expected string
	}{
		{SessionPending, "pending"},
		{SessionEstablishing, "establishing"},
		{SessionActive, "active"},
		{SessionExpired, "expired"},
		{SessionClosed, "closed"},
		{SessionStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("String() = %s, want %s", got, tt.expected)
			}
		})
	}
}
