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
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestNewSigningManager(t *testing.T) {
	sm := NewSigningManager()
	if sm == nil {
		t.Fatal("NewSigningManager returned nil")
	}
}

func TestSigningManager_SignAndVerify(t *testing.T) {
	sm := NewSigningManager()

	// Generate Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Test message
	message := HandshakeInvitation{
		Phase:              PhaseInvitation,
		FromDID:            "did:sage:ethereum:0xABC",
		ToDID:              "did:sage:ethereum:0xDEF",
		Nonce:              "test_nonce_123",
		EphemeralPublicKey: "ephemeral_pub_key",
		Timestamp:          time.Now(),
	}

	keyID := "did:sage:ethereum:0xABC#key-1"

	// Sign message
	signature, err := sm.SignMessage(message, privateKey, keyID)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	if signature.Algorithm != "EdDSA" {
		t.Errorf("Algorithm = %s, want EdDSA", signature.Algorithm)
	}

	if signature.KeyID != keyID {
		t.Errorf("KeyID = %s, want %s", signature.KeyID, keyID)
	}

	if signature.Value == "" {
		t.Error("Signature value is empty")
	}

	// Verify signature
	err = sm.VerifySignature(message, signature, publicKey)
	if err != nil {
		t.Errorf("VerifySignature failed: %v", err)
	}
}

func TestSigningManager_VerifyWithWrongKey(t *testing.T) {
	sm := NewSigningManager()

	// Generate two key pairs
	publicKey1, privateKey1, _ := ed25519.GenerateKey(rand.Reader)
	publicKey2, _, _ := ed25519.GenerateKey(rand.Reader)

	message := map[string]string{
		"from_did": "did:sage:ethereum:0xABC",
		"message":  "test",
	}

	keyID := "did:sage:ethereum:0xABC#key-1"

	// Sign with key1
	signature, err := sm.SignMessage(message, privateKey1, keyID)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Verify with key1 (should succeed)
	err = sm.VerifySignature(message, signature, publicKey1)
	if err != nil {
		t.Errorf("VerifySignature with correct key failed: %v", err)
	}

	// Verify with key2 (should fail)
	err = sm.VerifySignature(message, signature, publicKey2)
	if err == nil {
		t.Error("VerifySignature should fail with wrong key")
	}
}

func TestSigningManager_VerifyModifiedMessage(t *testing.T) {
	sm := NewSigningManager()

	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	originalMessage := map[string]string{
		"from_did": "did:sage:ethereum:0xABC",
		"message":  "original",
	}

	keyID := "did:sage:ethereum:0xABC#key-1"

	// Sign original message
	signature, err := sm.SignMessage(originalMessage, privateKey, keyID)
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}

	// Modify message
	modifiedMessage := map[string]string{
		"from_did": "did:sage:ethereum:0xABC",
		"message":  "modified",
	}

	// Verify with modified message (should fail)
	err = sm.VerifySignature(modifiedMessage, signature, publicKey)
	if err == nil {
		t.Error("VerifySignature should fail with modified message")
	}
}

func TestSigningManager_SignWithNilKey(t *testing.T) {
	sm := NewSigningManager()

	message := map[string]string{"test": "data"}
	keyID := "test-key"

	_, err := sm.SignMessage(message, nil, keyID)
	if err == nil {
		t.Error("SignMessage should fail with nil private key")
	}
}

func TestSigningManager_SignWithEmptyKeyID(t *testing.T) {
	sm := NewSigningManager()

	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	message := map[string]string{"test": "data"}

	_, err := sm.SignMessage(message, privateKey, "")
	if err == nil {
		t.Error("SignMessage should fail with empty keyID")
	}
}

func TestSigningManager_VerifyWithNilSignature(t *testing.T) {
	sm := NewSigningManager()

	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	message := map[string]string{"test": "data"}

	err := sm.VerifySignature(message, nil, publicKey)
	if err == nil {
		t.Error("VerifySignature should fail with nil signature")
	}
}

func TestSigningManager_VerifyWithNilPublicKey(t *testing.T) {
	sm := NewSigningManager()

	signature := &SignatureEnvelope{
		Algorithm: "EdDSA",
		KeyID:     "test-key",
		Value:     "dGVzdA==",
	}
	message := map[string]string{"test": "data"}

	err := sm.VerifySignature(message, signature, nil)
	if err == nil {
		t.Error("VerifySignature should fail with nil public key")
	}
}

