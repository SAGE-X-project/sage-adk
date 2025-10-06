# Protocol Layer Architecture

## Overview

The Protocol Layer in SAGE ADK provides a unified abstraction over two distinct agent communication protocols:

1. **A2A (Agent-to-Agent)**: Standard protocol for interoperability
2. **SAGE (Secure Agent Guarantee Engine)**: Enhanced security with blockchain identity

This layer enables seamless protocol switching, transparent security enforcement, and backward compatibility.

## Design Principles

### 1. Protocol Agnostic Core

The ADK core is designed to work with any protocol by defining a common `Protocol` interface:

```go
type Protocol interface {
    // Message Operations
    SendMessage(ctx context.Context, msg *Message) (*Response, error)
    StreamMessage(ctx context.Context, msg *Message) (<-chan *Event, error)

    // Task Operations
    GetTask(ctx context.Context, taskID string) (*Task, error)
    CancelTask(ctx context.Context, taskID string) error

    // Protocol Metadata
    Name() string
    Version() string
    RequiresSecurity() bool
}
```

### 2. Transparent Security Layer

Security features are added transparently without changing core message handling:

```go
// Same code works for both protocols
agent := adk.NewAgent("my-agent").
    WithProtocol(adk.ProtocolAuto).  // Auto-detect
    OnMessage(handleMessage).
    Build()
```

### 3. Progressive Enhancement

Start simple with A2A, add SAGE when needed:

```go
// Development: A2A only
WithProtocol(adk.ProtocolA2A)

// Production: Add SAGE security
WithProtocol(adk.ProtocolSAGE).
WithSAGE(sage.Options{
    DID: "did:sage:ethereum:0x...",
    Network: sage.NetworkEthereum,
})
```

## Protocol Implementations

### A2A Protocol Adapter

**Location**: `adapters/a2a/`

**Purpose**: Wraps `sage-a2a-go` library to provide A2A protocol support.

**Key Components**:

```go
type A2AProtocol struct {
    client       *a2aclient.Client
    server       *a2asrv.Server
    taskManager  taskmanager.TaskManager
    storage      storage.Storage
}

// Implementation
func (p *A2AProtocol) SendMessage(ctx context.Context, msg *Message) (*Response, error) {
    // Convert ADK message to A2A protocol message
    a2aMsg := convertToA2AMessage(msg)

    // Send via sage-a2a-go client
    params := protocol.SendMessageParams{
        Message: a2aMsg,
    }

    result, err := p.client.SendMessage(ctx, params)
    if err != nil {
        return nil, err
    }

    // Convert A2A response back to ADK response
    return convertFromA2AResponse(result), nil
}
```

**Features**:
- Task lifecycle management (submitted → working → completed)
- Message history with contextID
- Streaming via Server-Sent Events (SSE)
- Redis or memory-based storage
- Push notifications

### SAGE Protocol Adapter

**Location**: `adapters/sage/`

**Purpose**: Integrates SAGE security library for DID-based identity and message signatures.

**Key Components**:

```go
type SAGEProtocol struct {
    a2a          *A2AProtocol      // Wraps A2A for messaging
    didManager   *did.Manager      // DID management
    verifier     *rfc9421.Verifier // Signature verification
    handshake    *handshake.Client // Secure session establishment
    sessionMgr   *session.Manager  // Session key management
}

func (p *SAGEProtocol) SendMessage(ctx context.Context, msg *Message) (*Response, error) {
    // 1. Ensure session exists (perform handshake if needed)
    sess, err := p.ensureSession(ctx, msg.RecipientDID)
    if err != nil {
        return nil, err
    }

    // 2. Sign message with RFC 9421
    signature, err := p.signMessage(msg, sess)
    if err != nil {
        return nil, err
    }

    // 3. Attach signature to message
    msg.Security = &SecurityOptions{
        Mode:      SecurityModeSAGE,
        Signature: &signature,
        KeyID:     &sess.KeyID,
        DID:       &p.didManager.MyDID,
    }

    // 4. Send via underlying A2A protocol
    return p.a2a.SendMessage(ctx, msg)
}
```

**Features**:
- DID resolution from blockchain (Ethereum, Kaia)
- RFC 9421 HTTP message signatures
- HPKE-based handshake protocol
- Session key management
- Replay attack prevention (nonce cache)
- Message encryption (ChaCha20-Poly1305)

## Message Format

### Base A2A Message

```json
{
  "message_id": "msg-550e8400-e29b-41d4-a716-446655440000",
  "context_id": "ctx-7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "role": "user",
  "parts": [
    {
      "kind": "text",
      "text": "What is the weather today?"
    }
  ],
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### SAGE-Enhanced Message

```json
{
  "message_id": "msg-550e8400-e29b-41d4-a716-446655440000",
  "context_id": "ctx-7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "role": "user",
  "parts": [
    {
      "kind": "text",
      "text": "What is the weather today?"
    }
  ],
  "metadata": {
    "timestamp": "2024-01-15T10:30:00Z"
  },
  "security": {
    "mode": "sage",
    "signature": "keyid=\"key-abc123\", algorithm=\"ed25519\", created=1705315800, signature=\"base64-encoded-signature\"",
    "key_id": "key-abc123",
    "did": "did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678"
  }
}
```

## Protocol Selection Logic

### Auto-Detection Algorithm

```go
type ProtocolSelector struct {
    mode         ProtocolMode
    a2aProtocol  *A2AProtocol
    sageProtocol *SAGEProtocol
}

