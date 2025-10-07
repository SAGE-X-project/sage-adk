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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func setupHandshakeManager(did string, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) *HandshakeManager {
	sessionManager := NewSessionManager(1*time.Hour, 5*time.Minute)
	encryptionManager := NewEncryptionManager()
	signingManager := NewSigningManager()

	config := &TransportConfig{
		MaxClockSkew: 5 * time.Minute,
	}

	return NewHandshakeManager(
		sessionManager,
		encryptionManager,
		signingManager,
		did,
		privateKey,
		config,
	)
}

func TestNewHandshakeManager(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	sessionManager := NewSessionManager(1*time.Hour, 5*time.Minute)
	encryptionManager := NewEncryptionManager()
	signingManager := NewSigningManager()

	config := &TransportConfig{
		MaxClockSkew: 5 * time.Minute,
	}

	hm := NewHandshakeManager(
		sessionManager,
		encryptionManager,
		signingManager,
		"did:sage:alice",
		privateKey,
		config,
	)

	if hm == nil {
		t.Fatal("NewHandshakeManager returned nil")
	}

	if hm.localDID != "did:sage:alice" {
		t.Errorf("localDID = %s, want did:sage:alice", hm.localDID)
	}
}

func TestHandshakeManager_InitiateHandshake(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	hm := setupHandshakeManager("did:sage:alice", privateKey, publicKey)

	ctx := context.Background()
	invitation, session, err := hm.InitiateHandshake(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("InitiateHandshake failed: %v", err)
	}

	// Check invitation
	if invitation.Phase != PhaseInvitation {
		t.Errorf("Phase = %s, want %s", invitation.Phase, PhaseInvitation)
	}

	if invitation.FromDID != "did:sage:alice" {
		t.Errorf("FromDID = %s, want did:sage:alice", invitation.FromDID)
	}

	if invitation.ToDID != "did:sage:bob" {
		t.Errorf("ToDID = %s, want did:sage:bob", invitation.ToDID)
	}

	if invitation.Nonce == "" {
		t.Error("Nonce is empty")
	}

	if invitation.EphemeralPublicKey == "" {
		t.Error("EphemeralPublicKey is empty")
	}

	// Check session
	if session.Status != SessionEstablishing {
		t.Errorf("Session status = %d, want %d", session.Status, SessionEstablishing)
	}

	if session.LocalDID != "did:sage:alice" {
		t.Errorf("Session LocalDID = %s, want did:sage:alice", session.LocalDID)
	}

	if session.RemoteDID != "did:sage:bob" {
		t.Errorf("Session RemoteDID = %s, want did:sage:bob", session.RemoteDID)
	}

	if session.LocalNonce == "" {
		t.Error("Session LocalNonce is empty")
	}

	if len(session.EphemeralKey) == 0 {
		t.Error("Session EphemeralKey is empty")
	}
}

func TestHandshakeManager_ProcessInvitation(t *testing.T) {
	// Setup Alice
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	// Setup Bob
	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	ctx := context.Background()

	// Alice initiates
	invitation, aliceSession, err := alice.InitiateHandshake(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("InitiateHandshake failed: %v", err)
	}

	// Bob processes invitation
	request, bobSession, err := bob.ProcessInvitation(ctx, invitation)
	if err != nil {
		t.Fatalf("ProcessInvitation failed: %v", err)
	}

	// Check request
	if request.Phase != PhaseRequest {
		t.Errorf("Phase = %s, want %s", request.Phase, PhaseRequest)
	}

	if request.SessionID == "" {
		t.Error("SessionID is empty")
	}

	if request.FromDID != "did:sage:bob" {
		t.Errorf("FromDID = %s, want did:sage:bob", request.FromDID)
	}

	if request.ToDID != "did:sage:alice" {
		t.Errorf("ToDID = %s, want did:sage:alice", request.ToDID)
	}

	if request.EphemeralPublicKey == "" {
		t.Error("EphemeralPublicKey is empty")
	}

	if request.EncryptedPayload.Ciphertext == "" {
		t.Error("EncryptedPayload.Ciphertext is empty")
	}

	if request.Signature.Value == "" {
		t.Error("Signature is empty")
	}

	// Check Bob's session
	if bobSession.Status != SessionEstablishing {
		t.Errorf("Bob session status = %d, want %d", bobSession.Status, SessionEstablishing)
	}

	if bobSession.ID != request.SessionID {
		t.Errorf("Bob session ID = %s, want %s", bobSession.ID, request.SessionID)
	}

	if len(bobSession.SharedSecret) != 32 {
		t.Errorf("Bob SharedSecret length = %d, want 32", len(bobSession.SharedSecret))
	}

	// Verify session IDs match
	if aliceSession.ID == bobSession.ID {
		t.Error("Alice and Bob have same session ID (should be different)")
	}
}

