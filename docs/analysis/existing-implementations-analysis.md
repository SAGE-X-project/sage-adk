# Analysis: Existing Agent Implementations in SAGE-X-Project

**Date**: 2025-10-07
**Version**: 1.0
**Purpose**: Comprehensive analysis of current agent implementation patterns to inform SAGE ADK design decisions

## Executive Summary

This analysis examines three key implementations in the SAGE-X ecosystem:
- **sage-a2a-go**: Go implementation of A2A protocol with full task lifecycle management
- **sage**: Blockchain-based security framework with DID, handshake, and RFC 9421 signatures
- **A2A Specification**: Protocol definition for agent-to-agent communication

The findings reveal clear patterns and pain points that should guide the SAGE ADK's architecture to provide a superior developer experience while maintaining security and interoperability.

---

## 1. Current Agent Implementation Patterns

### 1.1 A2A Protocol Agent Pattern (sage-a2a-go)

#### Core Abstractions

**1. MessageProcessor Interface**
```go
type MessageProcessor interface {
    ProcessMessage(
        ctx context.Context,
        message protocol.Message,
        options ProcessOptions,
        taskHandler TaskHandler,
    ) (*MessageProcessingResult, error)
}
```

**Strengths:**
- Clean separation of concerns: protocol handling vs. business logic
- Single interface for developers to implement
- Supports both streaming and non-streaming modes
- Provides task management primitives through TaskHandler

**Weaknesses:**
- High cognitive overhead - developers need to understand:
  - Task lifecycle states (submitted, working, completed, failed, etc.)
  - Streaming vs. non-streaming semantics
  - Task building, subscribing, updating, and cleanup
- Lots of boilerplate for simple use cases
- No guidance on LLM integration
- Manual error handling and message construction

**2. TaskManager Implementation**
```go
type TaskManager interface {
    OnSendMessage(ctx, params) (*MessageResult, error)
    OnSendMessageStream(ctx, params) (<-chan StreamingMessageEvent, error)
    OnGetTask(ctx, params) (*Task, error)
    OnCancelTask(ctx, params) (*Task, error)
    OnPushNotificationSet(ctx, params) (*TaskPushNotificationConfig, error)
    OnPushNotificationGet(ctx, params) (*TaskPushNotificationConfig, error)
    OnResubscribe(ctx, params) (<-chan StreamingMessageEvent, error)
}
```

**Current Flow:**
1. HTTP request arrives → Server routes to JSON-RPC handler
2. Server calls TaskManager method (OnSendMessage, OnSendMessageStream, etc.)
3. TaskManager validates request, creates task/context
4. TaskManager calls MessageProcessor.ProcessMessage()
5. Developer's business logic executes
6. TaskManager handles state updates, notifications, cleanup

**Pain Points:**
- Two levels of abstraction (TaskManager + MessageProcessor) creates confusion
- TaskManager is essential but developers must understand its lifecycle
- Memory management (CleanTask) is manual and error-prone
- No built-in retry, timeout, or circuit breaker patterns

#### Agent Creation Workflow

**Current Pattern (Simple Example):**
```go
// 1. Implement MessageProcessor
type simpleMessageProcessor struct{}

func (p *simpleMessageProcessor) ProcessMessage(
    ctx context.Context,
    message protocol.Message,
    options taskmanager.ProcessOptions,
    handler taskmanager.TaskHandler,
) (*taskmanager.MessageProcessingResult, error) {
    // Extract text manually
    text := extractText(message)
    if text == "" {
        // Manual error handling
        errorMessage := protocol.NewMessage(
            protocol.MessageRoleAgent,
            []protocol.Part{protocol.NewTextPart("input message must contain text.")},
        )
        return &taskmanager.MessageProcessingResult{
            Result: &errorMessage,
        }, nil
    }

    // Process (no LLM helper)
    result := reverseString(text)

    // Different code paths for streaming vs non-streaming
    if !options.Streaming {
        responseMessage := protocol.NewMessage(
            protocol.MessageRoleAgent,
            []protocol.Part{protocol.NewTextPart(fmt.Sprintf("Processed result: %s", result))},
        )
        return &taskmanager.MessageProcessingResult{
            Result: &responseMessage,
        }, nil
    }

    // Streaming: build task, subscribe, spawn goroutine
    taskID, err := handler.BuildTask(nil, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to build task: %w", err)
    }

    subscriber, err := handler.SubscribeTask(&taskID)
    if err != nil {
        return nil, fmt.Errorf("failed to subscribe to task: %w", err)
    }

    go func() {
        defer func() {
            if subscriber != nil {
                subscriber.Close()
            }
            handler.CleanTask(&taskID)
        }()

        // Send multiple events...
        // (50+ lines of event sending code)
    }()

    return &taskmanager.MessageProcessingResult{
        StreamingEvents: subscriber,
    }, nil
}

// 2. Create AgentCard manually
agentCard := server.AgentCard{
    Name:        "Simple A2A Example Server",
    Description: "A simple example A2A server that reverses text",
    URL:         fmt.Sprintf("http://%s:%d/", *host, *port),
    Version:     "1.0.0",
    Provider: &server.AgentProvider{
        Organization: "tRPC-A2A-Go Examples",
        URL:          stringPtr(fmt.Sprintf("http://%s:%d/", *host, *port)),
    },
    Capabilities: server.AgentCapabilities{
        Streaming:              boolPtr(true),
        PushNotifications:      boolPtr(false),
        StateTransitionHistory: boolPtr(true),
    },
    DefaultInputModes:  []string{"text"},
    DefaultOutputModes: []string{"text"},
    Skills: []server.AgentSkill{
        {
            ID:          "text_reversal",
            Name:        "Text Reversal",
            Description: stringPtr("Reverses the input text"),
            Tags:        []string{"text", "processing"},
            Examples:    []string{"Hello, world!"},
            InputModes:  []string{"text"},
            OutputModes: []string{"text"},
        },
    },
}

// 3. Create TaskManager
processor := &simpleMessageProcessor{}
taskManager, err := taskmanager.NewMemoryTaskManager(processor)
if err != nil {
    log.Fatalf("Failed to create task manager: %v", err)
}

// 4. Create Server
srv, err := server.NewA2AServer(agentCard, taskManager)
if err != nil {
    log.Fatalf("Failed to create server: %v", err)
}

// 5. Start Server
if err := srv.Start(":8080"); err != nil {
    log.Fatalf("Server failed: %v", err)
}
```

**Analysis:**
- **~100+ lines** just to reverse a string!
- Manual text extraction (no helpers)
- No LLM integration guidance
- Streaming code is complex (task building, subscriptions, goroutines)
- Lots of pointer helpers (`stringPtr`, `boolPtr`)
- Error handling is verbose and repetitive

---

### 1.2 SAGE Protocol Agent Pattern

#### Core Components