func (s *ProtocolSelector) SelectProtocol(msg *Message) (Protocol, error) {
    switch s.mode {
    case ProtocolA2A:
        // Force A2A mode
        return s.a2aProtocol, nil

    case ProtocolSAGE:
        // Force SAGE mode
        return s.sageProtocol, nil

    case ProtocolAuto:
        // Auto-detect from message
        if msg.Security != nil && msg.Security.Mode == SecurityModeSAGE {
            return s.sageProtocol, nil
        }
        return s.a2aProtocol, nil

    default:
        return nil, fmt.Errorf("unknown protocol mode: %v", s.mode)
    }
}
```

### Configuration Options

```go
// Option 1: A2A Only (fastest, no blockchain)
agent := adk.NewAgent("agent").
    WithProtocol(adk.ProtocolA2A).
    Build()

// Option 2: SAGE Only (most secure, requires blockchain)
agent := adk.NewAgent("agent").
    WithProtocol(adk.ProtocolSAGE).
    WithSAGE(sage.Options{
        DID:     "did:sage:ethereum:0x...",
        Network: sage.NetworkEthereum,
    }).
    Build()

// Option 3: Auto-detect (flexible, recommended)
agent := adk.NewAgent("agent").
    WithProtocol(adk.ProtocolAuto).
    WithSAGE(sage.Optional()).
    Build()
```

## SAGE Security Features

### 1. DID-Based Identity

**DID Format**: `did:sage:<network>:<address>`

**Examples**:
- `did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678`
- `did:sage:kaia:0x9876543210fedcba9876543210fedcba98765432`

**DID Resolution**:
```go
// Resolve public key from blockchain
publicKey, err := didManager.ResolvePublicKey(ctx, "did:sage:ethereum:0x...")

// Verify agent is registered and active
agentInfo, err := didManager.GetAgentInfo(ctx, did)
```

### 2. RFC 9421 Message Signatures

**Signature Components**:
- `@method`: HTTP method (POST)
- `@authority`: Target host
- `@path`: Request path
- `content-type`: Message content type
- `content-digest`: SHA-256 digest of body
- `date`: Timestamp
- `x-agent-did`: Sender's DID

**Signature Header**:
```
Signature: keyid="key-abc123", algorithm="ed25519", created=1705315800,
           signature="MEUCIQDx7q...", nonce="random-nonce-xyz"
```

**Verification Process**:
```go
func (v *MessageVerifier) VerifyMessage(msg *Message) error {
    // 1. Extract DID from message
    did := msg.Security.DID

    // 2. Resolve public key from blockchain
    pubKey, err := v.didManager.ResolvePublicKey(ctx, *did)
    if err != nil {
        return fmt.Errorf("failed to resolve DID: %w", err)
    }

    // 3. Verify RFC 9421 signature
    signature := msg.Security.Signature
    if err := v.verifier.Verify(msg, *signature, pubKey); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    // 4. Check nonce (replay protection)
    if v.nonceCache.Exists(msg.Nonce) {
        return errors.New("nonce already used (replay attack)")
    }
    v.nonceCache.Add(msg.Nonce, time.Now().Add(5*time.Minute))

    return nil
}
```

### 3. Handshake Protocol

**4-Phase Handshake**:

```
Client Agent (A)                          Server Agent (B)
     │                                         │
     │  1. Invitation                          │
     ├────────────────────────────────────────>│
     │    - DID of A                           │
     │    - Signed with A's key                │
     │                                    ┌────▼────┐
     │                                    │ Verify  │
     │                                    │   DID   │
     │                                    └────┬────┘
     │  2. Response (Ephemeral Key B)          │
     │<────────────────────────────────────────┤
     │    - Encrypted with A's public key      │
     │                                         │
┌────▼────┐                                   │
│ Decrypt │                                   │
│ Key B   │                                   │
└────┬────┘                                   │
     │  3. Request (Ephemeral Key A)          │
     ├────────────────────────────────────────>│
     │    - Encrypted with B's public key      │
     │                                    ┌────▼────┐
     │                                    │ Decrypt │
     │                                    │  Key A  │
     │                                    └────┬────┘
     │                                    ┌────▼────┐
     │                                    │  HKDF   │
     │                                    │ Derive  │
     │                                    │ Session │
     │                                    └────┬────┘
     │  4. Complete                            │
     │<────────────────────────────────────────┤
     │    - Session established                │
┌────▼────┐                              ┌────▼────┐
│  HKDF   │                              │ Session │
│ Derive  │                              │  Ready  │
│ Session │                              └─────────┘
└────┬────┘
     │
