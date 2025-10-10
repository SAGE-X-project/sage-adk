# SAGE ADK Overview

## What is SAGE ADK?

**SAGE Agent Development Kit (ADK)** is a comprehensive Go framework for building secure, interoperable AI agents. It combines the standardized communication protocol of A2A (Agent-to-Agent) with the advanced security features of SAGE (Secure Agent Guarantee Engine).

### Design Philosophy

SAGE ADK is built on three core principles:

1. **Simplicity First**: Start with basic A2A protocol, add security when needed
2. **Protocol Transparency**: Seamlessly switch between A2A and SAGE modes
3. **Production Ready**: Built-in monitoring, health checks, and resilience patterns

## Why SAGE ADK?

### The Agent Communication Challenge

Modern AI systems require multiple specialized agents working together. However, building multi-agent systems faces several challenges:

- **Interoperability**: Agents built with different frameworks can't communicate
- **Security**: Messages can be intercepted, tampered, or replayed
- **Identity**: No standard way to verify agent authenticity
- **Integration**: Complex LLM provider APIs and storage backends

### The SAGE ADK Solution

SAGE ADK solves these challenges by providing:

```

                    SAGE ADK Layer                        
  (Unified API, Builder Pattern, Auto-Configuration)     

                                   
           
        A2A Protocol      SAGE Protocol  
         (Standard)         (Secure)     
           
                                   
               
                        
         
           sage-a2a-go + sage (libs)   
         
```

## Core Components

### 1. Agent Builder

Fluent API for creating agents with minimal boilerplate:

```go
agent := adk.NewAgent("my-agent").
    WithProtocol(adk.ProtocolAuto).
    WithLLM(llm.OpenAI()).
    WithStorage(storage.Redis(client)).
    OnMessage(handleMessage).
    Build()
```

### 2. Protocol Layer

**A2A Protocol** (Base Layer):
- JSON-RPC 2.0 over HTTP/HTTPS
- Task lifecycle management
- Message history and context
- Streaming support (SSE)

**SAGE Protocol** (Security Layer):
- DID-based agent identity (Ethereum, Kaia blockchains)
- RFC 9421 HTTP message signatures (Ed25519)
- HPKE end-to-end encryption
- Replay attack prevention

**Protocol Selection**:
```go
// Option 1: A2A Only (no blockchain dependency)
WithProtocol(adk.ProtocolA2A)

// Option 2: SAGE Only (full security)
WithProtocol(adk.ProtocolSAGE)

// Option 3: Auto-detect from message (recommended)
WithProtocol(adk.ProtocolAuto)
```

### 3. LLM Adapters

Unified interface for multiple LLM providers:

```go
type LLMProvider interface {
    Generate(ctx context.Context, prompt string, opts ...Option) (*Response, error)
    Stream(ctx context.Context, prompt string) (<-chan *Chunk, error)
}
```

Supported providers:
- **OpenAI**: GPT-3.5, GPT-4, GPT-4 Turbo, GPT-4o
- **Anthropic**: Claude 3 Sonnet, Opus, Haiku
- **Google**: Gemini Pro, Ultra, Flash

Configuration via environment variables:
```env
LLM_PROVIDER=openai
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4
```

### 4. Storage Backends

Pluggable storage for task history and conversation context:

- **Memory**: In-process storage (development)
- **Redis**: Production-ready with TTL and persistence
- **PostgreSQL**: Enterprise-grade with full ACID guarantees

```go
// Memory (default)
WithStorage(storage.Memory())

// Redis
WithStorage(storage.Redis(redisClient))

// PostgreSQL
WithStorage(storage.Postgres(dbConn))
```

### 5. Security Features

When SAGE mode is enabled:

**DID Management**:
```go
WithSAGE(sage.Options{
    DID:     "did:sage:ethereum:0x...",
    Network: sage.NetworkEthereum,
    PrivateKey: privateKey,
})
```

**Message Signing & Verification**:
- Automatic RFC 9421 signature generation
- Signature verification on incoming messages
- Public key resolution from blockchain

**Handshake Protocol**:
- 4-phase secure session establishment
- Ephemeral key exchange (X25519)
- Perfect forward secrecy

## Architecture Layers

