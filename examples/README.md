# SAGE ADK Examples

This directory contains example projects demonstrating how to use the SAGE Agent Development Kit (ADK).

## Examples Overview

### 1. Simple Agent (`simple-agent/`)

**Difficulty**: Beginner
**Protocol**: A2A
**Features**: OpenAI integration, basic message handling

A minimal chatbot agent using the A2A protocol with OpenAI. Perfect for getting started with SAGE ADK.

```bash
cd simple-agent
export OPENAI_API_KEY="your-key"
go run -tags examples main.go
```

**What you'll learn:**
- Builder pattern basics
- A2A protocol setup
- Message handling
- LLM integration
- Graceful shutdown

### 2. SAGE Agent (`sage-agent/`)

**Difficulty**: Intermediate
**Protocol**: SAGE
**Features**: Blockchain identity, secure messaging, DID resolution

A secure chatbot agent using the SAGE protocol with blockchain-based identity verification.

```bash
cd sage-agent
export OPENAI_API_KEY="your-key"
export SAGE_NETWORK="sepolia"
export SAGE_DID="did:sage:sepolia:0x123..."
export SAGE_RPC_ENDPOINT="https://eth-sepolia.example.com"
export SAGE_PRIVATE_KEY_PATH="./keys/agent.pem"
go run -tags examples main.go
```

**What you'll learn:**
- SAGE protocol setup
- `FromSAGEConfig()` builder
- DID-based identity
- Blockchain integration
- Secure key management
- Network configuration

### 3. SAGE-Enabled Agent (`sage-enabled-agent/`)

**Difficulty**: Advanced
**Protocol**: SAGE (Low-Level)
**Features**: Direct SAGE adapter usage, message signing, network layer

Demonstrates low-level SAGE adapter API for secure agent communication with Ed25519 signatures.

```bash
cd sage-enabled-agent
# Interactive mode (single process)
go run -tags examples main.go interactive

# Distributed mode (two terminals)
go run -tags examples main.go receiver  # Terminal 1
go run -tags examples main.go sender    # Terminal 2
```

**What you'll learn:**
- Low-level SAGE adapter API
- Direct message signing with Ed25519
- HTTP network layer implementation
- Security metadata (nonce, timestamp, signature)
- Replay attack protection
- Signature verification pipeline
- Distributed agent communication

### 4. Key Generation (`key-generation/`)

**Difficulty**: Beginner
**Protocol**: N/A (Utility)
**Features**: Ed25519 key generation, PEM/JWK formats

Generate cryptographic key pairs for SAGE agents.

```bash
cd key-generation
go run -tags examples main.go -output ./keys/agent.pem -show-public
```

**What you'll learn:**
- Ed25519 key generation
- Key format options (PEM, JWK)
- Security best practices
- DID registration workflow

## Quick Start

### Prerequisites

- Go 1.21 or later
- OpenAI API key (for agent examples)
- Blockchain RPC endpoint (for SAGE examples)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/sage-x-project/agent-develope-kit.git
   cd agent-develope-kit/sage-adk
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up environment variables:
   ```bash
   export OPENAI_API_KEY="your-openai-api-key"
   ```

### Running Examples

All examples use the `examples` build tag to avoid conflicts with tests:

```bash
# Run an example
cd examples/simple-agent
go run -tags examples main.go

# Or from root
go run -tags examples ./examples/simple-agent/main.go
```

## Examples Comparison

| Example | Protocol | Complexity | Use Case |
|---------|----------|------------|----------|
| `simple-agent` | A2A | ⭐ Basic | Quick start, learning basics |
| `sage-agent` | SAGE | ⭐⭐ Intermediate | Production security, blockchain identity |
| `sage-enabled-agent` | SAGE (Low-Level) | ⭐⭐⭐ Advanced | Understanding SAGE internals, custom transport |
| `key-generation` | N/A | ⭐ Basic | Key management utility |

## Protocol Comparison

### A2A Protocol (Agent-to-Agent)

**Best for:**
- Quick prototyping
- Simple agent communication
- Internal systems
- Development/testing

**Features:**
- HTTP-based communication
- JSON message format
- Lightweight and fast
- No blockchain required

**Trade-offs:**
- No cryptographic identity verification
- Message signing optional
- Trust-based communication

### SAGE Protocol (Secure Agent Guarantee Engine)

**Best for:**
- Production deployments
- High-security requirements
- Decentralized systems
- Multi-organization scenarios

**Features:**
- Blockchain-based identity (DID)
- Cryptographic message signing
- End-to-end encryption
- On-chain public key verification
- Audit trail via blockchain

