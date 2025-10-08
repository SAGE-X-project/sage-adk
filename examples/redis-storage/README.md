# Redis Storage Example

This example demonstrates how to use the SAGE ADK's Redis storage backend for persistent data storage.

## Features Demonstrated

1. **Basic Operations**: Store, retrieve, delete, and check existence
2. **Namespace Management**: Organize data with namespaces
3. **TTL Management**: Set, get, and update time-to-live for keys
4. **Batch Operations**: List all items in a namespace, clear namespaces
5. **Use Cases**: Conversation history, agent state, session storage
6. **Error Handling**: Handle not found errors properly

## Prerequisites

- Redis server running on `localhost:6379`
- Go 1.21 or later

## Installing Redis

### macOS (Homebrew)
```bash
brew install redis
brew services start redis
```

### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install redis-server
sudo systemctl start redis-server
```

### Docker
```bash
docker run -d -p 6379:6379 redis:7-alpine
```

## Running the Example

```bash
# Start Redis if not running
redis-server

# Run the example
cd sage-adk/examples/redis-storage
go run main.go
```

## Configuration

```go
config := storage.DefaultRedisConfig()
config.Address = "localhost:6379"  // Redis server address
config.Password = ""                // Redis password (if required)
config.DB = 0                       // Redis database number
config.TTL = 10 * time.Minute      // Default TTL for keys
config.PoolSize = 10                // Connection pool size
config.MaxRetries = 3               // Maximum retry attempts
```

## Code Examples

### Basic Store and Retrieve

```go
store, err := storage.NewRedisStorage(config)
if err != nil {
    log.Fatal(err)
}
defer store.Close()

ctx := context.Background()

// Store
err = store.Store(ctx, "users", "user:1", map[string]interface{}{
    "name": "Alice",
    "email": "alice@example.com",
})

// Retrieve
user, err := store.Get(ctx, "users", "user:1")
```

### List All Items

```go
users, err := store.List(ctx, "users")
for _, user := range users {
    fmt.Printf("%+v\n", user)
}
```

### TTL Management

```go
// Get TTL
ttl, err := store.GetTTL(ctx, "sessions", "session:123")

// Set TTL
err = store.SetTTL(ctx, "sessions", "session:123", 30*time.Second)

// Remove expiration
err = store.SetTTL(ctx, "sessions", "session:123", 0)
```

### Delete Operations

```go
// Delete single key
err = store.Delete(ctx, "users", "user:1")

// Clear entire namespace
err = store.Clear(ctx, "users")
```

### Error Handling

```go
user, err := store.Get(ctx, "users", "nonexistent")
if errors.Is(err, storage.ErrNotFound) {
    // Handle not found
}
```

## Use Cases

### 1. Conversation History

```go
conversationID := "conv:12345"
messages := []map[string]interface{}{
    {"role": "user", "content": "Hello!"},
    {"role": "assistant", "content": "Hi!"},
}

store.Store(ctx, "conversations", conversationID, messages)
```

### 2. Agent State

```go
agentState := map[string]interface{}{
    "agent_id": "agent:001",
    "status": "active",
    "context": map[string]interface{}{
        "current_task": "processing",
        "variables": map[string]string{
            "language": "en",
        },
    },
}

store.Store(ctx, "agent-states", "agent:001", agentState)
```

### 3. Session Management

```go
session := map[string]interface{}{
    "user_id": "user:123",
    "created_at": time.Now().Unix(),
    "expires_in": 3600,
}

store.Store(ctx, "sessions", "session:abc", session)
store.SetTTL(ctx, "sessions", "session:abc", 1*time.Hour)
```

## Namespace Organization

Recommended namespace patterns:

- `users:<id>` - User data
- `conversations:<id>` - Conversation history
- `agent-states:<id>` - Agent state
- `sessions:<id>` - User sessions
- `cache:<key>` - Cache data
- `tasks:<id>` - Task data
- `metadata:<id>` - Agent metadata

## Key Format

Redis keys are automatically prefixed with `sage:<namespace>:<key>`:

```
sage:users:user:1
sage:conversations:conv:12345
sage:agent-states:agent:001
```

## Performance Considerations

1. **Connection Pooling**: Default pool size is 10 connections
2. **Batch Operations**: Use `List()` instead of multiple `Get()` calls when possible
3. **TTL**: Set appropriate TTLs to prevent memory bloat
4. **Namespaces**: Use clear namespace patterns for easy management
5. **Cleanup**: Always `Close()` the storage when done

## Testing

Run integration tests (requires Redis):

```bash
cd sage-adk/storage
go test -v -tags=integration
```

## Monitoring

Check Redis status:

```bash
# CLI
redis-cli ping
redis-cli info

# Get all sage keys
redis-cli keys "sage:*"

# Monitor commands in real-time
redis-cli monitor
```

## Common Patterns

### Atomic Operations

Redis operations are atomic, so you can safely use them in concurrent environments:

```go
// Multiple goroutines can safely call Store
go store.Store(ctx, "counter", "count", currentCount)
```

### Expiring Cache

```go
// Store with short TTL for cache
config.TTL = 5 * time.Minute
store.Store(ctx, "cache", "result:123", data)
```

### Distributed Locking

For distributed locking, consider using Redis-specific features not in this interface.

## Troubleshooting

### Connection Refused

```
Failed to connect to Redis: dial tcp 127.0.0.1:6379: connect: connection refused
```

**Solution**: Ensure Redis is running:
```bash
redis-server
```

### Authentication Required

```
NOAUTH Authentication required
```

**Solution**: Set password in config:
```go
config.Password = "your-redis-password"
```

### Memory Issues

If Redis runs out of memory, consider:
- Setting appropriate TTLs on keys
- Using `Clear()` to remove old data
- Increasing Redis max memory limit
- Using Redis eviction policies

## Next Steps

- See `examples/stateful-agent/` for using Redis with state management
- See `storage/postgresql.go` for PostgreSQL alternative
- Check Redis documentation for advanced features: https://redis.io/docs/
