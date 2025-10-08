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
	fmt.Println("=== SAGE ADK PostgreSQL Storage Example ===\n")

	// Create PostgreSQL storage
	config := storage.DefaultPostgresConfig()
	config.Host = "localhost"
	config.Port = 5432
	config.User = "postgres"
	config.Password = "postgres"
	config.Database = "sage"
	config.SSLMode = "disable"

	store, err := storage.NewPostgresStorage(config)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test connection
	if err := store.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}
	fmt.Println("✓ Connected to PostgreSQL")

	// Example 1: Store and retrieve simple values
	fmt.Println("\n=== Example 1: Simple Key-Value Storage ===")

	if err := store.Store(ctx, "users", "user:1", map[string]interface{}{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   float64(30),
		"role":  "admin",
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
		"user:2": {"name": "Bob", "email": "bob@example.com", "age": float64(25), "role": "user"},
		"user:3": {"name": "Charlie", "email": "charlie@example.com", "age": float64(35), "role": "user"},
		"user:4": {"name": "Diana", "email": "diana@example.com", "age": float64(28), "role": "moderator"},
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

	// Example 5: Update existing item
	fmt.Println("\n=== Example 5: Update Item ===")

	updatedUser := map[string]interface{}{
		"name":       "Alice Smith",
		"email":      "alice.smith@example.com",
		"age":        float64(31),
		"role":       "super_admin",
		"updated_at": time.Now().Unix(),
	}

	if err := store.Store(ctx, "users", "user:1", updatedUser); err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}
	fmt.Println("✓ Updated user:1")

	user, err = store.Get(ctx, "users", "user:1")
	if err != nil {
		log.Fatalf("Failed to get updated user: %v", err)
	}
	fmt.Printf("✓ Retrieved updated user:1: %+v\n", user)

	// Example 6: Count items
	fmt.Println("\n=== Example 6: Count Items ===")

	count, err := store.Count(ctx, "users")
	if err != nil {
		log.Fatalf("Failed to count users: %v", err)
	}
	fmt.Printf("✓ Total users: %d\n", count)

	// Example 7: Delete operations
	fmt.Println("\n=== Example 7: Delete Operations ===")

	if err := store.Delete(ctx, "users", "user:4"); err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}
	fmt.Println("✓ Deleted user:4")

	exists, err = store.Exists(ctx, "users", "user:4")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("✓ user:4 exists after delete: %v\n", exists)

	// Example 8: Conversation history storage
	fmt.Println("\n=== Example 8: Conversation History ===")

	conversationID := "conv:12345"
	messages := []map[string]interface{}{
		{
			"role":      "user",
			"content":   "Hello!",
			"timestamp": time.Now().Unix(),
		},
		{
			"role":      "assistant",
			"content":   "Hi! How can I help you?",
			"timestamp": time.Now().Unix(),
		},
		{
			"role":      "user",
			"content":   "What's the weather?",
			"timestamp": time.Now().Unix(),
		},
		{
			"role":      "assistant",
			"content":   "It's sunny today!",
			"timestamp": time.Now().Unix(),
		},
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

	// Example 9: Get with metadata
	fmt.Println("\n=== Example 9: Get with Metadata ===")

	value, metadata, err := store.GetWithMetadata(ctx, "users", "user:1")
	if err != nil {
		log.Fatalf("Failed to get with metadata: %v", err)
	}
	fmt.Printf("✓ Value: %+v\n", value)
	fmt.Printf("✓ Metadata: %+v\n", metadata)

	// Example 10: List namespaces
	fmt.Println("\n=== Example 10: List Namespaces ===")

	namespaces, err := store.ListNamespaces(ctx)
	if err != nil {
		log.Fatalf("Failed to list namespaces: %v", err)
	}
	fmt.Printf("✓ Found %d namespace(s):\n", len(namespaces))
	for i, ns := range namespaces {
		count, _ := store.Count(ctx, ns)
		fmt.Printf("  %d. %s (%d items)\n", i+1, ns, count)
	}

	// Example 11: Error handling
	fmt.Println("\n=== Example 11: Error Handling ===")

	_, err = store.Get(ctx, "users", "nonexistent")
	if err == storage.ErrNotFound {
		fmt.Println("✓ Correctly handled ErrNotFound")
	} else {
		log.Printf("Unexpected error: %v", err)
	}

	// Clean up examples
	fmt.Println("\n=== Cleanup ===")
	_ = store.Clear(ctx, "conversations")
	_ = store.Clear(ctx, "users")
	fmt.Println("✓ Cleaned up example data")

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  • Store and retrieve values with namespaces")
	fmt.Println("  • List all items in a namespace")
	fmt.Println("  • Check key existence")
	fmt.Println("  • Update existing items (upsert)")
	fmt.Println("  • Count items in a namespace")
	fmt.Println("  • Delete individual keys")
	fmt.Println("  • Clear entire namespaces")
	fmt.Println("  • Conversation history storage")
	fmt.Println("  • Get with metadata (timestamps)")
	fmt.Println("  • List all namespaces")
	fmt.Println("  • Error handling")
	fmt.Println("  • Persistent storage (survives restarts)")
}
