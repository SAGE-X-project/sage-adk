# SAGE-ADK Architecture Refactoring Analysis

**Document Version:** 1.0
**Date:** 2025-10-13
**Status:** Analysis Complete - Implementation Pending

---

## Executive Summary

This document provides a comprehensive analysis of the architectural refactoring required for the `sage-adk` project following the restructuring of the `sage` project. The refactoring addresses dependency management, code duplication, and clarifies the boundaries between three interconnected projects: `sage`, `sage-a2a-go`, and `sage-adk`.

### Key Findings

1. **sage project successfully removed A2A dependency** - Go version restored to 1.23.0, transport layer abstracted
2. **Old A2A implementation remains in sage** - `pkg/agent/transport/a2a/` should be removed and moved to `sage-a2a-go`
3. **sage-adk contains 1000+ lines of duplicate code** - Reimplements functionality already in `sage` library
4. **Project role confusion** - `sage-a2a-go` needs clearer definition as a bridge project
5. **Go version mismatch** - `sage-adk` uses Go 1.24.4 while `sage` uses Go 1.23.0

### Refactoring Objectives

- **Eliminate code duplication** in `sage-adk`
- **Remove A2A code from sage** and move to `sage-a2a-go`
- **Define clear project boundaries** with proper dependency flow
- **Implement TDD approach** for stable, bug-free, extensible code
- **Align Go versions** across all projects

---

## 1. Project Role Clarification

### 1.1 Corrected Architecture

The proper dependency and responsibility flow:

```
┌─────────────────────────────────────────────────────────────┐
│                         sage-adk                             │
│  (AI Agent Development Kit - Integration Layer)              │
│                                                               │
│  - High-level APIs for AI agent development                  │
│  - Agent lifecycle management                                │
│  - Tool integration                                           │
│  - Uses sage + sage-a2a-go libraries                         │
└───────────────────┬─────────────────┬───────────────────────┘
                    │                 │
                    │                 │
        ┌───────────▼─────────┐   ┌───▼────────────────────┐
        │    sage-a2a-go      │   │       sage             │
        │  (Bridge Project)   │   │  (Security Primitives) │
        │                     │   │                        │
        │  Implements sage's  │◄──│  - RFC 9421 Signatures │
        │  transport interface│   │  - RFC 9180 Encryption │
        │  using A2A protocol │   │  - Handshake Protocol  │
        │                     │   │  - Interface Only      │
        │  - Secure mode      │   │  - No A2A dependency   │
        │    (with sage)      │   └────────────────────────┘
        │  - Plain mode       │
        │    (without sage)   │
        └──────────┬──────────┘
                   │
                   │
        ┌──────────▼──────────┐
        │         A2A         │
        │  (Agent Protocol)   │
        │                     │
        │  - gRPC/Protobuf    │
        │  - Message routing  │
        │  - Protocol core    │
        └─────────────────────┘
```

### 1.2 Project Responsibilities

#### sage (Security Primitives + Interfaces)

**Purpose:** Provide security layer for agent communication, independent of transport protocol

**Responsibilities:**
- RFC 9421: HTTP Message Signatures for message integrity
- RFC 9180: HPKE encryption for message confidentiality
- 4-phase handshake protocol (Invitation → Request → Response → Complete)
- Session management with AEAD encryption
- DID-based authentication
- **Transport interface definition** (NOT implementation)

**What it should NOT have:**
- ❌ A2A-specific code
- ❌ Concrete transport implementations (except HTTP/WebSocket for testing)
- ❌ A2A dependencies in go.mod

**Current Status:**
- ✅ Go 1.23.0 (correct version)
- ✅ No a2a-go dependency in go.mod
- ✅ Transport interface defined at `pkg/agent/transport/interface.go`
- ❌ Old A2A implementation at `pkg/agent/transport/a2a/client.go` (NEEDS REMOVAL)

#### sage-a2a-go (Bridge: sage ↔ A2A)

**Purpose:** Connect sage security layer with A2A protocol

**Responsibilities:**
- Implement `transport.MessageTransport` interface using A2A/gRPC
- Convert sage `SecureMessage` ↔ A2A protobuf messages
- Provide high-level secure A2A client (sage security + A2A transport)
- Support both secure mode (with sage) and plain mode (without sage)

**Project Structure:**
```
sage-a2a-go/
├── adapter/           # Implements sage's MessageTransport
│   ├── client.go      # A2ATransport struct
│   └── converter.go   # SecureMessage ↔ A2A protobuf
├── secure/            # High-level API with sage security
│   ├── client.go      # SecureA2AClient
│   └── session.go     # Session management integration
├── plain/             # Optional: A2A without sage
│   └── client.go      # Plain A2A client
└── examples/
    ├── secure_chat.go
    └── plain_chat.go
```

**Current Status:**
- ⚠️ Project needs to be created
- ⚠️ A2A implementation currently in sage (needs to be moved here)

#### sage-adk (AI Agent Development Kit)

**Purpose:** Simplify AI agent development by providing high-level integration

**Responsibilities:**
- Unified API for creating secure AI agents
- Agent lifecycle management
- Tool/capability integration
- Message routing and context management
- Examples and documentation
- **Use sage and sage-a2a-go as libraries** (not reimplement)