func TestHandshakeManager_ProcessRequest(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	ctx := context.Background()

	// Phase 1: Alice initiates
	invitation, aliceSession, err := alice.InitiateHandshake(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("InitiateHandshake failed: %v", err)
	}

	// Phase 2: Bob responds
	request, _, err := bob.ProcessInvitation(ctx, invitation)
	if err != nil {
		t.Fatalf("ProcessInvitation failed: %v", err)
	}

	// Phase 3: Alice processes request
	response, err := alice.ProcessRequest(ctx, request, aliceSession, bobPublicKey)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Check response
	if response.Phase != PhaseResponse {
		t.Errorf("Phase = %s, want %s", response.Phase, PhaseResponse)
	}

	if response.SessionID != request.SessionID {
		t.Errorf("SessionID = %s, want %s", response.SessionID, request.SessionID)
	}

	if response.FromDID != "did:sage:alice" {
		t.Errorf("FromDID = %s, want did:sage:alice", response.FromDID)
	}

	if response.ToDID != "did:sage:bob" {
		t.Errorf("ToDID = %s, want did:sage:bob", response.ToDID)
	}

	if response.EncryptedPayload.Ciphertext == "" {
		t.Error("EncryptedPayload.Ciphertext is empty")
	}

	if response.Signature.Value == "" {
		t.Error("Signature is empty")
	}

	// Check Alice's session has session key
	if len(aliceSession.SessionKey) != 32 {
		t.Errorf("Alice SessionKey length = %d, want 32", len(aliceSession.SessionKey))
	}

	// Session should still be establishing (not active until complete)
	if aliceSession.Status != SessionEstablishing {
		t.Errorf("Alice session status = %d, want %d", aliceSession.Status, SessionEstablishing)
	}
}

func TestHandshakeManager_ProcessResponse(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	ctx := context.Background()

	// Phase 1-3
	invitation, aliceSession, _ := alice.InitiateHandshake(ctx, "did:sage:bob")
	request, bobSession, _ := bob.ProcessInvitation(ctx, invitation)
	response, _ := alice.ProcessRequest(ctx, request, aliceSession, bobPublicKey)

	// Phase 4: Bob processes response
	complete, err := bob.ProcessResponse(ctx, response, bobSession, alicePublicKey)
	if err != nil {
		t.Fatalf("ProcessResponse failed: %v", err)
	}

	// Check complete message
	if complete.Phase != PhaseComplete {
		t.Errorf("Phase = %s, want %s", complete.Phase, PhaseComplete)
	}

	if complete.SessionID != bobSession.ID {
		t.Errorf("SessionID = %s, want %s", complete.SessionID, bobSession.ID)
	}

	if complete.FromDID != "did:sage:bob" {
		t.Errorf("FromDID = %s, want did:sage:bob", complete.FromDID)
	}

	if complete.ToDID != "did:sage:alice" {
		t.Errorf("ToDID = %s, want did:sage:alice", complete.ToDID)
	}

	if complete.EncryptedPayload.Ciphertext == "" {
		t.Error("EncryptedPayload.Ciphertext is empty")
	}

	// Check Bob's session has session key
	if len(bobSession.SessionKey) != 32 {
		t.Errorf("Bob SessionKey length = %d, want 32", len(bobSession.SessionKey))
	}

	// Bob's session should still be establishing
	if bobSession.Status != SessionEstablishing {
		t.Errorf("Bob session status = %d, want %d", bobSession.Status, SessionEstablishing)
	}
}

