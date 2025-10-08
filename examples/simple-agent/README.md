# Simple Chatbot Agent

A minimal example of building an AI chatbot agent using SAGE ADK with OpenAI integration and A2A protocol.

## Features

-  OpenAI-powered conversational AI
-  A2A (Agent-to-Agent) protocol support
-  Graceful shutdown handling
-  Clean and simple implementation

## Prerequisites

- Go 1.21 or later
- OpenAI API key

## Installation

1. Set your OpenAI API key:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

2. Run the agent:

```bash
# Full-featured chatbot with OpenAI
go run -tags examples main.go

# Or minimal echo agent
go run -tags examples minimal.go
```

The agent will start listening on `http://localhost:8080`.

**Note:** The `-tags examples` flag is required because these example files are excluded from normal builds to avoid conflicts with the test suite.

## Usage

The chatbot agent accepts messages via the A2A protocol. You can interact with it using:

1. **A2A Client** (from another SAGE ADK application)
2. **HTTP POST** to the A2A endpoint
3. **sage-a2a-go CLI tools**

### Example: Using the Test Client

The easiest way to test the agent is using the included client:

```bash
# In terminal 1: Start the agent
go run -tags examples main.go

# In terminal 2: Send a message
go run -tags examples client.go "Hello! How are you?"
```

### Example: Using A2A Client in Code

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
    // Create A2A client
    client, err := a2a.NewClient("http://localhost:8080/")
    if err != nil {
        log.Fatal(err)
    }

    // Create message
    msg := &types.Message{
        MessageID: types.GenerateMessageID(),
        Role:      types.MessageRoleUser,
        Parts: []types.Part{
            types.NewTextPart("Hello! How are you?"),
        },
    }

    // Send message and get response
    response, err := client.SendMessage(context.Background(), msg)
    if err != nil {
        log.Fatal(err)
    }

    // Print response
    for _, part := range response.Parts {
        if textPart, ok := part.(*types.TextPart); ok {
            fmt.Println("Agent:", textPart.Text)
        }
    }
}
```

### Example: Using HTTP POST

```bash
curl -X POST http://localhost:8080/a2a/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "user",
      "parts": [
        {
          "kind": "text",
          "text": "Hello! How are you?"
        }
      ]
    }
  }'
```

## Architecture

This example demonstrates the core concepts of SAGE ADK:

```
┌─────────────────┐
│   User/Client   │
└────────┬────────┘
         │ A2A Protocol
         ↓
┌─────────────────┐
│  Simple Agent   │
│  (Port 8080)    │
├─────────────────┤
│  Message        │
│  Handler        │
│     ↓           │
│  OpenAI LLM     │
│  Integration    │
└─────────────────┘
```

### Key Components

1. **Builder Pattern**: Fluent API for agent construction
   ```go
   chatbot := builder.NewAgent("simple-chatbot").
       WithLLM(provider).
       WithProtocol(protocol.ProtocolA2A).
       Build()
   ```

2. **Message Handler**: Processes incoming messages
   ```go
   OnMessage(func(ctx context.Context, msg agent.MessageContext) error {
       // Handle message and reply
       return msg.Reply(response)
   })
   ```

3. **Lifecycle Hooks**: BeforeStart and AfterStop callbacks
   ```go
   BeforeStart(func(ctx context.Context) error {
       log.Println("Agent starting...")
       return nil
   })
   ```

4. **Graceful Shutdown**: SIGINT/SIGTERM handling
   ```go
   signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
   // ... wait and stop gracefully
   ```

## Configuration

You can customize the agent by modifying:

- **LLM Model**: Change the OpenAI model in the provider
- **Port**: Modify the `:8080` address in `Start()`
- **System Prompt**: Update the system message in the LLM request
- **Timeout**: Adjust the A2A timeout in config

## Error Handling

The agent handles various error scenarios:

- Missing OpenAI API key → Fatal error at startup
- Empty messages → Returns error to client
- LLM failures → Logs error and returns to client
- Graceful shutdown → Stops accepting new requests

## Next Steps

- Add conversation history tracking
- Implement streaming responses
- Add custom tools/functions
- Enable multi-agent communication
- Add SAGE protocol security

## License

LGPL-3.0-or-later
