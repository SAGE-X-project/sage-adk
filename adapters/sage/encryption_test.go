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

	"golang.org/x/crypto/chacha20poly1305"
)

func TestNewEncryptionManager(t *testing.T) {
	em := NewEncryptionManager()
	if em == nil {
		t.Fatal("NewEncryptionManager returned nil")
	}
}

func TestEncryptionManager_GenerateEphemeralKeyPair(t *testing.T) {
	em := NewEncryptionManager()

	privKey, pubKey, err := em.GenerateEphemeralKeyPair()
	if err != nil {
		t.Fatalf("GenerateEphemeralKeyPair failed: %v", err)
	}

	if privKey == nil {
		t.Error("private key is nil")
	}

	if len(pubKey) != 32 {
		t.Errorf("public key length = %d, want 32", len(pubKey))
	}
}

func TestEncryptionManager_DeriveSharedSecret(t *testing.T) {
	em := NewEncryptionManager()

	// Generate two key pairs
	alicePriv, alicePub, err := em.GenerateEphemeralKeyPair()
	if err != nil {
		t.Fatalf("GenerateEphemeralKeyPair failed: %v", err)
	}

	bobPriv, bobPub, err := em.GenerateEphemeralKeyPair()
	if err != nil {
		t.Fatalf("GenerateEphemeralKeyPair failed: %v", err)
	}

	// Derive shared secrets
	aliceSecret, err := em.DeriveSharedSecret(alicePriv, bobPub)
	if err != nil {
		t.Fatalf("Alice DeriveSharedSecret failed: %v", err)
	}

	bobSecret, err := em.DeriveSharedSecret(bobPriv, alicePub)
	if err != nil {
		t.Fatalf("Bob DeriveSharedSecret failed: %v", err)
	}

	// Secrets should match
	if len(aliceSecret) != 32 {
		t.Errorf("Alice secret length = %d, want 32", len(aliceSecret))
	}

	if len(bobSecret) != 32 {
		t.Errorf("Bob secret length = %d, want 32", len(bobSecret))
	}

	// Compare secrets (should be equal)
	for i := range aliceSecret {
		if aliceSecret[i] != bobSecret[i] {
			t.Errorf("Shared secrets don't match at index %d", i)
			break
		}
	}
}

func TestEncryptionManager_GenerateSessionKey(t *testing.T) {
	em := NewEncryptionManager()

	key, err := em.GenerateSessionKey()
	if err != nil {
		t.Fatalf("GenerateSessionKey failed: %v", err)
	}

	if len(key) != chacha20poly1305.KeySize {
		t.Errorf("session key length = %d, want %d", len(key), chacha20poly1305.KeySize)
	}

	// Generate multiple keys and ensure they're different
	key2, _ := em.GenerateSessionKey()
	allSame := true
	for i := range key {
		if key[i] != key2[i] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("Generated identical session keys")
	}
}

func TestEncryptionManager_EncryptDecryptWithSharedSecret(t *testing.T) {
	em := NewEncryptionManager()

	// Generate shared secret
	alicePriv, _, _ := em.GenerateEphemeralKeyPair()
	_, bobPub, _ := em.GenerateEphemeralKeyPair()
	sharedSecret, _ := em.DeriveSharedSecret(alicePriv, bobPub)

	// Test data
	testData := map[string]interface{}{
		"message": "Hello, SAGE!",
		"nonce":   "test_nonce_123",
	}

	// Encrypt
	encrypted, err := em.EncryptWithSharedSecret(testData, sharedSecret)
	if err != nil {
		t.Fatalf("EncryptWithSharedSecret failed: %v", err)
	}

	if encrypted.Algorithm != "ChaCha20-Poly1305" {
		t.Errorf("Algorithm = %s, want ChaCha20-Poly1305", encrypted.Algorithm)
	}

	if encrypted.Ciphertext == "" {
		t.Error("Ciphertext is empty")
	}

	if encrypted.Nonce == "" {
		t.Error("Nonce is empty")
	}

	// Decrypt
	var decrypted map[string]interface{}
	err = em.DecryptWithSharedSecret(encrypted, sharedSecret, &decrypted)
	if err != nil {
		t.Fatalf("DecryptWithSharedSecret failed: %v", err)
	}

	if decrypted["message"] != testData["message"] {
		t.Errorf("Decrypted message = %v, want %v", decrypted["message"], testData["message"])
	}

	if decrypted["nonce"] != testData["nonce"] {
		t.Errorf("Decrypted nonce = %v, want %v", decrypted["nonce"], testData["nonce"])
	}
}