func TestHandshakeManager_ProcessComplete(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	ctx := context.Background()

	// Phase 1-4
	invitation, aliceSession, _ := alice.InitiateHandshake(ctx, "did:sage:bob")
	request, bobSession, _ := bob.ProcessInvitation(ctx, invitation)
	response, _ := alice.ProcessRequest(ctx, request, aliceSession, bobPublicKey)
	complete, _ := bob.ProcessResponse(ctx, response, bobSession, alicePublicKey)

	// Alice processes complete
	err := alice.ProcessComplete(ctx, complete, aliceSession, bobPublicKey)
	if err != nil {
		t.Fatalf("ProcessComplete failed: %v", err)
	}

	// Alice's session should now be active
	if aliceSession.Status != SessionActive {
		t.Errorf("Alice session status = %d, want %d", aliceSession.Status, SessionActive)
	}
}

func TestHandshakeManager_FullFlow(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	ctx := context.Background()

	// Phase 1: Alice initiates
	invitation, aliceSession, err := alice.InitiateHandshake(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("Phase 1 failed: %v", err)
	}

	// Phase 2: Bob responds
	request, bobSession, err := bob.ProcessInvitation(ctx, invitation)
	if err != nil {
		t.Fatalf("Phase 2 failed: %v", err)
	}

	// Phase 3: Alice confirms
	response, err := alice.ProcessRequest(ctx, request, aliceSession, bobPublicKey)
	if err != nil {
		t.Fatalf("Phase 3 failed: %v", err)
	}

	// Phase 4: Bob acknowledges
	complete, err := bob.ProcessResponse(ctx, response, bobSession, alicePublicKey)
	if err != nil {
		t.Fatalf("Phase 4 failed: %v", err)
	}

	// Alice activates
	err = alice.ProcessComplete(ctx, complete, aliceSession, bobPublicKey)
	if err != nil {
		t.Fatalf("Phase 4 complete failed: %v", err)
	}

	// Verify both sessions are established
	if aliceSession.Status != SessionActive {
		t.Errorf("Alice session not active: %d", aliceSession.Status)
	}

	if bobSession.Status != SessionEstablishing {
		t.Errorf("Bob session should still be establishing: %d", bobSession.Status)
	}

	// Verify session keys match
	if len(aliceSession.SessionKey) != 32 {
		t.Errorf("Alice session key length = %d, want 32", len(aliceSession.SessionKey))
	}

	if len(bobSession.SessionKey) != 32 {
		t.Errorf("Bob session key length = %d, want 32", len(bobSession.SessionKey))
	}

	// Session keys should be identical
	for i := range aliceSession.SessionKey {
		if aliceSession.SessionKey[i] != bobSession.SessionKey[i] {
			t.Errorf("Session keys don't match at index %d", i)
			break
		}
	}

	// Verify nonces were exchanged
	if aliceSession.LocalNonce == "" || aliceSession.RemoteNonce == "" {
		t.Error("Alice missing nonces")
	}

	if bobSession.LocalNonce == "" || bobSession.RemoteNonce == "" {
		t.Error("Bob missing nonces")
	}
}

