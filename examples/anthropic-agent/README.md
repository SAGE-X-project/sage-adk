# Anthropic Claude Agent

A chatbot agent powered by Anthropic's Claude AI using SAGE ADK and the A2A protocol.

## Features

- **Claude 3 Integration**: Uses Anthropic's latest Claude 3 models
- **A2A Protocol**: Standard agent-to-agent communication
- **Token Usage Tracking**: Monitors token consumption
- **Graceful Shutdown**: Clean lifecycle management
- **Multiple Models**: Support for Claude 3 Opus, Sonnet, and Haiku

## Prerequisites

- Go 1.21 or later
- Anthropic API key ([Get one here](https://console.anthropic.com/))

## Setup

1. Set your Anthropic API key:

```bash
export ANTHROPIC_API_KEY="sk-ant-your-api-key-here"
```

2. (Optional) Choose a Claude model:

```bash
# Claude 3 Opus - Most powerful, best for complex tasks
export ANTHROPIC_MODEL="claude-3-opus-20240229"

# Claude 3 Sonnet - Balanced performance and speed (default)
export ANTHROPIC_MODEL="claude-3-sonnet-20240229"

# Claude 3 Haiku - Fastest, most cost-effective
export ANTHROPIC_MODEL="claude-3-haiku-20240307"
```

3. Run the agent:

```bash
go run -tags examples main.go
```

The agent will start listening on `http://localhost:8080`.

## Claude Models Comparison

| Model | Strengths | Use Cases | Speed | Cost |
|-------|-----------|-----------|-------|------|
| **Claude 3 Opus** | Most intelligent, best at complex analysis | Research, strategy, detailed writing | Slow | High |
| **Claude 3 Sonnet** | Balanced performance and speed | General chatbots, customer service | Medium | Medium |
| **Claude 3 Haiku** | Fastest responses, cost-effective | Simple Q&A, high-volume tasks | Fast | Low |

## Usage

### Using the Test Client

```bash
# In terminal 1: Start the agent
go run -tags examples main.go

# In terminal 2: Send a message
cd ../simple-agent
go run -tags examples client.go "What makes Claude different from other AI assistants?"
```

### Using A2A Client in Code

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sage-x-project/sage-adk/adapters/a2a"
    "github.com/sage-x-project/sage-adk/pkg/types"
)

func main() {
    client, _ := a2a.NewClient("http://localhost:8080/")

    msg := &types.Message{
        MessageID: types.GenerateMessageID(),
        Role:      types.MessageRoleUser,
        Parts: []types.Part{
            types.NewTextPart("Explain quantum computing in simple terms"),
        },
    }

    response, err := client.SendMessage(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }

    for _, part := range response.Parts {
        if textPart, ok := part.(*types.TextPart); ok {
            fmt.Println("Claude:", textPart.Text)
        }
    }
}
```

### Using HTTP POST

```bash
curl -X POST http://localhost:8080/a2a/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "user",
      "parts": [
        {
          "kind": "text",
          "text": "What are the key principles of effective communication?"
        }
      ]
    }
  }'
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ANTHROPIC_API_KEY` | Anthropic API key (required) | - |
| `ANTHROPIC_MODEL` | Claude model to use | `claude-3-sonnet-20240229` |

### Programmatic Configuration

```go
provider := llm.Anthropic(&llm.AnthropicConfig{
    APIKey: "sk-ant-...",
    Model:  "claude-3-opus-20240229",
})

agent := builder.NewAgent("claude-agent").
    WithLLM(provider).
    Build()
```

## Advanced Features

### System Prompts

Customize Claude's behavior with system prompts:

```go
request := &llm.CompletionRequest{
    Messages: []llm.Message{
        {
            Role: llm.RoleSystem,
            Content: "You are a Python programming expert. " +
                   "Provide concise, executable code examples.",
        },
        {Role: llm.RoleUser, Content: userQuestion},
    },
}
```

### Temperature Control

Adjust response creativity:

```go
request := &llm.CompletionRequest{
    Messages:    messages,
    Temperature: 0.0,  // 0.0 = deterministic, 1.0 = creative
}
```

### Token Limits

Set maximum response length:

```go
request := &llm.CompletionRequest{
    Messages:  messages,
    MaxTokens: 1000,  // Limit response to 1000 tokens
}
```

## Token Usage Monitoring

The agent automatically logs token usage for each request:

```
📊 Usage - Prompt: 45 tokens, Completion: 128 tokens, Total: 173 tokens
```

**Token Costs** (approximate, as of 2024):
- **Claude 3 Opus**: $15/$75 per 1M input/output tokens
- **Claude 3 Sonnet**: $3/$15 per 1M input/output tokens
- **Claude 3 Haiku**: $0.25/$1.25 per 1M input/output tokens

## Example Interactions

### Complex Analysis

```
User: Analyze the trade-offs between microservices and monolithic architecture.

