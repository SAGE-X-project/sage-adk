## PostgreSQL Storage Example

This example demonstrates how to use the SAGE ADK's PostgreSQL storage backend for persistent, production-ready data storage.

## Features Demonstrated

1. **Basic Operations**: Store, retrieve, update, delete, and check existence
2. **Namespace Management**: Organize data with namespaces
3. **Batch Operations**: List all items in a namespace, count items, clear namespaces
4. **Metadata Tracking**: Automatic created_at and updated_at timestamps
5. **Use Cases**: User management, conversation history
6. **Advanced Queries**: List all namespaces, count items per namespace
7. **Error Handling**: Handle not found errors properly

## Prerequisites

- PostgreSQL server running on `localhost:5432`
- Database named `sage` created
- Go 1.21 or later

## Installing PostgreSQL

### macOS (Homebrew)
```bash
brew install postgresql@16
brew services start postgresql@16

# Create database
createdb sage
```

### Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql

# Create database
sudo -u postgres createdb sage
```

### Docker
```bash
docker run -d \
  -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=sage \
  --name sage-postgres \
  postgres:16-alpine
```

## Running the Example

```bash
# Start PostgreSQL if not running
# For Homebrew: brew services start postgresql@16
# For Docker: docker start sage-postgres

# Run the example
cd sage-adk/examples/postgres-storage
go run main.go
```

## Configuration

```go
config := storage.DefaultPostgresConfig()
config.Host = "localhost"          // PostgreSQL host
config.Port = 5432                 // PostgreSQL port
config.User = "postgres"           // Database user
config.Password = "postgres"       // Database password
config.Database = "sage"           // Database name
config.SSLMode = "disable"         // SSL mode (disable/require/verify-ca/verify-full)
config.TableName = "sage_storage"  // Table name for storage
config.MaxOpenConns = 25           // Max open connections
config.MaxIdleConns = 5            // Max idle connections
config.ConnMaxLifetime = 5*time.Minute  // Connection max lifetime
config.AutoMigrate = true          // Auto-create table
```

## Code Examples

### Basic Store and Retrieve

```go
store, err := storage.NewPostgresStorage(config)
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

### Update (Upsert)

```go
// Store overwrites existing values
err = store.Store(ctx, "users", "user:1", map[string]interface{}{
    "name": "Alice Smith",
    "email": "alice.smith@example.com",
})
```

### List All Items

```go
users, err := store.List(ctx, "users")
for _, user := range users {
    fmt.Printf("%+v\n", user)
}
```

### Count Items

```go
count, err := store.Count(ctx, "users")
fmt.Printf("Total users: %d\n", count)
```

### Get with Metadata

```go
value, metadata, err := store.GetWithMetadata(ctx, "users", "user:1")
fmt.Printf("Created: %v\n", metadata["created_at"])
fmt.Printf("Updated: %v\n", metadata["updated_at"])
```

### List Namespaces

```go
namespaces, err := store.ListNamespaces(ctx)
for _, ns := range namespaces {
    count, _ := store.Count(ctx, ns)
    fmt.Printf("%s: %d items\n", ns, count)
}
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

## Database Schema

The storage uses a simple table structure:

```sql
CREATE TABLE sage_storage (
    namespace VARCHAR(255) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (namespace, key)
);

CREATE INDEX idx_sage_storage_namespace ON sage_storage(namespace);
CREATE INDEX idx_sage_storage_created_at ON sage_storage(created_at);
CREATE INDEX idx_sage_storage_updated_at ON sage_storage(updated_at);
```

## Use Cases

### 1. User Management

```go
user := map[string]interface{}{
    "name": "Alice",
    "email": "alice@example.com",
    "role": "admin",
    "status": "active",
}

store.Store(ctx, "users", "user:123", user)
```

### 2. Conversation History

```go
messages := []map[string]interface{}{
    {"role": "user", "content": "Hello!"},
    {"role": "assistant", "content": "Hi!"},
}

store.Store(ctx, "conversations", "conv:abc", messages)
```

### 3. Agent State

```go
state := map[string]interface{}{
    "agent_id": "agent:001",
    "status": "active",
    "context": map[string]interface{}{
        "task": "processing",
        "variables": map[string]string{"lang": "en"},
    },
}

store.Store(ctx, "agent-states", "agent:001", state)
```

## Namespace Organization

Recommended namespace patterns:

- `users:<id>` - User data
- `conversations:<id>` - Conversation history
- `agent-states:<id>` - Agent state
- `tasks:<id>` - Task data
- `metadata:<id>` - Agent metadata
- `cache:<key>` - Cache data

## Performance Considerations

1. **Connection Pooling**: Default pool size is 25 connections
2. **Indexes**: Automatic indexes on namespace, created_at, updated_at
3. **JSONB**: Efficient JSON storage with indexing support
4. **Upsert**: Single query for insert or update
5. **Batch Operations**: Use transactions for bulk inserts (future)

## Advantages over Redis

1. **Persistence**: Data survives crashes and restarts
2. **Queries**: Complex queries with SQL
3. **Transactions**: ACID guarantees
4. **Joins**: Can join with other tables
5. **No Memory Limits**: Disk-based storage
6. **Timestamps**: Automatic created_at/updated_at

## Testing

Run integration tests (requires PostgreSQL):

```bash
cd sage-adk/storage
go test -v -tags=integration -run TestPostgres
```

## Monitoring

Check database status:

```bash
# psql CLI
psql -U postgres -d sage

# List tables
\dt

# View data
SELECT * FROM sage_storage LIMIT 10;

# Count by namespace
SELECT namespace, COUNT(*)
FROM sage_storage
GROUP BY namespace;

# View indexes
\di
```

## Maintenance

### Vacuum

```sql
-- Reclaim storage
VACUUM ANALYZE sage_storage;
```

### Backup

```bash
# Dump database
pg_dump sage > sage_backup.sql

# Restore database
psql sage < sage_backup.sql
```

### Clear old data

```sql
-- Delete old entries
DELETE FROM sage_storage
WHERE updated_at < NOW() - INTERVAL '30 days';
```

## Troubleshooting

### Connection Refused

```
Failed to connect to PostgreSQL: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solution**: Ensure PostgreSQL is running:
```bash
# Homebrew
brew services start postgresql@16

# Docker
docker start sage-postgres

# Ubuntu
sudo systemctl start postgresql
```

### Authentication Failed

```
pq: password authentication failed for user "postgres"
```

**Solution**: Set correct password in config or create user:
```sql
ALTER USER postgres PASSWORD 'postgres';
```

### Database Does Not Exist

```
pq: database "sage" does not exist
```

**Solution**: Create database:
```bash
createdb sage
# or
psql -U postgres -c "CREATE DATABASE sage;"
```

### Permission Denied

```
pq: permission denied for table sage_storage
```

**Solution**: Grant permissions:
```sql
GRANT ALL PRIVILEGES ON TABLE sage_storage TO postgres;
```

## Next Steps

- See `examples/stateful-agent/` for using PostgreSQL with state management
- See `storage/redis.go` for Redis alternative
- Check PostgreSQL documentation for advanced features: https://www.postgresql.org/docs/
