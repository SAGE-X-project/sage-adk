# SAGE ADK Task List v1.1
**Version**: 1.1
**Date**: 2025-10-07
**Status**: Phase 2A Complete, Moving to Phase 2B

## 📊 Overall Progress

| Phase | Status | Tasks | Completion |
|-------|--------|-------|------------|
| Phase 1: Foundation | ✅ Complete | 8/8 | 100% |
| Phase 2A: Make It Work | ✅ Complete | 6/6 | 100% |
| Phase 2B: Add Intelligence | 🔄 In Progress | 0/6 | 0% |
| Phase 2C: Add Security | ⏳ Planned | 0/4 | 0% |
| Phase 2D: Production Ready | ⏳ Planned | 0/4 | 0% |

---

## ✅ Phase 2A: Make It Work (COMPLETE)

### Summary
**Goal**: 5줄로 동작하는 AI agent 만들기 ✅

**Completed Tasks**:
1. ✅ Task 1: Builder API Implementation (3 days)
2. ✅ Task 2: OpenAI Provider Implementation (2 days)
3. ✅ Task 3: A2A Transport Layer (3 days)
4. ✅ Task 4: Agent Runtime (3 days)
5. ✅ Task 5: Simple Chatbot Example (2 days)
6. ✅ Task 6: Documentation & Polishing (2 days)

**Deliverables**:
- ✅ Fluent Builder API with progressive complexity
- ✅ OpenAI LLM integration (GPT-3.5, GPT-4, GPT-4o)
- ✅ A2A protocol client/server with type conversion
- ✅ Agent runtime with Start/Stop lifecycle
- ✅ Working examples (main.go, minimal.go, client.go)
- ✅ Complete documentation (README, godoc, examples)
- ✅ 253 tests passing (100% success rate)

**Key Achievements**:
```go
// This now works:
agent := builder.NewAgent("chatbot").
    WithLLM(llm.OpenAI()).
    OnMessage(handleMessage).
    MustBuild()

agent.Start(":8080")
```

---

## 🚀 Phase 2B: Add Intelligence (CURRENT FOCUS)

**Goal**: SAGE 보안 프로토콜 통합하기

**Rationale**:
- MCP는 외부 도구 의존성이 크고 복잡함
- SAGE는 프로젝트의 핵심 가치 (blockchain-based security)
- SAGE를 먼저 완성하면 차별화된 가치 제공
- 나중에 MCP를 추가하는 것이 더 자연스러움

**Duration**: 2 weeks (Week 3-4)

---

### Week 3: SAGE Core Implementation

#### Task 7: SAGE Transport Layer (5 days)
**Priority**: P1 - High (Critical for security)
**Status**: 🆕 Ready to Start

**Files**:
```
adapters/sage/
├── transport.go       # Handshake + transport
├── transport_test.go  # Unit tests
├── handshake.go       # 4-phase handshake
├── session.go         # Session management
├── encryption.go      # HPKE encryption
└── signing.go         # RFC 9421 signatures
```

**Requirements**:
```go
// Client side handshake:
transport := sage.NewTransport(sage.TransportConfig{
    LocalDID:    "did:sage:ethereum:0xABC...",
    RemoteDID:   "did:sage:ethereum:0xDEF...",
    Network:     sage.NetworkEthereum,
    RPCEndpoint: "https://eth-mainnet.alchemyapi.io/v2/...",
    PrivateKey:  privateKey,
})

// Perform handshake
session, err := transport.Handshake(ctx, remoteURL)

// Send encrypted + signed message
err := transport.SendMessage(ctx, session, msg)
```

**SAGE Handshake Flow** (4 phases):
```
Phase 1 (Invitation):
  Alice → Bob: { did_alice, nonce_a, public_key_a, capabilities }

Phase 2 (Request):
  Bob → Alice: { did_bob, nonce_b, public_key_b,
                 encrypted_with(public_key_a, shared_secret),
                 signature_bob }

Phase 3 (Response):
  Alice → Bob: { encrypted_with(shared_secret, session_key),
                 signature_alice, metadata }

Phase 4 (Complete):
  Bob → Alice: { encrypted_with(session_key, "ack"),
                 signature_bob }
```

