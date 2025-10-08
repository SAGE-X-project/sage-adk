// Stateful Agent Example

This example demonstrates conversation state management in SAGE ADK, showing how to maintain context, history, and user preferences across multiple interactions.

## Features

### State Management
- **Conversation History**: Maintains full conversation context
- **Session Persistence**: Each user gets a persistent session
- **Variable Storage**: Store and retrieve session-specific data
- **Message Limits**: Configurable maximum messages per session
- **Auto-Expiration**: Sessions automatically expire after TTL
- **Auto-Cleanup**: Expired sessions are automatically removed

### Special Commands

The agent supports several special commands:

- `/clear` - Clear conversation history
- `/stats` - Show session statistics
- `/name <name>` - Set your name (remembered in future conversations)
- `/help` - Show available commands

## Configuration

```go
stateConfig := &state.Config{
    DefaultTTL:        24 * time.Hour,  // Sessions expire after 24 hours
    MaxMessages:       50,               // Keep last 50 messages
    CleanupInterval:   1 * time.Hour,   // Run cleanup every hour
    EnableAutoCleanup: true,             // Enable automatic cleanup
}
```

## Running the Example

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY="your-api-key-here"
```

2. Run the agent:
```bash
go run -tags examples examples/stateful-agent/main.go
```

3. The agent will start on http://localhost:8080

## Testing the Agent

### Basic Conversation
```bash
# First message
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-001",
      "role": "user",
      "parts": [{"kind": "text", "text": "My favorite color is blue"}]
    }
  }'

# Second message (agent remembers context)
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-002",
      "role": "user",
      "parts": [{"kind": "text", "text": "What is my favorite color?"}]
    }
  }'
```

### Set Your Name
```bash
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-003",
      "role": "user",
      "parts": [{"kind": "text", "text": "/name Alice"}]
    }
  }'
```

### View Session Statistics
```bash
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-004",
      "role": "user",
      "parts": [{"kind": "text", "text": "/stats"}]
    }
  }'
```

### Clear Conversation History
```bash
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-005",
      "role": "user",
      "parts": [{"kind": "text", "text": "/clear"}]
    }
  }'
```

## How It Works

### Session Management

1. **Session Creation**: When a user sends their first message, a new session is created
2. **Message Storage**: All messages are stored in the session
3. **Context Building**: Recent messages are included in LLM prompts
4. **Variable Storage**: User preferences and metadata are stored per session

### Conversation Flow

```
User Message
    ↓
Extract/Create Session ID
    ↓
Get or Create Session State
    ↓
Add User Message to History
    ↓
Build Context from History
    ↓
Send to LLM with Context
    ↓
Add LLM Response to History
    ↓
Update Session Variables
    ↓
Return Response
```

### State Structure

Each session maintains:

```go
type State struct {
    SessionID string              // Unique session identifier
    AgentID   string              // Agent identifier
    Messages  []*types.Message    // Conversation history
    Variables map[string]interface{} // Session variables
    Metadata  map[string]interface{} // Session metadata
    CreatedAt time.Time           // Creation timestamp
    UpdatedAt time.Time           // Last update timestamp
    ExpiresAt *time.Time          // Expiration timestamp
}
```

## State Manager Interface

The state manager provides a simple interface:

```go
// Create new session
err := manager.Create(ctx, &state.State{
    SessionID: "session-123",
    AgentID:   "my-agent",
})

// Get session
session, err := manager.Get(ctx, "session-123")

// Add message to history
err = manager.AddMessage(ctx, "session-123", message)

// Get recent messages
messages, err := manager.GetMessages(ctx, "session-123", 10)

// Set session variable
err = manager.SetVariable(ctx, "session-123", "user_name", "Alice")

// Get session variable
value, err := manager.GetVariable(ctx, "session-123", "user_name")

// Clear message history
err = manager.Clear(ctx, "session-123")

// Delete session
err = manager.Delete(ctx, "session-123")
```

## Use Cases

### 1. Multi-Turn Conversations

The agent remembers previous messages:

```
User: "I have a cat named Whiskers"
Agent: "That's a lovely name for a cat! How long have you had Whiskers?"

User: "What's my cat's name?"
Agent: "Your cat's name is Whiskers!"
```

### 2. User Preferences

Store and recall user preferences:

```go
// Set preference
manager.SetVariable(ctx, sessionID, "language", "Spanish")

