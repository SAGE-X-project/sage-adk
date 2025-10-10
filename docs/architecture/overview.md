# Architecture Overview

## System Architecture

SAGE ADK is designed as a layered architecture that provides flexibility, security, and ease of use for building AI agents.

## High-Level Architecture

```

                     Application Layer                            
              (User-Defined Agent Business Logic)                 
                                                                   
                
     Message           LLM           Custom               
     Handlers       Processing       Skills               
                

                             

                        ADK Core Layer                            
                                                                   
                
      Agent          Message         Protocol             
     Builder         Router          Selector             
                
                                                                
                
    Lifecycle       Middleware       Security             
     Manager          Chain          Manager              
                

                             

                       Adapter Layer                              
                                                                   
                
       A2A             SAGE            LLM                
     Adapter         Adapter         Adapter              
                
                                                                
                
     Storage           DID           Provider             
     Adapter         Resolver        Manager              
                

                             

                  External Dependencies Layer                     
                                                                   
                
   sage-a2a-go         sage          LLM APIs             
     Library         Library                              
                
                                                                   
                                 
      Redis         Blockchain                               
     Storage           RPC                                   
                                 

```

## Core Components

### 1. Agent Builder

**Purpose**: Provides a fluent API for creating and configuring agents.

**Key Features**:
- Builder pattern for easy configuration
- Compile-time type safety
- Sensible defaults
- Validation before build

**Code Structure**:
```
builder/
 builder.go          # Main builder implementation
 options.go          # Configuration options
 validator.go        # Pre-build validation
 templates/          # Pre-configured templates
     simple.go       # Simple agent template
     conversational.go
     orchestrator.go
```

**Example Usage**:
```go
agent := adk.NewAgent("my-agent").
    WithProtocol(adk.ProtocolAuto).
    WithLLM(llm.OpenAI()).
    WithStorage(storage.Redis(client)).
    WithMiddleware(
        middleware.Logging(),
        middleware.Metrics(),
    ).
    OnMessage(handleMessage).
    Build()
```

### 2. Protocol Layer

**Purpose**: Abstracts protocol differences between A2A and SAGE.

**Components**:

#### Protocol Selector
```go
type ProtocolMode int

const (
    ProtocolA2A  ProtocolMode = iota  // A2A only
    ProtocolSAGE                      // SAGE only
    ProtocolAuto                      // Auto-detect
)

type ProtocolSelector interface {
    SelectProtocol(msg *Message) (Protocol, error)
}
```

#### Protocol Interface
```go
type Protocol interface {
    // Message handling
    SendMessage(ctx context.Context, msg *Message) (*Response, error)
    StreamMessage(ctx context.Context, msg *Message) (<-chan *Event, error)

    // Task management
    GetTask(ctx context.Context, taskID string) (*Task, error)
    CancelTask(ctx context.Context, taskID string) error

    // Protocol info
    Name() string
    Version() string
    RequiresSecurity() bool
}
```

### 3. Message Router

**Purpose**: Routes incoming messages to appropriate handlers based on content, skills, or routing rules.

**Routing Strategies**:
- **Content-based**: Route by message content (keywords, NLP)
- **Skill-based**: Route by agent capabilities
- **Round-robin**: Distribute load evenly
- **Custom**: User-defined routing logic

**Code Structure**:
```
core/message/
 router.go           # Main router
 strategy.go         # Routing strategies
 matcher.go          # Content matching
 middleware.go       # Message middleware
```

**Example**:
```go
router := message.NewRouter().
    AddRoute("/chat", chatHandler).
    AddRoute("/code", codeHandler).
    AddFallback(defaultHandler).
    WithMiddleware(
        middleware.Auth(),
        middleware.RateLimiting(),
    )
```

### 4. Security Manager

**Purpose**: Manages protocol switching and security enforcement.

**Responsibilities**:
- Detect security mode from message metadata
- Verify RFC 9421 signatures (SAGE mode)
- Resolve DIDs from blockchain
- Manage session keys
- Enforce security policies

**Code Structure**:
```
security/
 protocol_switch.go  # A2A ↔ SAGE switching
 signature.go        # RFC 9421 signing/verification
 did_resolver.go     # Blockchain DID resolution
 session.go          # Session key management
 policy.go           # Security policies
```

### 5. Storage Layer

**Purpose**: Provides unified interface for persisting tasks, messages, and sessions.

**Interface**:
```go
type Storage interface {
    // Task operations
    SaveTask(ctx context.Context, task *Task) error
    GetTask(ctx context.Context, taskID string) (*Task, error)

    // Message history
    SaveMessage(ctx context.Context, msg *Message) error
    GetHistory(ctx context.Context, contextID string, limit int) ([]*Message, error)

    // Session management
    SaveSession(ctx context.Context, session *Session) error
    GetSession(ctx context.Context, sessionID string) (*Session, error)
}
```

**Implementations**:
- **Memory**: In-process map with TTL
- **Redis**: Production-ready with persistence
- **PostgreSQL**: Enterprise with ACID guarantees

### 6. LLM Adapter Layer

**Purpose**: Abstracts differences between LLM providers.

**Unified Interface**:
```go
type LLMProvider interface {
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
    Stream(ctx context.Context, req *StreamRequest) (<-chan *Chunk, error)
    Name() string
    Model() string
}

type GenerateRequest struct {
    Prompt      string
    MaxTokens   int
    Temperature float64
    StopWords   []string
    Metadata    map[string]interface{}
}
```

**Providers**:
- `llm.OpenAI()` - GPT models
- `llm.Anthropic()` - Claude models
- `llm.Gemini()` - Google Gemini
- `llm.Custom()` - Custom implementations