┌────▼────┐
│ Session │
│  Ready  │
└─────────┘
```

**Session Keys**:
```go
type Session struct {
    SessionID       string
    KeyID           string
    EncryptionKey   []byte  // ChaCha20-Poly1305
    SigningKey      []byte  // HMAC-SHA256
    PeerDID         string
    CreatedAt       time.Time
    ExpiresAt       time.Time
}
```

### 4. Message Encryption

**AEAD Encryption** (ChaCha20-Poly1305):

```go
func (s *Session) Encrypt(plaintext []byte) ([]byte, error) {
    aead, err := chacha20poly1305.New(s.EncryptionKey)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, aead.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
    return ciphertext, nil
}
```

## Protocol Comparison

| Feature | A2A Protocol | SAGE Protocol |
|---------|-------------|---------------|
| **Identity** | Agent name (string) | DID on blockchain |
| **Authentication** | Optional API key | Required signature |
| **Message Integrity** | HTTPS only | RFC 9421 signatures |
| **Encryption** | TLS in transit | TLS + AEAD at rest |
| **Replay Protection** | None | Nonce cache |
| **Key Management** | None | Blockchain registry |
| **Latency** | ~10-50ms | ~100-200ms |
| **Blockchain Dependency** | No | Yes (Ethereum/Kaia) |
| **Use Case** | Internal agents | Cross-organization |

## Performance Optimization

### 1. DID Resolution Caching

```go
type DIDCache struct {
    cache map[string]*CacheEntry
    mu    sync.RWMutex
    ttl   time.Duration
}

type CacheEntry struct {
    PublicKey []byte
    ExpiresAt time.Time
}

func (c *DIDCache) Resolve(did string) ([]byte, error) {
    c.mu.RLock()
    entry, exists := c.cache[did]
    c.mu.RUnlock()

    if exists && time.Now().Before(entry.ExpiresAt) {
        return entry.PublicKey, nil
    }

    // Cache miss: resolve from blockchain
    pubKey, err := c.blockchain.ResolvePublicKey(did)
    if err != nil {
        return nil, err
    }

    c.mu.Lock()
    c.cache[did] = &CacheEntry{
        PublicKey: pubKey,
        ExpiresAt: time.Now().Add(c.ttl),
    }
    c.mu.Unlock()

    return pubKey, nil
}
```

### 2. Session Reuse

```go
// Reuse session for multiple messages
sess, err := agent.EstablishSession(ctx, peerDID)
for i := 0; i < 100; i++ {
    msg := adk.NewMessage("Hello " + strconv.Itoa(i))
    resp, err := agent.SendMessage(ctx, msg, adk.WithSession(sess))
}
```

### 3. Batch Verification

```go
// Verify multiple signatures in parallel
func (v *Verifier) VerifyBatch(messages []*Message) ([]error, error) {
    results := make([]error, len(messages))
    var wg sync.WaitGroup

    for i, msg := range messages {
        wg.Add(1)
        go func(idx int, m *Message) {
            defer wg.Done()
            results[idx] = v.VerifyMessage(m)
        }(i, msg)
    }

    wg.Wait()
    return results, nil
}
```

## Error Handling

### Protocol-Specific Errors

```go
// A2A Errors
var (
    ErrTaskNotFound = errors.New("task not found")
    ErrInvalidMessage = errors.New("invalid message format")
)

// SAGE Errors
var (
    ErrInvalidSignature = errors.New("invalid signature")
    ErrDIDNotResolved = errors.New("DID resolution failed")
    ErrSessionExpired = errors.New("session expired")
    ErrReplayAttack = errors.New("nonce already used")
)
```

### Graceful Degradation

```go
// Fallback to A2A if SAGE unavailable
func (s *ProtocolSelector) SelectWithFallback(msg *Message) Protocol {
    if msg.Security != nil && msg.Security.Mode == SecurityModeSAGE {
        // Try SAGE first
        if s.sageProtocol.IsAvailable() {
            return s.sageProtocol
        }

        // Fallback to A2A if blockchain unreachable
        log.Warn("SAGE protocol unavailable, falling back to A2A")
    }

    return s.a2aProtocol
}
```

## Testing

### Protocol Mocking

```go
type MockProtocol struct {
    SendMessageFunc func(ctx context.Context, msg *Message) (*Response, error)
}

func (m *MockProtocol) SendMessage(ctx context.Context, msg *Message) (*Response, error) {
    return m.SendMessageFunc(ctx, msg)
}

// Test
func TestAgentWithMockProtocol(t *testing.T) {
    mock := &MockProtocol{
        SendMessageFunc: func(ctx context.Context, msg *Message) (*Response, error) {
            return &Response{Text: "Mocked response"}, nil
        },
    }

    agent := adk.NewAgent("test").
        WithProtocol(mock).
        Build()

    resp, err := agent.SendMessage(ctx, msg)
    assert.NoError(t, err)
    assert.Equal(t, "Mocked response", resp.Text)
}
```

---

[← Architecture Overview](overview.md) | [Message Flow →](message-flow.md)
