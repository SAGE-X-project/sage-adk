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

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sage-x-project/sage-adk/storage"
)

func main() {
	fmt.Println("=== SAGE ADK Redis Storage Example ===\n")

	// Create Redis storage
	config := storage.DefaultRedisConfig()
	config.Address = "localhost:6379"
	config.TTL = 10 * time.Minute

	store, err := storage.NewRedisStorage(config)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test connection
	if err := store.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping Redis: %v", err)
	}
	fmt.Println("✓ Connected to Redis")

	// Example 1: Store and retrieve simple values
	fmt.Println("\n=== Example 1: Simple Key-Value Storage ===")

	if err := store.Store(ctx, "users", "user:1", map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   float64(30),
	}); err != nil {
		log.Fatalf("Failed to store user: %v", err)
	}
	fmt.Println("✓ Stored user:1")

	user, err := store.Get(ctx, "users", "user:1")
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}
	fmt.Printf("✓ Retrieved user:1: %+v\n", user)

	// Example 2: Store multiple items
	fmt.Println("\n=== Example 2: Multiple Items ===")

	users := map[string]map[string]interface{}{
		"user:2": {"name": "Bob", "email": "bob@example.com", "age": float64(25)},
		"user:3": {"name": "Charlie", "email": "charlie@example.com", "age": float64(35)},
		"user:4": {"name": "Diana", "email": "diana@example.com", "age": float64(28)},
	}

	for key, value := range users {
		if err := store.Store(ctx, "users", key, value); err != nil {
			log.Fatalf("Failed to store %s: %v", key, err)
		}
	}
	fmt.Printf("✓ Stored %d users\n", len(users))

	// Example 3: List all items in namespace
	fmt.Println("\n=== Example 3: List All Users ===")

	allUsers, err := store.List(ctx, "users")
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}
	fmt.Printf("✓ Found %d users:\n", len(allUsers))
	for i, u := range allUsers {
		fmt.Printf("  %d. %+v\n", i+1, u)
	}

	// Example 4: Check existence
	fmt.Println("\n=== Example 4: Check Existence ===")

	exists, err := store.Exists(ctx, "users", "user:1")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("✓ user:1 exists: %v\n", exists)

	exists, err = store.Exists(ctx, "users", "user:999")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("✓ user:999 exists: %v\n", exists)

	// Example 5: TTL management
	fmt.Println("\n=== Example 5: TTL Management ===")

	// Store a temporary session
	if err := store.Store(ctx, "sessions", "session:abc123", map[string]interface{}{
		"user_id":    "user:1",
		"created_at": time.Now().Unix(),
	}); err != nil {
		log.Fatalf("Failed to store session: %v", err)
	}
	fmt.Println("✓ Stored session with default TTL")

	// Get TTL
	ttl, err := store.GetTTL(ctx, "sessions", "session:abc123")
	if err != nil {
		log.Fatalf("Failed to get TTL: %v", err)
	}
	fmt.Printf("✓ Session TTL: %v\n", ttl)

	// Set custom TTL
	if err := store.SetTTL(ctx, "sessions", "session:abc123", 30*time.Second); err != nil {
		log.Fatalf("Failed to set TTL: %v", err)
	}
	fmt.Println("✓ Updated session TTL to 30 seconds")

	// Get updated TTL
	ttl, err = store.GetTTL(ctx, "sessions", "session:abc123")
	if err != nil {
		log.Fatalf("Failed to get TTL: %v", err)
	}
	fmt.Printf("✓ Updated session TTL: %v\n", ttl)

	// Example 6: Delete operations
	fmt.Println("\n=== Example 6: Delete Operations ===")

	if err := store.Delete(ctx, "users", "user:4"); err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}
	fmt.Println("✓ Deleted user:4")

	exists, err = store.Exists(ctx, "users", "user:4")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("✓ user:4 exists after delete: %v\n", exists)

	// Example 7: Conversation history storage
	fmt.Println("\n=== Example 7: Conversation History ===")

	conversationID := "conv:12345"
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello!", "timestamp": time.Now().Unix()},
		{"role": "assistant", "content": "Hi! How can I help you?", "timestamp": time.Now().Unix()},
		{"role": "user", "content": "What's the weather?", "timestamp": time.Now().Unix()},
		{"role": "assistant", "content": "It's sunny today!", "timestamp": time.Now().Unix()},
	}

	if err := store.Store(ctx, "conversations", conversationID, messages); err != nil {
		log.Fatalf("Failed to store conversation: %v", err)
	}
	fmt.Printf("✓ Stored conversation with %d messages\n", len(messages))

	retrieved, err := store.Get(ctx, "conversations", conversationID)
	if err != nil {
		log.Fatalf("Failed to get conversation: %v", err)
	}
	fmt.Println("✓ Retrieved conversation:")
	if msgs, ok := retrieved.([]interface{}); ok {
		for i, msg := range msgs {
			fmt.Printf("  %d. %+v\n", i+1, msg)
		}
	}

	// Example 8: Agent state storage
	fmt.Println("\n=== Example 8: Agent State ===")

	agentState := map[string]interface{}{
		"agent_id":     "agent:001",
		"status":       "active",
		"last_updated": time.Now().Unix(),
		"context": map[string]interface{}{
			"current_task": "processing_request",
			"variables": map[string]interface{}{
				"user_name": "Alice",
				"language":  "en",
			},
		},
	}

	if err := store.Store(ctx, "agent-states", "agent:001", agentState); err != nil {
		log.Fatalf("Failed to store agent state: %v", err)
	}
	fmt.Println("✓ Stored agent state")

	state, err := store.Get(ctx, "agent-states", "agent:001")
	if err != nil {
		log.Fatalf("Failed to get agent state: %v", err)
	}
	fmt.Printf("✓ Retrieved agent state: %+v\n", state)

	// Example 9: Clear namespace
	fmt.Println("\n=== Example 9: Clear Namespace ===")

	// Count items before clear
	beforeCount := len(allUsers)

	// Don't actually clear to keep other examples working
	// if err := store.Clear(ctx, "users"); err != nil {
	// 	log.Fatalf("Failed to clear users: %v", err)
	// }

	fmt.Printf("✓ Would clear namespace 'users' (%d items)\n", beforeCount)

	// Example 10: Error handling
	fmt.Println("\n=== Example 10: Error Handling ===")

	_, err = store.Get(ctx, "users", "nonexistent")
	if err == storage.ErrNotFound {
		fmt.Println("✓ Correctly handled ErrNotFound")
	} else {
		log.Printf("Unexpected error: %v", err)
	}

	// Clean up examples
	fmt.Println("\n=== Cleanup ===")
	_ = store.Clear(ctx, "sessions")
	_ = store.Clear(ctx, "conversations")
	_ = store.Clear(ctx, "agent-states")
	_ = store.Clear(ctx, "users")
	fmt.Println("✓ Cleaned up example data")

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  • Store and retrieve values with namespaces")
	fmt.Println("  • List all items in a namespace")
	fmt.Println("  • Check key existence")
	fmt.Println("  • TTL management (get, set, persist)")
	fmt.Println("  • Delete individual keys")
	fmt.Println("  • Clear entire namespaces")
	fmt.Println("  • Conversation history storage")
	fmt.Println("  • Agent state management")
	fmt.Println("  • Error handling")
}
