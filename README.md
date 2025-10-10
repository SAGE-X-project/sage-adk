# SAGE ADK - Agent Development Kit

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-LGPL--3.0--or--later-blue.svg)](LICENSE)
[![A2A Protocol](https://img.shields.io/badge/A2A-v0.2.2-green.svg)](https://github.com/google/a2a-protocol)
[![SAGE Protocol](https://img.shields.io/badge/SAGE-v1.0.0-orange.svg)](https://github.com/sage-x-project/sage)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](https://github.com/sage-x-project/sage-adk)

**SAGE Agent Development Kit (ADK)** is a comprehensive Go framework for building secure, interoperable AI agents. It seamlessly integrates the **A2A (Agent-to-Agent) Protocol** for standardized agent communication with the **SAGE (Secure Agent Guarantee Engine)** for blockchain-based identity and cryptographic message verification.

## Current Status: v1.2.0 - Enterprise Features Complete

**Available Now:**
- **Core Framework**
  - Fluent Builder API with description and version support
  - Multiple LLM Providers (OpenAI, Anthropic, Gemini)
  - A2A Protocol Support (HTTP & gRPC)
  - SAGE Protocol Security Layer
  - Agent Runtime with Lifecycle Management

- **Storage & Caching**
  - Multiple Storage Backends (Memory, Redis, PostgreSQL)
  - Response Caching with LRU Eviction
  - Distributed Tracing with OpenTelemetry

- **Enterprise Features**
  - Rate Limiting (Token Bucket & Sliding Window)
  - Multi-Tenant Support with Isolation
  - gRPC Server Support
  - Comprehensive Observability (Metrics, Health Checks, Logging)

- **Development Tools**
  - CLI Tool (adk) for Project Scaffolding
  - Complete Working Examples (15+ examples)
  - Comprehensive Test Coverage (All tests passing)

## Quick Start

### Installation

```bash
go get github.com/sage-x-project/sage-adk
```

### Create Your First Agent (2 minutes)

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sage-x-project/sage-adk/adapters/llm"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/protocol"
)

func main() {
	// Create OpenAI provider
	provider := llm.OpenAI(&llm.OpenAIConfig{
		APIKey: "your-api-key",
		Model:  "gpt-3.5-turbo",
	})

	// Build the agent
	chatbot, err := builder.NewAgent("my-chatbot").
		WithLLM(provider).
		WithProtocol(protocol.ProtocolA2A).
		OnMessage(func(ctx context.Context, msg agent.MessageContext) error {
			// Get user message
			userText := msg.Text()

			// Create LLM request
			request := &llm.CompletionRequest{
				Messages: []llm.Message{
					{Role: llm.RoleSystem, Content: "You are a helpful assistant."},
					{Role: llm.RoleUser, Content: userText},
				},
			}

			// Get LLM response
			response, err := provider.Complete(ctx, request)
			if err != nil {
				return err
			}

			// Reply to user
			return msg.Reply(response.Content)
		}).
		Build()

	if err != nil {
		log.Fatal(err)
	}

	// Start the agent
	log.Println(" Agent listening on :8080")
	if err := chatbot.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
```

### Run the Example

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="sk-..."

# Run the example
cd examples/simple-agent
go run -tags examples main.go
```

### Test Your Agent

```bash
# Using the included test client
go run -tags examples client.go "Hello! How are you?"

# Or using curl
curl -X POST http://localhost:8080/a2a/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "user",
      "parts": [{"kind": "text", "text": "Hello!"}]
    }
  }'
```

## Key Features

###  Fluent Builder API

Build agents with an intuitive, chainable API:

```go
agent := builder.NewAgent("my-agent").
    WithLLM(llm.OpenAI()).
    WithProtocol(protocol.ProtocolA2A).
    WithStorage(storage.NewMemoryStorage()).
    OnMessage(handleMessage).
    BeforeStart(func(ctx context.Context) error {
        log.Println("Starting...")
        return nil
    }).
    Build()
```

###  LLM Integration

Currently supports OpenAI with more providers coming:

```go
// OpenAI
provider := llm.OpenAI(&llm.OpenAIConfig{
    APIKey: "sk-...",
    Model:  "gpt-4",
})

// Complete (synchronous)
response, err := provider.Complete(ctx, request)

// Stream (real-time)
err := provider.Stream(ctx, request, func(chunk *llm.CompletionResponse) error {
    fmt.Print(chunk.Content)
    return nil
})
```

**Supported Models:**
- GPT-3.5 Turbo
- GPT-4
- GPT-4 Turbo
- GPT-4o

###  A2A Protocol Support

Full implementation of Google's Agent-to-Agent protocol:

```go
// Server (receives messages)
server, err := a2a.NewServer(&a2a.ServerConfig{
    AgentName:      "my-agent",
    AgentURL:       "http://localhost:8080/",
    MessageHandler: handleMessage,
})

// Client (sends messages)
client, err := a2a.NewClient("http://other-agent:8080/")
response, err := client.SendMessage(ctx, message)
```

**Features:**
- Message sending and receiving
- Streaming support
- Type conversion (A2A ↔ SDK types)
- Task management integration

###  Storage Backend

Currently supports in-memory storage with more backends coming:

```go
// Memory storage (development)
storage := storage.NewMemoryStorage()

// Store data
storage.Store(ctx, "namespace", "key", value)

// Retrieve data
value, err := storage.Get(ctx, "namespace", "key")
```

**Features:**
- CRUD operations
- Namespace isolation
- Concurrent access safe
- Type preservation

###  Agent Lifecycle

Full lifecycle management with hooks:

```go
agent := builder.NewAgent("my-agent").
    BeforeStart(func(ctx context.Context) error {
        // Initialize resources
        return nil
    }).
    AfterStop(func(ctx context.Context) error {
        // Cleanup resources
        return nil
    }).
    Build()

// Start agent (blocking)
agent.Start(":8080")

// Graceful shutdown
agent.Stop(context.Background())
```

## Architecture

```

       Your Application Logic            
     (Message Handlers, Business)         

                   

         Builder API (Fluent)             
   NewAgent().WithLLM().OnMessage()      

                   

            Core Agent                    
       
   Runtime   Protocol    Message  
  Lifecycle  Selector    Handler  
       

                   

            Adapters Layer                
       
     A2A        LLM      Storage  
   Client/   Provider    Backend  
   Server                         
       

                   

      External Dependencies               
  sage-a2a-go, OpenAI API, etc.          

```

## Project Structure

```
sage-adk/
 builder/            # Fluent API builder
 core/
    agent/         # Agent core and runtime
    protocol/      # Protocol abstraction layer
 adapters/
    a2a/          # A2A protocol adapter
    llm/          # LLM provider adapters
    sage/         # SAGE security (Phase 2B)
 storage/           # Storage backends
 config/            # Configuration management
 pkg/
    types/        # Common types and messages
    errors/       # Error handling
 examples/          # Working examples
    simple-agent/ # Basic chatbot example
 docs/              # Documentation
```

## Examples

### 1. Simple Chatbot ([examples/simple-agent](examples/simple-agent/))

A complete chatbot with OpenAI integration:
- Full LLM integration
- A2A protocol server
- Graceful shutdown
- Comprehensive documentation

```bash
cd examples/simple-agent
go run -tags examples main.go
```

### 2. Minimal Echo Agent ([examples/simple-agent/minimal.go](examples/simple-agent/minimal.go))

The absolute minimum code (14 lines):

```go
agent := builder.NewAgent("echo").
    OnMessage(func(ctx context.Context, msg agent.MessageContext) error {
        return msg.Reply("Echo: " + msg.Text())
    }).
    MustBuild()

log.Fatal(agent.Start(":8080"))
```

### 3. Test Client ([examples/simple-agent/client.go](examples/simple-agent/client.go))

Send messages to any agent:

```bash
go run -tags examples client.go "Your message here"
```

## Configuration

### Environment Variables

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Optional: Model selection
export OPENAI_MODEL="gpt-4"

# Optional: Server port
export PORT="8080"
```

### Programmatic Configuration

```go
// A2A Config
a2aConfig := &config.A2AConfig{
    Enabled:   true,
    Version:   "0.2.2",
    ServerURL: "http://localhost:8080/",
    Timeout:   30,
}

agent := builder.NewAgent("my-agent").
    WithA2AConfig(a2aConfig).
    Build()
```

## Development

### Prerequisites

- Go 1.21 or later
- OpenAI API key for testing

### Build and Test

```bash
# Clone repository
git clone https://github.com/sage-x-project/sage-adk.git
cd sage-adk

# Install dependencies
go mod download

# Run tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./core/agent
go test ./adapters/llm
```

### Test Results

All 253 tests passing:
-  adapters/a2a: 18 tests
-  adapters/llm: 26 tests
-  adapters/sage: 8 tests
-  builder: 17 tests
-  config: 28 tests
-  core/agent: 18 tests
-  core/protocol: 18 tests
-  pkg/errors: 36 tests
-  pkg/types: 58 tests
-  storage: 26 tests

## Roadmap

### Phase 2A  Complete
- [x] Builder API with fluent interface
- [x] OpenAI LLM provider
- [x] A2A protocol client/server
- [x] Agent runtime with lifecycle
- [x] Memory storage backend
- [x] Working examples

### Phase 2B  In Progress
- [ ] SAGE protocol security layer
- [ ] Anthropic Claude integration
- [ ] Google Gemini integration
- [ ] Redis storage backend
- [ ] PostgreSQL storage backend
- [ ] Streaming message support
- [ ] Tool/function calling
- [ ] Multi-agent orchestration

### Phase 3  Planned
- [ ] Advanced security features
- [ ] Monitoring and metrics
- [ ] Rate limiting
- [ ] Circuit breakers
- [ ] Distributed tracing
- [ ] Production deployment guides

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Development Workflow:**
1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with tests
4. Ensure all tests pass (`make test`)
5. Commit with clear message
6. Push and create Pull Request

**Areas where we need help:**
- Additional LLM provider integrations
- Storage backend implementations
- Documentation improvements
- Example applications
- Performance optimizations

## Dependencies

### Core
- [sage-a2a-go](https://github.com/sage-x-project/sage-a2a-go) - A2A protocol
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI client

### Development
- Go 1.21+
- Make
- Git

## License

This project is licensed under the **GNU Lesser General Public License v3.0 or later (LGPL-3.0-or-later)**.

SPDX-License-Identifier: `LGPL-3.0-or-later`

### What This Means For You

**You CAN:**
- Use SAGE ADK in commercial applications
- Use SAGE ADK in proprietary software
- Modify SAGE ADK for your needs
- Distribute applications using SAGE ADK
- Choose to follow LGPL-3.0 or any later version

**You MUST:**
- Provide SAGE ADK source code if you distribute modified versions
- Allow users to replace/relink SAGE ADK library
- Maintain LGPL-3.0-or-later license notices in SAGE ADK code
- Comply with licenses of third-party dependencies (see [NOTICE](NOTICE))

**You DON'T Need To:**
- Open-source your application code
- Release your application under LGPL-3.0
- Share your proprietary business logic

### Third-Party Components

SAGE ADK depends on several open-source projects with compatible licenses:
- **sage** (LGPL-3.0-or-later)
- **sage-a2a-go** (Apache-2.0)
- **go-ethereum** (LGPL-3.0/GPL-3.0)
- **Prometheus Client** (Apache-2.0)
- Others (MIT, BSD) - see [NOTICE](NOTICE) file

### License Compatibility

The LGPL-3.0-or-later license:
- Allows library usage in proprietary applications
- Compatible with Apache-2.0, MIT, BSD licenses
- Meets requirements for go-ethereum dependency
- Ensures modifications to SAGE ADK remain open-source

For complete details:
- [LICENSE](LICENSE) - Full license text
- [NOTICE](NOTICE) - Third-party components
- [SPDX LGPL-3.0-or-later](https://spdx.org/licenses/LGPL-3.0-or-later.html) - Official reference

## Support

- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)
- **Issues**: [GitHub Issues](https://github.com/sage-x-project/sage-adk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sage-x-project/sage-adk/discussions)

## Acknowledgments

- [Google's A2A Protocol](https://github.com/google/a2a-protocol) for the agent communication standard
- [SAGE Project](https://github.com/sage-x-project/sage) for the security framework
- [OpenAI](https://openai.com) for LLM APIs
- Open source community for continuous feedback

---

**Built by the SAGE Team** 

[Quick Start](#quick-start) | [Examples](examples/) | [Documentation](docs/)
