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

package sage_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/sage"
)

// Example_basicUsage demonstrates basic SAGE transport usage.
func Example_basicUsage() {
	// Generate key pairs for Alice and Bob
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)

	// Create transport managers
	alice := sage.NewTransportManager("did:sage:alice", alicePrivateKey, nil)
	bob := sage.NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Step 1: Alice initiates connection
	invitation, err := alice.Connect(ctx, "did:sage:bob")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Alice initiated connection")

	// Step 2: Bob processes invitation
	request, err := bob.HandleInvitation(ctx, invitation)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Bob responded to invitation")

	// Step 3: Alice processes request
	response, err := alice.HandleRequest(ctx, request, bobPublicKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Alice sent response")

	// Step 4: Bob completes handshake
	complete, err := bob.HandleResponse(ctx, response, alicePublicKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Bob completed handshake")

	// Step 5: Alice activates session
	err = alice.HandleComplete(ctx, complete, bobPublicKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Session established")

	// Step 6: Send encrypted message
	message := map[string]interface{}{
		"type": "greeting",
		"text": "Hello from Alice!",
	}

	appMsg, err := alice.SendMessage(ctx, "did:sage:bob", message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Alice sent encrypted message")

	// Step 7: Bob receives and handles message
	bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		var msg map[string]interface{}
		sage.DeserializeMessage(payload, &msg)
		fmt.Printf("Bob received: %s\n", msg["text"])
		return nil
	})

	err = bob.ReceiveMessage(ctx, appMsg, alicePublicKey)
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// Alice initiated connection
	// Bob responded to invitation
	// Alice sent response
	// Bob completed handshake
	// Session established
	// Alice sent encrypted message
	// Bob received: Hello from Alice!
}

// Example_bidirectionalCommunication demonstrates two-way message exchange.
func Example_bidirectionalCommunication() {
	// Setup
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)

	alice := sage.NewTransportManager("did:sage:alice", alicePrivateKey, nil)
	bob := sage.NewTransportManager("did:sage:bob", bobPrivateKey, nil)

	ctx := context.Background()

	// Establish session (handshake omitted for brevity)
	invitation, _ := alice.Connect(ctx, "did:sage:bob")
	request, _ := bob.HandleInvitation(ctx, invitation)
	response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
	complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
	alice.HandleComplete(ctx, complete, bobPublicKey)

	// Setup message handlers
	alice.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		var msg map[string]interface{}
		sage.DeserializeMessage(payload, &msg)
		fmt.Printf("Alice received: %s\n", msg["text"])
		return nil
	})

	bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
		var msg map[string]interface{}
		sage.DeserializeMessage(payload, &msg)
		fmt.Printf("Bob received: %s\n", msg["text"])
		return nil
	})

	// Alice sends message to Bob
	aliceMsg := map[string]interface{}{"type": "query", "text": "How are you?"}
	appMsg1, _ := alice.SendMessage(ctx, "did:sage:bob", aliceMsg)
	bob.ReceiveMessage(ctx, appMsg1, alicePublicKey)

	// Bob replies to Alice
	bobMsg := map[string]interface{}{"type": "response", "text": "I'm doing great!"}
	appMsg2, _ := bob.SendMessage(ctx, "did:sage:alice", bobMsg)
	alice.ReceiveMessage(ctx, appMsg2, bobPublicKey)

	// Output:
	// Bob received: How are you?
	// Alice received: I'm doing great!
}

// Example_messageWrapping demonstrates message envelope usage.
func Example_messageWrapping() {
	// Create a typed message
	payload := map[string]interface{}{
		"action": "transfer",
		"amount": 100,
		"token":  "ETH",
	}

	// Wrap message with type
	envelope, err := sage.WrapMessage("transaction", payload)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Message type: %s\n", envelope.Type)

	// Unwrap message
	var unwrapped map[string]interface{}
	err = sage.UnwrapMessage(envelope, &unwrapped)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Action: %s\n", unwrapped["action"])
	fmt.Printf("Amount: %.0f\n", unwrapped["amount"])

	// Output:
	// Message type: transaction
	// Action: transfer
	// Amount: 100
}

// Example_sessionManagement demonstrates session lifecycle management.
func Example_sessionManagement() {
	alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	charliePublicKey, charliePrivateKey, _ := ed25519.GenerateKey(rand.Reader)

	alice := sage.NewTransportManager("did:sage:alice", alicePrivateKey, nil)
	bob := sage.NewTransportManager("did:sage:bob", bobPrivateKey, nil)
	charlie := sage.NewTransportManager("did:sage:charlie", charliePrivateKey, nil)

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

	// List all sessions
	sessions := alice.ListSessions()
	fmt.Printf("Alice has %d active sessions\n", len(sessions))

	// Sort sessions by RemoteDID for deterministic output
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].RemoteDID < sessions[j].RemoteDID
	})

	for _, session := range sessions {
		fmt.Printf("- Session with %s: sess_...\n", session.RemoteDID)
	}

	// Get specific session
	bobSession, _ := alice.GetSession("did:sage:bob")
	fmt.Printf("Bob's session is active: %v\n", bobSession.IsActive())

	// Disconnect from Bob
	alice.Disconnect(ctx, "did:sage:bob")
	sessions = alice.ListSessions()
	fmt.Printf("After disconnect: %d sessions\n", len(sessions))

	// Output:
	// Alice has 2 active sessions
	// - Session with did:sage:bob: sess_...
	// - Session with did:sage:charlie: sess_...
	// Bob's session is active: true
	// After disconnect: 1 sessions
}

// Example_customConfiguration demonstrates custom transport configuration.
func Example_customConfiguration() {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)

	// Create custom configuration
	config := sage.DefaultTransportConfig()
	config.SessionTTL = 30 * time.Minute      // Custom session lifetime
	config.MaxClockSkew = 2 * time.Minute     // Stricter time validation
	config.HandshakeTimeout = 20 * time.Second // Faster timeout

	// Create transport manager with custom config
	tm := sage.NewTransportManager("did:sage:agent", privateKey, config)

	fmt.Printf("Session TTL: %v\n", config.SessionTTL)
	fmt.Printf("Max clock skew: %v\n", config.MaxClockSkew)
	fmt.Printf("Handshake timeout: %v\n", config.HandshakeTimeout)

	_ = tm // Use transport manager

	// Output:
	// Session TTL: 30m0s
	// Max clock skew: 2m0s
	// Handshake timeout: 20s
}
