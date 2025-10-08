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
	"path/filepath"
	"testing"
	"time"

	adkconfig "github.com/sage-x-project/sage-adk/config"
)

func TestNewTransportManagerFromConfig(t *testing.T) {
	// Create temp directory for test keys
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key.pem")

	// Generate and save a test key
	km := NewKeyManager()
	keyPair, err := km.Generate()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if err := km.SaveToFile(keyPair, keyPath); err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Create test config
	adkCfg := &adkconfig.SAGEConfig{
		Enabled:         true,
		Network:         "sepolia",
		DID:             "did:sage:sepolia:0x123",
		RPCEndpoint:     "https://test.example.com",
		ContractAddress: "0xABC",
		PrivateKeyPath:  keyPath,
		CacheTTL:        1 * time.Hour,
	}

	cfg, err := FromADKConfig(adkCfg)
	if err != nil {
		t.Fatalf("FromADKConfig() failed: %v", err)
	}

	// Test successful creation
	tm, err := NewTransportManagerFromConfig(cfg, km)
	if err != nil {
		t.Fatalf("NewTransportManagerFromConfig() error = %v", err)
	}

	if tm == nil {
		t.Fatal("NewTransportManagerFromConfig() returned nil")
	}

	if tm.localDID != cfg.LocalDID {
		t.Errorf("localDID = %s, want %s", tm.localDID, cfg.LocalDID)
	}

	// Test with nil config
	_, err = NewTransportManagerFromConfig(nil, km)
	if err == nil {
		t.Error("NewTransportManagerFromConfig() should fail with nil config")
	}

	// Test with nil key manager
	_, err = NewTransportManagerFromConfig(cfg, nil)
	if err == nil {
		t.Error("NewTransportManagerFromConfig() should fail with nil key manager")
	}

	// Test with invalid key path
	invalidCfg := &Config{
		Config:         cfg.Config,
		LocalDID:       cfg.LocalDID,
		PrivateKeyPath: "/nonexistent/key.pem",
	}
	_, err = NewTransportManagerFromConfig(invalidCfg, km)
	if err == nil {
		t.Error("NewTransportManagerFromConfig() should fail with invalid key path")
	}
}

func TestNewTransportManager(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	tm := NewTransportManager("did:sage:alice", privateKey, nil)
	if tm == nil {
		t.Fatal("NewTransportManager returned nil")
	}

	if tm.localDID != "did:sage:alice" {
		t.Errorf("localDID = %s, want did:sage:alice", tm.localDID)
	}

	if tm.config == nil {
		t.Error("config is nil")
	}

	if tm.activeHandshakes == nil {
		t.Error("activeHandshakes map is nil")
	}
}