## Data Flow

### Message Processing Pipeline

```
Incoming HTTP Request
        
        

 HTTP Handler  

        
        

 Protocol       Detect A2A vs SAGE from headers
 Detection     

        
        

 Security       Verify signature if SAGE mode
 Validation    

        
        

 Message        Parse A2A message format
 Parsing       

        
        

 Middleware     Auth, logging, metrics, rate limiting
 Chain         

        
        

 Message        Route to appropriate handler
 Router        

        
        

 User Handler   Your business logic + LLM processing

        
        

 Response       Build response message
 Builder       

        
        

 Signature      Sign response if SAGE mode
 Generation    

        
        

 Storage        Save task, message history

        
        
   HTTP Response
```

## Protocol Switching Mechanism

### Message Structure

```json
{
  "message": {
    "message_id": "msg-123",
    "context_id": "ctx-456",
    "role": "user",
    "parts": [
      {
        "kind": "text",
        "text": "Hello, can you help me?"
      }
    ]
  },
  "security": {
    "mode": "sage",
    "signature": "keyid=\"key-abc\", algorithm=\"ed25519\", ...",
    "key_id": "key-abc123",
    "did": "did:sage:ethereum:0x1234567890abcdef"
  }
}
```

### Detection Logic

```go
func (s *ProtocolSelector) SelectProtocol(msg *Message) (Protocol, error) {
    if s.mode == ProtocolA2A {
        return s.a2aProtocol, nil
    }

    if s.mode == ProtocolSAGE {
        return s.sageProtocol, nil
    }

    // Auto mode: detect from message
    if msg.Security != nil && msg.Security.Mode == "sage" {
        return s.sageProtocol, nil
    }

    return s.a2aProtocol, nil
}
```

### SAGE Signature Verification Flow

```

 Extract DID     
 from message    

         
         

 Resolve Public   Query blockchain registry
 Key from DID    

         
         

 Verify RFC 9421  Use sage library
 Signature       

         
         

 Check Nonce      Replay protection
 Cache           

         
         

 Accept/Reject   

```

## Agent Lifecycle

```

  Create    agent := adk.NewAgent("name")

     
     

Configure   .WithLLM(...).WithStorage(...)

     
     

 Validate   Check required fields, dependencies

     
     

  Build     .Build() → Returns Agent instance

     
     

  Start     agent.Start(":8080")

     
     

 Running    Handle messages, maintain sessions

     
     

Shutdown    Graceful shutdown, cleanup

```

## Concurrency Model

### Request Handling

- **HTTP Server**: Goroutine per request
- **Message Processing**: Concurrent with worker pool
- **LLM Calls**: Async with context cancellation
- **Storage**: Connection pooling (Redis, PostgreSQL)

### Synchronization

```go
type Agent struct {
    mu          sync.RWMutex
    sessions    map[string]*Session
    tasks       map[string]*Task

    // Worker pool for message processing
    workers     int
    jobQueue    chan *Job
    workerPool  chan chan *Job
}
```

## Error Handling

### Error Types

```go
type ErrorType int

const (
    ErrorTypeProtocol ErrorType = iota
    ErrorTypeSecurity
    ErrorTypeStorage
    ErrorTypeLLM
    ErrorTypeValidation
)

type Error struct {
    Type    ErrorType
    Code    string
    Message string
    Cause   error
}
```

### Recovery Strategy

- **Retry**: Exponential backoff for transient errors (network, LLM API)
- **Fallback**: Use default/cached response if LLM unavailable
- **Circuit Breaker**: Stop calling failing services temporarily
- **Graceful Degradation**: Continue with reduced functionality

## Configuration Management

### Configuration Sources (Priority Order)

1. **Explicit Code**: `WithLLM(llm.OpenAI("gpt-4"))`
2. **Environment Variables**: `LLM_MODEL=gpt-4`
3. **Config File**: `config.yaml`
4. **Defaults**: Built-in sensible defaults

### Configuration Validation

```go
type ConfigValidator interface {
    Validate(config *Config) error
}

// Pre-build validation
func (b *Builder) Build() (*Agent, error) {
    if err := b.validator.Validate(b.config); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    // ...
}
```

## Observability

### Metrics

```go
// Prometheus metrics
var (
    MessagesReceived = promauto.NewCounterVec(...)
    MessageDuration  = promauto.NewHistogramVec(...)
    ActiveSessions   = promauto.NewGauge(...)
    LLMCalls         = promauto.NewCounterVec(...)
)
```

### Logging

```go
// Structured logging
logger.Info("message received",
    "agent", agentName,
    "message_id", msg.ID,
    "protocol", protocol.Name(),
    "security_mode", msg.Security.Mode,
)
```

### Health Checks

```
GET /health

{
  "status": "healthy",
  "components": {
    "storage": "healthy",
    "llm": "healthy",
    "blockchain": "degraded"
  },
  "uptime": "2h15m30s"
}
```

## Security Considerations

### SAGE Mode Security

1. **DID Verification**: Always verify DID on blockchain
2. **Signature Validation**: Reject messages with invalid signatures
3. **Replay Protection**: Maintain nonce cache with TTL
4. **Key Rotation**: Support public key updates on blockchain
5. **Session Expiry**: Enforce max age and idle timeout

### A2A Mode Security

1. **TLS Required**: Always use HTTPS in production
2. **API Key Auth**: Optional authentication layer
3. **Rate Limiting**: Prevent DoS attacks
4. **Input Validation**: Sanitize all user inputs

---

[← Back to Documentation](../README.md) | [Protocol Layer →](protocol-layer.md)
