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
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewAdapter_Success(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	if adapter == nil {
		t.Fatal("NewAdapter() should not return nil")
	}

	if adapter.Name() != "sage" {
		t.Errorf("Name() = %v, want sage", adapter.Name())
	}
}

func TestNewAdapter_MissingDID(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "",
	}

	_, err := NewAdapter(cfg)
	if err == nil {
		t.Error("NewAdapter() should return error when DID is missing")
	}
}

func TestAdapter_Name(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	if adapter.Name() != "sage" {
		t.Errorf("Name() = %v, want sage", adapter.Name())
	}
}

func TestAdapter_SupportsStreaming(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	// Streaming not supported in Phase 1
	if adapter.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false in Phase 1")
	}
}

func TestAdapter_SendMessage_WithoutEndpoint(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// SendMessage without endpoint should prepare message only (no error)
	err := adapter.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage() without endpoint should succeed: %v", err)
	}

	// Verify message was prepared
	if msg.Security == nil {
		t.Error("Message security metadata should be added")
	}
}

func TestAdapter_ReceiveMessage_NotImplemented(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	// ReceiveMessage not implemented in Phase 1
	_, err := adapter.ReceiveMessage(context.Background())
	if err == nil {
		t.Error("ReceiveMessage() should return error (not implemented)")
	}
}

func TestAdapter_Verify_MissingSecurityMetadata(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Message without security metadata should fail
	err := adapter.Verify(context.Background(), msg)
	if err == nil {
		t.Error("Verify() should return error when security metadata is missing")
	}
}

func TestAdapter_Stream_NotImplemented(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	fn := func(chunk string) error {
		return nil
	}

	// Streaming not implemented in Phase 1
	err := adapter.Stream(context.Background(), fn)
	if err == nil {
		t.Error("Stream() should return error (not implemented)")
	}
}

func TestAdapter_Verify_InvalidProtocolMode(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Set A2A protocol mode (should fail for SAGE adapter)
	msg.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeA2A,
		AgentDID:  "did:sage:eth:0xABC",
		Nonce:     "test_nonce",
		Timestamp: time.Now(),
	}

	err := adapter.Verify(context.Background(), msg)
	if err == nil {
		t.Error("Verify() should return error for wrong protocol mode")
	}
}

func TestAdapter_Verify_ExpiredTimestamp(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Set timestamp 10 minutes in the past (should fail with 5 min tolerance)
	msg.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  "did:sage:eth:0xABC",
		Nonce:     "test_nonce_123",
		Timestamp: time.Now().Add(-10 * time.Minute),
	}

	err := adapter.Verify(context.Background(), msg)
	if err == nil {
		t.Error("Verify() should return error for expired timestamp")
	}
}

func TestAdapter_Verify_FutureTimestamp(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Set timestamp 10 minutes in the future (should fail with 5 min tolerance)
	msg.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  "did:sage:eth:0xABC",
		Nonce:     "test_nonce_456",
		Timestamp: time.Now().Add(10 * time.Minute),
	}

	err := adapter.Verify(context.Background(), msg)
	if err == nil {
		t.Error("Verify() should return error for future timestamp")
	}
}

func TestAdapter_Verify_ReplayAttack(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	// Create message with same nonce
	nonce := "test_nonce_replay"

	msg1 := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test1")},
	)
	msg1.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  "did:sage:eth:0xABC",
		Nonce:     nonce,
		Timestamp: time.Now(),
	}

	msg2 := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test2")},
	)
	msg2.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  "did:sage:eth:0xABC",
		Nonce:     nonce, // Same nonce - replay attack
		Timestamp: time.Now(),
	}

	// First message should pass nonce check
	err := adapter.Verify(context.Background(), msg1)
	// May fail on signature verification, but should pass nonce check

	// Second message with same nonce should fail
	err = adapter.Verify(context.Background(), msg2)
	if err == nil {
		t.Error("Verify() should detect replay attack (duplicate nonce)")
	}
}

func TestAdapter_Verify_ValidSecurityMetadata(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Set valid security metadata (no signature)
	msg.Security = &types.SecurityMetadata{
		Mode:      types.ProtocolModeSAGE,
		AgentDID:  "did:sage:eth:0xABC",
		Nonce:     "test_nonce_valid",
		Timestamp: time.Now(),
		Signature: nil, // No signature to verify
	}

	err := adapter.Verify(context.Background(), msg)
	if err != nil {
		t.Errorf("Verify() should succeed with valid security metadata: %v", err)
	}
}

// ==================== SendMessage Tests ====================

func TestAdapter_SendMessage_NilMessage(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	err := adapter.SendMessage(context.Background(), nil)
	if err == nil {
		t.Error("SendMessage() should return error for nil message")
	}
}

func TestAdapter_SendMessage_AddsSecurityMetadata(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test message")},
	)

	// Message should not have security metadata initially
	if msg.Security != nil {
		t.Error("Message should not have security metadata initially")
	}

	// SendMessage should add security metadata (even though transport fails)
	_ = adapter.SendMessage(context.Background(), msg)

	// Check that security metadata was added
	if msg.Security == nil {
		t.Error("SendMessage() should add security metadata")
	}

	if msg.Security.Mode != types.ProtocolModeSAGE {
		t.Errorf("Security mode = %s, want SAGE", msg.Security.Mode)
	}

	if msg.Security.AgentDID != cfg.DID {
		t.Errorf("AgentDID = %s, want %s", msg.Security.AgentDID, cfg.DID)
	}

	if msg.Security.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	if msg.Security.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestAdapter_SendMessage_UniqueNonces(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	// Send multiple messages and collect nonces
	nonces := make(map[string]bool)
	for i := 0; i < 10; i++ {
		msg := types.NewMessage(
			types.MessageRoleUser,
			[]types.Part{types.NewTextPart("test")},
		)

		_ = adapter.SendMessage(context.Background(), msg)

		if msg.Security != nil {
			if nonces[msg.Security.Nonce] {
				t.Error("Duplicate nonce generated")
			}
			nonces[msg.Security.Nonce] = true
		}
	}

	if len(nonces) != 10 {
		t.Errorf("Expected 10 unique nonces, got %d", len(nonces))
	}
}

