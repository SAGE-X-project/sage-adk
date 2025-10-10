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

//go:build examples
// +build examples

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/sage"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// This example demonstrates low-level SAGE adapter usage with two agents
// communicating securely over HTTP with message signing and verification.

func main() {
	// Parse command line arguments
	mode := "interactive"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case "sender":
		runSender()
	case "receiver":
		runReceiver()
	case "interactive":
		runInteractive()
	default:
		log.Fatalf("Unknown mode: %s. Use 'sender', 'receiver', or 'interactive'", mode)
	}
}

// runInteractive demonstrates two agents exchanging messages in a single process
func runInteractive() {
	log.Println("🚀 SAGE Interactive Demo - Two agents exchanging secure messages")
	log.Println("=" + string(make([]byte, 70)))

	ctx := context.Background()

	// Generate key pairs for Alice and Bob
	log.Println("\n📋 Step 1: Generating Ed25519 key pairs for Alice and Bob...")
	alicePublicKey, alicePrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate Alice's key: %v", err)
	}
	bobPublicKey, bobPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate Bob's key: %v", err)
	}
	log.Printf("✅ Alice's public key: %x", alicePublicKey[:8])
	log.Printf("✅ Bob's public key: %x", bobPublicKey[:8])

	// Save keys temporarily
	aliceKeyPath := "/tmp/alice-key.json"
	bobKeyPath := "/tmp/bob-key.json"
	if err := saveKey(aliceKeyPath, alicePrivateKey); err != nil {
		log.Fatalf("Failed to save Alice's key: %v", err)
	}
	if err := saveKey(bobKeyPath, bobPrivateKey); err != nil {
		log.Fatalf("Failed to save Bob's key: %v", err)
	}
	defer os.Remove(aliceKeyPath)
	defer os.Remove(bobKeyPath)

	// Create SAGE adapters for Alice and Bob
	log.Println("\n📋 Step 2: Creating SAGE adapters...")
	aliceAdapter, err := sage.NewAdapter(&config.SAGEConfig{
		DID:            "did:sage:alice",
		Network:        "local",
		PrivateKeyPath: aliceKeyPath,
	})
	if err != nil {
		log.Fatalf("Failed to create Alice's adapter: %v", err)
	}

	bobAdapter, err := sage.NewAdapter(&config.SAGEConfig{
		DID:            "did:sage:bob",
		Network:        "local",
		PrivateKeyPath: bobKeyPath,
	})
	if err != nil {
		log.Fatalf("Failed to create Bob's adapter: %v", err)
	}
	log.Println("✅ Alice's adapter created")
	log.Println("✅ Bob's adapter created")

	// Start Bob's HTTP server to receive messages
	log.Println("\n📋 Step 3: Starting Bob's HTTP server on :18080...")
	var receivedMessage *types.Message
	server := sage.NewNetworkServer(":18080", func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		log.Printf("\n📨 Bob received message from %s", msg.Security.AgentDID)

		// Verify the message
		if err := bobAdapter.Verify(ctx, msg); err != nil {
			log.Printf("❌ Message verification failed: %v", err)
			return nil, err
		}
		log.Println("✅ Message signature verified successfully")

		// Store received message
		receivedMessage = msg

		// Log message content
		if len(msg.Parts) > 0 {
			if textPart, ok := msg.Parts[0].(*types.TextPart); ok {
				log.Printf("📝 Message content: %s", textPart.Text)
			}
		}

		return nil, nil
	})

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	defer server.Stop(ctx)
	time.Sleep(100 * time.Millisecond) // Wait for server to start
	log.Println("✅ Bob's server is running")

	// Configure Alice to send to Bob's endpoint
	log.Println("\n📋 Step 4: Configuring Alice to send messages to Bob...")
	aliceAdapter.SetRemoteEndpoint("http://localhost:18080/sage/message")
	log.Printf("✅ Alice configured to send to: %s", aliceAdapter.GetRemoteEndpoint())

	// Alice sends a message to Bob
	log.Println("\n📋 Step 5: Alice sending encrypted message to Bob...")
	message := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello Bob! This is a secure SAGE message from Alice."),
	})

	if err := aliceAdapter.SendMessage(ctx, message); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	log.Println("✅ Message sent successfully")

	// Wait for message to be received
	time.Sleep(200 * time.Millisecond)

	// Verify message was received
	log.Println("\n📋 Step 6: Verifying message delivery...")
	if receivedMessage == nil {
		log.Fatal("❌ Message was not received")
	}
	log.Println("✅ Message delivered and verified successfully")

	// Display security metadata
	log.Println("\n📊 Security Metadata:")
	log.Printf("  Protocol Mode: %s", receivedMessage.Security.Mode)
	log.Printf("  Agent DID: %s", receivedMessage.Security.AgentDID)
	log.Printf("  Timestamp: %s", receivedMessage.Security.Timestamp.Format(time.RFC3339))
	log.Printf("  Nonce: %s", receivedMessage.Security.Nonce[:16]+"...")
	if receivedMessage.Security.Signature != nil {
		log.Printf("  Signature Algorithm: %s", receivedMessage.Security.Signature.Algorithm)
		log.Printf("  Signature KeyID: %s", receivedMessage.Security.Signature.KeyID)
		log.Printf("  Signature Length: %d bytes", len(receivedMessage.Security.Signature.Signature))
	}

	log.Println("\n🎉 SAGE Interactive Demo completed successfully!")
	log.Println("=" + string(make([]byte, 70)))
}

