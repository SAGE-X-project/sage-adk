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

// Package storage provides storage abstraction for the SAGE ADK.
//
// This package allows AI agents to store and retrieve data using a
// unified interface that works with multiple storage backends.
//
// # Phase 1 Implementation
//
// The current implementation (Phase 1) focuses on the storage abstraction
// and an in-memory implementation for testing and development.
//
// Implemented:
//   - Storage interface definition
//   - MemoryStorage implementation (in-memory, thread-safe)
//   - Namespace-based organization
//   - Full CRUD operations
//
// Not Implemented (Future):
//   - Redis backend for distributed agents
//   - PostgreSQL backend for production deployments
//   - Advanced querying (filtering, pagination)
//   - TTL support for temporary data
//   - Data versioning and migration
//
// # Storage Interface
//
// All storage backends implement the Storage interface:
//
//	type Storage interface {
//	    Store(ctx context.Context, namespace, key string, value interface{}) error
//	    Get(ctx context.Context, namespace, key string) (interface{}, error)
//	    List(ctx context.Context, namespace string) ([]interface{}, error)
//	    Delete(ctx context.Context, namespace, key string) error
//	    Clear(ctx context.Context, namespace string) error
//	    Exists(ctx context.Context, namespace, key string) (bool, error)
//	}
//
// # Basic Usage
//
//	// Create storage
//	store := storage.NewMemoryStorage()
//
//	// Store a message
//	msg := &types.Message{MessageID: "msg-1", Content: "Hello"}
//	err := store.Store(ctx, "history:agent1", msg.MessageID, msg)
//
//	// Retrieve message
//	retrieved, err := store.Get(ctx, "history:agent1", "msg-1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	msg := retrieved.(*types.Message)
//
//	// List all messages
//	messages, err := store.List(ctx, "history:agent1")
//	for _, item := range messages {
//	    msg := item.(*types.Message)
//	    fmt.Println(msg.Content)
//	}
//
//	// Delete a message
//	err = store.Delete(ctx, "history:agent1", "msg-1")
//
//	// Clear all messages
//	err = store.Clear(ctx, "history:agent1")
//
// # Namespace Organization
//
// Use namespaces to organize different types of data:
//
//	// Message history for an agent
//	store.Store(ctx, "history:agent1", msgID, message)
//
//	// Agent metadata
//	store.Store(ctx, "metadata:agent1", "config", config)
//
//	// Conversation context
//	store.Store(ctx, "context:ctx-123", "data", contextData)
//
//	// Agent state
//	store.Store(ctx, "state:agent1", "current", stateData)
//
// # Integration with Agent
//
//	agent, _ := agent.NewAgent("storage-agent").
//	    OnMessage(func(ctx context.Context, msg MessageContext) error {
//	        // Create storage
//	        store := storage.NewMemoryStorage()
//
//	        // Store incoming message
//	        store.Store(ctx, "history:agent1", msg.ID(), msg.Message())
//
//	        // Retrieve conversation history
//	        history, err := store.List(ctx, "history:agent1")
//	        if err != nil {
//	            return err
//	        }
//
//	        // Process with full context
//	        return processWithHistory(msg, history)
//	    }).
//	    Build()
//
// # Memory Storage
//
// MemoryStorage provides an in-memory, thread-safe storage implementation:
//
//	store := storage.NewMemoryStorage()
//
//	// Safe for concurrent use
//	var wg sync.WaitGroup
//	for i := 0; i < 100; i++ {
//	    wg.Add(1)
//	    go func(n int) {
//	        defer wg.Done()
//	        key := fmt.Sprintf("key-%d", n)
//	        store.Store(ctx, "test", key, n)
//	    }(i)
//	}
//	wg.Wait()
//
// Characteristics:
//   - O(1) access time for Get/Store/Delete
//   - O(n) for List operations
//   - No serialization overhead
//   - Data lost when process exits
//   - Suitable for testing and single-instance deployments
//
// # Type Preservation
//
// MemoryStorage preserves types without serialization:
//
//	// Store different types
//	store.Store(ctx, "data", "string", "hello")
//	store.Store(ctx, "data", "int", 42)
//	store.Store(ctx, "data", "struct", myStruct)
//
//	// Retrieve with type assertions
//	str, _ := store.Get(ctx, "data", "string")
//	message := str.(string)
//
//	num, _ := store.Get(ctx, "data", "int")
//	count := num.(int)
//
//	obj, _ := store.Get(ctx, "data", "struct")
//	s := obj.(MyStruct)
//
// # Error Handling
//
// The storage package uses custom error types:
//
//	val, err := store.Get(ctx, "test", "nonexistent")
//	if errors.Is(err, errors.ErrNotFound) {
//	    // Key not found
//	}
//
//	err = store.Store(ctx, "", "key", "value")
//	if errors.Is(err, errors.ErrInvalidInput) {
//	    // Empty namespace or key
//	}
//
// # Namespace Isolation
//
// Namespaces are completely isolated:
//
//	store.Store(ctx, "ns1", "key", "value1")
//	store.Store(ctx, "ns2", "key", "value2")
//
//	val1, _ := store.Get(ctx, "ns1", "key")  // "value1"
//	val2, _ := store.Get(ctx, "ns2", "key")  // "value2"
//
//	// Clearing one namespace doesn't affect others
//	store.Clear(ctx, "ns1")
//	val2, _ = store.Get(ctx, "ns2", "key")  // Still "value2"
//
// # Design Principles
//
// Based on AI agent development research:
//
//   - Simple Interface: Familiar CRUD operations
//   - Namespace Organization: Logical separation of data types
//   - Type Flexibility: Accept any Go type (interface{})
//   - Thread Safety: Safe for concurrent agent operations
//   - Progressive Disclosure: Simple by default, powerful when needed
//
// # Future Enhancements
//
// Phase 2: Redis Backend
//   - Distributed storage for multi-instance deployments
//   - Pub/Sub for agent communication
//   - TTL support for temporary data
//
// Phase 3: PostgreSQL Backend
//   - Persistent storage for production
//   - Complex querying and filtering
//   - Transaction support
//
// Phase 4: Advanced Features
//   - Pagination for large datasets
//   - Full-text search
//   - Data versioning
//   - Backup and restore utilities
package storage
