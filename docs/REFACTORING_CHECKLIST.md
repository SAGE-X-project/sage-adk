# SAGE-ADK Refactoring Checklist

**Version**: 1.0
**Created**: 2025-10-10
**Status**: In Progress
**Target Completion**: 5 weeks (25 working days)

## Overview

This checklist tracks the integration of SAGE security framework features from `sage/` into `sage-adk/`. The work is organized into 5 phases based on priority and dependencies.

**Progress Summary:**
- **Phase 1 (Critical)**: 0/5 completed (Week 1)
- **Phase 2 (High)**: 0/5 completed (Weeks 2-3)
- **Phase 3 (Medium)**: 0/5 completed (Weeks 4-5)
- **Phase 4 (Low)**: 0/4 completed (Future)
- **Testing & Documentation**: 0/3 completed (Continuous)

---

## Phase 1: Critical Foundation (Week 1 - 5 days)

### 1.1 Crypto Integration (2 days)
- [ ] Create `sage-adk/internal/crypto/` package structure
- [ ] Implement crypto manager wrapper around `sage/crypto`
- [ ] Add key generation methods (Ed25519, Secp256k1, X25519, RSA256)
- [ ] Add key storage abstraction (memory, file, vault)
- [ ] Update `builder/builder.go` with key management methods
  - [ ] `WithKeyManager(km crypto.Manager) *Builder`
  - [ ] `WithKeyPair(keyPair crypto.KeyPair) *Builder`
  - [ ] `WithKeyPath(path string) *Builder`
- [ ] Write unit tests for crypto wrapper
- [ ] Write integration tests for key generation and storage
- [ ] Update documentation for crypto integration

**Dependencies**: None
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 1.2 DID Integration (2 days)
- [ ] Create `sage-adk/internal/did/` package structure
- [ ] Implement DID manager wrapper around `sage/did`
- [ ] Add multi-chain DID support (Ethereum, Solana, Kaia)
- [ ] Implement DID resolution methods
- [ ] Add DID document caching
- [ ] Update `builder/builder.go` with DID methods
  - [ ] `WithDIDManager(dm did.Manager) *Builder`
  - [ ] `WithDID(did string) *Builder`
  - [ ] `WithDIDResolver(resolver did.Resolver) *Builder`
- [ ] Write unit tests for DID wrapper
- [ ] Write integration tests for multi-chain DID resolution
- [ ] Update documentation for DID integration

**Dependencies**: 1.1 Crypto Integration
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 1.3 RFC 9421 Integration (1 day)
- [ ] Create `sage-adk/internal/rfc9421/` package structure
- [ ] Implement RFC 9421 message signature wrapper
- [ ] Add signature generation for outgoing messages
- [ ] Add signature verification for incoming messages
- [ ] Implement covered components (digest, signature params)
- [ ] Update `adapters/sage/adapter.go` with RFC 9421 methods
- [ ] Write unit tests for signature generation/verification
- [ ] Write integration tests with sample HTTP messages
- [ ] Update documentation for RFC 9421 integration

**Dependencies**: 1.1 Crypto Integration, 1.2 DID Integration
**Estimated Time**: 1 day
**Assignee**: TBD
**Status**: Not Started

---

## Phase 2: High Priority Features (Weeks 2-3 - 10 days)

### 2.1 Session Management (2 days)
- [ ] Create `sage-adk/internal/session/` package structure
- [ ] Implement session manager wrapper around `sage/session`
- [ ] Add session creation with key derivation
- [ ] Add session encryption/decryption methods
- [ ] Add session metadata tracking
- [ ] Implement session lifecycle management
- [ ] Add session timeout and cleanup
- [ ] Update `core/agent/agent.go` with session support
- [ ] Write unit tests for session management
- [ ] Write integration tests for session lifecycle
- [ ] Update documentation for session management

**Dependencies**: 1.1 Crypto Integration, 1.2 DID Integration
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 2.2 Handshake Protocol (3 days)
- [ ] Create `sage-adk/internal/handshake/` package structure
- [ ] Implement handshake manager wrapper around `sage/handshake`
- [ ] Implement Phase 1: INVITE message handling
- [ ] Implement Phase 2: ACCEPT message handling
- [ ] Implement Phase 3: CONFIRM message handling
- [ ] Implement Phase 4: COMPLETE message handling
- [ ] Add handshake state machine
- [ ] Add handshake timeout handling
- [ ] Add handshake event callbacks
- [ ] Update `adapters/sage/adapter.go` with handshake support
- [ ] Write unit tests for each handshake phase
- [ ] Write integration tests for complete handshake flow
- [ ] Update documentation for handshake protocol