```

                  Application Layer                       
              (Your Agent Business Logic)                 

                       

                   ADK Core Layer                         
                    
    Agent      Message      Router                
   Builder    Processor                           
                    

                       

                  Adapter Layer                           
                    
     A2A         SAGE        LLM                  
   Adapter     Adapter     Adapter                
                    

                       

              External Dependencies Layer                 
                    
  sage-a2a-      sage     LLM APIs                
     go         library                           
                    

```

## Message Flow

### Standard A2A Flow

```
Client Agent                    Server Agent
                                    
        POST /message/send          
     >
        (A2A Message)                
                                    
                               
                                Process 
                                 with   
                                 LLM    
                               
                                    
        < Response Message 
                                    
```

### SAGE-Enhanced Flow

```
Client Agent                    Server Agent
                                    
        1. Handshake Request        
     >
        (DID + Ephemeral Key)       
                                    
        < Handshake Response 
        (Session Key Established)   
                                    
        2. Signed Message           
     >
        (RFC 9421 Signature)        
                               
                                Verify      
                                Signature   
                                (Blockchain)
                               
                               
                                Process 
                                 with   
                                 LLM    
                               
                                    
        < Signed Response 
        (RFC 9421 Signature)        
                                    
```

## Protocol Switching Logic

Messages include a security metadata field:

```json
{
  "message": {
    "role": "user",
    "parts": [{"kind": "text", "text": "Hello"}]
  },
  "security": {
    "mode": "sage",
    "signature": "keyid=..., signature=...",
    "key_id": "key-abc123",
    "did": "did:sage:ethereum:0x..."
  }
}
```

ADK automatically:
1. Detects `security.mode` field in incoming messages
2. Routes to appropriate protocol handler
3. Applies security validation if `mode == "sage"`
4. Returns response in same security mode

## Use Cases

### 1. **Simple Chatbot** (A2A Only)
- No blockchain dependency
- Fast development
- Standard A2A protocol

### 2. **Financial Services Agent** (SAGE Recommended)
- Identity verification required
- Message integrity critical
- Audit trail needed

### 3. **Multi-Agent Orchestrator** (Hybrid)
- Internal agents: A2A mode (fast)
- External agents: SAGE mode (secure)
- Auto-detect protocol per connection

### 4. **Enterprise AI Assistant** (Full SAGE)
- Blockchain-based access control
- Encrypted communication
- Compliance requirements (SOC2, GDPR)

## Performance Characteristics

### A2A Mode
- **Latency**: ~10-50ms (HTTP overhead only)
- **Throughput**: 1000+ msg/sec per agent
- **Storage**: Redis recommended for production

### SAGE Mode
- **Latency**: ~100-200ms (signature verification + blockchain lookup)
- **Throughput**: 100-500 msg/sec per agent
- **Storage**: Redis + blockchain RPC required

### Optimization Tips
- Cache DID → Public Key mappings (TTL: 1 hour)
- Use session keys to avoid repeated handshakes
- Batch blockchain queries when possible
- Use Auto mode to mix protocols (80% A2A, 20% SAGE)

## Development vs Production

### Development Setup
```go
agent := adk.NewAgent("dev-agent").
    WithProtocol(adk.ProtocolA2A).      // No blockchain
    WithLLM(llm.Mock()).                 // Mock LLM
    WithStorage(storage.Memory()).       // In-memory
    Build()
```

### Production Setup
```go
agent := adk.NewAgent("prod-agent").
    WithProtocol(adk.ProtocolAuto).      // Support both
    WithSAGE(sage.FromEnv()).            // Load from env
    WithLLM(llm.FromEnv()).              // Production LLM
    WithStorage(storage.Redis(client)).  // Persistent storage
    WithMetrics(true).                   // Enable monitoring
    WithHealthCheck(true).               // Health endpoint
    Build()
```

## Next Steps

- **Quick Start**: [Build your first agent](guides/quick-start.md)
- **Configuration**: [Environment setup guide](guides/configuration.md)
- **Architecture**: [Deep dive into components](architecture/overview.md)
- **Examples**: [Sample agent implementations](examples/simple-agent.md)

---

[← Back to Documentation Home](README.md) | [Quick Start Guide →](guides/quick-start.md)