Claude: When comparing microservices and monolithic architectures, several key
trade-offs emerge:

**Microservices Advantages:**
1. Independent scaling of services
2. Technology diversity
3. Fault isolation
...
```

### Creative Writing

```
User: Write a haiku about artificial intelligence.

Claude: Silicon neurons
Learning patterns, thoughts emerge
Mind without the flesh
```

### Code Generation

```
User: Create a Python function to calculate Fibonacci numbers recursively.

Claude: Here's a recursive Fibonacci function:

def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
...
```

## Error Handling

The agent handles various error scenarios:

| Error | Handling |
|-------|----------|
| Missing API key | Fatal error at startup |
| Invalid API key | Returns error to client |
| Rate limit exceeded | Returns error with retry suggestion |
| Empty message | Returns error to client |
| Service unavailable | Logs error and returns to client |

## Best Practices

### 1. API Key Security

```bash
# Use environment variables
export ANTHROPIC_API_KEY="sk-ant-..."

# Never hardcode in code
# ❌ Bad
provider := llm.Anthropic(&llm.AnthropicConfig{
    APIKey: "sk-ant-hardcoded-key",
})

# ✅ Good
provider := llm.Anthropic()  // Uses ANTHROPIC_API_KEY env var
```

### 2. Model Selection

Choose the right model for your use case:

```bash
# Complex reasoning, analysis, creative writing
export ANTHROPIC_MODEL="claude-3-opus-20240229"

# General chatbot, balanced performance
export ANTHROPIC_MODEL="claude-3-sonnet-20240229"

# High-volume, simple tasks, cost-sensitive
export ANTHROPIC_MODEL="claude-3-haiku-20240307"
```

### 3. Context Management

Keep conversations focused:

```go
// ❌ Sending too much context
messages := []llm.Message{
    {Role: llm.RoleSystem, Content: veryLongSystemPrompt},
    // ... 50 previous messages ...
    {Role: llm.RoleUser, Content: currentQuestion},
}

// ✅ Summarize or truncate context
messages := []llm.Message{
    {Role: llm.RoleSystem, Content: "You are a helpful assistant."},
    {Role: llm.RoleAssistant, Content: summarizeConversation(prevMessages)},
    {Role: llm.RoleUser, Content: currentQuestion},
}
```

### 4. Error Recovery

Implement retry logic for transient errors:

```go
func handleMessageWithRetry(ctx context.Context, msg agent.MessageContext) error {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        resp, err := provider.Complete(ctx, request)
        if err == nil {
            return msg.Reply(resp.Content)
        }

        // Check if error is retryable
        if isRateLimitError(err) || isServiceUnavailable(err) {
            time.Sleep(time.Duration(i+1) * time.Second)
            continue
        }

        return err
    }
    return errors.New("max retries exceeded")
}
```

## Troubleshooting

### "ANTHROPIC_API_KEY environment variable is required"

Set your API key:
```bash
export ANTHROPIC_API_KEY="sk-ant-your-key"
```

### "invalid API key"

- Check that your key starts with `sk-ant-`
- Verify key in [Anthropic Console](https://console.anthropic.com/)
- Ensure no extra spaces or quotes

### "rate limit exceeded"

- You've exceeded your API quota
- Wait and retry, or upgrade your plan
- Implement exponential backoff

### High token usage

- Use Claude 3 Haiku for simple tasks
- Implement conversation summarization
- Set `MaxTokens` limits
- Truncate long prompts

## Comparison with OpenAI

| Feature | Claude (Anthropic) | GPT (OpenAI) |
|---------|-------------------|--------------|
| Latest Model | Claude 3 Opus | GPT-4 Turbo |
| Context Window | 200K tokens | 128K tokens |
| System Prompt | Native support | Native support |
| Streaming | ✅ Yes | ✅ Yes |
| Function Calling | ✅ Yes | ✅ Yes |
| JSON Mode | ✅ Yes | ✅ Yes |
| Vision | ✅ Yes | ✅ Yes |

### Migration from OpenAI

The SAGE ADK makes it easy to switch:

```go
// OpenAI
provider := llm.OpenAI(&llm.OpenAIConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})

// Anthropic (same interface!)
provider := llm.Anthropic(&llm.AnthropicConfig{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-3-opus-20240229",
})

// Rest of code remains unchanged!
agent := builder.NewAgent("chatbot").
    WithLLM(provider).
    Build()
```

## Next Steps

- Add streaming responses for real-time interaction
- Implement conversation history management
- Add multi-turn dialogue support
- Integrate with SAGE protocol for blockchain identity
- Deploy to production with monitoring

## Resources

- [Anthropic Documentation](https://docs.anthropic.com/)
- [Claude API Reference](https://docs.anthropic.com/claude/reference/)
- [Model Comparison](https://docs.anthropic.com/claude/docs/models-overview)
- [Best Practices](https://docs.anthropic.com/claude/docs/prompt-engineering)

## License

LGPL-3.0-or-later