**Dependencies**: 2.1 Session Management, 1.3 RFC 9421 Integration
**Estimated Time**: 3 days
**Assignee**: TBD
**Status**: Not Started

### 2.3 HPKE Integration (1 day)
- [ ] Create `sage-adk/internal/hpke/` package structure
- [ ] Implement HPKE wrapper around `sage/hpke`
- [ ] Add HPKE session key derivation
- [ ] Add HPKE encryption/decryption
- [ ] Integrate HPKE with session management
- [ ] Write unit tests for HPKE operations
- [ ] Write integration tests for HPKE session derivation
- [ ] Update documentation for HPKE integration

**Dependencies**: 2.1 Session Management
**Estimated Time**: 1 day
**Assignee**: TBD
**Status**: Not Started

### 2.4 Message Validation (2 days)
- [ ] Create `sage-adk/internal/validation/` package structure
- [ ] Implement validation wrapper around `sage/core/message`
- [ ] Add nonce management and validation
- [ ] Add duplicate message detection
- [ ] Add message ordering validation
- [ ] Add replay attack prevention
- [ ] Update `adapters/sage/adapter.go` with validation
- [ ] Write unit tests for each validation type
- [ ] Write integration tests for attack scenarios
- [ ] Update documentation for message validation

**Dependencies**: 1.3 RFC 9421 Integration, 2.1 Session Management
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 2.5 Complete SAGE Adapter (2 days)
- [ ] Remove stub implementations from `adapters/sage/adapter.go`
- [ ] Implement `SendMessage()` with full SAGE protocol
- [ ] Implement `ReceiveMessage()` with full SAGE protocol
- [ ] Implement `Verify()` with RFC 9421 + validation
- [ ] Implement `InitiateHandshake()` method
- [ ] Implement `RespondToHandshake()` method
- [ ] Add proper error handling and logging
- [ ] Add metrics for SAGE operations
- [ ] Write comprehensive unit tests
- [ ] Write end-to-end integration tests
- [ ] Update documentation for SAGE adapter

**Dependencies**: All Phase 1 and Phase 2.1-2.4
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

---

## Phase 3: Medium Priority Enhancements (Weeks 4-5 - 10 days)

### 3.1 Config Unification (2 days)
- [ ] Analyze differences between `sage/config` and `sage-adk/config`
- [ ] Design unified configuration schema
- [ ] Create `sage-adk/config/unified.go`
- [ ] Migrate SAGE config to unified schema
- [ ] Migrate A2A config to unified schema
- [ ] Add config validation
- [ ] Add config migration utilities
- [ ] Update builder to use unified config
- [ ] Write unit tests for config parsing
- [ ] Write integration tests for config migration
- [ ] Update all documentation and examples

**Dependencies**: Phase 2 complete
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 3.2 Health Check Integration (1 day)
- [ ] Create `sage-adk/internal/health/` package structure
- [ ] Implement health check wrapper around `sage/health`
- [ ] Add liveness probes
- [ ] Add readiness probes
- [ ] Add startup probes
- [ ] Add component-level health checks
- [ ] Integrate with existing `observability/health/`
- [ ] Write unit tests for health checks
- [ ] Write integration tests for probe endpoints
- [ ] Update documentation for health checks

**Dependencies**: Phase 2 complete
**Estimated Time**: 1 day
**Assignee**: TBD
**Status**: Not Started

### 3.3 Smart Contract Integration (3 days)
- [ ] Create `sage-adk/internal/contracts/` package structure
- [ ] Add Ethereum contract bindings from `sage/contracts/ethereum`
- [ ] Add Solana contract bindings from `sage/contracts/solana`
- [ ] Implement DID registry interactions
- [ ] Implement challenge-response verification
- [ ] Add multi-chain support
- [ ] Add gas optimization
- [ ] Write unit tests for contract interactions
- [ ] Write integration tests with testnet
- [ ] Update documentation for contract integration

**Dependencies**: 1.2 DID Integration
**Estimated Time**: 3 days
**Assignee**: TBD
**Status**: Not Started

### 3.4 Key Rotation (2 days)
- [ ] Create `sage-adk/internal/rotation/` package structure
- [ ] Implement key rotation wrapper around `sage/crypto/rotation`
- [ ] Add automated key rotation policies
- [ ] Add rotation scheduling
- [ ] Add key archival and rollback
- [ ] Integrate with crypto manager
- [ ] Write unit tests for rotation logic
- [ ] Write integration tests for rotation scenarios
- [ ] Update documentation for key rotation

