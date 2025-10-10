# Multi-Agent Chat Example

This example demonstrates a multi-agent system where multiple AI agents collaborate to answer user questions through intelligent routing and specialization.

## Architecture

```
┌──────────┐      ┌─────────────┐      ┌──────────────┐
│  Client  │─────>│ Coordinator │─────>│ Specialist   │
└──────────┘      │   Agent     │      │   Agents     │
                  └─────────────┘      └──────────────┘
                         │                    │
                         └────────────────────┘
                          Collaboration Flow
```

## Components

### 1. Coordinator Agent (Port 8090)
- Routes incoming questions to appropriate specialist agents
- Uses keyword matching to determine the best agent for each question
- Aggregates responses and returns them to the client

### 2. Math Agent (Port 8091)
- Specializes in mathematical questions
- Handles calculations, equations, and numerical problems
- Keywords: "math", "calculate", "number"

### 3. Code Agent (Port 8092)
- Specializes in programming questions
- Answers questions about code, algorithms, and programming concepts
- Keywords: "code", "program", "function"

### 4. General Agent (Port 8093)
- Handles general knowledge questions
- Default agent for questions that don't match other specializations
- Covers history, science, culture, etc.

## Features

✅ **Intelligent Routing**: Automatically routes questions to the right specialist
✅ **Agent Collaboration**: Agents communicate through HTTP/A2A protocol
✅ **Scalable Architecture**: Easy to add new specialist agents
✅ **Graceful Shutdown**: All agents shut down cleanly
✅ **Interactive Demo**: Built-in demonstration mode

## Running the Example

### Start the System

```bash
# Run all agents
go run main.go

# Run with interactive demo
go run main.go --demo
```

### Test with curl

```bash
# Ask a math question
curl -X POST http://localhost:8090/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "messageId": "msg-1",
    "role": "user",
    "parts": [{
      "kind": "text",
      "text": "What is 123 times 456?"
    }]
  }'

# Ask a coding question
curl -X POST http://localhost:8090/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "messageId": "msg-2",
    "role": "user",
    "parts": [{
      "kind": "text",
      "text": "How do I write a function in Go?"
    }]
  }'

# Ask a general question
curl -X POST http://localhost:8090/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "messageId": "msg-3",
    "role": "user",
    "parts": [{
      "kind": "text",
      "text": "What is the capital of France?"
    }]
  }'
```

### Using the Client SDK

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sage-x-project/sage-adk/client"
    "github.com/sage-x-project/sage-adk/pkg/types"
)

func main() {
    ctx := context.Background()

    // Connect to coordinator
    c, err := client.NewClient("http://localhost:8090")
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Send question
    msg := types.NewMessage(types.MessageRoleUser, []types.Part{
        types.NewTextPart("Calculate 10 factorial"),
    })

    response, err := c.SendMessage(ctx, msg)
    if err != nil {
        log.Fatal(err)
    }

    // Print response
    for _, part := range response.Parts {
        if textPart, ok := part.(*types.TextPart); ok {
            fmt.Println(textPart.Text)
        }
    }
}
```

## Logs Example

```
🚀 Starting Multi-Agent Chat System...
✅ All agents started successfully

📋 Available Agents:
  - Coordinator (port 8090): Routes questions to specialists
  - Math Agent (port 8091): Answers mathematical questions
  - Code Agent (port 8092): Answers programming questions
  - General Agent (port 8093): Handles general knowledge

📨 [Coordinator] Received: What is 123 times 456?
🔀 [Coordinator] Routing to Math Agent at http://localhost:8091
🔢 [Math Agent] Processing: What is 123 times 456?
💬 Response:
[Routed to Math Agent]
Math analysis: I can help with mathematical problems!
Your question: 'What is 123 times 456?'
```

## Extending the System

### Add a New Specialist Agent

```go
func startHistoryAgent(ctx context.Context) *agent.AgentImpl {
    handler := func(ctx context.Context, msg agent.MessageContext) error {
        question := msg.Text()
        log.Printf("📚 [History Agent] Processing: %s", question)

        // Your history-specific logic here
        response := "History answer: ..."

        return msg.Reply(response)
    }

    return createSpecialistAgent(
        "history-agent",
        "History specialist",
        handler,
        ":8094",
    )
}
```

### Update Coordinator Routing

```go
case strings.Contains(questionLower, "history") ||
     strings.Contains(questionLower, "historical"):
    targetURL = "http://localhost:8094"
    agentName = "History Agent"
```

## Integration with LLMs

To connect real LLM providers:

```go
import "github.com/sage-x-project/sage-adk/adapters/llm"

func startMathAgent(ctx context.Context) *agent.AgentImpl {
    // Create LLM provider
    provider := llm.OpenAI(&llm.OpenAIConfig{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4",
    })

    handler := func(ctx context.Context, msg agent.MessageContext) error {
        // Use LLM to process question
        systemPrompt := "You are a mathematics expert."
        response, err := provider.Chat(ctx, systemPrompt, msg.Text())
        if err != nil {
            return err
        }

        return msg.Reply(response)
    }

    b := builder.NewAgent("math-agent").
        WithLLM(provider).
        OnMessage(handler)

    // ... rest of setup
}
```

## Production Considerations

### Load Balancing
- Deploy multiple instances of each specialist agent
- Use a load balancer to distribute requests
- Implement health checks for automatic failover

### Monitoring
- Add metrics for routing decisions
- Track specialist agent performance
- Monitor message latency end-to-end

### Security
- Add authentication between agents
- Use HTTPS for production
- Implement rate limiting

### Storage
- Use Redis/PostgreSQL for shared state
- Store conversation history
- Cache common responses

## Troubleshooting

### Agents Not Starting
```bash
# Check if ports are available
lsof -i :8090
lsof -i :8091
lsof -i :8092
lsof -i :8093
```

### Connection Errors
```bash
# Verify agents are running
curl http://localhost:8090/health
curl http://localhost:8091/health
curl http://localhost:8092/health
curl http://localhost:8093/health
```

### High Latency
- Check network between agents
- Verify no resource constraints
- Review logs for errors

## Learn More

- [SAGE ADK Documentation](../../README.md)
- [Client SDK](../../client/doc.go)
- [Agent Builder](../../builder/doc.go)
- [Protocol Documentation](../../core/protocol/doc.go)

## License

LGPL-3.0-or-later