**Cryptographic Primitives** (from sage library):
- **Key Agreement**: HPKE (Hybrid Public Key Encryption)
- **Signatures**: EdDSA (Ed25519) via RFC 9421
- **Encryption**: ChaCha20-Poly1305 for message payload
- **Hashing**: BLAKE3 for nonces

**Acceptance Criteria**:
- [ ] Handshake Phase 1-4 implemented
- [ ] HPKE key agreement works
- [ ] Session keys derived correctly
- [ ] Message encryption/decryption works
- [ ] RFC 9421 signature creation/verification
- [ ] Nonce generation and validation
- [ ] Session state management
- [ ] Error recovery (network failures, invalid signatures)
- [ ] Test coverage ≥ 80%
- [ ] Integration test with sage library

**Dependencies**:
- ✅ sage library (already exists in sage-x-project/sage)
- ✅ Core types (pkg/types/security.go already has SAGE types)

**Reference Implementation**:
- `sage/core/handshake/` - Handshake logic
- `sage/core/session/` - Session management
- `sage/crypto/` - Cryptographic operations

**Testing Strategy**:
1. Unit tests for each handshake phase
2. Mock blockchain for DID resolution
3. Test vector validation (known inputs → outputs)
4. Error injection tests (invalid signatures, replayed nonces)
5. Integration test with real sage library

---

#### Task 8: SAGE Configuration & DID Management (2 days)
**Priority**: P1 - High
**Status**: 🔜 Blocked by Task 7

**Files**:
```
adapters/sage/
├── config.go          # SAGE configuration
├── config_test.go     # Config validation tests
├── did.go             # DID resolution
├── did_test.go        # DID tests
├── keys.go            # Key management
└── blockchain.go      # Blockchain interaction
```

**Requirements**:
```go
// Configuration
sageConfig := &config.SAGEConfig{
    DID:             "did:sage:ethereum:0xABC...",
    Network:         "ethereum",
    RPCEndpoint:     "https://eth-mainnet.alchemyapi.io/v2/...",
    ContractAddress: "0xDEF...",
    PrivateKeyPath:  "keys/agent.key",
    CacheExpiry:     300, // seconds
}

// Agent with SAGE
agent := builder.NewAgent("secure-agent").
    WithProtocol(protocol.ProtocolSAGE).
    WithSAGEConfig(sageConfig).
    WithLLM(llm.OpenAI()).
    OnMessage(handleSecureMessage).
    Build()
```

**DID Resolution Flow**:
```
1. Check local cache first
2. If not cached, query blockchain:
   - Connect to RPC endpoint
   - Call AgentRegistry contract
   - Get agent record (publicKeys, status, capabilities)
3. Validate agent status (must be "active")
4. Cache result (with TTL)
5. Return DID document
```

**Key Management**:
```go
// Key loading
keyManager := sage.NewKeyManager()
err := keyManager.LoadFromFile("keys/agent.key", password)

// Key types (all three required for SAGE):
// - Ed25519: Signing key (RFC 9421)
// - X25519:  Key agreement (HPKE)
// - Secp256k1: Ethereum signing (for blockchain)

// Key generation (if needed)
keys, err := keyManager.GenerateKeySet()
keyManager.SaveToFile("keys/agent.key", password)
```

**Acceptance Criteria**:
- [ ] Config validation (required fields, valid URLs)
- [ ] DID resolution from blockchain
- [ ] DID caching (with expiry)
- [ ] Key loading (Ed25519, X25519, Secp256k1)
- [ ] Key generation helper
- [ ] Password-protected key storage
- [ ] Blockchain connection validation
- [ ] Error handling (invalid DID, inactive agent, network errors)
- [ ] Test coverage ≥ 85%

**Dependencies**:
- ✅ Task 7 (SAGE Transport)
- ✅ sage library DID resolver
- ✅ Ethereum RPC client

**Testing Strategy**:
1. Mock blockchain for unit tests
2. Test DID caching behavior
3. Key generation/loading round-trip
4. Config validation edge cases
5. Integration test with local blockchain (Hardhat)

---

### Week 4: SAGE Integration & Examples

#### Task 9: SAGE Server Implementation (3 days)
**Priority**: P1 - High
**Status**: 🔜 Blocked by Task 7, 8

**Files**:
```
adapters/sage/
├── server.go          # SAGE server
├── server_test.go     # Server tests
├── handler.go         # Message handling
└── middleware.go      # SAGE middleware
```

