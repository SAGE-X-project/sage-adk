# Streaming Agent

A real-time streaming chatbot agent that supports OpenAI, Anthropic, and Gemini with live response streaming.

## Features

- **Real-Time Streaming**: Responses arrive token-by-token in real-time
- **Multi-Provider Support**: Works with OpenAI, Anthropic, and Gemini
- **Provider Switching**: Easy switching between providers via environment variables
- **Chunk Monitoring**: Tracks and logs streaming chunks
- **Low Latency**: See responses as they're generated

## Prerequisites

- Go 1.21 or later
- API key for at least one provider (OpenAI, Anthropic, or Gemini)

## Setup

### OpenAI (Default)

```bash
export LLM_PROVIDER="openai"
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_MODEL="gpt-3.5-turbo"  # Optional
```

### Anthropic Claude

```bash
export LLM_PROVIDER="anthropic"
export ANTHROPIC_API_KEY="sk-ant-your-api-key"
export ANTHROPIC_MODEL="claude-3-sonnet-20240229"  # Optional
```

### Google Gemini

```bash
export LLM_PROVIDER="gemini"
export GEMINI_API_KEY="AIza-your-api-key"
export GEMINI_MODEL="gemini-pro"  # Optional
```

## Running the Agent

```bash
go run -tags examples main.go
```

The agent will start on `http://localhost:8080`.

## Usage

### Using Test Client

```bash
# In terminal 1: Start streaming agent
export LLM_PROVIDER="openai"
export OPENAI_API_KEY="sk-..."
go run -tags examples main.go

# In terminal 2: Send message
cd ../simple-agent
go run -tags examples client.go "Tell me a story about AI"
```

You'll see in the server logs:
```
 Received message: Tell me a story about AI
 Streaming response...
 First chunk received: "Once"
 Streaming complete - Total chunks: 147, Total length: 523 characters
```

### Streaming Output Example

```
 Streaming response...
 First chunk received: "Once"
 Chunk 2: " upon"
 Chunk 3: " a"
 Chunk 4: " time"
...
 Streaming complete - Total chunks: 147
```

## How Streaming Works

### Traditional (Non-Streaming) Response

```
User sends request → Wait... → Complete response arrives → Display
              [========= 5-10 seconds =========]
```

### Streaming Response

```
User sends request → First tokens → More tokens → ... → Complete
                    [0.1s]        [0.2s]        ...   [5s total]
                      ↓             ↓                    ↓
                   Display       Display              Display
```

### Code Example

```go
// Non-streaming
response, err := provider.Complete(ctx, request)
fmt.Println(response.Content)  // Waits for complete response

// Streaming
var fullResponse strings.Builder
err := provider.Stream(ctx, request, func(chunk string) error {
    fullResponse.WriteString(chunk)
    fmt.Print(chunk)  // Display immediately!
    return nil
})
```

## Architecture

```

   Client    

        HTTP POST
       ↓

  Streaming      
  Agent          

  OnMessage()    
     ↓           
  provider.      
  Stream()       
     ↓           
  Collect        
  chunks         

      Stream chunks
     ↓

  LLM Provider   
  (OpenAI/       
   Anthropic/    
   Gemini)       

```

## Streaming Benefits

### 1. Improved User Experience

- Users see responses immediately
- Perceived latency is much lower
- More engaging interaction

### 2. Lower Time-to-First-Token

```
Non-streaming: 5-10 seconds to first word
Streaming:     0.1-0.5 seconds to first word
```

### 3. Better for Long Responses

```
Non-streaming: Wait 30s for essay
Streaming:     Start reading after 0.5s, full text in 30s
```

### 4. Graceful Handling of Interruptions

- User can stop generation early
- Save on API costs
- Better control

## Advanced Usage

### Custom Chunk Processing

```go
func processChunks(provider llm.Provider) agent.MessageHandler {
    return func(ctx context.Context, msg agent.MessageContext) error {
        var buffer strings.Builder
        wordCount := 0

        err := provider.Stream(ctx, request, func(chunk string) error {
            buffer.WriteString(chunk)

            // Count words
            if strings.Contains(chunk, " ") {
                wordCount++
            }

            // Send to WebSocket
            sendToWebSocket(chunk)

            // Log every 10 words
            if wordCount%10 == 0 {
                log.Printf("Progress: %d words", wordCount)
            }

            return nil
        })

        return msg.Reply(buffer.String())
    }
}
```

### Streaming to WebSocket

```go
func handleWebSocketStream(ws *websocket.Conn, provider llm.Provider) error {
    err := provider.Stream(ctx, request, func(chunk string) error {
        // Send each chunk to WebSocket client
        return ws.WriteMessage(websocket.TextMessage, []byte(chunk))
    })

    // Send completion signal
    ws.WriteMessage(websocket.TextMessage, []byte("[DONE]"))
    return err
}
```

### Progress Indicators