func TestTransportManager_Connect(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	tm := NewTransportManager("did:sage:alice", privateKey, nil)

	ctx := context.Background()
	invitation, err := tm.Connect(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if invitation.FromDID != "did:sage:alice" {
		t.Errorf("FromDID = %s, want did:sage:alice", invitation.FromDID)
	}

	if invitation.ToDID != "did:sage:bob" {
		t.Errorf("ToDID = %s, want did:sage:bob", invitation.ToDID)
	}

	// Check that handshake is tracked
	tm.mu.RLock()
	state, exists := tm.activeHandshakes["did:sage:bob"]
	tm.mu.RUnlock()

	if !exists {
		t.Error("Handshake not tracked")
	}

	if state.Phase != PhaseInvitation {
		t.Errorf("Phase = %s, want %s", state.Phase, PhaseInvitation)
	}
}

func TestTransportManager_Connect_AlreadyConnected(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	tm := NewTransportManager("did:sage:alice", privateKey, nil)

	ctx := context.Background()

	// First connection
	_, err := tm.Connect(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("First Connect failed: %v", err)
	}

	// Try to connect again while handshake in progress
	_, err = tm.Connect(ctx, "did:sage:bob")
	if err == nil {
		t.Error("Connect should fail when handshake already in progress")
	}
}

func TestTransportManager_FullHandshakeFlow(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Phase 1: Alice initiates
	invitation, err := alice.Connect(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("Phase 1 failed: %v", err)
	}

	// Phase 2: Bob processes invitation
	request, err := bob.HandleInvitation(ctx, invitation)
	if err != nil {
		t.Fatalf("Phase 2 failed: %v", err)
	}

	// Phase 3: Alice processes request
	response, err := alice.HandleRequest(ctx, request, bobPublicKey)
	if err != nil {
		t.Fatalf("Phase 3 failed: %v", err)
	}

	// Phase 4: Bob processes response
	complete, err := bob.HandleResponse(ctx, response, alicePublicKey)
	if err != nil {
		t.Fatalf("Phase 4 failed: %v", err)
	}

	// Alice processes complete
	err = alice.HandleComplete(ctx, complete, bobPublicKey)
	if err != nil {
		t.Fatalf("Phase 4 complete failed: %v", err)
	}

	// Verify Alice's session is active
	aliceSession, err := alice.GetSession("did:sage:bob")
	if err != nil {
		t.Fatalf("Failed to get Alice's session: %v", err)
	}

	if !aliceSession.IsActive() {
		t.Error("Alice's session is not active")
	}

	// Verify Bob's session exists
	bobSession, err := bob.GetSession("did:sage:alice")
	if err != nil {
		t.Fatalf("Failed to get Bob's session: %v", err)
	}

	if bobSession.Status != SessionEstablishing && bobSession.Status != SessionActive {
		t.Errorf("Bob's session has unexpected status: %d", bobSession.Status)
	}

	// Verify both have session keys
	if len(aliceSession.SessionKey) != 32 {
		t.Errorf("Alice's session key length = %d, want 32", len(aliceSession.SessionKey))
	}

	if len(bobSession.SessionKey) != 32 {
		t.Errorf("Bob's session key length = %d, want 32", len(bobSession.SessionKey))
	}

	// Session keys should be identical
	for i := range aliceSession.SessionKey {
		if aliceSession.SessionKey[i] != bobSession.SessionKey[i] {
			t.Errorf("Session keys don't match at index %d", i)
			break
		}
	}
}

func TestTransportManager_SendReceiveMessage(t *testing.T) {
	// Setup Alice and Bob with established session
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Perform full handshake
	invitation, _ := alice.Connect(ctx, "did:sage:bob")
	request, _ := bob.HandleInvitation(ctx, invitation)
	response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
	complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
	alice.HandleComplete(ctx, complete, bobPublicKey)

	// Test message sending
	testPayload := map[string]interface{}{
		"type":    "greeting",
		"message": "Hello Bob!",
		"data":    []string{"foo", "bar", "baz"},
	}

	message, err := alice.SendMessage(ctx, "did:sage:bob", testPayload)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if message.FromDID != "did:sage:alice" {
		t.Errorf("FromDID = %s, want did:sage:alice", message.FromDID)
	}

	if message.ToDID != "did:sage:bob" {
		t.Errorf("ToDID = %s, want did:sage:bob", message.ToDID)
	}

	if message.Signature.Value == "" {
		t.Error("Message signature is empty")
	}

	// Test message receiving with handler
	var receivedPayload []byte
	var receivedFrom string

	bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		receivedFrom = fromDID
		receivedPayload = payload
		return nil
	})

	err = bob.ReceiveMessage(ctx, message, alicePublicKey)
	if err != nil {
		t.Fatalf("ReceiveMessage failed: %v", err)
	}

	if receivedFrom != "did:sage:alice" {
		t.Errorf("receivedFrom = %s, want did:sage:alice", receivedFrom)
	}

	if len(receivedPayload) == 0 {
		t.Error("receivedPayload is empty")
	}

	// Verify payload contents
	var decoded map[string]interface{}
	if err := DeserializeMessage(receivedPayload, &decoded); err != nil {
		t.Fatalf("Failed to deserialize payload: %v", err)
	}

	if decoded["message"] != "Hello Bob!" {
		t.Errorf("message = %v, want Hello Bob!", decoded["message"])
	}
}

func TestTransportManager_SendMessage_NoSession(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	tm := NewTransportManager("did:sage:alice", privateKey, nil)

	ctx := context.Background()
	payload := map[string]string{"test": "data"}

	_, err := tm.SendMessage(ctx, "did:sage:bob", payload)
	if err == nil {
		t.Error("SendMessage should fail when no session exists")
	}
}

func TestTransportManager_ReceiveMessage_InvalidSignature(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Establish session between Alice and Bob
	invitation, _ := alice.Connect(ctx, "did:sage:bob")
	request, _ := bob.HandleInvitation(ctx, invitation)
	response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
	complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
	alice.HandleComplete(ctx, complete, bobPublicKey)

	// Alice sends a message
	message, _ := alice.SendMessage(ctx, "did:sage:bob", map[string]string{"test": "data"})

	// Eve tries to receive the message with wrong public key
	evePublicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	err := bob.ReceiveMessage(ctx, message, evePublicKey)
	if err == nil {
		t.Error("ReceiveMessage should fail with wrong public key")
	}
}

