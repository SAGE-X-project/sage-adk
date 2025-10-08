# Middleware Agent Example

This example demonstrates the middleware chain system in SAGE ADK, showing how to use builtin middleware and create custom middleware for request processing.

## Features

The example agent uses a comprehensive middleware chain with:

### Builtin Middleware

1. **Recovery** - Recovers from panics and converts them to errors
2. **Logger** - Logs incoming requests and responses with timing
3. **RequestID** - Adds unique request ID to context
4. **Timer** - Tracks execution time and adds to response metadata
5. **Validator** - Validates message structure before processing
6. **ContentFilter** - Filters out prohibited content
7. **RateLimiter** - Limits requests to 10 per minute
8. **Timeout** - Sets 30-second timeout for requests
9. **Metadata** - Adds service metadata to responses

### Custom Middleware

The example also shows how to create custom middleware:

```go
customLogger := func(next middleware.Handler) middleware.Handler {
    return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
        log.Printf("🔵 Processing message: %s", msg.MessageID)

        // Add custom context
        ctx = middleware.ContextWithMetadata(ctx, map[string]interface{}{
            "custom_processor": "middleware-agent",
            "timestamp":        time.Now().Unix(),
        })

        resp, err := next(ctx, msg)

        if err != nil {
            log.Printf("🔴 Processing failed: %v", err)
        } else {
            log.Printf("🟢 Processing completed successfully")
        }

        return resp, err
    }
}
```

## Running the Example

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY="your-api-key-here"
```

2. Run the agent:
```bash
go run -tags examples examples/middleware-agent/main.go
```

3. The agent will start on http://localhost:8080

## Testing the Agent

You can test the agent using curl or the A2A protocol:

### Send a normal message:
```bash
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-001",
      "role": "user",
      "parts": [{"kind": "text", "text": "Hello!"}]
    }
  }'
```

### Test content filter (will be blocked):
```bash
curl -X POST http://localhost:8080/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "messageId": "msg-002",
      "role": "user",
      "parts": [{"kind": "text", "text": "This contains spam"}]
    }
  }'
```

### Test rate limiting (send 11 requests quickly):
```bash
for i in {1..11}; do
  curl -X POST http://localhost:8080/v1/message \
    -H "Content-Type: application/json" \
    -d "{
      \"message\": {
        \"messageId\": \"msg-$i\",
        \"role\": \"user\",
        \"parts\": [{\"kind\": \"text\", \"text\": \"Request $i\"}]
      }
    }"
done
```

## Middleware Execution Order

Middleware executes in the order added to the chain:

```
Request Flow (Before Handler):
1. Recovery (outermost)
2. Logger
3. Custom Logger
4. RequestID
5. Timer
6. Validator
7. ContentFilter
8. RateLimiter
9. Timeout
10. Metadata
→ Handler

Response Flow (After Handler):
← Handler
10. Metadata (innermost)
9. Timeout
8. RateLimiter
7. ContentFilter
6. Validator
5. Timer
4. RequestID
3. Custom Logger
2. Logger
1. Recovery (outermost)
```

## Middleware Capabilities

### Request Processing
- Add request ID for tracing
- Validate message structure
- Filter prohibited content
- Enforce rate limits
- Set request timeouts

### Response Enhancement
- Add execution timing metadata
- Include service metadata
- Log request/response pairs

### Error Handling
- Recover from panics
- Log errors with context
- Return structured error responses

### Context Management
- Store request-scoped data
- Pass metadata between middleware
- Track request lifecycle

## Creating Custom Middleware

To create custom middleware:

```go
func MyMiddleware(config MyConfig) middleware.Middleware {
    return func(next middleware.Handler) middleware.Handler {
        return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
            // Before handler: Pre-processing
            log.Println("Before handler")

            // Modify context
            ctx = context.WithValue(ctx, "my-key", "my-value")

            // Call next middleware/handler
            resp, err := next(ctx, msg)

            // After handler: Post-processing
            log.Println("After handler")

            // Modify response
            if resp != nil && resp.Metadata == nil {
                resp.Metadata = make(map[string]interface{})
            }
            resp.Metadata["my-metadata"] = "my-value"

            return resp, err
        }
    }
}
```

## Use Cases

### Content Moderation
Use ContentFilter to block inappropriate content:
```go
moderationFilter := middleware.ContentFilter(func(content string) (bool, string) {
    if containsInappropriate(content) {
        return false, "inappropriate content detected"
    }
    return true, ""
})
```

### API Rate Limiting
Protect your service from abuse:
```go
rateLimiter := middleware.RateLimiter(middleware.RateLimiterConfig{
    MaxRequests: 100,
    Window:      1 * time.Minute,
})
```

### Request Tracing
Track requests across services:
```go
tracer := func(next middleware.Handler) middleware.Handler {
    return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
        traceID := generateTraceID()
        ctx = context.WithValue(ctx, "trace-id", traceID)

        log.Printf("Starting trace: %s", traceID)
        resp, err := next(ctx, msg)
        log.Printf("Ending trace: %s", traceID)

        return resp, err
    }
}
```

### Performance Monitoring
Monitor request performance:
```go
monitor := func(next middleware.Handler) middleware.Handler {
    return func(ctx context.Context, msg *types.Message) (*types.Message, error) {
        start := time.Now()
        resp, err := next(ctx, msg)
        duration := time.Since(start)

        // Report to monitoring system
        reportMetric("request_duration", duration)

        return resp, err
    }
}
```

## Best Practices

1. **Order Matters**: Place middleware in the correct order:
   - Recovery should be first (outermost)
   - Validation early to fail fast
   - Timeout/RateLimit before expensive operations
   - Metadata last (innermost)

2. **Error Handling**: Always handle errors gracefully:
   ```go
   resp, err := next(ctx, msg)
   if err != nil {
       log.Printf("Error: %v", err)
       // Don't swallow errors
       return resp, err
   }
   ```

3. **Context Values**: Use typed keys for context values:
   ```go
   type contextKey string
   const myKey contextKey = "my-key"
   ctx = context.WithValue(ctx, myKey, value)
   ```

4. **Immutability**: Don't modify the original message, create copies:
   ```go
   newMsg := *msg // Copy
   newMsg.Metadata = make(map[string]interface{})
   ```

5. **Performance**: Keep middleware lightweight:
   - Avoid blocking operations
   - Use goroutines for async work
   - Cache expensive computations

## Learning Resources

- [Middleware Design Pattern](../../docs/architecture/middleware.md)
- [Core Middleware Package](../../core/middleware/)
- [Creating Custom Middleware](../../docs/guides/custom-middleware.md)
- [A2A Protocol](../../docs/protocols/a2a.md)