// runSender runs Alice as a standalone sender
func runSender() {
	log.Println("🚀 SAGE Sender (Alice)")

	ctx := context.Background()

	// Generate or load key
	keyPath := getEnvOrDefault("ALICE_KEY_PATH", "/tmp/alice-key.json")
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	if err := saveKey(keyPath, privateKey); err != nil {
		log.Fatalf("Failed to save key: %v", err)
	}
	defer os.Remove(keyPath)

	// Create adapter
	adapter, err := sage.NewAdapter(&config.SAGEConfig{
		DID:            "did:sage:alice",
		Network:        "local",
		PrivateKeyPath: keyPath,
	})
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Set Bob's endpoint
	bobEndpoint := getEnvOrDefault("BOB_ENDPOINT", "http://localhost:18080/sage/message")
	adapter.SetRemoteEndpoint(bobEndpoint)

	log.Printf("📤 Sending message to: %s", bobEndpoint)

	// Send message
	message := types.NewMessage(types.MessageRoleUser, []types.Part{
		types.NewTextPart("Hello from standalone Alice!"),
	})

	if err := adapter.SendMessage(ctx, message); err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	log.Println("✅ Message sent successfully")
}

// runReceiver runs Bob as a standalone receiver
func runReceiver() {
	log.Println("🚀 SAGE Receiver (Bob)")
	log.Println("Listening on :18080")

	ctx := context.Background()

	// Generate or load key
	keyPath := getEnvOrDefault("BOB_KEY_PATH", "/tmp/bob-key.json")
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate key: %v", err)
	}
	if err := saveKey(keyPath, privateKey); err != nil {
		log.Fatalf("Failed to save key: %v", err)
	}
	defer os.Remove(keyPath)

	// Create adapter
	adapter, err := sage.NewAdapter(&config.SAGEConfig{
		DID:            "did:sage:bob",
		Network:        "local",
		PrivateKeyPath: keyPath,
	})
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Start server
	server := sage.NewNetworkServer(":18080", func(ctx context.Context, msg *types.Message) (*types.Message, error) {
		log.Printf("\n📨 Received message from %s", msg.Security.AgentDID)

		// Verify the message
		if err := adapter.Verify(ctx, msg); err != nil {
			log.Printf("❌ Verification failed: %v", err)
			return nil, err
		}
		log.Println("✅ Message verified")

		// Display content
		if len(msg.Parts) > 0 {
			if textPart, ok := msg.Parts[0].(*types.TextPart); ok {
				log.Printf("📝 Content: %s", textPart.Text)
			}
		}

		return nil, nil
	})

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("✅ Server started. Press Ctrl+C to stop.")

	<-sigChan
	log.Println("\n📥 Shutting down...")
	server.Stop(ctx)
}

// Helper functions

func saveKey(path string, privateKey ed25519.PrivateKey) error {
	data := map[string]interface{}{
		"kty": "OKP",
		"crv": "Ed25519",
		"d":   privateKey.Seed(),
		"x":   []byte(privateKey.Public().(ed25519.PublicKey)),
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