func TestSigningManager_ValidateTimestamp(t *testing.T) {
	sm := NewSigningManager()

	tests := []struct {
		name        string
		timestamp   time.Time
		maxSkew     time.Duration
		expectError bool
	}{
		{
			name:        "current time",
			timestamp:   time.Now(),
			maxSkew:     5 * time.Minute,
			expectError: false,
		},
		{
			name:        "1 minute ago",
			timestamp:   time.Now().Add(-time.Minute),
			maxSkew:     5 * time.Minute,
			expectError: false,
		},
		{
			name:        "1 minute future",
			timestamp:   time.Now().Add(time.Minute),
			maxSkew:     5 * time.Minute,
			expectError: false,
		},
		{
			name:        "10 minutes ago (exceeds skew)",
			timestamp:   time.Now().Add(-10 * time.Minute),
			maxSkew:     5 * time.Minute,
			expectError: true,
		},
		{
			name:        "10 minutes future (exceeds skew)",
			timestamp:   time.Now().Add(10 * time.Minute),
			maxSkew:     5 * time.Minute,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.ValidateTimestamp(tt.timestamp, tt.maxSkew)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateTimestamp() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestNonceCache_Check(t *testing.T) {
	cache := NewNonceCache(100)

	nonce1 := "nonce_123"
	nonce2 := "nonce_456"

	// First check should succeed
	err := cache.Check(nonce1)
	if err != nil {
		t.Errorf("First Check failed: %v", err)
	}

	// Second check with same nonce should fail (replay)
	err = cache.Check(nonce1)
	if err == nil {
		t.Error("Check should fail on replay")
	}

	// Check with different nonce should succeed
	err = cache.Check(nonce2)
	if err != nil {
		t.Errorf("Check with different nonce failed: %v", err)
	}
}

func TestNonceCache_Cleanup(t *testing.T) {
	cache := NewNonceCache(5) // Small cache for testing

	// Add more nonces than max size
	for i := 0; i < 10; i++ {
		nonce := string(rune('a' + i))
		cache.Check(nonce)
	}

	// Cache should not exceed max size (with some tolerance for cleanup logic)
	if len(cache.nonces) > 6 {
		t.Errorf("Cache size = %d, should be around maxSize (5)", len(cache.nonces))
	}
}

func TestSigningManager_SignatureBaseConsistency(t *testing.T) {
	sm := NewSigningManager()

	// Same message should produce same signature base
	message := map[string]string{
		"from_did":  "did:sage:ethereum:0xABC",
		"to_did":    "did:sage:ethereum:0xDEF",
		"timestamp": "2025-10-07T10:00:00Z",
	}

	base1, err := sm.createSignatureBase(message)
	if err != nil {
		t.Fatalf("createSignatureBase failed: %v", err)
	}

	base2, err := sm.createSignatureBase(message)
	if err != nil {
		t.Fatalf("createSignatureBase failed: %v", err)
	}

	// Bases should be identical (deterministic)
	if base1 != base2 {
		t.Error("Signature bases are not consistent for same message")
	}
}

func TestSigningManager_SignHandshakeMessages(t *testing.T) {
	sm := NewSigningManager()
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	tests := []struct {
		name    string
		message interface{}
	}{
		{
			name: "Invitation",
			message: HandshakeInvitation{
				Phase:   PhaseInvitation,
				FromDID: "did:sage:ethereum:0xABC",
				ToDID:   "did:sage:ethereum:0xDEF",
				Nonce:   "test_nonce",
			},
		},
		{
			name: "Request",
			message: HandshakeRequest{
				Phase:     PhaseRequest,
				SessionID: "sess_123",
				FromDID:   "did:sage:ethereum:0xDEF",
				ToDID:     "did:sage:ethereum:0xABC",
				Nonce:     "test_nonce_2",
			},
		},
		{
			name: "Response",
			message: HandshakeResponse{
				Phase:     PhaseResponse,
				SessionID: "sess_123",
				FromDID:   "did:sage:ethereum:0xABC",
				ToDID:     "did:sage:ethereum:0xDEF",
			},
		},
		{
			name: "Complete",
			message: HandshakeComplete{
				Phase:     PhaseComplete,
				SessionID: "sess_123",
				FromDID:   "did:sage:ethereum:0xDEF",
				ToDID:     "did:sage:ethereum:0xABC",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyID := "did:sage:ethereum:0xABC#key-1"

			// Sign
			signature, err := sm.SignMessage(tt.message, privateKey, keyID)
			if err != nil {
				t.Fatalf("SignMessage failed: %v", err)
			}

			// Verify
			err = sm.VerifySignature(tt.message, signature, publicKey)
			if err != nil {
				t.Errorf("VerifySignature failed: %v", err)
			}
		})
	}
}