**1. Handshake Protocol (4-Phase)**
```go
// Client side
client := handshake.NewClient(conn, myKeyPair)

// Phase 1: Invitation
inv := &handshake.InvitationMessage{
    BaseMessage: message.BaseMessage{ContextID: contextID},
}
resp, err := client.Invitation(ctx, *inv, myDID)

// Phase 2: Request (with ephemeral key)
req := &handshake.RequestMessage{
    BaseMessage:     message.BaseMessage{ContextID: contextID},
    EphemeralPubKey: myEphemeralKey,
}
resp, err = client.Request(ctx, *req, peerPublicKey, myDID)

// Phase 3: Response (server sends back)
// Phase 4: Complete
comp := &handshake.CompleteMessage{
    BaseMessage: message.BaseMessage{ContextID: contextID},
}
resp, err = client.Complete(ctx, *comp, myDID)
```

**Server side** (event-driven):
```go
events := &MyEventHandler{
    sessionManager: session.NewManager(),
}
server := handshake.NewServer(keyPair, events, resolver, nil, cleanupInterval)

// Implement event callbacks:
// - OnInvitation(ctx, msg, peerDID) error
// - OnRequest(ctx, msg, peerDID, peerEph) error
// - OnResponse(ctx, msg, peerDID, peerEph) error
// - OnComplete(ctx, contextID, sharedSecret, metadata) error
```

**Strengths:**
- Strong security guarantees (forward secrecy, replay protection)
- Event-driven architecture on server side
- Clean separation of handshake from message processing
- Session management built-in

**Weaknesses:**
- Complex 4-phase handshake requires deep understanding
- Manual DID management and resolution
- No abstraction over blockchain interactions
- Developers must implement event handlers for all phases

**2. Session Management**
```go
// Sessions created automatically after handshake
sess, ok := sessionManager.GetByKeyID(keyID)

// Sessions provide:
// - Encryption/decryption (ChaCha20-Poly1305)
// - Message signing (HMAC-SHA256)
// - Nonce tracking (replay protection)
// - Automatic cleanup (TTL, idle timeout)
```

**3. RFC 9421 Message Signatures**
```go
// Create HTTP message builder
builder := rfc9421.NewMessageBuilder()
msg := builder.
    Method("POST").
    Authority("api.example.com").
    Path("/api/v1/chat").
    Header("Content-Type", "application/json").
    Body([]byte(requestBody)).
    Build()

// Sign the message
verifier := rfc9421.NewHTTPVerifier(sess, sessionManager)
signature, err := verifier.SignHTTPMessage(msg, keyID, []string{
    "@method", "@authority", "@path", "content-type", "content-digest",
})

// Verify signature
err = verifier.VerifyHTTPSignature(msg, signature, keyID)
```

**Strengths:**
- Standards-compliant (RFC 9421)
- Flexible field selection for signing
- Integrated with session management

