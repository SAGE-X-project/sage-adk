# SAGE ADK Documentation

Welcome to the **SAGE Agent Development Kit (ADK)** documentation. This comprehensive guide will help you build secure, interoperable AI agents using the A2A protocol with optional SAGE security enhancements.

##  Documentation Structure

### Getting Started
- [Overview](overview.md) - Introduction to SAGE ADK
- [Quick Start](guides/quick-start.md) - Build your first agent in 5 minutes
- [Installation](guides/installation.md) - Setup and dependencies
- [Configuration](guides/configuration.md) - Environment variables and settings

### Architecture
- [Architecture Overview](architecture/overview.md) - System design and components
- [Protocol Layer](architecture/protocol-layer.md) - A2A and SAGE protocol integration
- [Message Flow](architecture/message-flow.md) - How messages are processed
- [Security Model](architecture/security-model.md) - Security features and best practices

### Guides
- [Building Agents](guides/building-agents.md) - Step-by-step agent development
- [A2A Protocol](guides/a2a-protocol.md) - Using pure A2A protocol
- [SAGE Integration](guides/sage-integration.md) - Enabling SAGE security features
- [LLM Providers](guides/llm-providers.md) - Integrating language models
- [Storage Backends](guides/storage-backends.md) - Memory, Redis, PostgreSQL
- [Multi-Agent Systems](guides/multi-agent-systems.md) - Building agent networks

### API Reference
- [Agent API](api/agent.md) - Agent builder and methods
- [Message API](api/message.md) - Message types and handlers
- [Protocol API](api/protocol.md) - Protocol selection and options
- [LLM API](api/llm.md) - LLM provider interfaces
- [Storage API](api/storage.md) - Storage abstractions

### Examples
- [Simple Agent](examples/simple-agent.md) - Basic A2A agent
- [SAGE-Enabled Agent](examples/sage-agent.md) - Secure agent with SAGE
- [Multi-LLM Agent](examples/multi-llm-agent.md) - Using multiple LLM providers
- [Orchestrator Agent](examples/orchestrator-agent.md) - Agent coordination
- [Production Deployment](examples/production-deployment.md) - Deploy to production

##  Key Features

### 1. **Dual Protocol Support**
- **A2A Protocol**: Standard agent-to-agent communication
- **SAGE Protocol**: Enhanced security with blockchain-based identity and message signatures
- **Auto Mode**: Automatic protocol detection based on message metadata

### 2. **LLM Integration**
- OpenAI (GPT-3.5, GPT-4, GPT-4o)
- Anthropic (Claude 3 Sonnet, Opus, Haiku)
- Google (Gemini Pro, Ultra)
- Environment-based configuration

### 3. **Flexible Storage**
- In-memory (development)
- Redis (production)
- PostgreSQL (enterprise)

### 4. **Production-Ready**
- Health checks and metrics
- Structured logging
- Circuit breakers and retry logic
- CORS support
- Authentication middleware

##  Quick Example

```go
package main

import (
    "context"
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
)

func main() {
    // Create a simple agent with OpenAI
    agent := adk.NewAgent("my-agent").
        WithLLM(llm.OpenAI()).
        OnMessage(func(ctx context.Context, msg *adk.Message) error {
            // Process message with LLM
            response, err := msg.LLM().Generate(ctx, msg.Text())
            if err != nil {
                return err
            }
            return msg.Reply(response)
        }).
        Build()

    // Start the agent server
    agent.Start(":8080")
}
```

##  Core Concepts

### Agent
An autonomous entity that can:
- Receive and send A2A messages
- Process requests using LLM
- Maintain conversation context
- Interact with other agents

### Protocol Mode
Choose how your agent communicates:
- **A2A Only**: Standard protocol, no blockchain dependency
- **SAGE Only**: Full security with DID and message signatures
- **Auto**: Detect protocol from incoming messages

### Security Options
When using SAGE mode:
- **DID-based Identity**: Blockchain-registered agent identity
- **Message Signatures**: RFC 9421 HTTP message signatures
- **End-to-End Encryption**: HPKE-based secure channels
- **Replay Protection**: Nonce-based attack prevention

##  Development Workflow

1. **Initialize Project**
   ```bash
   adk init my-agent
   cd my-agent
   ```

2. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

3. **Implement Agent Logic**
   ```go
   // main.go
   agent := adk.NewAgent("my-agent").
       WithLLM(llm.FromEnv()).
       OnMessage(handleMessage).
       Build()
   ```

4. **Run Locally**
   ```bash
   go run main.go
   ```

5. **Deploy**
   ```bash
   make build
   docker build -t my-agent .
   ```

## External Resources

- [A2A Protocol Specification](https://github.com/google/a2a-protocol)
- [SAGE Security Framework](https://github.com/sage-x-project/sage)
- [sage-a2a-go Implementation](https://github.com/sage-x-project/sage-a2a-go)

##  Support

- **Issues**: [GitHub Issues](https://github.com/sage-x-project/sage-adk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sage-x-project/sage-adk/discussions)
- **Discord**: [SAGE Community](https://discord.gg/sage-x)

## License

SAGE ADK is licensed under the LGPL-3.0 License. See [LICENSE](../LICENSE) for details.

---

**Next Steps**: Start with the [Overview](overview.md) to understand SAGE ADK architecture, then follow the [Quick Start Guide](guides/quick-start.md) to build your first agent.
