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
// # Implementation Status
//
// Implemented:
//   - Storage interface definition
//   - MemoryStorage implementation (in-memory, thread-safe)
//   - RedisStorage implementation (distributed, with TTL support)
//   - Namespace-based organization
//   - Full CRUD operations
//   - TTL management (Redis)
//
// Not Implemented (Future):
//   - PostgreSQL backend for production deployments
//   - Advanced querying (filtering, pagination)
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
// # Redis Storage
//
// RedisStorage provides distributed storage with TTL support:
//
//	config := storage.DefaultRedisConfig()
//	config.Address = "localhost:6379"
//	config.TTL = 10 * time.Minute
//
//	store, err := storage.NewRedisStorage(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
//	// Store with automatic TTL
//	store.Store(ctx, "sessions", "session:123", sessionData)
//
//	// Manage TTL
//	ttl, _ := store.GetTTL(ctx, "sessions", "session:123")
//	store.SetTTL(ctx, "sessions", "session:123", 30*time.Second)
//
// Characteristics:
//   - Distributed: Share data across multiple agent instances
//   - Persistent: Data survives process restarts
//   - TTL Support: Automatic expiration for temporary data
//   - JSON Serialization: Automatic marshaling/unmarshaling
//   - Connection Pooling: Efficient resource usage
//
// # PostgreSQL Storage
//
// PostgresStorage provides production-ready persistent storage:
//
//	config := storage.DefaultPostgresConfig()
//	config.Host = "localhost"
//	config.Database = "sage"
//	config.User = "postgres"
//	config.Password = "postgres"
//
//	store, err := storage.NewPostgresStorage(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close()
//
//	// Store with automatic timestamps
//	store.Store(ctx, "users", "user:1", userData)
//
//	// Get with metadata
//	value, metadata, _ := store.GetWithMetadata(ctx, "users", "user:1")
//	fmt.Println(metadata["created_at"])
//
//	// Count items
//	count, _ := store.Count(ctx, "users")
//
//	// List all namespaces
//	namespaces, _ := store.ListNamespaces(ctx)
//
// Characteristics:
//   - Persistent: Data survives restarts and crashes
//   - ACID: Transaction support (future)
//   - Scalable: Disk-based storage
//   - Queryable: JSONB support for complex queries
//   - Automatic Timestamps: created_at, updated_at
//   - Connection Pooling: Efficient resource usage
//   - Indexes: Optimized queries on namespace and timestamps
//
// # Storage Backend Comparison
//
//	| Feature          | Memory  | Redis   | PostgreSQL |
//	|------------------|---------|---------|------------|
//	| Persistence      | No      | Yes     | Yes        |
//	| Distributed      | No      | Yes     | Yes        |
//	| TTL Support      | No      | Yes     | No*        |
//	| Complex Queries  | No      | Limited | Yes        |
//	| Transactions     | No      | Limited | Yes        |
//	| Timestamps       | No      | No      | Yes        |
//	| Best For         | Testing | Cache   | Production |
//
// * PostgreSQL can implement TTL with triggers or cron jobs
//
// # Future Enhancements
//
// Phase 4: Advanced Features
//   - Pagination for large datasets
//   - Full-text search with PostgreSQL
//   - Data versioning and audit logs
//   - Backup and restore utilities
//   - Transaction support for batch operations
//   - Query builder for complex filters
package storage