**Weaknesses:**
- HTTP-focused (doesn't directly map to A2A protocol)
- Developers must understand HTTP signature semantics
- No automatic signing for common A2A message patterns

**4. DID Management**
```go
// Register agent on blockchain
./build/bin/sage-did register \
  --chain ethereum \
  --key keys/ethereum.key \
  --name "My AI Agent" \
  --endpoint "https://api.myagent.com" \
  --capabilities "chat,code,analysis"

// Resolve DID
./build/bin/sage-did resolve did:sage:ethereum:0x...

// In code:
resolver := did.NewResolver(chainConfig)
doc, err := resolver.Resolve(ctx, "did:sage:ethereum:0x...")
```

**Strengths:**
- Blockchain-backed identity (immutable, verifiable)
- Multi-chain support (Ethereum, Kaia, Solana)
- CLI tools for management

**Weaknesses:**
- Requires blockchain node access (RPC endpoint)
- Gas costs for registration/updates
- No automatic DID provisioning for development
- Developers must manage private keys securely

---

### 1.3 Multi-Agent Orchestration Pattern

**Example: Root Agent with Subagents**
```go
type rootAgentProcessor struct {
    llm                 *googleai.GoogleAI
    creativeClient      *client.A2AClient
    exchangeClient      *client.A2AClient
    reimbursementClient *client.A2AClient
}

func (p *rootAgentProcessor) ProcessMessage(...) (*MessageProcessingResult, error) {
    text := extractText(message)

    // Route to subagent using LLM
    subagent, err := p.routeTaskToSubagent(ctx, text)

    // Forward to appropriate subagent
    switch subagent {
    case "creative":
        result, err = p.callCreativeAgent(ctx, text)
    case "exchange":
        result, err = p.callExchangeAgent(ctx, text)
    case "reimbursement":
        result, err = p.callReimbursementAgent(ctx, text)
    }

    // Return result
    responseMessage := protocol.NewMessage(
        protocol.MessageRoleAgent,
        []protocol.Part{protocol.NewTextPart(result)},
    )
    return &MessageProcessingResult{Result: &responseMessage}, nil
}

func (p *rootAgentProcessor) callCreativeAgent(ctx context.Context, text string) (string, error) {
    message := protocol.NewMessage(
        protocol.MessageRoleUser,
        []protocol.Part{protocol.NewTextPart(text)},
    )

    params := protocol.SendMessageParams{Message: message}
    result, err := p.creativeClient.SendMessage(ctx, params)
    if err != nil {
        return "", fmt.Errorf("failed to send message to creative agent: %w", err)
    }

    // Extract text from response
    switch result.Result.GetKind() {
    case protocol.KindMessage:
        msg := result.Result.(*protocol.Message)
        return extractText(*msg), nil
    case protocol.KindTask:
        task := result.Result.(*protocol.Task)
        if task.Status.Message != nil {
            return extractText(*task.Status.Message), nil
        }
        return "", fmt.Errorf("no response message from creative agent")
    }
}
```

**Strengths:**
- Clean separation of orchestration logic
- LLM-based routing is flexible
- Uses standard A2A client for subagent communication

**Weaknesses:**
- Lots of boilerplate for each subagent call
- No built-in retry or error handling patterns
- Manual result extraction (Message vs Task)
- No parallel execution support
- No subagent discovery mechanism
- Hard-coded URLs (no service registry)

---

## 2. A2A Protocol Agent Pattern Details

### 2.1 Key Interfaces

**1. Protocol Types**
```go
// Message represents communication unit
type Message struct {
    MessageID        string
    ContextID        *string
    Role             MessageRole  // user | agent
    Parts            []Part       // TextPart | FilePart | DataPart
    Kind             string       // "message"
    Metadata         map[string]interface{}
    TaskID           *string
    ReferenceTaskIDs []string
    Extensions       []string
}

// Task represents work unit
type Task struct {
    ID           string
    ContextID    string
    Status       TaskStatus
    Artifacts    []Artifact
    Created      time.Time
    Updated      time.Time
    Metadata     map[string]interface{}
}

// TaskStatus represents task lifecycle
type TaskStatus struct {
    State   TaskState  // submitted | working | completed | failed | canceled
    Message *Message   // Final result message
}
```

**2. Agent Card (Discovery)**
```go
type AgentCard struct {
    Name               string
    Description        *string
    URL                string
    Version            string
    Provider           *AgentProvider
    Capabilities       AgentCapabilities
    DefaultInputModes  []string
    DefaultOutputModes []string
    Skills             []AgentSkill
    Extensions         []string
}

type AgentCapabilities struct {
    Streaming              *bool
    PushNotifications      *bool
    StateTransitionHistory *bool
}

type AgentSkill struct {
    ID          string
    Name        string
    Description *string
    Tags        []string
    Examples    []string
    InputModes  []string
    OutputModes []string
}
```

**3. Client API**
```go
// Create client
client, err := client.NewA2AClient("http://localhost:8080/")

// Send message (blocking)
result, err := client.SendMessage(ctx, protocol.SendMessageParams{
    Message: message,
})

// Stream message (SSE)
eventChan, err := client.StreamMessage(ctx, params)
for event := range eventChan {
    // Handle event
}

// Get task
task, err := client.GetTasks(ctx, protocol.TaskQueryParams{ID: taskID})

// Cancel task
task, err := client.CancelTasks(ctx, protocol.TaskIDParams{ID: taskID})
```

**Strengths:**
- Well-defined protocol types
- Supports rich content (text, files, data)
- Streaming and non-streaming modes
- Task lifecycle management
- Agent discovery via AgentCard

**Weaknesses:**
- Many optional fields (pointers everywhere)
- Message construction is verbose
- No helpers for common patterns (text-only messages, etc.)
- Task state machine is complex

### 2.2 Message Handling Patterns

**Pattern 1: Simple Request-Response**
```go
func (p *processor) ProcessMessage(...) (*MessageProcessingResult, error) {
    text := extractText(message)
    result := processText(text)

    responseMessage := protocol.NewMessage(
        protocol.MessageRoleAgent,
        []protocol.Part{protocol.NewTextPart(result)},
    )

    return &MessageProcessingResult{Result: &responseMessage}, nil
}
```

**Pattern 2: Task-Based (Non-Streaming)**
```go
func (p *processor) ProcessMessage(...) (*MessageProcessingResult, error) {
    text := extractText(message)

    // Create task
    taskID, _ := handler.BuildTask(nil, nil)

    // Process asynchronously
    go func() {
        defer handler.CleanTask(&taskID)

        result := processText(text)

        // Update task state
        msg := protocol.NewMessage(protocol.MessageRoleAgent,
            []protocol.Part{protocol.NewTextPart(result)})
        handler.UpdateTaskState(&taskID, protocol.TaskStateCompleted, &msg)
    }()

    // Return task immediately
    task, _ := handler.GetTask(&taskID)
    return &MessageProcessingResult{Result: task.Task()}, nil
}
```

**Pattern 3: Streaming**
```go
func (p *processor) ProcessMessage(...) (*MessageProcessingResult, error) {
    text := extractText(message)

    taskID, _ := handler.BuildTask(nil, nil)
    subscriber, _ := handler.SubscribeTask(&taskID)

    go func() {
        defer func() {
            subscriber.Close()
            handler.CleanTask(&taskID)
        }()

        // Send working status
        subscriber.Send(protocol.StreamingMessageEvent{
            Result: &protocol.TaskStatusUpdateEvent{
                TaskID: taskID,
                Status: protocol.TaskStatus{State: protocol.TaskStateWorking},
            },
        })

        // Process and stream chunks
        for chunk := range processStreamingText(text) {
            msg := protocol.NewMessage(protocol.MessageRoleAgent,
                []protocol.Part{protocol.NewTextPart(chunk)})
            subscriber.Send(protocol.StreamingMessageEvent{Result: &msg})
        }

        // Send completion
        subscriber.Send(protocol.StreamingMessageEvent{
            Result: &protocol.TaskStatusUpdateEvent{
                TaskID: taskID,
                Status: protocol.TaskStatus{State: protocol.TaskStateCompleted},
                Final:  true,
            },
        })
    }()

    return &MessageProcessingResult{StreamingEvents: subscriber}, nil
}
```

**Analysis:**
- Three distinct patterns for common scenarios
- Lots of boilerplate in each pattern
- Manual state management
- Easy to make mistakes (forgetting CleanTask, not setting Final flag, etc.)

### 2.3 Task Lifecycle Management

**States:**
```
submitted → working → completed
                   ↘ failed
                   ↘ canceled
                   ↘ input-required → working → completed
```

**State Transitions:**
```go
// Submit (automatic when message received)
taskID, _ := handler.BuildTask(nil, nil)

// Working
handler.UpdateTaskState(&taskID, protocol.TaskStateWorking, nil)

// Input Required
handler.UpdateTaskState(&taskID, protocol.TaskStateInputRequired, &promptMsg)

// Completed
handler.UpdateTaskState(&taskID, protocol.TaskStateCompleted, &resultMsg)

// Failed
handler.UpdateTaskState(&taskID, protocol.TaskStateFailed, &errorMsg)

// Cleanup (manual!)
defer handler.CleanTask(&taskID)
```

**Pain Points:**
- Manual state management (easy to forget transitions)
- CleanTask is manual and error-prone
- No automatic timeout/expiration
- No built-in retry for transient failures
- Task IDs are strings (easy to mistype or lose)

---

## 3. SAGE Protocol Agent Pattern Details

### 3.1 Key Interfaces

**1. Handshake Messages**
```go
// Phase 1: Invitation
type InvitationMessage struct {
    BaseMessage
    // No encryption, signed with DID key
}

// Phase 2: Request
type RequestMessage struct {
    BaseMessage
    EphemeralPubKey []byte  // X25519 public key
    // Encrypted with peer's Ed25519 public key
}

// Phase 3: Response
type ResponseMessage struct {
    BaseMessage
    EphemeralPubKey []byte
    // Encrypted with peer's Ed25519 public key
}

// Phase 4: Complete
type CompleteMessage struct {
    BaseMessage
    // No encryption, signed with DID key
}
```

**2. Session Interface**
```go
type SecureSession interface {
    // Encryption
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(ciphertext []byte) ([]byte, error)

    // Signing
    Sign(data []byte) ([]byte, error)
    Verify(data, signature []byte) error

    // Metadata
    SessionID() string
    PeerDID() string
    Created() time.Time
    LastUsed() time.Time

    // Lifecycle
    IsExpired() bool
    Close() error
}
```

**3. DID Resolution**
```go
type Resolver interface {
    Resolve(ctx context.Context, did string) (*DIDDocument, error)
}

type DIDDocument struct {
    DID            string
    PublicKey      crypto.PublicKey
    AgentEndpoint  string
    Capabilities   []string
    ChainID        int64
    ContractAddr   string
    IsActive       bool
}
```

**Strengths:**
- Strong cryptographic guarantees
- Standards-compliant (W3C DID, RFC 9421)
- Blockchain-backed identity
- Forward secrecy via ephemeral keys

**Weaknesses:**
- Complex multi-phase handshake
- Requires blockchain infrastructure
- Manual session lifecycle management
- No automatic key rotation

### 3.2 Handshake Protocol Flow

**Client (Initiator):**
```
1. Create ephemeral X25519 key pair
2. Send Invitation (clear, signed with DID key)
3. Receive Invitation response
4. Send Request (encrypted with peer's Ed25519 key, contains ephemeral pub key)
5. Receive Response (encrypted, contains peer's ephemeral pub key)
6. Derive shared secret using ECDH
7. Send Complete (clear, signed)
8. Create session from shared secret
```

**Server (Responder):**
```
1. Receive Invitation → Verify signature → Resolve sender's DID
2. Send Invitation response
3. Receive Request → Decrypt → Extract peer's ephemeral key
4. Create ephemeral X25519 key pair
5. Send Response (encrypted with peer's Ed25519 key)
6. Receive Complete → Verify signature
7. Derive shared secret using ECDH
8. Create session from shared secret
```

**Session Derivation:**
```go
// Both parties compute:
sharedSecret := ECDH(myEphemeralPriv, peerEphemeralPub)

// Derive session keys using HKDF
sessionID := HKDF(sharedSecret, "session-id")
encKeyC2S := HKDF(sharedSecret, "client-to-server-enc")
encKeyS2C := HKDF(sharedSecret, "server-to-client-enc")
sigKeyC2S := HKDF(sharedSecret, "client-to-server-sig")
sigKeyS2C := HKDF(sharedSecret, "server-to-client-sig")
```

**Pain Points:**
- 4 round trips before first real message
- Manual key management (ephemeral keys)
- Error handling at each phase
- No automatic retry on failure
- Difficult to test (requires blockchain)

### 3.3 DID and Signature Handling

**DID Registration:**
```go
// On blockchain (Ethereum example)
registry.RegisterAgent(
    did="did:sage:ethereum:0x1234...",
    name="My Agent",
    endpoint="https://api.myagent.com",
    publicKey=ed25519PubKeyBytes,
    capabilities=["chat", "analysis"],
)

// Smart contract validates:
// - Public key ownership (challenge-response)
// - Key format (length, zero-key check)
// - No revoked keys
```

**Message Signing (RFC 9421):**
```go
// Sign HTTP message
builder := rfc9421.NewMessageBuilder()
msg := builder.
    Method("POST").
    Authority("api.example.com").
    Path("/message/send").
    Header("Content-Type", "application/json").
    Header("Content-Digest", "sha-256=<hash>").
    Body(jsonPayload).
    Build()

// Create signature
verifier := rfc9421.NewHTTPVerifier(session, sessionManager)
signature, _ := verifier.SignHTTPMessage(msg, keyID, []string{
    "@method",
    "@authority",
    "@path",
    "content-type",
    "content-digest",
})

// Signature format: keyid="<id>", signature="<base64>", fields="@method @authority..."
```

**Signature Verification:**
```go
// Parse signature
sig, _ := verifier.ParseSignature(signatureHeader)

// Resolve signer's DID
doc, _ := resolver.Resolve(ctx, sig.KeyID)

// Rebuild canonical message
canonical := verifier.BuildCanonicalMessage(msg, sig.SignedFields)

// Verify signature
err := crypto.VerifyEd25519(doc.PublicKey, canonical, sig.Signature)
```

**Pain Points:**
- HTTP signatures don't map directly to A2A protocol (uses different message structure)
- Manual field selection for signing (what should be signed?)
- DID resolution adds latency (requires blockchain query)
- No caching guidance for DID documents
- Signature parsing and verification is complex

---

## 4. Pain Points to Address in SAGE ADK

### 4.1 Developer Experience Pain Points

**1. High Boilerplate for Simple Tasks**

*Current:* 100+ lines to create a basic echo agent
```go
// Implement MessageProcessor
type processor struct{}
func (p *processor) ProcessMessage(...) (*MessageProcessingResult, error) {
    text := extractText(message)  // Manual extraction
    if text == "" {
        // Manual error handling
        errorMessage := protocol.NewMessage(
            protocol.MessageRoleAgent,
            []protocol.Part{protocol.NewTextPart("error message")},
        )
        return &MessageProcessingResult{Result: &errorMessage}, nil
    }

    result := strings.ToUpper(text)

    // Manual response construction
    responseMessage := protocol.NewMessage(
        protocol.MessageRoleAgent,
        []protocol.Part{protocol.NewTextPart(result)},
    )
    return &MessageProcessingResult{Result: &responseMessage}, nil
}

// Create AgentCard (30+ lines of configuration)
agentCard := server.AgentCard{...}

// Create TaskManager
processor := &processor{}
taskManager, _ := taskmanager.NewMemoryTaskManager(processor)

// Create Server
srv, _ := server.NewA2AServer(agentCard, taskManager)

// Start
srv.Start(":8080")
```

*Desired:* 10-20 lines for same functionality
```go
agent := adk.NewAgent("echo-agent").
    OnMessage(func(ctx context.Context, msg *adk.Message) error {
        return msg.Reply(strings.ToUpper(msg.Text()))
    }).
    Build()

agent.Start(":8080")
```

**2. No LLM Integration Guidance**

*Current:* Developers must:
- Choose LLM provider
- Handle API keys and configuration
- Manage rate limits and errors
- Convert between A2A messages and LLM formats
- Implement streaming if needed

*Desired:*
```go
agent := adk.NewAgent("my-agent").
    WithLLM(llm.OpenAI()).  // Auto-detects from env
    OnMessage(func(ctx context.Context, msg *adk.Message) error {
        response, _ := msg.LLM().Generate(ctx, msg.Text())
        return msg.Reply(response)
    }).
    Build()
```

**3. Complex Task Lifecycle Management**

*Current:*
- Manual task creation: `taskID, _ := handler.BuildTask(nil, nil)`
- Manual state updates: `handler.UpdateTaskState(&taskID, state, msg)`
- Manual cleanup: `defer handler.CleanTask(&taskID)`
- Manual artifact management
- Manual streaming event coordination

*Desired:*
```go
// Simple response (automatic task creation/cleanup)
return msg.Reply("response")

// Streaming (automatic task management)
return msg.Stream(func(stream *adk.Stream) error {
    for chunk := range processInChunks() {
        stream.Send(chunk)
    }
    return nil
})
```

**4. Streaming Complexity**

*Current:* ~50 lines for streaming
```go
taskID, _ := handler.BuildTask(nil, nil)
subscriber, _ := handler.SubscribeTask(&taskID)

go func() {
    defer func() {
        subscriber.Close()
        handler.CleanTask(&taskID)
    }()

    // Send working status
    subscriber.Send(protocol.StreamingMessageEvent{
        Result: &protocol.TaskStatusUpdateEvent{...},
    })

    // Send chunks
    for chunk := range chunks {
        msg := protocol.NewMessage(...)
        subscriber.Send(protocol.StreamingMessageEvent{Result: &msg})
    }

    // Send completion
    subscriber.Send(protocol.StreamingMessageEvent{
        Result: &protocol.TaskStatusUpdateEvent{Final: true},
    })
}()

return &MessageProcessingResult{StreamingEvents: subscriber}, nil
```

*Desired:*
```go
return msg.Stream(func(s *adk.Stream) error {
    for chunk := range chunks {
        s.Send(chunk)
    }
    return nil
})
```

**5. Manual Error Handling**

*Current:*
```go
if text == "" {
    errorMessage := protocol.NewMessage(
        protocol.MessageRoleAgent,
        []protocol.Part{protocol.NewTextPart("Error: empty input")},
    )
    return &MessageProcessingResult{Result: &errorMessage}, nil
}

result, err := callLLM(text)
if err != nil {
    errorMessage := protocol.NewMessage(
        protocol.MessageRoleAgent,
        []protocol.Part{protocol.NewTextPart(fmt.Sprintf("Error: %v", err))},
    )
    return &MessageProcessingResult{Result: &errorMessage}, nil
}
```

*Desired:*
```go
if text == "" {
    return msg.Error("empty input")
}

result, err := callLLM(text)
if err != nil {
    return err  // Auto-converted to error message
}
```

### 4.2 Security Integration Pain Points

**1. SAGE Handshake is Too Manual**

*Current:*
- Must implement all 4 phases manually
- Must manage ephemeral keys
- Must handle DID resolution
- Must derive session keys
- No error recovery

*Desired:*
```go
// Client
agent := adk.NewAgent("client").
    WithSAGE(sage.FromEnv()).
    Build()

// Server (automatic handshake handling)
agent := adk.NewAgent("server").
    WithSAGE(sage.FromEnv()).
    OnMessage(handleMessage).  // Only called after successful handshake
    Build()
```

**2. DID Management is Complex**

*Current:*
- CLI tools for registration
- Manual blockchain interactions
- Gas cost considerations
- Private key management
- No test/dev mode

*Desired:*
```go
// Production
agent.WithSAGE(sage.Options{
    DID:     "did:sage:ethereum:0x...",
    Network: sage.NetworkEthereum,
    PrivateKey: loadFromEnv(),
})

// Development (auto-generates local DID)
agent.WithSAGE(sage.Development())
```

**3. Signature Verification is Low-Level**

*Current:*
```go
builder := rfc9421.NewMessageBuilder()
msg := builder.Method("POST").Authority(...).Path(...).Body(json).Build()
verifier := rfc9421.NewHTTPVerifier(sess, sessionMgr)
signature, _ := verifier.SignHTTPMessage(msg, keyID, fields)
// ... verification ...
```

*Desired:*
```go
// Automatic signing/verification for A2A messages
// Developers don't need to think about it
agent.WithSAGE(sage.FromEnv())  // Signatures handled automatically
```

**4. Protocol Switching is Unclear**

*Current:*
- No guidance on when to use A2A vs SAGE
- No automatic detection
- Hard to support both in same agent

*Desired:*
```go
// Auto-detect protocol from incoming messages
agent.WithProtocol(adk.ProtocolAuto)

// Or explicit modes
agent.WithProtocol(adk.ProtocolA2A)   // A2A only
agent.WithProtocol(adk.ProtocolSAGE)  // SAGE only
```

### 4.3 Configuration Pain Points

**1. Many Environment Variables**

*Current:* Developers must know and set:
```
A2A_STORAGE_TYPE=redis
A2A_REDIS_URL=redis://localhost:6379
SAGE_ENABLED=true
SAGE_DID=did:sage:ethereum:0x...
SAGE_NETWORK=ethereum
ETHEREUM_RPC_URL=https://...
ETHEREUM_CONTRACT_ADDRESS=0x...
SAGE_PRIVATE_KEY=0x...
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
LLM_MODEL=gpt-4
ADK_SERVER_PORT=8080
LOG_LEVEL=info
METRICS_ENABLED=true
```

*Desired:*
```go
// Sensible defaults, minimal config
// Only need:
OPENAI_API_KEY=sk-...

// Optional SAGE:
SAGE_DID=did:sage:ethereum:0x...
SAGE_PRIVATE_KEY=0x...
```

**2. AgentCard is Verbose**

*Current:* 30-50 lines to describe agent
```go
agentCard := server.AgentCard{
    Name: "...",
    Description: stringPtr("..."),
    URL: "...",
    Version: "...",
    Provider: &server.AgentProvider{
        Organization: "...",
        URL: stringPtr("..."),
    },
    Capabilities: server.AgentCapabilities{
        Streaming: boolPtr(true),
        PushNotifications: boolPtr(false),
        StateTransitionHistory: boolPtr(true),
    },
    DefaultInputModes: []string{"text"},
    DefaultOutputModes: []string{"text"},
    Skills: []server.AgentSkill{...},
}
```

*Desired:*
```go
agent := adk.NewAgent("my-agent").
    WithDescription("Agent that does X").
    WithSkill(adk.Skill{
        ID:   "chat",
        Name: "Chat",
    }).
    Build()

// AgentCard generated automatically
```

### 4.4 Multi-Agent Pain Points

**1. No Subagent Discovery**

*Current:*
- Hard-coded URLs
- Manual client creation
- No service registry
- No health checking

*Desired:*
```go
// Discover and call subagents
creative := agent.SubAgent("creative")  // Auto-discovers
result, _ := creative.Send(ctx, "write a poem")
```

**2. No Orchestration Helpers**

*Current:*
- Manual routing logic
- No parallel execution
- No retry/timeout patterns
- Manual result aggregation

*Desired:*
```go
// Parallel execution
results := agent.ParallelSend(ctx, message,
    agent.SubAgent("agent1"),
    agent.SubAgent("agent2"),
    agent.SubAgent("agent3"),
)

// Conditional routing
agent.Route(func(msg *adk.Message) string {
    if containsKeyword(msg, "creative") {
        return "creative-agent"
    }
    return "general-agent"
})
```

**3. No Tool/Capability Registry**

*Current:*
- Each agent has Skills in AgentCard
- No standard way to query capabilities
- No dynamic tool discovery

*Desired:*
```go
// Register tools
agent.WithTool(tools.Calculator())
agent.WithTool(tools.WebSearch())

// Query agent capabilities
caps := await client.GetCapabilities("did:sage:ethereum:0x...")
if caps.HasTool("calculator") {
    result := client.CallTool("calculator", params)
}
```

---

## 5. Integration Patterns Analysis

### 5.1 LLM Integration Patterns

**Current State: No Standard Pattern**

Different examples use different approaches:

**Pattern 1: Direct API Calls**
```go
// Using langchaingo
llm, _ := googleai.New(ctx, googleai.WithAPIKey(apiKey))
completion, _ := llm.Call(ctx, prompt, llms.WithTemperature(0.7))
```

**Pattern 2: Custom Wrappers**
```go
// Custom LLM client
type LLMClient interface {
    Generate(ctx context.Context, prompt string) (string, error)
    GenerateStream(ctx context.Context, prompt string) (<-chan string, error)
}
```

**Pain Points:**
- No standard abstraction
- Each agent reinvents the wheel
- No prompt engineering helpers
- No conversation history management
- No automatic retries or fallbacks
- No cost tracking

**Desired Pattern:**
```go
// Unified LLM interface
agent.WithLLM(llm.OpenAI())

// In message handler
func handleMessage(ctx context.Context, msg *adk.Message) error {
    // Simple generation
    response, _ := msg.LLM().Generate(ctx, msg.Text())
    return msg.Reply(response)

    // Streaming
    return msg.LLM().GenerateStream(ctx, msg.Text(), func(chunk string) error {
        return msg.SendChunk(chunk)
    })

    // With history
    response, _ := msg.LLM().GenerateWithHistory(ctx, msg.Text(), msg.History())

    // With tools
    response, _ := msg.LLM().GenerateWithTools(ctx, msg.Text(), msg.Tools())
}
```

### 5.2 State Management Patterns

**Current State: Manual Session Management**

**Pattern 1: In-Memory Map**
```go
type processor struct {
    sessions map[string]*SessionData
    mu       sync.RWMutex
}

func (p *processor) ProcessMessage(...) {
    p.mu.Lock()
    session, ok := p.sessions[contextID]
    if !ok {
        session = &SessionData{History: []Message{}}
        p.sessions[contextID] = session
    }
    p.mu.Unlock()

    // Use session...
}
```

**Pattern 2: Redis**
```go
// Store in Redis
sessionKey := "session:" + contextID
data, _ := json.Marshal(session)
redis.Set(ctx, sessionKey, data, time.Hour)

// Retrieve
data, _ := redis.Get(ctx, sessionKey)
json.Unmarshal(data, &session)
```

**Pattern 3: TaskManager's Context**
```go
// Limited to current message processing
history := handler.GetMessageHistory()
contextID := handler.GetContextID()
metadata, _ := handler.GetMetadata()
```

**Pain Points:**
- No standard state management
- Manual serialization/deserialization
- No TTL management
- No cleanup strategy
- No transactional updates
- Can't share state across agents

**Desired Pattern:**
```go
// Agent-level state
agent.WithState(state.Redis(redisClient))

// In message handler
func handleMessage(ctx context.Context, msg *adk.Message) error {
    // Get/set user state
    userState, _ := msg.State().Get("user:" + msg.UserID())
    userState.MessageCount++
    msg.State().Set("user:" + msg.UserID(), userState)

    // Get conversation history
    history := msg.History()

    // Get shared state (across agents)
    sharedData, _ := msg.SharedState().Get("global:config")
}
```

### 5.3 Tool/Capability Integration

**Current State: No Standard Pattern**

**Pattern 1: Direct Function Calls**
```go
func (p *processor) ProcessMessage(...) {
    text := extractText(message)

    if strings.Contains(text, "calculate") {
        result := calculator.Evaluate(extractExpression(text))
        return &MessageProcessingResult{Result: makeMessage(result)}, nil
    }

    if strings.Contains(text, "weather") {
        weather := weatherAPI.Get(extractLocation(text))
        return &MessageProcessingResult{Result: makeMessage(weather)}, nil
    }
}
```

**Pattern 2: LLM Tool Calling**
```go
// Using LangChain tools
tools := []llms.Tool{
    calculatorTool,
    weatherTool,
}

response, _ := llm.CallWithTools(ctx, prompt, tools)
```

**Pattern 3: Subagent Delegation**
```go
// Forward to specialized agent
if needsCalculation {
    result, _ := calculatorAgent.SendMessage(ctx, message)
}
```

**Pain Points:**
- No standard tool interface
- No automatic tool discovery
- Manual tool selection/routing
- No parameter validation
- No tool result formatting
- Can't share tools across agents

**Desired Pattern:**
```go
// Register tools
agent.WithTool(tools.Calculator())
agent.WithTool(tools.Weather())
agent.WithTool(tools.WebSearch())

// Automatic tool calling via LLM
agent.WithLLM(llm.OpenAI()).
    WithTools(tools.All()).  // LLM can call any tool
    OnMessage(func(ctx context.Context, msg *adk.Message) error {
        // LLM automatically decides which tools to use
        response, _ := msg.LLM().Generate(ctx, msg.Text())
        return msg.Reply(response)
    })

// Manual tool calling
func handleMessage(ctx context.Context, msg *adk.Message) error {
    result, _ := msg.CallTool("calculator", map[string]any{
        "expression": "2 + 2",
    })
    return msg.Reply(result)
}
```

---

## 6. Recommendations for SAGE ADK

### 6.1 High-Priority Improvements

**1. Fluent Builder API**

Design a clean, discoverable API:
```go
agent := adk.NewAgent("my-agent").
    WithDescription("Agent that does X").
    WithLLM(llm.OpenAI()).
    WithProtocol(adk.ProtocolAuto).
    WithStorage(storage.Redis(client)).
    WithTool(tools.Calculator()).
    OnMessage(handleMessage).
    Build()
```

Benefits:
- Minimal boilerplate
- Self-documenting
- Type-safe
- Easy to test
- Familiar pattern (similar to http.Server, grpc.Server, etc.)

**2. Simplified Message Handling**

Provide high-level abstractions:
```go
// Simple handler
func handleMessage(ctx context.Context, msg *adk.Message) error {
    // Auto-extraction
    text := msg.Text()

    // Auto-LLM integration
    response, _ := msg.LLM().Generate(ctx, text)

    // Auto-response construction
    return msg.Reply(response)
}

// Error handling
if err != nil {
    return err  // Automatically converted to error message
}

// Streaming
return msg.Stream(func(s *adk.Stream) error {
    for chunk := range chunks {
        s.Send(chunk)
    }
    return nil
})
```

**3. Automatic Protocol Handling**

Hide protocol complexity:
```go
// A2A mode: no security overhead
agent.WithProtocol(adk.ProtocolA2A)

// SAGE mode: automatic handshake + signing
agent.WithProtocol(adk.ProtocolSAGE).
    WithSAGE(sage.FromEnv())

// Auto mode: detect from incoming messages
agent.WithProtocol(adk.ProtocolAuto).
    WithSAGE(sage.Optional())
```

SAGE integration should be transparent:
- Automatic handshake handling
- Automatic message signing/verification
- Automatic DID resolution (with caching)
- Automatic session management

**4. LLM Provider Abstraction**

Unified interface for all LLM providers:
```go
// Provider selection
agent.WithLLM(llm.OpenAI())
agent.WithLLM(llm.Anthropic())
agent.WithLLM(llm.Gemini())
agent.WithLLM(llm.FromEnv())  // Auto-detect

// Usage (same interface for all providers)
response, _ := msg.LLM().Generate(ctx, prompt)

// Streaming (same for all)
msg.LLM().GenerateStream(ctx, prompt, func(chunk string) {
    msg.SendChunk(chunk)
})

// With history
response, _ := msg.LLM().GenerateWithHistory(ctx, prompt, msg.History())

// With tools
response, _ := msg.LLM().GenerateWithTools(ctx, prompt, msg.Tools())
```

**5. Automatic Task Management**

Remove manual task lifecycle management:
```go
// Simple response (task created/cleaned automatically)
return msg.Reply("response")

// Streaming (task managed automatically)
return msg.Stream(func(s *adk.Stream) error {
    for chunk := range processInChunks() {
        s.Send(chunk)
    }
    return nil
})

// Long-running task (automatic state tracking)
return msg.Task(func(task *adk.Task) error {
    task.UpdateStatus("Starting processing...")

    result := longRunningOperation()

    task.UpdateStatus("Completed")
    task.SetResult(result)
    return nil
})
```

### 6.2 Medium-Priority Improvements

**6. State Management Abstraction**
```go
agent.WithState(state.Redis(client))

// In handler
userState := msg.State().Get("user:" + msg.UserID())
msg.State().Set("key", value, state.WithTTL(time.Hour))
```

**7. Tool/Capability Registry**
```go
agent.WithTool(tools.Calculator())
agent.WithTool(tools.Weather())

// Tools automatically available to LLM
// Or manually callable
result := msg.CallTool("calculator", params)
```

**8. Multi-Agent Orchestration**
```go
// Discover subagents
creative := agent.SubAgent("creative")

// Parallel execution
results := agent.ParallelSend(ctx, message,
    agent1, agent2, agent3)

// Conditional routing
agent.Route(func(msg *adk.Message) string {
    return selectAgent(msg)
})
```

**9. Resilience Patterns**
```go
agent.WithRetry(retry.Exponential(3, time.Second))
agent.WithTimeout(30 * time.Second)
agent.WithCircuitBreaker(breaker.Default())
agent.WithRateLimit(100, time.Minute)
```

**10. Observability**
```go
agent.WithMetrics(metrics.Prometheus())
agent.WithLogging(logging.JSON())
agent.WithTracing(tracing.Jaeger())

// Automatic metrics:
// - message_count
// - message_duration
// - llm_calls
// - llm_tokens
// - error_rate
// - task_states
```

### 6.3 Developer Experience Principles

**1. Progressive Disclosure**

Start simple, add complexity as needed:
```go
// Level 1: Minimal (for learning)
agent := adk.NewAgent("echo").
    OnMessage(func(ctx, msg) error {
        return msg.Reply(msg.Text())
    }).
    Build()

// Level 2: Add LLM
agent := adk.NewAgent("chat").
    WithLLM(llm.OpenAI()).
    OnMessage(func(ctx, msg) error {
        response, _ := msg.LLM().Generate(ctx, msg.Text())
        return msg.Reply(response)
    }).
    Build()

// Level 3: Add SAGE security
agent := adk.NewAgent("secure").
    WithProtocol(adk.ProtocolSAGE).
    WithSAGE(sage.FromEnv()).
    WithLLM(llm.OpenAI()).
    OnMessage(handleMessage).
    Build()

// Level 4: Production features
agent := adk.NewAgent("production").
    WithProtocol(adk.ProtocolAuto).
    WithSAGE(sage.Optional()).
    WithLLM(llm.OpenAI()).
    WithStorage(storage.Redis(client)).
    WithMetrics(metrics.Prometheus()).
    WithRetry(retry.Exponential(3, time.Second)).
    OnMessage(handleMessage).
    Build()
```

**2. Sensible Defaults**

Minimize configuration:
```go
// These are identical:
agent := adk.NewAgent("simple").OnMessage(handler).Build()

agent := adk.NewAgent("explicit").
    WithProtocol(adk.ProtocolA2A).  // Default
    WithStorage(storage.Memory()).   // Default
    WithLogging(logging.Text()).     // Default
    WithPort(8080).                  // Default
    OnMessage(handler).
    Build()
```

**3. Type Safety**

Use Go's type system to prevent errors:
```go
// Wrong type: compile error
agent.WithLLM("openai")  // Error: expected llm.Provider

// Correct
agent.WithLLM(llm.OpenAI())

// Invalid configuration: caught at Build()
agent := adk.NewAgent("test").
    WithProtocol(adk.ProtocolSAGE).
    Build()  // Error: SAGE protocol requires WithSAGE()
```

**4. Clear Error Messages**

Help developers fix problems:
```go
// Bad error:
// Error: invalid configuration

// Good error:
// Error: SAGE protocol requires a DID.
// Either:
//   1. Set SAGE_DID environment variable, or
//   2. Call agent.WithSAGE(sage.Options{DID: "..."})
// For development, you can use:
//   agent.WithSAGE(sage.Development())
```

**5. Comprehensive Examples**

Provide examples for every use case:
```
examples/
├── 01-simple-agent/          # Echo agent (5 lines)
├── 02-llm-agent/             # OpenAI agent (10 lines)
├── 03-sage-agent/            # SAGE-secured agent
├── 04-streaming-agent/       # Streaming responses
├── 05-multi-agent/           # Orchestration
├── 06-tool-agent/            # Tool calling
├── 07-production-agent/      # Full production setup
└── 08-migration-guide/       # From sage-a2a-go to ADK
```

---

## 7. Specific Code Examples for ADK

### 7.1 Simple Agent (Target: 5-10 lines)

**Goal:** Make "Hello World" trivial

```go
package main

import "github.com/sage-x-project/sage-adk/adk"

func main() {
    adk.NewAgent("echo").
        OnMessage(func(ctx, msg) error {
            return msg.Reply("You said: " + msg.Text())
        }).
        Start(":8080")
}
```

Compare to current sage-a2a-go: **100+ lines**

### 7.2 LLM Agent (Target: 10-15 lines)

**Goal:** LLM integration should be trivial

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
)