**What it should NOT have:**
- ❌ Duplicate implementations of sage functionality
- ❌ Direct A2A protocol handling (use sage-a2a-go)
- ❌ Custom transport implementations (use sage's)

**Current Status:**
- ❌ Go 1.24.4 (should be 1.23.0)
- ❌ Duplicate code in `adapters/sage/` (~1000+ lines)
- ❌ Reimplements TransportManager, HandshakeManager, etc.
- ✅ Has sage-a2a-go dependency

---

## 2. Current State Analysis

### 2.1 sage Project

#### File: `sage/go.mod`

**Status:** ✅ Correct

```go
module github.com/sage-x-project/sage
go 1.23.0

require (
    github.com/ethereum/go-ethereum v1.14.12
    golang.org/x/crypto v0.31.0
    // NO A2A dependencies - successfully removed!
)
```

**Analysis:**
- Successfully removed a2a-go dependency
- Go version set to 1.23.0 (appropriate for project)
- No direct protocol dependencies

#### File: `sage/pkg/agent/transport/interface.go`

**Status:** ✅ Excellent design - Keep as-is

```go
// MessageTransport is the transport layer abstraction interface.
type MessageTransport interface {
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}

type SecureMessage struct {
    ID        string // Unique message ID (UUID)
    ContextID string // Conversation context ID
    TaskID    string // Task identifier
    Payload   []byte // Encrypted message content
    DID       string // Sender DID (did:sage:ethereum:...)
    Signature []byte // RFC 9421 signature or DID signature
    Metadata  map[string]string
    Role      string // "user" or "agent"
}

type Response struct {
    Success   bool
    MessageID string
    TaskID    string
    Data      []byte
    Error     error
}
```

**Analysis:**
- Clean interface following dependency inversion principle
- Protocol-agnostic design
- Contains all necessary security metadata
- Allows multiple transport implementations

**Location:** `sage/pkg/agent/transport/interface.go:28-98`

#### File: `sage/pkg/agent/transport/mock.go`

**Status:** ✅ Keep - Essential for testing

```go
type MockTransport struct {
    SendFunc     func(ctx context.Context, msg *SecureMessage) (*Response, error)
    SentMessages []*SecureMessage
    mu           sync.Mutex
}

func (m *MockTransport) Send(ctx context.Context, msg *SecureMessage) (*Response, error) {
    m.mu.Lock()
    m.SentMessages = append(m.SentMessages, msg)
    m.mu.Unlock()

    if m.SendFunc != nil {
        return m.SendFunc(ctx, msg)
    }

    return &Response{
        Success:   true,
        MessageID: msg.ID,
        TaskID:    msg.TaskID,
        Data:      []byte("mock response"),
    }, nil
}
```

**Analysis:**
- Excellent testing infrastructure
- Thread-safe implementation
- Allows test behavior injection
- Essential for TDD approach

**Location:** `sage/pkg/agent/transport/mock.go:41-72`

#### File: `sage/pkg/agent/transport/http/client.go`

**Status:** ✅ Keep - Valid transport implementation

```go
type HTTPTransport struct {
    baseURL    string
    httpClient *http.Client
}

func NewHTTPTransport(baseURL string) *HTTPTransport {
    return &HTTPTransport{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (t *HTTPTransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    wireMsg := toWireMessage(msg)
    jsonData, err := json.Marshal(wireMsg)
    // ... HTTP POST implementation
}
```

**Analysis:**
- Non-A2A transport implementation
- No external protocol dependencies
- Good example of interface implementation
- Should remain in sage

**Location:** `sage/pkg/agent/transport/http/client.go:45-164`

#### File: `sage/pkg/agent/transport/a2a/client.go`

**Status:** ❌ REMOVE - Move to sage-a2a-go

```go
//go:build a2a
// +build a2a

package a2a

import (
    a2apb "github.com/a2aproject/a2a/grpc"
    "google.golang.org/grpc"
    "github.com/sage-x-project/sage/pkg/agent/transport"
)

type A2ATransport struct {
    client a2apb.A2AServiceClient
}

func NewA2ATransport(conn grpc.ClientConnInterface) *A2ATransport {
    return &A2ATransport{
        client: a2apb.NewA2AServiceClient(conn),
    }
}

func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    // Convert SecureMessage → A2A protobuf
    // ... implementation
}
```

**Issues:**
1. ❌ A2A implementation belongs in sage-a2a-go, not sage
2. ❌ Violates project separation goals
3. ❌ Old code from before refactoring

**Action Required:**
- DELETE entire `sage/pkg/agent/transport/a2a/` directory
- Move this code to `sage-a2a-go/adapter/`
- Update package paths and dependencies

**Location:** `sage/pkg/agent/transport/a2a/client.go:22-162`

#### File: `sage/pkg/agent/handshake/client.go`

**Status:** ✅ Perfect - Shows correct usage pattern

```go
type Client struct {
    transport transport.MessageTransport  // ← Interface, not concrete type
    key       sagecrypto.KeyPair
}

func NewClient(t transport.MessageTransport, key sagecrypto.KeyPair) *Client {
    return &Client{
        transport: t,
        key:       key,
    }
}

func (c *Client) Invitation(ctx context.Context, msg *InvitationMessage, recipientDID string) (*InvitationResponse, error) {
    // Prepare secure message
    secureMsg := &transport.SecureMessage{
        ID:        uuid.New().String(),
        Payload:   encryptedPayload,
        DID:       c.key.DID(),
        Signature: signature,
        Role:      "user",
    }

    // Send via transport interface
    resp, err := c.transport.Send(ctx, secureMsg)
    // ...
}
```

**Analysis:**
- ✅ Depends on interface, not concrete implementation
- ✅ Follows dependency inversion principle
- ✅ Allows any transport to be injected
- ✅ Clean separation of concerns

**Key Pattern:**
```go
// Client accepts interface
func NewClient(transport transport.MessageTransport, key crypto.KeyPair)

// Not concrete type
func NewClient(transport *a2a.A2ATransport, key crypto.KeyPair) // ❌ Wrong
```

### 2.2 sage-adk Project

#### File: `sage-adk/go.mod`

**Status:** ⚠️ Needs updates

```go
module github.com/sage-x-project/sage-adk
go 1.24.4  // ← Should be 1.23.0

require (
    github.com/sage-x-project/sage v2.0.0
    trpc.group/trpc-go/trpc-a2a-go v0.0.0  // This is sage-a2a-go
    // ... other dependencies
)
```

**Issues:**
1. ❌ Go version mismatch (1.24.4 vs 1.23.0)
2. ⚠️ Dependency on sage-a2a-go exists but project not yet created

**Action Required:**
- Change `go 1.24.4` → `go 1.23.0`
- Wait for sage-a2a-go to be created
- Update import paths once sage-a2a-go is ready

#### File: `sage-adk/adapters/sage/transport.go` (520 lines)

**Status:** ❌ DELETE - Duplicate implementation

**Current Content (problematic):**
```go
package sage

type TransportManager struct {
    handshakeManager  *HandshakeManager
    sessionManager    *SessionManager
    encryptionManager *EncryptionManager
    signatureManager  *SignatureManager
}

func (tm *TransportManager) SendSecureMessage(ctx context.Context, msg *Message) error {
    // Reimplements sage functionality!
    encrypted := tm.encryptionManager.Encrypt(msg.Payload)
    signature := tm.signatureManager.Sign(encrypted)
    // ... 500 more lines
}
```

**Issues:**
1. ❌ Duplicates `sage/pkg/agent/handshake/`
2. ❌ Reimplements encryption logic (already in sage)
3. ❌ Reimplements session management (already in sage)
4. ❌ 520 lines of unnecessary code

**Correct Approach:**
```go
package adapters

import (
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage-a2a-go/secure"
)

type AgentAdapter struct {
    secureClient *secure.SecureA2AClient  // Use library
}

func NewAgentAdapter(did string, key crypto.PrivateKey, grpcConn *grpc.ClientConn) *AgentAdapter {
    return &AgentAdapter{
        secureClient: secure.NewSecureA2AClient(did, key, grpcConn),
    }
}

func (a *AgentAdapter) SendMessage(ctx context.Context, msg string) error {
    return a.secureClient.SendSecure(ctx, msg)  // Use library method
}
```

**Action Required:**
- DELETE `sage-adk/adapters/sage/transport.go`
- Create thin adapter using sage + sage-a2a-go libraries
- Remove all duplicate security implementations

#### File: `sage-adk/adapters/sage/handshake.go` (543 lines)

**Status:** ❌ DELETE - Duplicate implementation

**Current Content (problematic):**
```go
package sage

type HandshakeManager struct {
    keyPair      KeyPair
    didRegistry  *DIDRegistry
    cryptoEngine *CryptoEngine
}

func (hm *HandshakeManager) PerformHandshake(ctx context.Context, recipientDID string) (*Session, error) {
    // Phase 1: Invitation
    inv := hm.createInvitation()
    // Phase 2: Request
    req := hm.createRequest()
    // ... 500 more lines duplicating sage/pkg/agent/handshake/
}
```

**Issues:**
1. ❌ Completely duplicates `sage/pkg/agent/handshake/client.go`
2. ❌ Reimplements 4-phase handshake protocol
3. ❌ Maintenance burden (must keep in sync with sage)
4. ❌ 543 lines of unnecessary code

**Correct Approach:**
```go
package adapters

import (
    "github.com/sage-x-project/sage/pkg/agent/handshake"
    sagecrypto "github.com/sage-x-project/sage/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/adapter"
)

type HandshakeAdapter struct {
    client *handshake.Client  // Use sage library
}

func NewHandshakeAdapter(key sagecrypto.KeyPair, grpcConn *grpc.ClientConn) *HandshakeAdapter {
    transport := adapter.NewA2ATransport(grpcConn)
    client := handshake.NewClient(transport, key)

    return &HandshakeAdapter{
        client: client,
    }
}

func (h *HandshakeAdapter) EstablishSession(ctx context.Context, recipientDID string) (*Session, error) {
    // Use sage's handshake client, just wrap it
    return h.client.PerformHandshake(ctx, recipientDID)
}
```

**Action Required:**
- DELETE `sage-adk/adapters/sage/handshake.go`
- Create thin wrapper using `sage/pkg/agent/handshake`
- Add sage-adk-specific convenience methods only

---

## 3. Detailed Refactoring Plan

### 3.1 Phase 1: Clean Up sage (Week 1)

#### Task 1.1: Remove A2A Implementation from sage

**Files to Delete:**
```bash
rm -rf sage/pkg/agent/transport/a2a/
```

**Files to Verify:**
- Check no imports of `sage/pkg/agent/transport/a2a` remain
- Verify build tags (`//go:build a2a`) are not needed elsewhere
- Update documentation to remove A2A examples

**Testing:**
```bash
cd sage
go build ./...  # Should succeed without a2a build tag
go test ./...   # All tests should pass
```

**Branch:** `refactor/remove-a2a-implementation`
**Commit:** `refactor: remove A2A transport implementation from sage`

**Expected Result:**
- sage has NO A2A-specific code
- Only HTTP, WebSocket, and Mock transports remain
- All tests pass

#### Task 1.2: Verify sage Interface Completeness

**Check:**
1. `transport.MessageTransport` interface has all needed methods
2. `transport.SecureMessage` has all required fields
3. `transport.Response` is sufficient for all transports

**Add if missing:**
- Error codes/categories in Response
- Transport metadata fields
- Retry/timeout configuration

**Documentation:**
```go
// File: pkg/agent/transport/interface.go

// MessageTransport is the transport layer abstraction interface.
//
// Implementations:
//   - HTTP: sage/pkg/agent/transport/http (included)
//   - WebSocket: sage/pkg/agent/transport/websocket (included)
//   - A2A: sage-a2a-go/adapter (external package)
//   - Mock: sage/pkg/agent/transport/mock (testing)
//
// Example:
//   // Using HTTP transport
//   transport := http.NewHTTPTransport("https://agent.example.com")
//   client := handshake.NewClient(transport, keyPair)
type MessageTransport interface {
    Send(ctx context.Context, msg *SecureMessage) (*Response, error)
}
```

### 3.2 Phase 2: Create sage-a2a-go Bridge (Week 2)

#### Task 2.1: Project Structure Setup

**Create new repository:** `github.com/sage-x-project/sage-a2a-go`

**Directory Structure:**
```
sage-a2a-go/
├── go.mod                      # Go 1.23.0
├── README.md
├── LICENSE                     # LGPL v3
├── adapter/                    # Implements sage's interface
│   ├── client.go              # A2ATransport struct
│   ├── client_test.go         # TDD tests
│   ├── converter.go           # SecureMessage ↔ A2A protobuf
│   └── converter_test.go
├── secure/                     # High-level secure API
│   ├── client.go              # SecureA2AClient (sage + A2A)
│   ├── client_test.go
│   ├── session.go             # Session lifecycle
│   └── session_test.go
├── plain/                      # Optional: A2A without sage
│   ├── client.go              # Plain A2A client
│   └── client_test.go
├── internal/
│   └── proto/                 # A2A protobuf helpers
└── examples/
    ├── secure_chat/           # Example with sage security
    │   └── main.go
    └── plain_chat/            # Example without sage
        └── main.go
```

**go.mod:**
```go
module github.com/sage-x-project/sage-a2a-go

go 1.23.0

require (
    github.com/sage-x-project/sage v2.0.0
    github.com/a2aproject/a2a v0.1.0  // A2A protocol
    google.golang.org/grpc v1.60.0
    google.golang.org/protobuf v1.32.0
)
```

#### Task 2.2: Implement adapter Package (TDD)

**Test First (adapter/client_test.go):**
```go
package adapter_test

import (
    "context"
    "testing"

    "github.com/sage-x-project/sage/pkg/agent/transport"
    "github.com/sage-x-project/sage-a2a-go/adapter"
    "google.golang.org/grpc"
)

func TestA2ATransport_ImplementsInterface(t *testing.T) {
    // Verify it implements transport.MessageTransport
    var _ transport.MessageTransport = (*adapter.A2ATransport)(nil)
}

func TestA2ATransport_Send_Success(t *testing.T) {
    // Setup mock gRPC connection
    mockConn := newMockGRPCConn()
    transport := adapter.NewA2ATransport(mockConn)

    // Create test message
    msg := &transport.SecureMessage{
        ID:        "test-123",
        ContextID: "ctx-456",
        TaskID:    "task-789",
        Payload:   []byte("encrypted data"),
        DID:       "did:sage:ethereum:0x123",
        Signature: []byte("signature"),
        Role:      "user",
    }

    // Send message
    resp, err := transport.Send(context.Background(), msg)

    // Assertions
    if err != nil {
        t.Fatalf("Send failed: %v", err)
    }
    if !resp.Success {
        t.Errorf("Expected success=true, got false")
    }
    if resp.MessageID != "test-123" {
        t.Errorf("MessageID mismatch: got %s", resp.MessageID)
    }
}

func TestA2ATransport_Send_NetworkError(t *testing.T) {
    // Test network failure handling
    mockConn := newMockGRPCConnWithError()
    transport := adapter.NewA2ATransport(mockConn)

    msg := &transport.SecureMessage{ID: "test"}

    resp, err := transport.Send(context.Background(), msg)

    if err == nil {
        t.Fatal("Expected error, got nil")
    }
    if resp.Success {
        t.Error("Expected success=false on error")
    }
}
```

**Implementation (adapter/client.go):**
```go
package adapter

import (
    "context"
    "fmt"

    a2apb "github.com/a2aproject/a2a/grpc"
    "google.golang.org/grpc"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// A2ATransport implements transport.MessageTransport using A2A protocol.
//
// This adapter bridges sage's transport interface with the A2A gRPC protocol,
// allowing sage security features to work over A2A communication channels.
type A2ATransport struct {
    client a2apb.A2AServiceClient
}

// NewA2ATransport creates a new A2A transport from a gRPC connection.
//
// Example:
//   conn, _ := grpc.Dial("agent.example.com:50051")
//   transport := adapter.NewA2ATransport(conn)
//   handshakeClient := handshake.NewClient(transport, keyPair)
func NewA2ATransport(conn grpc.ClientConnInterface) *A2ATransport {
    return &A2ATransport{
        client: a2apb.NewA2AServiceClient(conn),
    }
}

// Send implements transport.MessageTransport.Send
func (t *A2ATransport) Send(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
    if msg == nil {
        return nil, fmt.Errorf("nil message")
    }

    // Convert sage SecureMessage → A2A protobuf
    a2aMsg, metadata, err := secureMessageToA2A(msg)
    if err != nil {
        return nil, fmt.Errorf("convert to a2a: %w", err)
    }

    // Send via gRPC
    req := &a2apb.SendMessageRequest{
        Request:  a2aMsg,
        Metadata: metadata,
    }

    resp, err := t.client.SendMessage(ctx, req)
    if err != nil {
        return &transport.Response{
            Success:   false,
            MessageID: msg.ID,
            TaskID:    msg.TaskID,
            Error:     fmt.Errorf("grpc send: %w", err),
        }, err
    }

    // Convert A2A response → transport.Response
    return a2aResponseToTransport(resp, msg.ID, msg.TaskID)
}
```

**Converter (adapter/converter.go):**
```go
package adapter

import (
    "encoding/base64"
    "encoding/json"
    "fmt"

    a2apb "github.com/a2aproject/a2a/grpc"
    "google.golang.org/protobuf/types/known/structpb"

    "github.com/sage-x-project/sage/pkg/agent/transport"
)

// secureMessageToA2A converts transport.SecureMessage to A2A protobuf format
func secureMessageToA2A(msg *transport.SecureMessage) (*a2apb.Message, *structpb.Struct, error) {
    // Parse payload as JSON (if needed, or use raw bytes)
    var payloadMap map[string]interface{}
    if err := json.Unmarshal(msg.Payload, &payloadMap); err != nil {
        // If not JSON, wrap in a data field
        payloadMap = map[string]interface{}{
            "data": base64.StdEncoding.EncodeToString(msg.Payload),
        }
    }

    payloadStruct, err := structpb.NewStruct(payloadMap)
    if err != nil {
        return nil, nil, fmt.Errorf("convert payload: %w", err)
    }

    // Build metadata
    metadataMap := make(map[string]interface{})
    for k, v := range msg.Metadata {
        metadataMap[k] = v
    }
    metadataMap["did"] = msg.DID
    metadataMap["signature"] = base64.StdEncoding.EncodeToString(msg.Signature)

    metadata, err := structpb.NewStruct(metadataMap)
    if err != nil {
        return nil, nil, fmt.Errorf("convert metadata: %w", err)
    }

    // Determine role
    role := a2apb.Role_ROLE_USER
    if msg.Role == "agent" {
        role = a2apb.Role_ROLE_AGENT
    }

    // Create A2A message
    a2aMsg := &a2apb.Message{
        MessageId: msg.ID,
        ContextId: msg.ContextID,
        TaskId:    msg.TaskID,
        Role:      role,
        Content: []*a2apb.Part{
            {
                Part: &a2apb.Part_Data{
                    Data: &a2apb.DataPart{
                        Data: payloadStruct,
                    },
                },
            },
        },
    }

    return a2aMsg, metadata, nil
}

// a2aResponseToTransport converts A2A response to transport.Response
func a2aResponseToTransport(resp *a2apb.SendMessageResponse, msgID, taskID string) (*transport.Response, error) {
    if resp == nil {
        return nil, fmt.Errorf("nil a2a response")
    }

    var responseData []byte
    var responseMessageID, responseTaskID string

    if msg := resp.GetMsg(); msg != nil {
        responseMessageID = msg.MessageId
        responseTaskID = msg.TaskId

        // Extract data from content
        if len(msg.Content) > 0 {
            if dataPart := msg.Content[0].GetData(); dataPart != nil && dataPart.Data != nil {
                jsonBytes, err := json.Marshal(dataPart.Data.AsMap())
                if err == nil {
                    responseData = jsonBytes
                }
            }
        }
    }

    // Use provided IDs if response doesn't include them
    if responseMessageID == "" {
        responseMessageID = msgID
    }
    if responseTaskID == "" {
        responseTaskID = taskID
    }

    return &transport.Response{
        Success:   true,
        MessageID: responseMessageID,
        TaskID:    responseTaskID,
        Data:      responseData,
        Error:     nil,
    }, nil
}
```

#### Task 2.3: Implement secure Package (High-level API)

**Test First (secure/client_test.go):**
```go
package secure_test

import (
    "context"
    "testing"

    "github.com/sage-x-project/sage-a2a-go/secure"
    sagecrypto "github.com/sage-x-project/sage/pkg/crypto"
)

func TestSecureA2AClient_Creation(t *testing.T) {
    keyPair := sagecrypto.GenerateKeyPair()
    mockConn := newMockGRPCConn()

    client := secure.NewSecureA2AClient("did:sage:ethereum:0x123", keyPair, mockConn)

    if client == nil {
        t.Fatal("Expected non-nil client")
    }
}

func TestSecureA2AClient_EstablishSession(t *testing.T) {
    keyPair := sagecrypto.GenerateKeyPair()
    mockConn := newMockGRPCConn()
    client := secure.NewSecureA2AClient("did:sage:ethereum:0x123", keyPair, mockConn)

    recipientDID := "did:sage:ethereum:0x456"

    session, err := client.EstablishSession(context.Background(), recipientDID)

    if err != nil {
        t.Fatalf("EstablishSession failed: %v", err)
    }
    if session == nil {
        t.Fatal("Expected non-nil session")
    }
    if session.RecipientDID != recipientDID {
        t.Errorf("RecipientDID mismatch")
    }
}

func TestSecureA2AClient_SendSecureMessage(t *testing.T) {
    // Test sending encrypted message over A2A
    keyPair := sagecrypto.GenerateKeyPair()
    mockConn := newMockGRPCConn()
    client := secure.NewSecureA2AClient("did:sage:ethereum:0x123", keyPair, mockConn)

    // Establish session first
    session, _ := client.EstablishSession(context.Background(), "did:sage:ethereum:0x456")

    // Send message
    plaintext := "Hello, secure world!"
    resp, err := client.SendSecure(context.Background(), session, plaintext)

    if err != nil {
        t.Fatalf("SendSecure failed: %v", err)
    }
    if !resp.Success {
        t.Error("Expected success=true")
    }
}
```

**Implementation (secure/client.go):**
```go
package secure

import (
    "context"
    "fmt"

    "google.golang.org/grpc"

    "github.com/sage-x-project/sage/pkg/agent/handshake"
    "github.com/sage-x-project/sage/pkg/agent/session"
    sagecrypto "github.com/sage-x-project/sage/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/adapter"
)

// SecureA2AClient provides high-level API for secure A2A communication.
//
// This client combines sage's security features (handshake, encryption, signing)
// with A2A's transport protocol, providing a simple API for developers.
type SecureA2AClient struct {
    localDID         string
    keyPair          sagecrypto.KeyPair
    handshakeClient  *handshake.Client
    sessionManager   *session.Manager
    transport        *adapter.A2ATransport
}

// NewSecureA2AClient creates a new secure A2A client.
//
// Parameters:
//   - localDID: This agent's DID (e.g., "did:sage:ethereum:0x123")
//   - keyPair: Cryptographic key pair for signing and encryption
//   - grpcConn: gRPC connection to A2A server
//
// Example:
//   keyPair := crypto.GenerateKeyPair()
//   conn, _ := grpc.Dial("agent.example.com:50051")
//   client := secure.NewSecureA2AClient("did:sage:ethereum:0x123", keyPair, conn)
func NewSecureA2AClient(localDID string, keyPair sagecrypto.KeyPair, grpcConn *grpc.ClientConn) *SecureA2AClient {
    transport := adapter.NewA2ATransport(grpcConn)
    handshakeClient := handshake.NewClient(transport, keyPair)
    sessionManager := session.NewManager()

    return &SecureA2AClient{
        localDID:        localDID,
        keyPair:         keyPair,
        handshakeClient: handshakeClient,
        sessionManager:  sessionManager,
        transport:       transport,
    }
}

// EstablishSession performs 4-phase handshake with recipient.
//
// Returns a session that can be used for encrypted communication.
func (c *SecureA2AClient) EstablishSession(ctx context.Context, recipientDID string) (*session.Session, error) {
    // Perform 4-phase handshake
    sessionData, err := c.handshakeClient.PerformHandshake(ctx, recipientDID)
    if err != nil {
        return nil, fmt.Errorf("handshake failed: %w", err)
    }

    // Create session
    sess := c.sessionManager.CreateSession(recipientDID, sessionData)

    return sess, nil
}

// SendSecure encrypts and sends a message over an established session.
//
// The message is encrypted using the session key, signed with the local key,
// and sent via A2A transport.
func (c *SecureA2AClient) SendSecure(ctx context.Context, sess *session.Session, plaintext string) (*Response, error) {
    // Encrypt message using session key
    encrypted, err := sess.Encrypt([]byte(plaintext))
    if err != nil {
        return nil, fmt.Errorf("encryption failed: %w", err)
    }

    // Sign encrypted message
    signature, err := c.keyPair.Sign(encrypted)
    if err != nil {
        return nil, fmt.Errorf("signing failed: %w", err)
    }

    // Create secure message
    msg := &transport.SecureMessage{
        ID:        uuid.New().String(),
        ContextID: sess.ContextID,
        TaskID:    sess.TaskID,
        Payload:   encrypted,
        DID:       c.localDID,
        Signature: signature,
        Role:      "user",
    }

    // Send via A2A transport
    resp, err := c.transport.Send(ctx, msg)
    if err != nil {
        return nil, fmt.Errorf("send failed: %w", err)
    }

    return &Response{
        Success:   resp.Success,
        MessageID: resp.MessageID,
        Data:      resp.Data,
    }, nil
}
```

### 3.3 Phase 3: Refactor sage-adk (Week 3)

#### Task 3.1: Update Go Version

**File:** `sage-adk/go.mod`

**Change:**
```diff
  module github.com/sage-x-project/sage-adk
- go 1.24.4
+ go 1.23.0

  require (
      github.com/sage-x-project/sage v2.0.0
-     trpc.group/trpc-go/trpc-a2a-go v0.0.0
+     github.com/sage-x-project/sage-a2a-go v1.0.0
  )
```

**Test:**
```bash
cd sage-adk
go mod tidy
go build ./...
```

#### Task 3.2: Remove Duplicate Code

**Delete Files:**
```bash
rm sage-adk/adapters/sage/transport.go
rm sage-adk/adapters/sage/handshake.go
rm sage-adk/adapters/sage/encryption.go
rm sage-adk/adapters/sage/session.go
# ... any other files duplicating sage functionality
```

**Create Thin Adapter (adapters/sage/client.go):**
```go
package sage

import (
    "context"
    "fmt"

    "google.golang.org/grpc"

    sagecrypto "github.com/sage-x-project/sage/pkg/crypto"
    "github.com/sage-x-project/sage-a2a-go/secure"
)

// Client is sage-adk's thin wrapper around sage-a2a-go.
//
// This provides convenience methods specific to sage-adk's use cases
// while delegating all security and transport logic to the libraries.
type Client struct {
    secureClient *secure.SecureA2AClient
    did          string
}

// NewClient creates a new sage-adk client.
//
// Example:
//   keyPair := crypto.GenerateKeyPair()
//   conn, _ := grpc.Dial("agent.example.com:50051")
//   client := sage.NewClient("did:sage:ethereum:0x123", keyPair, conn)
func NewClient(did string, keyPair sagecrypto.KeyPair, grpcConn *grpc.ClientConn) *Client {
    secureClient := secure.NewSecureA2AClient(did, keyPair, grpcConn)

    return &Client{
        secureClient: secureClient,
        did:          did,
    }
}

// ConnectToAgent establishes a secure session with another agent.
func (c *Client) ConnectToAgent(ctx context.Context, recipientDID string) (*AgentConnection, error) {
    session, err := c.secureClient.EstablishSession(ctx, recipientDID)
    if err != nil {
        return nil, fmt.Errorf("connect failed: %w", err)
    }

    return &AgentConnection{
        client:       c.secureClient,
        session:      session,
        recipientDID: recipientDID,
    }, nil
}

// AgentConnection represents a secure connection to another agent.
type AgentConnection struct {
    client       *secure.SecureA2AClient
    session      *session.Session
    recipientDID string
}

// Send sends a message to the connected agent.
func (c *AgentConnection) Send(ctx context.Context, message string) error {
    _, err := c.client.SendSecure(ctx, c.session, message)
    return err
}
```

**Total LOC reduction:** ~1000+ lines removed (duplicate code)
**New code:** ~100 lines (thin adapter)

#### Task 3.3: Update Examples

**Before (sage-adk/examples/secure_agent/main.go):**
```go
// OLD: Using duplicate implementation
func main() {
    tm := sage.NewTransportManager(...)  // 520 lines of duplicate code
    hm := sage.NewHandshakeManager(...)  // 543 lines of duplicate code

    session := hm.PerformHandshake(...)
    tm.SendSecure(session, "Hello")
}
```

**After (sage-adk/examples/secure_agent/main.go):**
```go
// NEW: Using libraries
import (
    "github.com/sage-x-project/sage/pkg/crypto"
    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    // Generate or load key pair
    keyPair := crypto.GenerateKeyPair()

    // Connect to A2A server
    conn, err := grpc.Dial("agent.example.com:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    // Create client
    client := sage.NewClient("did:sage:ethereum:0x123", keyPair, conn)

    // Connect to agent
    agentConn, err := client.ConnectToAgent(context.Background(), "did:sage:ethereum:0x456")
    if err != nil {
        log.Fatal(err)
    }

    // Send secure message
    err = agentConn.Send(context.Background(), "Hello, secure world!")
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## 4. Testing Strategy (TDD Approach)

### 4.1 Test-First Development Cycle

**Red-Green-Refactor:**
1. **Red:** Write failing test that defines expected behavior
2. **Green:** Write minimal code to make test pass
3. **Refactor:** Improve code while keeping tests green

### 4.2 Test Coverage Goals

**sage-a2a-go/adapter:**
- Unit tests: 90%+ coverage
- Integration tests with mock gRPC
- Converter tests (SecureMessage ↔ A2A protobuf)

**sage-a2a-go/secure:**
- Unit tests: 85%+ coverage
- Integration tests with sage + mock A2A
- End-to-end handshake tests

**sage-adk:**
- Unit tests for adapters: 80%+ coverage
- Integration tests using real sage-a2a-go
- Example tests to verify documentation

### 4.3 Example Test Suite

**adapter/client_test.go:**
```go
func TestA2ATransport_InterfaceCompliance(t *testing.T) {
    // Verify interface implementation
}

func TestA2ATransport_Send_Success(t *testing.T) {
    // Happy path: successful message send
}

func TestA2ATransport_Send_NilMessage(t *testing.T) {
    // Edge case: nil message
}

func TestA2ATransport_Send_NetworkError(t *testing.T) {
    // Error case: network failure
}

func TestA2ATransport_Send_InvalidResponse(t *testing.T) {
    // Error case: malformed A2A response
}

func TestA2ATransport_Send_ContextCancellation(t *testing.T) {
    // Concurrency: context cancellation
}
```

**secure/client_test.go:**
```go
func TestSecureA2AClient_EstablishSession(t *testing.T) {
    // Test 4-phase handshake
}

func TestSecureA2AClient_SendSecure_Encryption(t *testing.T) {
    // Verify message is encrypted
}

func TestSecureA2AClient_SendSecure_Signature(t *testing.T) {
    // Verify message is signed
}

func TestSecureA2AClient_SessionReuse(t *testing.T) {
    // Test session management
}
```

### 4.4 Mock Infrastructure

**Use sage's MockTransport:**
```go
import "github.com/sage-x-project/sage/pkg/agent/transport"

func TestWithMock(t *testing.T) {
    mock := &transport.MockTransport{
        SendFunc: func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
            // Custom test behavior
            return &transport.Response{Success: true}, nil
        },
    }

    client := handshake.NewClient(mock, keyPair)
    // Test client behavior with mock transport
}
```

---

## 5. Implementation Timeline

### Week 1: sage Cleanup
- **Day 1-2:** Remove `sage/pkg/agent/transport/a2a/`
- **Day 3:** Verify all tests pass, update documentation
- **Day 4-5:** Code review, merge to main

### Week 2: sage-a2a-go Creation
- **Day 1:** Project setup, go.mod, directory structure
- **Day 2-3:** Implement `adapter/` package (TDD)
- **Day 4-5:** Implement `secure/` package (TDD)
- **Day 6:** Documentation, examples
- **Day 7:** Code review, initial release v1.0.0

### Week 3: sage-adk Refactoring
- **Day 1:** Update go.mod, add sage-a2a-go dependency
- **Day 2:** Delete duplicate files (~1000 lines)
- **Day 3-4:** Implement thin adapter layer
- **Day 5:** Update all examples
- **Day 6:** Integration testing
- **Day 7:** Documentation update, code review

### Week 4: Integration & Documentation
- **Day 1-2:** End-to-end testing across all projects
- **Day 3-4:** Performance testing, optimization
- **Day 5:** Update README files, architecture docs
- **Day 6:** Create migration guide for existing users
- **Day 7:** Release preparation

---

## 6. Success Criteria

### Functional Requirements
- ✅ sage has NO A2A dependencies
- ✅ sage-a2a-go implements `transport.MessageTransport`
- ✅ sage-adk uses libraries (no duplicate code)
- ✅ All existing functionality preserved
- ✅ All tests pass

### Quality Requirements
- ✅ Test coverage ≥85% for new code
- ✅ No breaking changes to public APIs
- ✅ Documentation complete and accurate
- ✅ Examples run without errors
- ✅ Code review approved

### Performance Requirements
- ✅ No performance regression vs. old code
- ✅ Handshake latency <100ms
- ✅ Message throughput ≥1000 msg/sec
- ✅ Memory usage stable

---

## 7. Risk Mitigation

### Risk 1: Breaking Changes
**Mitigation:**
- Maintain backward compatibility in sage
- Provide migration guide for sage-adk users
- Version bumps follow semantic versioning

### Risk 2: A2A Protocol Changes
**Mitigation:**
- sage-a2a-go isolated to A2A changes
- sage interface remains stable
- Version pinning in go.mod

### Risk 3: Performance Degradation
**Mitigation:**
- Benchmark tests before/after
- Profile hot paths
- Optimize converter functions

### Risk 4: Incomplete Test Coverage
**Mitigation:**
- TDD approach enforces tests
- CI/CD blocks on coverage drops
- Integration tests mandatory

---

## 8. Migration Guide (For Existing sage-adk Users)

### Before Refactoring
```go
import "github.com/sage-x-project/sage-adk/adapters/sage"