```go
func streamWithProgress(provider llm.Provider) error {
    var buffer strings.Builder
    chunkCount := 0
    startTime := time.Now()

    err := provider.Stream(ctx, request, func(chunk string) error {
        buffer.WriteString(chunk)
        chunkCount++

        // Show progress every 0.5s
        if time.Since(startTime) > 500*time.Millisecond {
            fmt.Printf("\rReceived %d chunks, %d chars...",
                chunkCount, buffer.Len())
            startTime = time.Now()
        }

        return nil
    })

    fmt.Printf("\rComplete: %d chunks, %d chars\n",
        chunkCount, buffer.Len())
    return err
}
```

### Streaming with Timeout

```go
func streamWithTimeout(provider llm.Provider) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err := provider.Stream(ctx, request, func(chunk string) error {
        // Process chunk
        return nil
    })

    if ctx.Err() == context.DeadlineExceeded {
        return fmt.Errorf("streaming timeout after 30s")
    }

    return err
}
```

## Provider Comparison

| Feature | OpenAI | Anthropic | Gemini |
|---------|--------|-----------|--------|
| Streaming |  SSE |  SSE |  SSE |
| First Chunk Latency | ~300ms | ~400ms | ~200ms |
| Chunk Size | 1-5 tokens | 1-3 tokens | 1-5 tokens |
| Max Tokens/Chunk | ~5 | ~3 | ~5 |

## Performance Tips

### 1. Adjust Temperature

Lower temperature = faster, more deterministic:
```go
request := &llm.CompletionRequest{
    Temperature: 0.3,  // Faster than 0.9
}
```

### 2. Set Max Tokens

Limit response length for faster completion:
```go
request := &llm.CompletionRequest{
    MaxTokens: 500,  // Limits total response
}
```

### 3. Buffer Chunks

Reduce UI updates by buffering:
```go
var buffer strings.Builder
bufferSize := 0

provider.Stream(ctx, request, func(chunk string) error {
    buffer.WriteString(chunk)
    bufferSize += len(chunk)

    // Update UI every 50 chars
    if bufferSize >= 50 {
        updateUI(buffer.String())
        buffer.Reset()
        bufferSize = 0
    }

    return nil
})
```

## Troubleshooting

### "Streaming response incomplete"

- Check context isn't timing out
- Verify network connection is stable
- Increase timeout if needed

### "High latency between chunks"

- Check network speed
- Try different provider
- Verify API region/endpoint

### "Chunks arriving out of order"

- This shouldn't happen with SSE
- Check for proxy/CDN issues
- Verify HTTP/2 support

### "Memory usage grows during streaming"

```go
//  Bad - grows indefinitely
var allChunks []string
provider.Stream(ctx, req, func(chunk string) error {
    allChunks = append(allChunks, chunk)
    return nil
})

//  Good - constant memory
var buffer strings.Builder
provider.Stream(ctx, req, func(chunk string) error {
    buffer.WriteString(chunk)
    // Process immediately
    return nil
})
```

## Best Practices

### 1. Handle Errors Gracefully

```go
err := provider.Stream(ctx, req, func(chunk string) error {
    if err := processChunk(chunk); err != nil {
        log.Printf("Chunk processing error: %v", err)
        return err  // Stop streaming
    }
    return nil
})

if err != nil {
    // Send partial response + error notice
    return msg.Reply(buffer.String() + "\n[Error: Stream interrupted]")
}
```

### 2. Implement Backpressure

```go
chunkChan := make(chan string, 10)  // Buffer 10 chunks

go func() {
    provider.Stream(ctx, req, func(chunk string) error {
        select {
        case chunkChan <- chunk:
            return nil
        case <-time.After(5 * time.Second):
            return fmt.Errorf("backpressure timeout")
        }
    })
    close(chunkChan)
}()

for chunk := range chunkChan {
    processChunk(chunk)
}
```

### 3. Monitor Performance

```go
metrics := struct {
    firstChunkLatency time.Duration
    totalChunks       int
    totalChars        int
    duration          time.Duration
}{}

startTime := time.Now()
firstChunk := true

provider.Stream(ctx, req, func(chunk string) error {
    if firstChunk {
        metrics.firstChunkLatency = time.Since(startTime)
        firstChunk = false
    }
    metrics.totalChunks++
    metrics.totalChars += len(chunk)
    return nil
})

metrics.duration = time.Since(startTime)
log.Printf("Streaming metrics: %+v", metrics)
```

## Example Output

```
Streaming Chatbot Agent starting...
Provider: openai
Model: gpt-3.5-turbo
Listening on http://localhost:8080
Responses will be streamed in real-time!

 Received message: Tell me a story about AI
 Streaming response...
 First chunk received: "Once"
 Streaming complete - Total chunks: 147, Total length: 523 characters
 Full response: Once upon a time, in a world not too different from ours...
```

## Next Steps

- Integrate with WebSocket for browser streaming
- Add Server-Sent Events (SSE) endpoint
- Implement streaming UI components
- Add streaming metrics dashboard

## Resources

- [OpenAI Streaming](https://platform.openai.com/docs/api-reference/streaming)
- [Anthropic Streaming](https://docs.anthropic.com/claude/reference/streaming)
- [Gemini Streaming](https://ai.google.dev/tutorials/rest_streaming)

## License

LGPL-3.0-or-later