func main() {
    adk.NewAgent("chat").
        WithLLM(llm.OpenAI()).
        OnMessage(func(ctx, msg) error {
            response, _ := msg.LLM().Generate(ctx, msg.Text())
            return msg.Reply(response)
        }).
        Start(":8080")
}
```

With `.env`:
```
OPENAI_API_KEY=sk-...
```

### 7.3 SAGE-Secured Agent (Target: 15-20 lines)

**Goal:** Security should be opt-in and simple

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    adk.NewAgent("secure-chat").
        WithProtocol(adk.ProtocolSAGE).
        WithSAGE(sage.FromEnv()).
        WithLLM(llm.OpenAI()).
        OnMessage(func(ctx, msg) error {
            // Message already verified via SAGE
            response, _ := msg.LLM().Generate(ctx, msg.Text())
            return msg.Reply(response)
        }).
        Start(":8080")
}
```

With `.env`:
```
OPENAI_API_KEY=sk-...
SAGE_DID=did:sage:ethereum:0x...
SAGE_PRIVATE_KEY=0x...
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/...
```

### 7.4 Streaming Agent (Target: 15-20 lines)

**Goal:** Streaming should be as easy as non-streaming

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
)

func main() {
    adk.NewAgent("streaming-chat").
        WithLLM(llm.OpenAI()).
        OnMessage(func(ctx, msg) error {
            return msg.LLM().GenerateStream(ctx, msg.Text(),
                func(chunk string) error {
                    return msg.SendChunk(chunk)
                })
        }).
        Start(":8080")
}
```

Compare to current: **~50 lines** of task management code

### 7.5 Multi-Agent Orchestrator (Target: 25-30 lines)

**Goal:** Orchestration should be declarative

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
)

func main() {
    adk.NewAgent("orchestrator").
        WithLLM(llm.OpenAI()).
        OnMessage(func(ctx, msg) error {
            // LLM-based routing
            agent := routeToAgent(msg)

            // Forward to subagent
            result, err := msg.Forward(ctx, agent)
            if err != nil {
                return err
            }

            return msg.Reply(result)
        }).
        Start(":8080")
}

func routeToAgent(msg *adk.Message) string {
    // Could use LLM for routing
    if containsKeyword(msg, "creative") {
        return "creative-agent"
    }
    return "general-agent"
}
```

