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
	"sync"
	"testing"
	"time"

	adkconfig "github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// TestIntegration_FullCommunicationFlow tests the complete SAGE communication flow
// from connection establishment to bidirectional message exchange.
func TestIntegration_FullCommunicationFlow(t *testing.T) {
	// Setup Alice
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	// Setup Bob
	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Step 1: Alice initiates connection
	invitation, err := alice.Connect(ctx, "did:sage:bob")
	if err != nil {
		t.Fatalf("Alice Connect failed: %v", err)
	}
	t.Logf("Alice initiated connection with invitation")

	// Step 2: Bob receives invitation and responds
	request, err := bob.HandleInvitation(ctx, invitation)
	if err != nil {
		t.Fatalf("Bob HandleInvitation failed: %v", err)
	}
	t.Logf("Bob processed invitation and sent request")

	// Step 3: Alice receives request and responds
	response, err := alice.HandleRequest(ctx, request, bobPublicKey)
	if err != nil {
		t.Fatalf("Alice HandleRequest failed: %v", err)
	}
	t.Logf("Alice processed request and sent response")

	// Step 4: Bob receives response and completes handshake
	complete, err := bob.HandleResponse(ctx, response, alicePublicKey)
	if err != nil {
		t.Fatalf("Bob HandleResponse failed: %v", err)
	}
	t.Logf("Bob processed response and sent complete")

	// Step 5: Alice receives complete and activates session
	err = alice.HandleComplete(ctx, complete, bobPublicKey)
	if err != nil {
		t.Fatalf("Alice HandleComplete failed: %v", err)
	}
	t.Logf("Alice processed complete - handshake finished")

	// Verify sessions are active
	aliceSession, err := alice.GetSession("did:sage:bob")
	if err != nil {
		t.Fatalf("Failed to get Alice's session: %v", err)
	}
	if !aliceSession.IsActive() {
		t.Error("Alice's session is not active")
	}

	bobSession, err := bob.GetSession("did:sage:alice")
	if err != nil {
		t.Fatalf("Failed to get Bob's session: %v", err)
	}
	if !bobSession.IsActive() {
		t.Error("Bob's session is not active")
	}
	t.Logf("Both sessions are active")

	// Step 6: Alice sends message to Bob
	aliceMessage := map[string]interface{}{
		"type":    "greeting",
		"message": "Hello Bob!",
		"data": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"count":     1,
		},
	}

	aliceAppMsg, err := alice.SendMessage(ctx, "did:sage:bob", aliceMessage)
	if err != nil {
		t.Fatalf("Alice SendMessage failed: %v", err)
	}
	t.Logf("Alice sent encrypted message to Bob")

	// Setup Bob's message handler
	var bobReceivedPayload []byte
	var bobReceivedFrom string
	bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		bobReceivedFrom = fromDID
		bobReceivedPayload = payload
		return nil
	})

	// Step 7: Bob receives message from Alice
	err = bob.ReceiveMessage(ctx, aliceAppMsg, alicePublicKey)
	if err != nil {
		t.Fatalf("Bob ReceiveMessage failed: %v", err)
	}

	if bobReceivedFrom != "did:sage:alice" {
		t.Errorf("Bob received from %s, want did:sage:alice", bobReceivedFrom)
	}

	var bobDecodedMsg map[string]interface{}
	err = DeserializeMessage(bobReceivedPayload, &bobDecodedMsg)
	if err != nil {
		t.Fatalf("Failed to deserialize Bob's received message: %v", err)
	}

	if bobDecodedMsg["message"] != "Hello Bob!" {
		t.Errorf("Bob received message = %v, want 'Hello Bob!'", bobDecodedMsg["message"])
	}
	t.Logf("Bob received and decrypted message from Alice")

	// Step 8: Bob sends reply to Alice
	bobMessage := map[string]interface{}{
		"type":    "reply",
		"message": "Hi Alice! Nice to meet you.",
		"status":  "received",
	}

	bobAppMsg, err := bob.SendMessage(ctx, "did:sage:alice", bobMessage)
	if err != nil {
		t.Fatalf("Bob SendMessage failed: %v", err)
	}
	t.Logf("Bob sent encrypted reply to Alice")

	// Setup Alice's message handler
	var aliceReceivedPayload []byte
	var aliceReceivedFrom string
	alice.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		aliceReceivedFrom = fromDID
		aliceReceivedPayload = payload
		return nil
	})

	// Step 9: Alice receives reply from Bob
	err = alice.ReceiveMessage(ctx, bobAppMsg, bobPublicKey)
	if err != nil {
		t.Fatalf("Alice ReceiveMessage failed: %v", err)
	}

	if aliceReceivedFrom != "did:sage:bob" {
		t.Errorf("Alice received from %s, want did:sage:bob", aliceReceivedFrom)
	}

	var aliceDecodedMsg map[string]interface{}
	err = DeserializeMessage(aliceReceivedPayload, &aliceDecodedMsg)
	if err != nil {
		t.Fatalf("Failed to deserialize Alice's received message: %v", err)
	}

	if aliceDecodedMsg["message"] != "Hi Alice! Nice to meet you." {
		t.Errorf("Alice received message = %v", aliceDecodedMsg["message"])
	}
	t.Logf("Alice received and decrypted reply from Bob")

	// Step 10: Verify session keys match
	for i := range aliceSession.SessionKey {
		if aliceSession.SessionKey[i] != bobSession.SessionKey[i] {
			t.Errorf("Session keys don't match at index %d", i)
			break
		}
	}
	t.Logf("Session keys are identical")

	t.Log("Full communication flow completed successfully")
}

