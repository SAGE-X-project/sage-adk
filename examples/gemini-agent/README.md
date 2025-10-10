# Google Gemini Agent

A chatbot agent powered by Google's Gemini AI using SAGE ADK and the A2A protocol.

## Features

- **Gemini Pro Integration**: Uses Google's latest Gemini AI models
- **A2A Protocol**: Standard agent-to-agent communication
- **Token Usage Tracking**: Monitors token consumption
- **Graceful Shutdown**: Clean lifecycle management
- **Multiple Models**: Support for Gemini Pro and Gemini Pro Vision

## Prerequisites

- Go 1.21 or later
- Google AI API key ([Get one here](https://makersuite.google.com/app/apikey))

## Setup

1. Set your Google AI API key:

```bash
# Option 1: Gemini-specific key
export GEMINI_API_KEY="AIza...your-api-key-here"

# Option 2: General Google API key
export GOOGLE_API_KEY="AIza...your-api-key-here"
```

2. (Optional) Choose a Gemini model:

```bash
# Gemini Pro - Text generation (default)
export GEMINI_MODEL="gemini-pro"

# Gemini Pro Vision - Multimodal (text + images)
export GEMINI_MODEL="gemini-pro-vision"

# Gemini 1.5 Pro - Extended context window
export GEMINI_MODEL="gemini-1.5-pro-latest"
```

3. Run the agent:

```bash
go run -tags examples main.go
```

The agent will start listening on `http://localhost:8080`.

## Gemini Models

| Model | Context Window | Capabilities | Best For |
|-------|---------------|--------------|----------|
| **gemini-pro** | 30,720 tokens | Text generation | General chatbots, Q&A |
| **gemini-pro-vision** | 12,288 tokens | Text + images | Image analysis, OCR |
| **gemini-1.5-pro** | 1M+ tokens | Extended context | Long documents, code |

## Usage

### Using the Test Client

```bash
# In terminal 1: Start the agent
go run -tags examples main.go

# In terminal 2: Send a message
cd ../simple-agent
go run -tags examples client.go "What are the key features of Google Gemini?"
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
            types.NewTextPart("Explain how neural networks work"),
        },
    }

    response, err := client.SendMessage(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }

    for _, part := range response.Parts {
        if textPart, ok := part.(*types.TextPart); ok {
            fmt.Println("Gemini:", textPart.Text)
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
          "text": "What makes Google Gemini unique?"
        }
      ]
    }
  }'
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GEMINI_API_KEY` | Gemini API key (required) | - |
| `GOOGLE_API_KEY` | Google API key (fallback) | - |
| `GEMINI_MODEL` | Gemini model to use | `gemini-pro` |

### Programmatic Configuration

```go
provider := llm.Gemini(&llm.GeminiConfig{
    APIKey: "AIza...",
    Model:  "gemini-pro",
})

agent := builder.NewAgent("gemini-agent").
    WithLLM(provider).
    Build()
```

## Advanced Features

### System Instructions

Customize Gemini's behavior with system instructions:

```go
request := &llm.CompletionRequest{
    Messages: []llm.Message{
        {
            Role: llm.RoleSystem,
            Content: "You are a JavaScript expert. " +
                   "Provide modern ES6+ code examples.",
        },
        {Role: llm.RoleUser, Content: userQuestion},
    },
}
```

### Temperature Control

Adjust response creativity (0.0 - 1.0):

```go
request := &llm.CompletionRequest{
    Messages:    messages,
    Temperature: 0.9,  // Higher = more creative
}
```

### Token Limits

Set maximum response length:

```go
request := &llm.CompletionRequest{
    Messages:  messages,
    MaxTokens: 2048,
}
```

### Top-P (Nucleus Sampling)

Control diversity of responses:

```go
request := &llm.CompletionRequest{
    Messages: messages,
    TopP:     0.95,  // 0.0 - 1.0
}
```

## Token Usage Monitoring

The agent automatically logs token usage:

```
 Usage - Prompt: 52 tokens, Completion: 143 tokens, Total: 195 tokens
```

**Pricing** (as of 2024):
- **Gemini Pro**: Free for up to 60 requests/minute
- **Gemini Pro Vision**: Free for up to 60 requests/minute
- **Gemini 1.5 Pro**: Pay-as-you-go pricing

## Example Interactions

### General Knowledge

```
User: What is quantum entanglement?

Gemini: Quantum entanglement is a phenomenon where two or more particles become
correlated in such a way that the state of one particle instantly influences the
state of the other, regardless of the distance between them...
```

### Code Generation

```
User: Write a Python function to reverse a string.

Gemini: Here's a Python function to reverse a string:

def reverse_string(text):
    return text[::-1]

# Example usage
result = reverse_string("Hello, World!")
print(result)  # Output: !dlroW ,olleH
```

### Creative Writing

```
User: Write a short poem about AI.

Gemini: Silicon dreams awake,
Patterns learned from data's lake,
Neural pathways intertwine,
Human thought and code combine.
```

## Error Handling

The agent handles various error scenarios:

| Error | Handling |
|-------|----------|
| Missing API key | Fatal error at startup |
| Invalid API key | Returns error to client |
| Permission denied | Returns error about quota/permissions |
| Rate limit exceeded | Returns error with retry suggestion |
| Empty message | Returns error to client |
| Service unavailable | Logs error and returns to client |

## Gemini-Specific Features

### 1. Long Context Support

Gemini 1.5 Pro supports up to 1M tokens:

```go
provider := llm.Gemini(&llm.GeminiConfig{
    Model: "gemini-1.5-pro-latest",
})

// Can process entire books or large codebases
request := &llm.CompletionRequest{
    Messages: []llm.Message{
        {Role: llm.RoleUser, Content: veryLongDocument + "\n\nSummarize this."},
    },
}
```

### 2. System Instructions

Gemini has native support for system instructions:

```go
// System message is automatically converted to systemInstruction
messages := []llm.Message{
    {
        Role: llm.RoleSystem,
        Content: "You are a medical expert. Provide evidence-based answers.",
    },
    {Role: llm.RoleUser, Content: "What causes diabetes?"},
}
```

### 3. Safety Settings

Gemini includes built-in safety filters for:
- Harassment
- Hate speech
- Sexually explicit content
- Dangerous content

These are automatically applied by the API.

## Best Practices

### 1. API Key Security

```bash
# Use environment variables
export GEMINI_API_KEY="AIza..."

# Never hardcode in code
#  Bad
provider := llm.Gemini(&llm.GeminiConfig{
    APIKey: "AIza-hardcoded-key",
})

#  Good
provider := llm.Gemini()  // Uses environment variable
```

### 2. Model Selection

```bash
# Text-only tasks
export GEMINI_MODEL="gemini-pro"

# Tasks involving images
export GEMINI_MODEL="gemini-pro-vision"

# Large documents (books, codebases)
export GEMINI_MODEL="gemini-1.5-pro-latest"
```

### 3. Rate Limit Management

```go
// Implement exponential backoff
func makeRequestWithRetry(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        resp, err := provider.Complete(ctx, req)
        if err == nil {
            return resp, nil
        }

        if strings.Contains(err.Error(), "rate limit") {
            time.Sleep(time.Duration(math.Pow(2, float64(i))) * time.Second)
            continue
        }

        return nil, err
    }
    return nil, errors.New("max retries exceeded")
}
```

### 4. Context Window Management

```go
// Keep track of conversation length
const maxTokens = 30000  // Leave room for response

func truncateHistory(messages []llm.Message) []llm.Message {
    // Estimate ~4 chars per token
    totalChars := 0
    for _, msg := range messages {
        totalChars += len(msg.Content)
    }

    if totalChars/4 > maxTokens {
        // Keep system message and recent messages
        return append(messages[:1], messages[len(messages)-5:]...)
    }

    return messages
}
```

## Troubleshooting

### "GEMINI_API_KEY or GOOGLE_API_KEY environment variable is required"

Set your API key:
```bash
export GEMINI_API_KEY="AIza-your-key"
# or
export GOOGLE_API_KEY="AIza-your-key"
```

### "invalid API key" or "API key not valid"

- Get a new key from [Google AI Studio](https://makersuite.google.com/app/apikey)
- Verify key starts with `AIza`
- Check for extra spaces or quotes

### "API key lacks permissions or quota exceeded"

- You've exceeded free tier limits
- Enable billing in Google Cloud Console
- Check API quotas in Cloud Console

### "rate limit exceeded"

- Free tier: 60 requests/minute
- Wait and retry with exponential backoff
- Consider upgrading to paid tier

### Response quality issues

- Adjust temperature (0.7-1.0 for creativity)
- Provide clearer system instructions
- Break complex queries into smaller parts
- Use examples in your prompts

## Comparison with Other Providers

| Feature | Gemini | GPT-4 | Claude 3 |
|---------|--------|-------|----------|
| Context Window | 1M+ tokens | 128K | 200K |
| Free Tier |  60 req/min |  No |  No |
| Vision |  Yes |  Yes |  Yes |
| Streaming |  Yes |  Yes |  Yes |
| Native Safety |  Built-in |  Moderation API |  Limited |
| Pricing (1M tokens) | Free tier + paid | $10 | $3-15 |

### Migration from OpenAI/Anthropic

The SAGE ADK makes switching easy:

```go
// OpenAI
provider := llm.OpenAI(&llm.OpenAIConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})

// Gemini (same interface!)
provider := llm.Gemini(&llm.GeminiConfig{
    APIKey: os.Getenv("GEMINI_API_KEY"),
    Model:  "gemini-pro",
})

// Rest of code unchanged!
agent := builder.NewAgent("chatbot").
    WithLLM(provider).
    Build()
```

## Gemini Advantages

1. **Free Tier**: Generous free tier for development
2. **Long Context**: Up to 1M tokens with Gemini 1.5 Pro
3. **Multimodal**: Native image understanding
4. **Safety**: Built-in content filtering
5. **Integration**: Easy integration with Google services

## Next Steps

- Add streaming responses for real-time interaction
- Implement conversation history management
- Add image input support (Gemini Pro Vision)
- Integrate with SAGE protocol for security
- Deploy to production with monitoring

## Resources

- [Google AI Studio](https://makersuite.google.com/)
- [Gemini API Documentation](https://ai.google.dev/docs)
- [Model Card](https://ai.google.dev/models/gemini)
- [Pricing](https://ai.google.dev/pricing)
- [Safety Settings](https://ai.google.dev/docs/safety_setting_gemini)

## License

LGPL-3.0-or-later