func TestEncryptionManager_EncryptDecryptWithSessionKey(t *testing.T) {
	em := NewEncryptionManager()

	// Generate session key
	sessionKey, err := em.GenerateSessionKey()
	if err != nil {
		t.Fatalf("GenerateSessionKey failed: %v", err)
	}

	// Test data
	testData := map[string]string{
		"ack":        "session_established",
		"session_id": "sess_123",
	}

	// Encrypt
	encrypted, err := em.EncryptWithSessionKey(testData, sessionKey)
	if err != nil {
		t.Fatalf("EncryptWithSessionKey failed: %v", err)
	}

	// Decrypt
	var decrypted map[string]string
	err = em.DecryptWithSessionKey(encrypted, sessionKey, &decrypted)
	if err != nil {
		t.Fatalf("DecryptWithSessionKey failed: %v", err)
	}

	if decrypted["ack"] != testData["ack"] {
		t.Errorf("Decrypted ack = %v, want %v", decrypted["ack"], testData["ack"])
	}

	if decrypted["session_id"] != testData["session_id"] {
		t.Errorf("Decrypted session_id = %v, want %v", decrypted["session_id"], testData["session_id"])
	}
}

func TestEncryptionManager_DecryptWithWrongKey(t *testing.T) {
	em := NewEncryptionManager()

	// Generate two different session keys
	sessionKey1, _ := em.GenerateSessionKey()
	sessionKey2, _ := em.GenerateSessionKey()

	// Test data
	testData := map[string]string{"message": "secret"}

	// Encrypt with key1
	encrypted, err := em.EncryptWithSessionKey(testData, sessionKey1)
	if err != nil {
		t.Fatalf("EncryptWithSessionKey failed: %v", err)
	}

	// Try to decrypt with key2 (should fail)
	var decrypted map[string]string
	err = em.DecryptWithSessionKey(encrypted, sessionKey2, &decrypted)
	if err == nil {
		t.Error("DecryptWithSessionKey should fail with wrong key")
	}
}

func TestEncryptionManager_InvalidSessionKeySize(t *testing.T) {
	em := NewEncryptionManager()

	testData := map[string]string{"message": "test"}
	invalidKey := []byte("too_short")

	// Try to encrypt with invalid key
	_, err := em.EncryptWithSessionKey(testData, invalidKey)
	if err == nil {
		t.Error("EncryptWithSessionKey should fail with invalid key size")
	}

	// Try to decrypt with invalid key
	encrypted := &EncryptedPayload{
		Algorithm:  "ChaCha20-Poly1305",
		Ciphertext: "dGVzdA==",
		Nonce:      "dGVzdA==",
	}
	var decrypted map[string]string
	err = em.DecryptWithSessionKey(encrypted, invalidKey, &decrypted)
	if err == nil {
		t.Error("DecryptWithSessionKey should fail with invalid key size")
	}
}

func TestEncryptionManager_EncryptPayloadForPublicKey(t *testing.T) {
	em := NewEncryptionManager()

	// Generate recipient key pair
	_, recipientPubKey, err := em.GenerateEphemeralKeyPair()
	if err != nil {
		t.Fatalf("GenerateEphemeralKeyPair failed: %v", err)
	}

	// Test data
	testData := map[string]string{
		"invitation_nonce": "nonce_a",
		"response_nonce":   "nonce_b",
	}

	// Encrypt for public key
	encrypted, ephemeralPub, err := em.EncryptPayloadForPublicKey(testData, recipientPubKey)
	if err != nil {
		t.Fatalf("EncryptPayloadForPublicKey failed: %v", err)
	}

	if encrypted.Algorithm != "HPKE" {
		t.Errorf("Algorithm = %s, want HPKE", encrypted.Algorithm)
	}

	if len(ephemeralPub) != 32 {
		t.Errorf("Ephemeral public key length = %d, want 32", len(ephemeralPub))
	}
}

func TestEncryptionManager_DecryptWithNilPayload(t *testing.T) {
	em := NewEncryptionManager()
	sessionKey, _ := em.GenerateSessionKey()

	var decrypted map[string]string
	err := em.DecryptWithSessionKey(nil, sessionKey, &decrypted)
	if err == nil {
		t.Error("DecryptWithSessionKey should fail with nil payload")
	}
}

func TestEncryptionManager_DeriveSharedSecret_InvalidPublicKey(t *testing.T) {
	em := NewEncryptionManager()

	privKey, _, _ := em.GenerateEphemeralKeyPair()
	invalidPubKey := []byte("invalid_key")

	_, err := em.DeriveSharedSecret(privKey, invalidPubKey)
	if err == nil {
		t.Error("DeriveSharedSecret should fail with invalid public key")
	}
}