// TestIntegration_MultipleMessages tests sending multiple messages in sequence.
func TestIntegration_MultipleMessages(t *testing.T) {
	// Setup Alice and Bob with established session
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Establish session
	invitation, _ := alice.Connect(ctx, "did:sage:bob")
	request, _ := bob.HandleInvitation(ctx, invitation)
	response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
	complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
	alice.HandleComplete(ctx, complete, bobPublicKey)

	// Track received messages
	var receivedMessages []map[string]interface{}
	var mu sync.Mutex

	bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		var msg map[string]interface{}
		DeserializeMessage(payload, &msg)
		mu.Lock()
		receivedMessages = append(receivedMessages, msg)
		mu.Unlock()
		return nil
	})

	// Send multiple messages
	messageCount := 10
	for i := 0; i < messageCount; i++ {
		message := map[string]interface{}{
			"sequence": i,
			"data":     "Message number " + string(rune('A'+i)),
		}

		appMsg, err := alice.SendMessage(ctx, "did:sage:bob", message)
		if err != nil {
			t.Fatalf("SendMessage %d failed: %v", i, err)
		}

		err = bob.ReceiveMessage(ctx, appMsg, alicePublicKey)
		if err != nil {
			t.Fatalf("ReceiveMessage %d failed: %v", i, err)
		}
	}

	// Verify all messages received
	mu.Lock()
	defer mu.Unlock()

	if len(receivedMessages) != messageCount {
		t.Errorf("Received %d messages, want %d", len(receivedMessages), messageCount)
	}

	for i, msg := range receivedMessages {
		seq := int(msg["sequence"].(float64))
		if seq != i {
			t.Errorf("Message %d has sequence %d", i, seq)
		}
	}

	t.Logf("Successfully sent and received %d messages", messageCount)
}

// TestIntegration_ConcurrentConnections tests multiple concurrent agent connections.
func TestIntegration_ConcurrentConnections(t *testing.T) {
	// Setup Alice
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	ctx := context.Background()

	// Create multiple peer agents
	peerCount := 5
	peers := make([]*TransportManager, peerCount)
	peerKeys := make([]ed25519.PublicKey, peerCount)

	for i := 0; i < peerCount; i++ {
		pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
		did := "did:sage:peer" + string(rune('A'+i))
		peers[i] = NewTransportManager(did, privKey, nil)
		peerKeys[i] = pubKey
	}

	// Establish connections concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, peerCount)

	for i := 0; i < peerCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			peer := peers[idx]
			peerPublicKey := peerKeys[idx]

			// Full handshake
			invitation, err := alice.Connect(ctx, peer.localDID)
			if err != nil {
				errChan <- err
				return
			}

			request, err := peer.HandleInvitation(ctx, invitation)
			if err != nil {
				errChan <- err
				return
			}

			response, err := alice.HandleRequest(ctx, request, peerPublicKey)
			if err != nil {
				errChan <- err
				return
			}

			complete, err := peer.HandleResponse(ctx, response, alicePublicKey)
			if err != nil {
				errChan <- err
				return
			}

			err = alice.HandleComplete(ctx, complete, peerPublicKey)
			if err != nil {
				errChan <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			t.Fatalf("Concurrent connection failed: %v", err)
		}
	}

	// Verify all sessions established
	sessions := alice.ListSessions()
	if len(sessions) != peerCount {
		t.Errorf("Alice has %d sessions, want %d", len(sessions), peerCount)
	}

	for _, session := range sessions {
		if !session.IsActive() {
			t.Errorf("Session %s is not active", session.ID)
		}
	}

	t.Logf("Successfully established %d concurrent connections", peerCount)
}