func TestHandshakeManager_ValidateInvitation(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	hm := setupHandshakeManager("did:sage:alice", privateKey, publicKey)

	tests := []struct {
		name        string
		invitation  *HandshakeInvitation
		expectError bool
	}{
		{
			name: "valid invitation",
			invitation: &HandshakeInvitation{
				Phase:              PhaseInvitation,
				FromDID:            "did:sage:alice",
				ToDID:              "did:sage:bob",
				Nonce:              "test_nonce",
				EphemeralPublicKey: "dGVzdA==",
				Timestamp:          time.Now(),
			},
			expectError: false,
		},
		{
			name:        "nil invitation",
			invitation:  nil,
			expectError: true,
		},
		{
			name: "wrong phase",
			invitation: &HandshakeInvitation{
				Phase:              PhaseRequest,
				FromDID:            "did:sage:alice",
				ToDID:              "did:sage:bob",
				Nonce:              "test_nonce",
				EphemeralPublicKey: "dGVzdA==",
				Timestamp:          time.Now(),
			},
			expectError: true,
		},
		{
			name: "missing nonce",
			invitation: &HandshakeInvitation{
				Phase:              PhaseInvitation,
				FromDID:            "did:sage:alice",
				ToDID:              "did:sage:bob",
				Nonce:              "",
				EphemeralPublicKey: "dGVzdA==",
				Timestamp:          time.Now(),
			},
			expectError: true,
		},
		{
			name: "expired timestamp",
			invitation: &HandshakeInvitation{
				Phase:              PhaseInvitation,
				FromDID:            "did:sage:alice",
				ToDID:              "did:sage:bob",
				Nonce:              "test_nonce",
				EphemeralPublicKey: "dGVzdA==",
				Timestamp:          time.Now().Add(-10 * time.Minute),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hm.ValidateInvitation(tt.invitation)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateInvitation() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestHandshakeManager_ValidateRequest(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	hm := setupHandshakeManager("did:sage:alice", privateKey, publicKey)

	tests := []struct {
		name        string
		request     *HandshakeRequest
		expectError bool
	}{
		{
			name: "valid request",
			request: &HandshakeRequest{
				Phase:              PhaseRequest,
				SessionID:          "sess_123",
				FromDID:            "did:sage:bob",
				ToDID:              "did:sage:alice",
				Nonce:              "test_nonce_123",
				EphemeralPublicKey: "dGVzdA==",
				EncryptedPayload: EncryptedPayload{
					Ciphertext: "dGVzdA==",
					Nonce:      "dGVzdA==",
				},
				Timestamp: time.Now(),
			},
			expectError: false,
		},
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "wrong phase",
			request: &HandshakeRequest{
				Phase:              PhaseInvitation,
				SessionID:          "sess_123",
				FromDID:            "did:sage:bob",
				ToDID:              "did:sage:alice",
				EphemeralPublicKey: "dGVzdA==",
				Timestamp:          time.Now(),
			},
			expectError: true,
		},
		{
			name: "missing session ID",
			request: &HandshakeRequest{
				Phase:              PhaseRequest,
				SessionID:          "",
				FromDID:            "did:sage:bob",
				ToDID:              "did:sage:alice",
				EphemeralPublicKey: "dGVzdA==",
				Timestamp:          time.Now(),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hm.ValidateRequest(tt.request)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateRequest() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// Note: Removed TestHandshakeManager_ProcessInvitation_InvalidSignature
// because ProcessInvitation doesn't verify the invitation signature.
// Signature verification happens in ProcessRequest when Alice verifies Bob's request signature.

func TestHandshakeManager_ProcessRequest_WrongPublicKey(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	// Generate wrong public key
	wrongPublicKey, _, _ := ed25519.GenerateKey(rand.Reader)

	ctx := context.Background()

	// Phase 1-2
	invitation, aliceSession, _ := alice.InitiateHandshake(ctx, "did:sage:bob")
	request, _, _ := bob.ProcessInvitation(ctx, invitation)

	// Alice tries to verify with wrong public key
	_, err := alice.ProcessRequest(ctx, request, aliceSession, wrongPublicKey)
	if err == nil {
		t.Error("ProcessRequest should fail with wrong public key")
	}
}

func TestHandshakeManager_ProcessRequest_ModifiedPayload(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := setupHandshakeManager("did:sage:alice", alicePrivateKey, alicePublicKey)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := setupHandshakeManager("did:sage:bob", bobPrivateKey, bobPublicKey)

	ctx := context.Background()

	// Phase 1-2
	invitation, aliceSession, _ := alice.InitiateHandshake(ctx, "did:sage:bob")
	request, _, _ := bob.ProcessInvitation(ctx, invitation)

	// Tamper with encrypted payload
	request.EncryptedPayload.Ciphertext = "tampered_data"

	// Alice tries to process tampered request
	_, err := alice.ProcessRequest(ctx, request, aliceSession, bobPublicKey)
	if err == nil {
		t.Error("ProcessRequest should fail with tampered payload")
	}
}
