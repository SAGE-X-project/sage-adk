# SAGE ADK - Agent Development Kit

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-LGPL--3.0-blue.svg)](LICENSE)
[![A2A Protocol](https://img.shields.io/badge/A2A-v0.2.2-green.svg)](https://github.com/google/a2a-protocol)
[![SAGE Protocol](https://img.shields.io/badge/SAGE-v1.0.0-orange.svg)](https://github.com/sage-x-project/sage)

**SAGE Agent Development Kit (ADK)** is a comprehensive Go framework for building secure, interoperable AI agents. It seamlessly integrates the **A2A (Agent-to-Agent) Protocol** for standardized agent communication with the **SAGE (Secure Agent Guarantee Engine)** for blockchain-based identity and cryptographic message verification.

## Project Vision

ADK simplifies the development of production-ready AI agents by providing:

1. **Protocol Flexibility**: Start with simple A2A protocol, optionally enhance with SAGE security
2. **LLM Integration**: Unified interface for OpenAI, Anthropic, and Google Gemini
3. **Production Features**: Built-in authentication, monitoring, health checks, and resilience patterns
4. **Developer Experience**: Fluent API, sensible defaults, comprehensive documentation

## Key Features

### Dual Protocol Support

- **A2A Protocol**: Standard agent-to-agent communication (Google's open standard)
- **SAGE Protocol**: Enhanced security with blockchain identity and RFC 9421 message signatures
- **Auto-Detection**: Automatically switch between protocols based on message metadata

```go
// Simple A2A agent (no blockchain dependency)
agent := adk.NewAgent("my-agent").
    WithProtocol(adk.ProtocolA2A).
    WithLLM(llm.OpenAI()).
    Build()

// SAGE-secured agent (blockchain-verified identity)
agent := adk.NewAgent("secure-agent").
    WithProtocol(adk.ProtocolSAGE).
    WithSAGE(sage.FromEnv()).
    WithLLM(llm.OpenAI()).
    Build()

// Hybrid agent (auto-detect from messages)
agent := adk.NewAgent("hybrid-agent").
    WithProtocol(adk.ProtocolAuto).
    WithSAGE(sage.Optional()).
    WithLLM(llm.OpenAI()).
    Build()
```

### LLM Provider Abstraction

Unified interface for multiple LLM providers with environment-based configuration:

```go
// OpenAI
agent.WithLLM(llm.OpenAI())

// Anthropic Claude
agent.WithLLM(llm.Anthropic())

// Google Gemini
agent.WithLLM(llm.Gemini())

// Auto-detect from environment
agent.WithLLM(llm.FromEnv())
```

Supported models:
- **OpenAI**: GPT-3.5, GPT-4, GPT-4 Turbo, GPT-4o
- **Anthropic**: Claude 3 Sonnet, Opus, Haiku
- **Google**: Gemini Pro, Ultra, Flash

### SAGE Security Features

When SAGE mode is enabled, you get:

- **DID-based Identity**: Blockchain-registered agent identity (Ethereum, Kaia)
- **Message Signatures**: RFC 9421 HTTP message signatures with Ed25519
- **Handshake Protocol**: 4-phase secure session establishment with HPKE
- **End-to-End Encryption**: ChaCha20-Poly1305 AEAD encryption
- **Replay Protection**: Nonce-based attack prevention
- **Public Key Resolution**: Resolve agent public keys from blockchain

### Flexible Storage Backends

Choose the right storage for your use case:

- **Memory**: In-process storage for development and testing
- **Redis**: Production-ready with persistence and TTL
- **PostgreSQL**: Enterprise-grade with ACID guarantees

```go
// Development: in-memory
agent.WithStorage(storage.Memory())

// Production: Redis
agent.WithStorage(storage.Redis(redisClient))

// Enterprise: PostgreSQL
agent.WithStorage(storage.Postgres(dbConn))
```

### Production-Ready Features

- **Health Checks**: `/health` endpoint with component status
- **Metrics**: Prometheus metrics for monitoring
- **Structured Logging**: JSON or text format with configurable levels
- **CORS Support**: Configurable cross-origin resource sharing
- **Rate Limiting**: Protect against abuse
- **Circuit Breakers**: Prevent cascading failures
- **Graceful Shutdown**: Clean resource cleanup

## Quick Start

### Installation

```bash
go get github.com/sage-x-project/sage-adk
```

### Simple Agent (5 minutes)

Create `main.go`:

```go
package main

import (
    "context"
    "log"

    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
)

func main() {
    // Create agent with OpenAI
    agent := adk.NewAgent("my-first-agent").
        WithLLM(llm.OpenAI()).
        OnMessage(handleMessage).
        Build()

    // Start server
    log.Println("Starting agent on :8080")
    if err := agent.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}

func handleMessage(ctx context.Context, msg *adk.Message) error {
    // Get user's message text
    userText := msg.Text()

    // Generate response using LLM
    response, err := msg.LLM().Generate(ctx, userText)
    if err != nil {
        return err
    }

    // Reply to user
    return msg.Reply(response)
}
```

Create `.env`:

```bash
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
LLM_MODEL=gpt-4
```

Run:

```bash
go run main.go
```

Test:

```bash
curl -X POST http://localhost:8080/message/send \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "role": "user",
      "parts": [{"kind": "text", "text": "Hello!"}]
    }
  }'
```

### SAGE-Secured Agent

For production deployments with security requirements:

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    agent := adk.NewAgent("secure-agent").
        WithProtocol(adk.ProtocolSAGE).
        WithSAGE(sage.Options{
            DID:     "did:sage:ethereum:0x...",
            Network: sage.NetworkEthereum,
            RPC:     "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
            ContractAddress: "0x...",
            PrivateKey: loadPrivateKey(),
        }).
        WithLLM(llm.FromEnv()).
        OnMessage(handleSecureMessage).
        Build()

    agent.Start(":8080")
}

func handleSecureMessage(ctx context.Context, msg *adk.Message) error {
    // Message is automatically verified via SAGE
    // - DID resolved from blockchain
    // - Signature verified (RFC 9421)
    // - Nonce checked (replay protection)

    // Process message securely
    return processMessage(ctx, msg)
}
```

## Architecture

SAGE ADK is built on a layered architecture:

```
┌─────────────────────────────────────────────────────────┐
│                  Application Layer                       │
│              (Your Agent Business Logic)                 │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                   ADK Core Layer                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  Agent   │  │ Message  │  │ Protocol │              │
│  │ Builder  │  │ Router   │  │ Selector │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│                  Adapter Layer                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │   A2A    │  │   SAGE   │  │   LLM    │              │
│  │ Adapter  │  │ Adapter  │  │ Adapter  │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└──────────────────────┬──────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────┐
│              External Dependencies                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │sage-a2a- │  │   sage   │  │ LLM APIs │              │
│  │   go     │  │ library  │  │          │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
```

### Protocol Switching

Messages include a security metadata field that determines protocol:

```json
{
  "message": {
    "role": "user",
    "parts": [{"kind": "text", "text": "Hello"}]
  },
  "security": {
    "mode": "sage",
    "signature": "keyid=..., signature=...",
    "did": "did:sage:ethereum:0x..."
  }
}
```

ADK automatically:
1. Detects `security.mode` field
2. Routes to A2A or SAGE protocol handler
3. Applies security validation if SAGE mode
4. Returns response in same protocol

## Project Structure

```
sage-adk/
├── core/               # Core agent functionality
│   ├── agent/         # Agent abstraction
│   ├── protocol/      # Protocol layer (A2A/SAGE)
│   └── message/       # Message routing
├── adapters/          # External integrations
│   ├── a2a/          # sage-a2a-go wrapper
│   ├── sage/         # SAGE security wrapper
│   └── llm/          # LLM providers
├── builder/           # Fluent API builder
├── server/            # HTTP/gRPC server
├── client/            # Client SDK
├── storage/           # Storage backends
├── config/            # Configuration
├── examples/          # Example projects
└── docs/              # Documentation
```

## Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

- [**Overview**](docs/overview.md) - Introduction and key concepts
- [**Architecture**](docs/architecture/overview.md) - System design and components
- [**Configuration**](docs/guides/configuration.md) - Environment variables and settings
- [**Protocol Layer**](docs/architecture/protocol-layer.md) - A2A and SAGE integration details
- [**Project Structure**](docs/project-structure.md) - Directory layout and package descriptions

## Configuration

### Environment Variables

```bash
# Protocol
ADK_PROTOCOL_MODE=auto              # a2a | sage | auto

# A2A
A2A_STORAGE_TYPE=redis              # memory | redis | postgres
A2A_REDIS_URL=redis://localhost:6379

# SAGE (optional)
SAGE_ENABLED=true
SAGE_DID=did:sage:ethereum:0x...
SAGE_NETWORK=ethereum               # ethereum | kaia | sepolia
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
ETHEREUM_CONTRACT_ADDRESS=0x...
SAGE_PRIVATE_KEY=0x...

# LLM
LLM_PROVIDER=openai                 # openai | anthropic | gemini
OPENAI_API_KEY=sk-...
LLM_MODEL=gpt-4

# Server
ADK_SERVER_PORT=8080
LOG_LEVEL=info
METRICS_ENABLED=true
```

See [Configuration Guide](docs/guides/configuration.md) for complete reference.

## Use Cases

### 1. Simple Chatbot (A2A Only)
- No blockchain dependency
- Fast development iteration
- Standard A2A protocol compliance

### 2. Financial Services Agent (SAGE Recommended)
- Identity verification required
- Message integrity critical
- Audit trail via blockchain

### 3. Multi-Agent Orchestrator (Hybrid)
- Internal agents use A2A (performance)
- External agents use SAGE (security)
- Auto-detect protocol per connection

### 4. Enterprise AI Platform (Full SAGE)
- Blockchain-based access control
- End-to-end encrypted communication
- Compliance requirements (SOC2, GDPR)

## Examples

Explore working examples in the [`examples/`](examples/) directory:

- **[simple-agent](examples/simple-agent/)** - Basic A2A agent with OpenAI
- **[sage-enabled-agent](examples/sage-enabled-agent/)** - SAGE-secured agent with blockchain
- **[multi-llm-agent](examples/multi-llm-agent/)** - Agent using multiple LLM providers
- **[orchestrator](examples/orchestrator/)** - Multi-agent orchestration system

## Development

### Prerequisites

- Go 1.24 or higher
- Redis (for production storage)
- Ethereum/Kaia RPC endpoint (for SAGE mode)

### Build

```bash
# Clone repository
git clone https://github.com/sage-x-project/sage-adk.git
cd sage-adk

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run with coverage
make test-coverage
```

### Run Examples

```bash
# Simple agent
cd examples/simple-agent
cp .env.example .env
# Edit .env with your API keys
go run main.go

# SAGE-enabled agent
cd examples/sage-enabled-agent
cp .env.example .env
# Edit .env with blockchain configuration
go run main.go
```

## Dependencies

### Core Dependencies

- **[sage-a2a-go](https://github.com/sage-x-project/sage-a2a-go)** - A2A protocol implementation
- **[sage](https://github.com/sage-x-project/sage)** - SAGE security library
- **[go-openai](https://github.com/sashabaranov/go-openai)** - OpenAI API client
- **[go-redis](https://github.com/redis/go-redis)** - Redis client

### Protocol Standards

- **[A2A Protocol v0.2.2](https://github.com/google/a2a-protocol)** - Agent-to-agent communication
- **[RFC 9421](https://datatracker.ietf.org/doc/rfc9421/)** - HTTP message signatures
- **[W3C DID](https://www.w3.org/TR/did-core/)** - Decentralized identifiers

## Performance

### Benchmarks

| Mode | Latency (p50) | Latency (p99) | Throughput |
|------|---------------|---------------|------------|
| A2A Only | ~20ms | ~50ms | 1000+ msg/sec |
| SAGE Mode | ~150ms | ~300ms | 200+ msg/sec |
| Auto (80% A2A) | ~40ms | ~200ms | 800+ msg/sec |

*Note: SAGE latency includes blockchain DID resolution (cached after first lookup)*

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## License

This project is licensed under the **GNU Lesser General Public License v3.0 (LGPL-3.0)**.

**What this means:**

**You CAN:**
- Use SAGE ADK in commercial applications
- Use SAGE ADK in proprietary software
- Modify SAGE ADK for your needs
- Distribute SAGE ADK

**You MUST:**
- Provide SAGE ADK source code if you distribute it
- Allow users to replace/relink the SAGE ADK library
- Maintain LGPL-3.0 license notices

**You DON'T Need To:**
- Open-source your application that uses SAGE ADK
- Release your application under LGPL-3.0

See [LICENSE](LICENSE) for full details.

## Acknowledgments

- [Google's A2A Protocol](https://github.com/google/a2a-protocol) team for the agent communication standard
- [SAGE Project](https://github.com/sage-x-project/sage) team for the security framework
- [OpenAI](https://openai.com), [Anthropic](https://anthropic.com), and [Google](https://deepmind.google/technologies/gemini/) for LLM APIs
- Open source community for continuous feedback

## Support

- **Issues**: [GitHub Issues](https://github.com/sage-x-project/sage-adk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/sage-x-project/sage-adk/discussions)
- **Documentation**: [docs/](docs/)

---

**Built by the SAGE Team**

[Get Started](docs/overview.md) | [Documentation](docs/) | [Examples](examples/)