**Dependencies**: 1.1 Crypto Integration
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 3.5 Examples and Tutorials (2 days)
- [ ] Create `examples/sage-security/` directory
- [ ] Write basic SAGE handshake example
- [ ] Write secure messaging example
- [ ] Write multi-chain DID example
- [ ] Write key rotation example
- [ ] Update `examples/README.md`
- [ ] Create tutorial documentation
- [ ] Add code comments and explanations
- [ ] Test all examples
- [ ] Update main documentation

**Dependencies**: Phase 2 complete
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

---

## Phase 4: Low Priority / Future Work

### 4.1 OIDC Integration (Future)
- [ ] Evaluate OIDC integration requirements
- [ ] Create `sage-adk/internal/oidc/` package structure
- [ ] Implement OIDC wrapper around `sage/oidc`
- [ ] Add OIDC provider configuration
- [ ] Add token validation
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation

**Dependencies**: Phase 1-3 complete
**Estimated Time**: 3 days
**Assignee**: TBD
**Status**: Not Started

### 4.2 Advanced Metrics (Future)
- [ ] Create `sage-adk/internal/metrics/` package structure
- [ ] Implement metrics wrapper around `sage/internal/metrics`
- [ ] Add SAGE-specific metrics
- [ ] Add performance counters
- [ ] Integrate with Prometheus
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation

**Dependencies**: Phase 1-3 complete
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 4.3 Vault Integration (Future)
- [ ] Create `sage-adk/internal/vault/` package structure
- [ ] Implement vault wrapper around `sage/crypto/vault`
- [ ] Add HashiCorp Vault support
- [ ] Add secret rotation
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation

**Dependencies**: 1.1 Crypto Integration
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

### 4.4 Tracing Integration (Future)
- [ ] Evaluate tracing requirements (OpenTelemetry)
- [ ] Create `sage-adk/observability/tracing/` package
- [ ] Add distributed tracing
- [ ] Add span context propagation
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Update documentation

**Dependencies**: Phase 1-3 complete
**Estimated Time**: 2 days
**Assignee**: TBD
**Status**: Not Started

---

## Testing & Quality Assurance (Continuous)

### Unit Testing
- [ ] Achieve 90%+ test coverage for all new code
- [ ] Add table-driven tests for all functions
- [ ] Add error case testing
- [ ] Add race condition testing (`-race` flag)
- [ ] Review and update existing tests

**Target**: 90%+ coverage
**Status**: In Progress

### Integration Testing
- [ ] Create integration test suite in `test/integration/sage/`
- [ ] Test complete handshake flows
- [ ] Test multi-chain DID resolution
- [ ] Test session lifecycle
- [ ] Test message validation scenarios
- [ ] Test error recovery

**Target**: All critical paths covered
**Status**: Not Started

### Documentation
- [ ] Update `docs/architecture/overview.md`
- [ ] Update `docs/architecture/protocol-layer.md`
- [ ] Create `docs/security/sage-integration.md`
- [ ] Update `CLAUDE.md` with new architecture
- [ ] Update `README.md` with SAGE features
- [ ] Create API documentation
- [ ] Create migration guide

**Target**: Complete documentation
**Status**: Not Started

---

## Risk Mitigation

### Breaking Changes
- [ ] Identify all breaking API changes
- [ ] Create deprecation plan
- [ ] Write migration guide
- [ ] Version bump strategy

**Status**: Not Started

### Performance Testing
- [ ] Benchmark crypto operations
- [ ] Benchmark handshake protocol
- [ ] Benchmark message validation
- [ ] Optimize bottlenecks

**Status**: Not Started

### Security Audit
- [ ] Code review for security vulnerabilities
- [ ] Penetration testing
- [ ] Dependency audit
- [ ] Security documentation

**Status**: Not Started

---

## Success Metrics

### Functional Metrics
- [ ] All SAGE protocol features implemented
- [ ] All tests passing (100%)
- [ ] Test coverage ≥90%
- [ ] Zero known security vulnerabilities

### Performance Metrics
- [ ] Handshake completion <100ms
- [ ] Message signing <10ms
- [ ] Message verification <15ms
- [ ] Session creation <50ms

### Code Quality Metrics
- [ ] No critical linter warnings
- [ ] All public APIs documented
- [ ] All examples working
- [ ] Migration guide complete

---

## Notes

- Update this checklist as work progresses
- Mark items complete with `[x]` when finished
- Add actual assignees and dates as work begins
- Track blockers and dependencies
- Update progress summary at the top

**Last Updated**: 2025-10-10