Compare to current: **~150 lines** with manual client management

### 7.6 Production Agent (Target: 30-40 lines)

**Goal:** Production features should be composable

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
    "github.com/sage-x-project/sage-adk/adapters/sage"
    "github.com/sage-x-project/sage-adk/storage"
    "github.com/sage-x-project/sage-adk/resilience"
    "github.com/sage-x-project/sage-adk/observability"
)

func main() {
    agent := adk.NewAgent("production").
        // Protocol
        WithProtocol(adk.ProtocolAuto).
        WithSAGE(sage.Optional()).

        // LLM
        WithLLM(llm.FromEnv()).

        // Storage
        WithStorage(storage.Redis(redisClient)).

        // Resilience
        WithRetry(resilience.Exponential(3, time.Second)).
        WithTimeout(30 * time.Second).
        WithCircuitBreaker(resilience.DefaultBreaker()).

        // Observability
        WithMetrics(observability.Prometheus()).
        WithLogging(observability.JSON()).
        WithTracing(observability.Jaeger()).

        // Health checks
        WithHealthCheck(healthcheck.Default()).

        // Message handler
        OnMessage(handleMessage).
        Build()

    // Graceful shutdown
    agent.StartWithShutdown(":8080")
}

func handleMessage(ctx context.Context, msg *adk.Message) error {
    response, err := msg.LLM().Generate(ctx, msg.Text())
    if err != nil {
        return err
    }
    return msg.Reply(response)
}
```

---

## 8. Implementation Priorities

### Phase 1: Foundation (Weeks 1-2)
1. Core type system (Message, Task, AgentCard)
2. Error handling framework
3. Configuration management
4. Builder pattern API

### Phase 2: A2A Integration (Weeks 3-4)
5. A2A protocol adapter (wraps sage-a2a-go)
6. TaskManager abstraction
7. Message handler interface
8. HTTP server implementation

### Phase 3: LLM Integration (Weeks 5-6)
9. LLM provider interface
10. OpenAI adapter
11. Anthropic adapter
12. Gemini adapter
13. Streaming support

### Phase 4: SAGE Integration (Weeks 7-8)
14. SAGE protocol adapter (wraps sage)
15. Handshake automation
16. DID management
17. Signature handling
18. Protocol auto-detection

### Phase 5: Production Features (Weeks 9-10)
19. State management (Redis, PostgreSQL)
20. Resilience patterns (retry, timeout, circuit breaker)
21. Observability (metrics, logging, tracing)
22. Health checks

### Phase 6: Advanced Features (Weeks 11-12)
23. Tool/capability registry
24. Multi-agent orchestration
25. Subagent discovery
26. Advanced routing

---

## 9. Key Takeaways

### What Works Well (Keep)
1. **A2A Protocol Design**: Well-thought-out message structure, task lifecycle
2. **SAGE Security Model**: Strong cryptographic guarantees, blockchain identity
3. **Separation of Concerns**: Protocol vs. business logic
4. **Streaming Support**: Essential for LLM applications
5. **Extensibility**: Plugins, custom storage, etc.

### What Needs Improvement (Fix)
1. **Developer Experience**: Too much boilerplate
2. **LLM Integration**: No standard pattern
3. **SAGE Complexity**: 4-phase handshake is manual
4. **Task Management**: Manual lifecycle is error-prone
5. **Configuration**: Too many environment variables
6. **Multi-Agent**: No orchestration helpers

### Core Philosophy for SAGE ADK
1. **Simplicity First**: Default to easy, allow advanced
2. **Progressive Disclosure**: Start simple, add complexity as needed
3. **Convention over Configuration**: Sensible defaults
4. **Type Safety**: Catch errors at compile time
5. **Developer Joy**: Make common tasks trivial

### Success Metrics
- **Simple agent**: 5-10 lines (vs. current 100+)
- **LLM agent**: 10-15 lines
- **SAGE agent**: 15-20 lines (vs. current manual implementation)
- **Production agent**: 30-40 lines (vs. current 150+)
- **Time to first agent**: 5 minutes (vs. current 30+ minutes)

---

## Appendix A: Pain Point Severity Matrix

| Pain Point | Severity | Impact | Effort | Priority |
|-----------|----------|--------|---------|----------|
| High boilerplate | Critical | High | Medium | P0 |
| No LLM integration | Critical | High | Medium | P0 |
| Manual task management | High | High | Medium | P1 |
| Streaming complexity | High | Medium | Medium | P1 |
| SAGE handshake manual | High | High | High | P1 |
| DID management complex | High | Medium | High | P2 |
| No state management | Medium | Medium | Low | P2 |
| No tool registry | Medium | Low | Medium | P3 |
| No orchestration helpers | Medium | Low | Medium | P3 |
| Verbose AgentCard | Low | Low | Low | P3 |

## Appendix B: Developer Personas

**Persona 1: Beginner Developer**
- Goal: Build first AI agent quickly
- Needs: Simple API, clear examples, minimal config
- Pain Points: Overwhelmed by complexity, doesn't understand protocols
- Success: "Hello World" agent in 5 minutes

**Persona 2: Experienced Developer (No AI Background)**
- Goal: Integrate AI into existing application
- Needs: Clear LLM integration, good docs, production features
- Pain Points: Too much boilerplate, unclear how to integrate
- Success: Production agent with OpenAI in 1 hour

**Persona 3: AI/ML Engineer**
- Goal: Build sophisticated multi-agent system
- Needs: Advanced features, orchestration, tool calling
- Pain Points: No standard patterns, manual wiring
- Success: Multi-agent orchestrator with custom tools in 1 day

**Persona 4: Security-Conscious Enterprise Developer**
- Goal: Build agent with blockchain identity and E2E encryption
- Needs: SAGE integration, compliance, audit trails
- Pain Points: Complex handshake, DID management, signatures
- Success: SAGE-secured agent in production in 1 week

---

**End of Analysis**