// TestIntegration_SessionExpiration tests session expiration and cleanup.
func TestIntegration_SessionExpiration(t *testing.T) {
	// Setup with short session TTL
	config := DefaultTransportConfig()
	config.SessionTTL = 100 * time.Millisecond

	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, config)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, config)

	ctx := context.Background()

	// Establish session
	invitation, _ := alice.Connect(ctx, "did:sage:bob")
	request, _ := bob.HandleInvitation(ctx, invitation)
	response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
	complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
	alice.HandleComplete(ctx, complete, bobPublicKey)

	// Verify session is active
	session, err := alice.GetSession("did:sage:bob")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if !session.IsActive() {
		t.Error("Session should be active")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Session should be expired
	if session.IsActive() {
		t.Error("Session should be expired")
	}

	// Sending message should fail
	_, err = alice.SendMessage(ctx, "did:sage:bob", map[string]string{"test": "data"})
	if err == nil {
		t.Error("SendMessage should fail with expired session")
	}

	t.Log("Session expiration works correctly")
}

// TestIntegration_MessageTypes tests different message payload types.
func TestIntegration_MessageTypes(t *testing.T) {
	// Setup Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	alice := NewTransportManager("did:sage:alice", alicePrivateKey, nil)

	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bob := NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Establish session
	invitation, _ := alice.Connect(ctx, "did:sage:bob")
	request, _ := bob.HandleInvitation(ctx, invitation)
	response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
	complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
	alice.HandleComplete(ctx, complete, bobPublicKey)

	// Test different message types
	testCases := []struct {
		name    string
		payload interface{}
	}{
		{
			name: "string map",
			payload: map[string]string{
				"type": "text",
				"data": "Hello World",
			},
		},
		{
			name: "mixed map",
			payload: map[string]interface{}{
				"type":    "mixed",
				"number":  42,
				"boolean": true,
				"array":   []string{"a", "b", "c"},
			},
		},
		{
			name: "nested structure",
			payload: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  30,
					"tags": []string{"developer", "crypto"},
				},
				"metadata": map[string]string{
					"version": "1.0",
					"type":    "profile",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var receivedPayload []byte

			bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
				receivedPayload = payload
				return nil
			})

			// Send message
			appMsg, err := alice.SendMessage(ctx, "did:sage:bob", tc.payload)
			if err != nil {
				t.Fatalf("SendMessage failed: %v", err)
			}

			// Receive message
			err = bob.ReceiveMessage(ctx, appMsg, alicePublicKey)
			if err != nil {
				t.Fatalf("ReceiveMessage failed: %v", err)
			}

			// Verify received
			if len(receivedPayload) == 0 {
				t.Error("No payload received")
			}

			var decoded map[string]interface{}
			err = DeserializeMessage(receivedPayload, &decoded)
			if err != nil {
				t.Fatalf("Failed to deserialize: %v", err)
			}

			t.Logf("Successfully transmitted %s payload", tc.name)
		})
	}
}

// ==================== Adapter + Network Layer Integration Tests ====================