**Trade-offs:**
- Requires blockchain infrastructure
- Higher setup complexity
- Transaction costs (gas fees)
- Blockchain dependency

## Configuration Methods

### 1. Environment Variables (Recommended)

```bash
export OPENAI_API_KEY="sk-..."
export SAGE_NETWORK="sepolia"
export SAGE_DID="did:sage:sepolia:0x123..."
```

**Pros**: Easy to change per environment, secure
**Cons**: Requires shell setup

### 2. YAML Configuration

```yaml
# config.yaml
sage:
  enabled: true
  network: sepolia
  did: did:sage:sepolia:0x123...
  rpc_endpoint: https://eth-sepolia.example.com
  private_key_path: ./keys/agent.pem
```

Load with:
```go
cfg, _ := config.LoadFromFile("config.yaml")
agent := builder.FromSAGEConfig(&cfg.SAGE).Build()
```

**Pros**: Centralized config, version control friendly
**Cons**: Sensitive data must be handled carefully

### 3. Programmatic Configuration

```go
sageConfig := &config.SAGEConfig{
    Enabled:         true,
    Network:         "sepolia",
    DID:             "did:sage:sepolia:0x123...",
    RPCEndpoint:     "https://eth-sepolia.example.com",
    ContractAddress: "0xABC...",
    PrivateKeyPath:  "./keys/agent.pem",
}

agent := builder.FromSAGEConfig(sageConfig).Build()
```

**Pros**: Type-safe, IDE support, compile-time validation
**Cons**: Requires code changes for config updates

## Common Patterns

### 1. Builder Pattern

```go
agent := builder.NewAgent("my-agent").
    WithLLM(provider).
    WithProtocol(protocol.ProtocolA2A).
    OnMessage(handleMessage).
    Build()
```

### 2. SAGE Config Shortcut

```go
// Instead of:
agent := builder.NewAgent("agent").
    WithProtocol(protocol.ProtocolSAGE).
    WithSAGEConfig(cfg).
    Build()

// Use:
agent := builder.FromSAGEConfig(cfg).Build()
```

### 3. Message Handler

```go
func handleMessage(provider llm.Provider) agent.MessageHandler {
    return func(ctx context.Context, msg agent.MessageContext) error {
        text := msg.Text()
        response, _ := provider.Complete(ctx, &llm.CompletionRequest{
            Messages: []llm.Message{
                {Role: llm.RoleUser, Content: text},
            },
        })
        return msg.Reply(response.Content)
    }
}
```

### 4. Graceful Shutdown

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    agent.Start(":8080")
}()

<-sigChan
agent.Stop(context.Background())
```

## Troubleshooting

### "undefined: llm.NewMock"
Use `llm.NewMockProvider()` instead of `llm.NewMock()`.

### "Failed to load private key"
- Check file path is correct
- Verify file permissions (should be readable)
- Ensure key format is Ed25519 PEM or JWK

### "DID resolution failed"
- Verify DID is registered on blockchain
- Check RPC endpoint connectivity
- Confirm contract address is correct

### "LLM API key must not be empty"
Set the API key via environment variable:
```bash
export OPENAI_API_KEY="your-key"
```

### Build tag errors
Always use `-tags examples` when running examples:
```bash
go run -tags examples main.go
```

## Next Steps

After exploring the examples:

1. **Read the Documentation**:
   - Architecture overview: `../docs/architecture/`
   - API reference: `../docs/api/`
   - Development guide: `../docs/DEVELOPMENT_ROADMAP_v1.0_20251006-235205.md`

2. **Build Your Own Agent**:
   - Start with simple-agent template
   - Add custom business logic
   - Integrate with your LLM provider
   - Deploy to your infrastructure

3. **Add SAGE Security**:
   - Generate keys with key-generation tool
   - Register DID on blockchain
   - Switch to SAGE protocol
   - Configure network settings

4. **Production Deployment**:
   - Set up monitoring
   - Configure logging
   - Implement rate limiting
   - Add error handling
   - Set up CI/CD pipeline

## Contributing

To add a new example:

1. Create directory: `examples/my-example/`
2. Add `main.go` with `//go:build examples` tag
3. Write comprehensive `README.md`
4. Test thoroughly
5. Update this file with example description
6. Submit PR

## Resources

- **Documentation**: `../docs/`
- **API Reference**: `../pkg/`
- **SAGE Protocol**: See `../../sage/` repository
- **A2A Specification**: See `../../A2A/` repository
- **Issue Tracker**: GitHub Issues
- **Community**: Discord / Slack

## License

All examples are licensed under LGPL-3.0-or-later.