func TestTransportManager_Disconnect(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	tm := NewTransportManager("did:sage:alice", privateKey, nil)

	ctx := context.Background()

	// Connect
	_, err := tm.Connect(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Disconnect
	err = tm.Disconnect(ctx, "did:sage:bob")
	if err != nil {
		t.Errorf("Disconnect failed: %v", err)
	}

	// Verify handshake is removed
	tm.mu.RLock()
	_, exists := tm.activeHandshakes["did:sage:bob"]
	tm.mu.RUnlock()

	if exists {
		t.Error("Handshake should be removed after disconnect")
	}

	// Disconnect non-existent session should not error
	err = tm.Disconnect(ctx, "did:sage:nonexistent")
	if err != nil {
		t.Errorf("Disconnect non-existent should not error: %v", err)
	}
}

func TestTransportManager_ListSessions(t *testing.T) {
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	charliePublicKey, charliePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	charlie := NewTransportManager("did:sage:charlie", charliePrivateKey, nil)

	ctx := context.Background()

	// Establish session with Bob
	invitation1, _ := alice.Connect(ctx, "did:sage:bob")
	request1, _ := bob.HandleInvitation(ctx, invitation1)
	response1, _ := alice.HandleRequest(ctx, request1, bobPublicKey)
	complete1, _ := bob.HandleResponse(ctx, response1, alicePublicKey)
	alice.HandleComplete(ctx, complete1, bobPublicKey)

	// Establish session with Charlie
	invitation2, _ := alice.Connect(ctx, "did:sage:charlie")
	request2, _ := charlie.HandleInvitation(ctx, invitation2)
	response2, _ := alice.HandleRequest(ctx, request2, charliePublicKey)
	complete2, _ := charlie.HandleResponse(ctx, response2, alicePublicKey)
	alice.HandleComplete(ctx, complete2, charliePublicKey)

	// List sessions
	sessions := alice.ListSessions()
	if len(sessions) != 2 {
		t.Errorf("ListSessions returned %d sessions, want 2", len(sessions))
	}
}

func TestTransportManager_Close(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	tm := NewTransportManager("did:sage:alice", privateKey, nil)

	err := tm.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestWrapUnwrapMessage(t *testing.T) {
	original := map[string]string{
		"type":    "test",
		"message": "Hello",
	}

	// Wrap
	envelope, err := WrapMessage("test_message", original)
	if err != nil {
		t.Fatalf("WrapMessage failed: %v", err)
	}

	if envelope.Type != "test_message" {
		t.Errorf("Type = %s, want test_message", envelope.Type)
	}

	// Unwrap
	var decoded map[string]string
	err = UnwrapMessage(envelope, &decoded)
	if err != nil {
		t.Fatalf("UnwrapMessage failed: %v", err)
	}

	if decoded["type"] != "test" {
		t.Errorf("type = %s, want test", decoded["type"])
	}

	if decoded["message"] != "Hello" {
		t.Errorf("message = %s, want Hello", decoded["message"])
	}
}

func TestSerializeDeserializeMessage(t *testing.T) {
	original := map[string]interface{}{
		"string": "test",
		"number": float64(42),
		"array":  []interface{}{"a", "b", "c"},
	}

	// Serialize
	data, err := SerializeMessage(original)
	if err != nil {
		t.Fatalf("SerializeMessage failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Serialized data is empty")
	}

	// Deserialize
	var decoded map[string]interface{}
	err = DeserializeMessage(data, &decoded)
	if err != nil {
		t.Fatalf("DeserializeMessage failed: %v", err)
	}

	if decoded["string"] != "test" {
		t.Errorf("string = %v, want test", decoded["string"])
	}

	if decoded["number"] != float64(42) {
		t.Errorf("number = %v, want 42", decoded["number"])
	}
}

func TestEncodeDecodeMessage(t *testing.T) {
	original := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}

	// Encode
	encoded, err := EncodeMessage(original)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	if encoded == "" {
		t.Error("Encoded message is empty")
	}

	// Decode
	var decoded map[string]string
	err = DecodeMessage(encoded, &decoded)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	if decoded["field1"] != "value1" {
		t.Errorf("field1 = %s, want value1", decoded["field1"])
	}

	if decoded["field2"] != "value2" {
		t.Errorf("field2 = %s, want value2", decoded["field2"])
	}
}

func TestDefaultTransportConfig(t *testing.T) {
	config := DefaultTransportConfig()

	if config.MaxClockSkew != 5*time.Minute {
		t.Errorf("MaxClockSkew = %v, want 5m", config.MaxClockSkew)
	}

	if config.HandshakeTimeout != 30*time.Second {
		t.Errorf("HandshakeTimeout = %v, want 30s", config.HandshakeTimeout)
	}

	if config.SessionTTL != 1*time.Hour {
		t.Errorf("SessionTTL = %v, want 1h", config.SessionTTL)
	}

	if config.MaxMessageSize != 10*1024*1024 {
		t.Errorf("MaxMessageSize = %d, want 10MB", config.MaxMessageSize)
	}
}