// TestEndToEnd_AdapterMessageTransmission tests complete message sending and receiving
// using Adapter + NetworkClient.
func TestEndToEnd_AdapterMessageTransmission(t *testing.T) {
	// Create temporary key files for both agents
	tmpDir := t.TempDir()

	// Generate keys for sender
	senderKeyPath := tmpDir + "/sender-key.jwk"
	senderKM := NewKeyManager()
	senderKeyPair, err := senderKM.Generate()
	if err != nil {
		t.Fatalf("Failed to generate sender key: %v", err)
	}
	if err := senderKM.SaveToFile(senderKeyPair, senderKeyPath); err != nil {
		t.Fatalf("Failed to save sender key: %v", err)
	}

	// Generate keys for receiver
	receiverKeyPath := tmpDir + "/receiver-key.jwk"
	receiverKM := NewKeyManager()
	receiverKeyPair, err := receiverKM.Generate()
	if err != nil {
		t.Fatalf("Failed to generate receiver key: %v", err)
	}
	if err := receiverKM.SaveToFile(receiverKeyPair, receiverKeyPath); err != nil {
		t.Fatalf("Failed to save receiver key: %v", err)
	}

	// Track received messages
	var receivedMessage *types.Message
	var mu sync.Mutex

	// Create message handler
	handler := func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		mu.Lock()
		receivedMessage = msg
		mu.Unlock()
		return nil, nil
	}

	// Start receiver server
	server := NewNetworkServer(":18082", handler)
	go func() {
		if err := server.Start(); err != nil && err.Error() != "http: Server closed" {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Ensure server cleanup
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	// Create sender adapter
	senderCfg := &adkconfig.SAGEConfig{
		Enabled:        true,
		Network:        "ethereum",
		DID:            "did:sage:eth:0xSENDER",
		PrivateKeyPath: senderKeyPath,
	}

	senderAdapter, err := NewAdapter(senderCfg)
	if err != nil {
		t.Fatalf("Failed to create sender adapter: %v", err)
	}

	// Set remote endpoint
	senderAdapter.SetRemoteEndpoint("http://localhost:18082/sage/message")

	// Create test message
	testMessage := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("Hello from integration test!")},
	)

	// Send message
	err = senderAdapter.SendMessage(context.Background(), testMessage)
	if err != nil {
		t.Fatalf("SendMessage() failed: %v", err)
	}

	// Wait for message to be received
	time.Sleep(100 * time.Millisecond)

	// Verify message was received
	mu.Lock()
	defer mu.Unlock()

	if receivedMessage == nil {
		t.Fatal("No message was received")
	}

	// Verify message content
	if len(receivedMessage.Parts) == 0 {
		t.Fatal("Received message has no parts")
	}

	textPart, ok := receivedMessage.Parts[0].(*types.TextPart)
	if !ok {
		t.Fatal("First part is not a text part")
	}

	if textPart.Text != "Hello from integration test!" {
		t.Errorf("Received text = %s, want 'Hello from integration test!'", textPart.Text)
	}

	// Verify security metadata was added
	if receivedMessage.Security == nil {
		t.Fatal("Received message has no security metadata")
	}

	if receivedMessage.Security.Mode != types.ProtocolModeSAGE {
		t.Errorf("Security mode = %s, want SAGE", receivedMessage.Security.Mode)
	}

	if receivedMessage.Security.AgentDID != senderCfg.DID {
		t.Errorf("Agent DID = %s, want %s", receivedMessage.Security.AgentDID, senderCfg.DID)
	}

	if receivedMessage.Security.Nonce == "" {
		t.Error("Nonce should not be empty")
	}

	if receivedMessage.Security.Signature == nil {
		t.Error("Message should be signed")
	}

	t.Logf("✅ End-to-end test passed: message sent, received, and verified")
}

// TestEndToEnd_WithoutEndpoint tests that messages can be prepared without endpoint.
func TestEndToEnd_WithoutEndpoint(t *testing.T) {
	// Create adapter without endpoint
	cfg := &adkconfig.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0xTEST",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Create message
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("Test message")},
	)

	// Send message without endpoint (should prepare only)
	err = adapter.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage() should succeed without endpoint: %v", err)
	}

	// Verify security metadata was added
	if msg.Security == nil {
		t.Fatal("Security metadata should be added")
	}

	if msg.Security.Mode != types.ProtocolModeSAGE {
		t.Errorf("Security mode = %s, want SAGE", msg.Security.Mode)
	}

	t.Logf("✅ Message prepared successfully without network transmission")
}

// TestEndToEnd_SetRemoteEndpoint tests endpoint configuration.
func TestEndToEnd_SetRemoteEndpoint(t *testing.T) {
	cfg := &adkconfig.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0xTEST",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Initially no endpoint
	if adapter.GetRemoteEndpoint() != "" {
		t.Error("Initial endpoint should be empty")
	}

	// Set endpoint
	testEndpoint := "http://test.example.com/sage/message"
	adapter.SetRemoteEndpoint(testEndpoint)

	// Verify endpoint was set
	if adapter.GetRemoteEndpoint() != testEndpoint {
		t.Errorf("Endpoint = %s, want %s", adapter.GetRemoteEndpoint(), testEndpoint)
	}

	t.Logf("✅ Endpoint configuration works correctly")
}