**Requirements**:
```go
// Server configuration
serverConfig := &sage.ServerConfig{
    AgentName:      "secure-agent",
    AgentURL:       "https://secure-agent.example.com/",
    SAGEConfig:     sageConfig,
    MessageHandler: handleMessage,
}

server, err := sage.NewServer(serverConfig)

// Start server (implements agent.Server interface)
err = server.Start(":8080")
```

**Server Responsibilities**:
1. **Accept handshake initiations** (Phase 1: Invitation)
2. **Manage active sessions** (track session state)
3. **Decrypt incoming messages** (using session keys)
4. **Verify signatures** (RFC 9421)
5. **Route to message handler** (call user's OnMessage)
6. **Encrypt responses** (using session keys)
7. **Sign outgoing messages** (RFC 9421)

**Middleware Pipeline**:
```
Incoming Message Flow:
  HTTP Request
    → SAGE Middleware (decrypt, verify)
      → Session Validation
        → Message Handler (user code)
          → Response Builder
            → SAGE Middleware (encrypt, sign)
              → HTTP Response
```

**Acceptance Criteria**:
- [ ] Server accepts handshake invitations
- [ ] Session management (create, lookup, expire)
- [ ] Message decryption works
- [ ] Signature verification works
- [ ] Message handler integration
- [ ] Response encryption works
- [ ] Response signing works
- [ ] Error handling (invalid signature, expired session)
- [ ] Graceful shutdown (drain connections)
- [ ] Test coverage ≥ 85%

**Dependencies**:
- ✅ Task 7 (SAGE Transport)
- ✅ Task 8 (SAGE Config)
- ✅ agent.Server interface (from core/agent)

**Testing Strategy**:
1. Unit tests for each middleware component
2. Mock session manager
3. Test handshake acceptance
4. Test message routing
5. Integration test (client → server)

---

#### Task 10: SAGE Example (3 days)
**Priority**: P1 - High
**Status**: 🔜 Blocked by Task 7, 8, 9

**Files**:
```
examples/secure-agent/
├── main.go            # Secure agent
├── README.md          # Setup guide
├── scripts/
│   ├── setup.sh       # All-in-one setup
│   ├── generate-keys.sh
│   ├── start-blockchain.sh
│   └── register-did.sh
├── docker-compose.yml # Local blockchain + agent
├── .env.example       # Configuration template
└── test.sh            # E2E test
```

**Requirements**:
```go
// main.go - Secure agent with SAGE
package main

import (
    "context"
    "log"
    "os"

    "github.com/sage-x-project/sage-adk/adapters/llm"
    "github.com/sage-x-project/sage-adk/builder"
    "github.com/sage-x-project/sage-adk/config"
    "github.com/sage-x-project/sage-adk/core/agent"
    "github.com/sage-x-project/sage-adk/core/protocol"
)

func main() {
    // SAGE configuration
    sageConfig := &config.SAGEConfig{
        DID:             os.Getenv("AGENT_DID"),
        Network:         "ethereum",
        RPCEndpoint:     os.Getenv("ETH_RPC_URL"),
        ContractAddress: os.Getenv("REGISTRY_ADDRESS"),
        PrivateKeyPath:  "keys/agent.key",
    }

    // Build secure agent
    agent := builder.NewAgent("secure-chatbot").
        WithProtocol(protocol.ProtocolSAGE).
        WithSAGEConfig(sageConfig).
        WithLLM(llm.OpenAI()).
        OnMessage(handleMessage).
        BeforeStart(func(ctx context.Context) error {
            log.Println("🔐 Secure agent starting...")
            return nil
        }).
        MustBuild()

    log.Fatal(agent.Start(":8080"))
}

func handleMessage(ctx context.Context, msg agent.MessageContext) error {
    // Get LLM response
    request := &llm.CompletionRequest{
        Messages: []llm.Message{
            {Role: llm.RoleSystem, Content: "You are a secure AI assistant."},
            {Role: llm.RoleUser, Content: msg.Text()},
        },
    }

    // Access LLM from context (injected by builder)
    provider := ctx.Value("llm").(llm.Provider)
    response, err := provider.Complete(ctx, request)
    if err != nil {
        return err
    }

    return msg.Reply(response.Content)
}
```

**Setup Scripts**:

1. **setup.sh** (all-in-one):
```bash
#!/bin/bash
# Install dependencies
npm install -g hardhat

# Generate keys
./scripts/generate-keys.sh

# Start local blockchain
./scripts/start-blockchain.sh &

# Wait for blockchain
sleep 5

# Register DID
./scripts/register-did.sh

echo "✅ Setup complete! Run: go run main.go"
```

2. **docker-compose.yml**:
```yaml
version: '3.8'
services:
  blockchain:
    image: trufflesuite/ganache:latest
    ports:
      - "8545:8545"
    command: --deterministic --accounts 10

  agent:
    build: .
    ports:
      - "8080:8080"
    environment:
      - AGENT_DID=did:sage:ethereum:0x...
      - ETH_RPC_URL=http://blockchain:8545
      - REGISTRY_ADDRESS=0x...
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    depends_on:
      - blockchain
```

**Test Flow** (test.sh):
```bash
#!/bin/bash
set -e

echo "1. Starting blockchain..."
docker-compose up -d blockchain
sleep 5

echo "2. Deploying contract..."
cd contracts && npx hardhat run scripts/deploy.js --network localhost

echo "3. Starting agent..."
docker-compose up -d agent
sleep 3

echo "4. Testing secure communication..."
# Create test client
go run test-client.go "Hello secure agent!"

echo "✅ All tests passed!"
```

**Acceptance Criteria**:
- [ ] Setup scripts work on fresh machine
- [ ] Keys generated correctly
- [ ] Local blockchain starts
- [ ] DID registered on blockchain
- [ ] Agent starts and accepts secure messages
- [ ] Handshake completes successfully
- [ ] Messages encrypted and signed
- [ ] README clear and complete
- [ ] Docker setup works
- [ ] Test script validates E2E flow

**Dependencies**:
- ✅ Task 7, 8, 9 (SAGE fully implemented)

**Testing Strategy**:
1. Test on fresh Ubuntu VM
2. Test on macOS
3. Time the setup (should be < 15 minutes)
4. Validate all cryptographic operations
5. Test error scenarios (wrong key, invalid DID)

---

#### Task 11: Protocol Auto-Detection (2 days)
**Priority**: P1 - High
**Status**: 🔜 Blocked by Task 7, 8, 9

**Files**:
```
examples/hybrid-agent/
├── main.go           # Auto-detect A2A vs SAGE
├── README.md
├── test-a2a.sh       # Test A2A path
└── test-sage.sh      # Test SAGE path
```

**Requirements**:
```go
// Agent that handles both A2A and SAGE
agent := builder.NewAgent("hybrid").
    WithProtocol(protocol.ProtocolAuto).  // Auto-detect!
    WithA2AConfig(a2aConfig).             // Optional A2A
    WithSAGEConfig(sageConfig).           // Optional SAGE
    WithLLM(llm.OpenAI()).
    OnMessage(handleMessage).
    Build()

// Detection logic (in core/protocol/selector.go):
func DetectProtocol(msg *types.Message) ProtocolMode {
    if msg.Security != nil && msg.Security.AgentDID != "" {
        return ProtocolSAGE  // Has security metadata
    }
    return ProtocolA2A  // No security metadata
}
```

**Auto-Detection Flow**:
```
Incoming Message:
  1. Parse JSON
  2. Check for "security" field
  3. If present → SAGE path (decrypt, verify, handle)
  4. If absent → A2A path (handle directly)
  5. Response uses same protocol as request
```

**Acceptance Criteria**:
- [ ] Auto-detection works for A2A messages
- [ ] Auto-detection works for SAGE messages
- [ ] Response protocol matches request protocol
- [ ] Error handling (ambiguous messages)
- [ ] Test coverage ≥ 85%
- [ ] Example demonstrates both paths

**Dependencies**:
- ✅ Task 3 (A2A Transport)
- ✅ Task 7, 8, 9 (SAGE Transport)

**Testing Strategy**:
1. Send A2A message → verify A2A response
2. Send SAGE message → verify SAGE response
3. Alternate between protocols
4. Test edge cases (malformed messages)

---

#### Task 12: Documentation Update (1 day)
**Priority**: P1 - High
**Status**: 🔜 Blocked by Task 7-11

**Files**:
```
docs/
├── sage-protocol.md   # SAGE protocol guide
├── security.md        # Security best practices
└── examples.md        # Updated examples
```

**Content**:
1. **SAGE Protocol Guide**:
   - What is SAGE?
   - Why blockchain-based security?
   - Handshake flow explained
   - Cryptographic primitives
   - Setup guide
   - Troubleshooting

2. **Security Best Practices**:
   - Key management (never commit keys!)
   - DID registration
   - Session management
   - Network security
   - Monitoring and alerts

3. **Examples Update**:
   - Add SAGE examples
   - Add hybrid example
   - Update architecture diagrams

**Acceptance Criteria**:
- [ ] SAGE protocol documented
- [ ] Security best practices listed
- [ ] Examples section updated
- [ ] All code examples tested
- [ ] Diagrams added (handshake flow, architecture)
- [ ] Troubleshooting section complete

---

## ⏳ Phase 2C: Add Intelligence (MCP) - DEFERRED

**Original Goal**: MCP 도구 통합
**New Status**: Moved to Phase 3
**Reason**: SAGE security is higher priority for project differentiation

**Deferred Tasks**:
- Task 13: MCP Client Implementation (4 days) → Phase 3
- Task 14: MCP Server Implementations (3 days) → Phase 3
- Task 15: LLM + MCP Integration (3 days) → Phase 3
- Task 16: MCP Agent Example (2 days) → Phase 3

---

## ⏳ Phase 2D: Production Ready - UNCHANGED

**Goal**: 프로덕션 환경에서 사용 가능하게

**Planned Tasks**:
- Task 17: Redis Storage (3 days)
- Task 18: PostgreSQL Storage (3 days)
- Task 19: Metrics & Monitoring (2 days)
- Task 20: Multi-Agent Orchestrator Example (4 days)

---

## 📊 Updated Timeline

| Week | Phase | Focus | Deliverable |
|------|-------|-------|-------------|
| 1-2 | 2A | Builder + Runtime | ✅ Working agent |
| 3 | 2B | SAGE Core | SAGE transport + config |
| 4 | 2B | SAGE Integration | SAGE example + auto-detect |
| 5-6 | 2D | Storage | Redis + PostgreSQL |
| 7-8 | 2D | Production | Metrics + orchestrator |

**Total**: 6 weeks to SAGE-secured, production-ready framework

---

## 🎯 Success Metrics

### Phase 2B Success (Week 4 End)
- [ ] SAGE handshake completes successfully
- [ ] Messages encrypted and signed
- [ ] DID resolution from blockchain works
- [ ] Example runs on fresh machine
- [ ] Test coverage ≥80%
- [ ] Documentation complete

### Phase 2D Success (Week 8 End)
- [ ] Redis storage working
- [ ] PostgreSQL storage working
- [ ] Metrics exposed (Prometheus)
- [ ] Orchestrator example working
- [ ] Docker + K8s deployment tested
- [ ] Test coverage ≥90%

---

## 🚦 Decision Gate: End of Week 4

**Question**: Does SAGE security work end-to-end?

**Pass Criteria**:
- [ ] Handshake (4 phases) completes
- [ ] Messages encrypted/decrypted correctly
- [ ] Signatures verified
- [ ] DID resolution works
- [ ] Example works on fresh machine in < 30 min
- [ ] Test coverage ≥80%

**If PASS**: Proceed to Phase 2D (Storage + Production)
**If FAIL**: Simplify SAGE, mark as experimental, focus on A2A

---

## 📝 Changes from v1.0

1. **Re-prioritized Phase 2B**: SAGE security before MCP
2. **Moved MCP to Phase 3**: Deferred for later
3. **Updated timeline**: 6 weeks instead of 8 weeks
4. **Merged Phase 2C into 2B**: SAGE is now Phase 2B
5. **Phase 2D unchanged**: Storage + Production still planned

**Rationale**:
- SAGE is core differentiator (blockchain-based security)
- MCP has external dependencies (Node.js servers)
- SAGE completion unblocks production use cases
- MCP can be added later without blocking other features

---

**Document Version**: 1.1
**Last Updated**: 2025-10-07
**Next Review**: End of Week 4 (Phase 2B completion)