func TestAdapter_SendMessage_TimestampRecent(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	before := time.Now()
	_ = adapter.SendMessage(context.Background(), msg)
	after := time.Now()

	if msg.Security == nil {
		t.Fatal("Security metadata should be added")
	}

	// Timestamp should be between before and after
	if msg.Security.Timestamp.Before(before) || msg.Security.Timestamp.After(after) {
		t.Error("Timestamp should be recent (between before and after SendMessage)")
	}
}

// ==================== Message Signing Tests ====================

func TestAdapter_SendMessage_WithoutPrivateKey_NoSignature(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
		// No PrivateKeyPath - signing should be optional
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test message")},
	)

	_ = adapter.SendMessage(context.Background(), msg)

	// Security metadata should be added even without signing
	if msg.Security == nil {
		t.Fatal("Security metadata should be added")
	}

	// Signature should be nil when no private key is provided
	if msg.Security.Signature != nil {
		t.Error("Signature should be nil when no private key is provided")
	}
}

func TestAdapter_SendMessage_WithPrivateKey_AddsSignature(t *testing.T) {
	// Create temporary key file for testing
	tmpDir := t.TempDir()
	keyPath := tmpDir + "/test-key.jwk"

	// Generate and save test key
	km := NewKeyManager()
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err = km.SaveToFile(keyPair, keyPath)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Create adapter with private key
	cfg := &config.SAGEConfig{
		Enabled:        true,
		Network:        "ethereum",
		DID:            "did:sage:eth:0x1234567890abcdef",
		PrivateKeyPath: keyPath,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test message")},
	)

	_ = adapter.SendMessage(context.Background(), msg)

	// Security metadata should be added
	if msg.Security == nil {
		t.Fatal("Security metadata should be added")
	}

	// Signature should be present when private key is provided
	if msg.Security.Signature == nil {
		t.Error("Signature should be present when private key is provided")
	}

	// Verify signature fields
	if msg.Security.Signature.Algorithm == "" {
		t.Error("Signature algorithm should not be empty")
	}

	if msg.Security.Signature.KeyID == "" {
		t.Error("Signature key ID should not be empty")
	}

	if len(msg.Security.Signature.Signature) == 0 {
		t.Error("Signature bytes should not be empty")
	}

	if len(msg.Security.Signature.SignedFields) == 0 {
		t.Error("Signed fields should not be empty")
	}
}

func TestAdapter_SendMessage_SignedMessageCanBeVerified(t *testing.T) {
	// Create temporary key file for testing
	tmpDir := t.TempDir()
	keyPath := tmpDir + "/test-key.jwk"

	// Generate and save test key
	km := NewKeyManager()
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err = km.SaveToFile(keyPair, keyPath)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Create adapter with private key
	cfg := &config.SAGEConfig{
		Enabled:        true,
		Network:        "ethereum",
		DID:            "did:sage:eth:0x1234567890abcdef",
		PrivateKeyPath: keyPath,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Create and sign message
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test message")},
	)

	_ = adapter.SendMessage(context.Background(), msg)

	// Verify that signature was added
	if msg.Security == nil || msg.Security.Signature == nil {
		t.Fatal("Message should have security metadata with signature")
	}

	// Extract public key for verification
	privateKey, err := km.ExtractEd25519PrivateKey(keyPair)
	if err != nil {
		t.Fatalf("Failed to extract private key: %v", err)
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatal("Failed to convert public key to ed25519.PublicKey")
	}

	// Verify signature using signing manager
	signingManager := NewSigningManager()
	signatureEnvelope := &SignatureEnvelope{
		Algorithm: string(msg.Security.Signature.Algorithm),
		KeyID:     msg.Security.Signature.KeyID,
		Value:     base64Encode(msg.Security.Signature.Signature),
	}

	err = signingManager.VerifySignature(msg, signatureEnvelope, publicKey)
	if err != nil {
		t.Errorf("Signature verification failed: %v", err)
	}
}

func TestAdapter_SendMessage_SignatureKeyID_MatchesDID(t *testing.T) {
	// Create temporary key file for testing
	tmpDir := t.TempDir()
	keyPath := tmpDir + "/test-key.jwk"

	// Generate and save test key
	km := NewKeyManager()
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	err = km.SaveToFile(keyPair, keyPath)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Create adapter with private key
	did := "did:sage:eth:0x1234567890abcdef"
	cfg := &config.SAGEConfig{
		Enabled:        true,
		Network:        "ethereum",
		DID:            did,
		PrivateKeyPath: keyPath,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test message")},
	)

	_ = adapter.SendMessage(context.Background(), msg)

	// Verify key ID matches DID pattern
	if msg.Security == nil || msg.Security.Signature == nil {
		t.Fatal("Message should have security metadata with signature")
	}

	expectedKeyID := did + "#key-1"
	if msg.Security.Signature.KeyID != expectedKeyID {
		t.Errorf("Key ID = %s, want %s", msg.Security.Signature.KeyID, expectedKeyID)
	}
}