func main() {
    tm := sage.NewTransportManager(...)
    hm := sage.NewHandshakeManager(...)

    session := hm.PerformHandshake(ctx, recipientDID)
    tm.SendSecure(session, "message")
}
```

### After Refactoring
```go
import (
    "github.com/sage-x-project/sage/pkg/crypto"
    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    keyPair := crypto.GenerateKeyPair()
    conn, _ := grpc.Dial("agent.example.com:50051")

    client := sage.NewClient("did:sage:ethereum:0x123", keyPair, conn)
    agentConn, _ := client.ConnectToAgent(ctx, "did:sage:ethereum:0x456")
    agentConn.Send(ctx, "message")
}
```

**Key Changes:**
1. Replace `TransportManager` → `sage.Client`
2. Replace `HandshakeManager.PerformHandshake()` → `client.ConnectToAgent()`
3. Replace `SendSecure()` → `agentConn.Send()`
4. Simpler API, same functionality

---

## 9. Appendix: Code Inventory

### Files to Delete (sage)
```
sage/pkg/agent/transport/a2a/client.go
sage/pkg/agent/transport/a2a/client_test.go
```

### Files to Delete (sage-adk)
```
sage-adk/adapters/sage/transport.go        (520 lines)
sage-adk/adapters/sage/handshake.go        (543 lines)
sage-adk/adapters/sage/encryption.go       (estimated 200 lines)
sage-adk/adapters/sage/session.go          (estimated 150 lines)
sage-adk/adapters/sage/signature.go        (estimated 100 lines)
---
Total: ~1500 lines to delete
```

### Files to Create (sage-a2a-go)
```
sage-a2a-go/go.mod
sage-a2a-go/README.md
sage-a2a-go/LICENSE
sage-a2a-go/adapter/client.go              (~150 lines)
sage-a2a-go/adapter/client_test.go         (~300 lines)
sage-a2a-go/adapter/converter.go           (~200 lines)
sage-a2a-go/adapter/converter_test.go      (~200 lines)
sage-a2a-go/secure/client.go               (~250 lines)
sage-a2a-go/secure/client_test.go          (~400 lines)
sage-a2a-go/secure/session.go              (~100 lines)
sage-a2a-go/plain/client.go                (~100 lines)
sage-a2a-go/examples/secure_chat/main.go   (~100 lines)
sage-a2a-go/examples/plain_chat/main.go    (~80 lines)
---
Total: ~1880 lines to create
```

### Files to Create (sage-adk)
```
sage-adk/adapters/sage/client.go           (~150 lines)
sage-adk/adapters/sage/client_test.go      (~200 lines)
---
Total: ~350 lines to create
```

### Net LOC Change
```
Deleted:  ~1500 lines (sage-adk duplicates)
Created:  ~2230 lines (sage-a2a-go + new adapters)
---
Net:      +730 lines (but properly separated, tested, and maintainable)
```

---

## 10. Next Steps

1. **Review this document** with team/stakeholders
2. **Get approval** for refactoring approach
3. **Create sage-a2a-go repository**
4. **Begin Week 1 tasks** (sage cleanup)
5. **Set up CI/CD pipelines** for all three projects
6. **Schedule code reviews** for each phase

---

## Document History

| Version | Date       | Author | Changes |
|---------|------------|--------|---------|
| 1.0     | 2025-10-13 | Claude | Initial comprehensive analysis |

---

## References

- [SAGE Architecture Refactoring Proposal](../sage/docs/ARCHITECTURE_REFACTORING_PROPOSAL.md)
- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [Dependency Inversion Principle](https://en.wikipedia.org/wiki/Dependency_inversion_principle)
- [Test-Driven Development](https://en.wikipedia.org/wiki/Test-driven_development)