// Use in system message
if lang, err := manager.GetVariable(ctx, sessionID, "language"); err == nil {
    systemMsg += fmt.Sprintf("User prefers %s. ", lang)
}
```

### 3. Conversation Context

Maintain context across multiple interactions:

```go
// Build context from last 10 messages
messages, _ := manager.GetMessages(ctx, sessionID, 10)

// Include in LLM prompt
for _, msg := range messages {
    // Add to conversation history
}
```

### 4. Session Analytics

Track conversation metrics:

```go
// Count interactions
count, _ := manager.GetVariable(ctx, sessionID, "interaction_count")
manager.SetVariable(ctx, sessionID, "interaction_count", count+1)

// Track topics discussed
topics, _ := manager.GetVariable(ctx, sessionID, "topics")
manager.SetVariable(ctx, sessionID, "topics", append(topics, newTopic))
```

## Advanced Features

### Custom Session ID Extraction

Extract session ID from headers or message metadata:

```go
func extractSessionID(msg agent.MessageContext) string {
    // From HTTP headers
    if sessionID := getHeader("X-Session-ID"); sessionID != "" {
        return sessionID
    }

    // From message metadata
    if metadata := msg.Metadata(); metadata != nil {
        if sid, ok := metadata["sessionId"].(string); ok {
            return sid
        }
    }

    // Generate new session ID
    return generateSessionID()
}
```

### Message Filtering

Get messages by criteria:

```go
// Get messages after timestamp
after := time.Now().Add(-1 * time.Hour)
messages := session.GetMessagesAfter(after)

// Get only user messages
userMessages := make([]*types.Message, 0)
for _, msg := range session.Messages {
    if msg.Role == types.MessageRoleUser {
        userMessages = append(userMessages, msg)
    }
}
```

### Session Expiration

Control session lifecycle:

```go
// Set custom expiration
expiresAt := time.Now().Add(2 * time.Hour)
session.ExpiresAt = &expiresAt

// Check if expired
if session.IsExpired() {
    // Handle expired session
}

// Manual cleanup
count, err := manager.Cleanup(ctx)
log.Printf("Cleaned up %d expired sessions", count)
```

### Message Limits

Prevent memory issues with message limits:

```go
// Truncate to last N messages
session.TruncateMessages(20)

// Or configure automatic truncation
config := &state.Config{
    MaxMessages: 50, // Automatically keep only last 50
}
```

## Best Practices

1. **Session ID Generation**: Use secure, unique session IDs
   ```go
   sessionID := uuid.New().String()
   ```

2. **Error Handling**: Gracefully handle state errors
   ```go
   session, err := manager.Get(ctx, sessionID)
   if err == state.ErrStateNotFound {
       // Create new session
   } else if err != nil {
       // Handle other errors
   }
   ```

3. **Context Limits**: Don't send entire history to LLM
   ```go
   // Get only recent messages
   messages, _ := manager.GetMessages(ctx, sessionID, 10)
   ```

4. **Cleanup**: Enable auto-cleanup or run periodic cleanup
   ```go
   config.EnableAutoCleanup = true
   config.CleanupInterval = 1 * time.Hour
   ```

5. **Variable Naming**: Use consistent variable names
   ```go
   const (
       VarUserName = "user_name"
       VarLanguage = "language"
       VarTheme    = "theme"
   )
   ```

6. **Session Validation**: Validate session before use
   ```go
   if session.IsExpired() {
       return errors.New("session expired")
   }
   if err := session.Validate(); err != nil {
       return err
   }
   ```

## Storage Backends

The example uses in-memory storage, but you can implement custom backends:

```go
type CustomManager struct {
    db *sql.DB
}

func (m *CustomManager) Get(ctx context.Context, sessionID string) (*state.State, error) {
    // Query from database
}

func (m *CustomManager) Create(ctx context.Context, state *state.State) error {
    // Insert into database
}

// Implement other methods...
```

Future storage backends:
- Redis (for distributed systems)
- PostgreSQL (for persistent storage)
- DynamoDB (for serverless)
- MongoDB (for document storage)

## Performance Considerations

- **Memory Usage**: Monitor session count and message count
- **Cleanup Interval**: Balance between cleanup frequency and performance
- **Message Limits**: Set appropriate limits based on your use case
- **Session TTL**: Set TTL based on expected conversation duration

## Learning Resources

- [State Management Package](../../core/state/)
- [Manager Interface](../../core/state/types.go)
- [Memory Implementation](../../core/state/memory.go)
- [Agent Integration](../../docs/guides/state-integration.md)
