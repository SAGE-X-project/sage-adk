# SAGE-ADK Refactoring Implementation Plan

**Version**: 1.0
**Created**: 2025-10-10
**Author**: Development Team
**Status**: Ready for Implementation

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Phase 1: Critical Foundation](#phase-1-critical-foundation)
4. [Phase 2: High Priority Features](#phase-2-high-priority-features)
5. [Phase 3: Medium Priority Enhancements](#phase-3-medium-priority-enhancements)
6. [Phase 4: Future Work](#phase-4-future-work)
7. [Testing Strategy](#testing-strategy)
8. [Migration Guide](#migration-guide)
9. [Risk Management](#risk-management)

---

## Executive Summary

### Current State
- **sage-adk**: Has stub SAGE adapter with no real implementation
- **sage**: Has production-ready SAGE security framework with complete features

### Goal
Integrate SAGE security framework from `sage/` into `sage-adk/` to enable:
- Secure handshake protocol
- RFC 9421 HTTP message signatures
- Multi-chain DID resolution
- Session management with encryption
- Message validation and replay protection

### Approach
- **Strategy**: Wrapper pattern to isolate sage/ dependencies
- **Timeline**: 5 weeks (25 working days)
- **Team Size**: 2-3 developers
- **Test Coverage**: ≥90%

---

## Architecture Overview

### Current Architecture

```
sage-adk/
├── adapters/
│   ├── a2a/           ✅ Complete
│   ├── llm/           ✅ Complete
│   └── sage/          ❌ Stub only
├── builder/           ✅ Complete
├── core/
│   ├── agent/         ✅ Complete (no security)
│   └── protocol/      ✅ Complete
└── storage/           ✅ Complete
```

### Target Architecture

```
sage-adk/
├── adapters/
│   ├── a2a/           ✅ Complete
│   ├── llm/           ✅ Complete
│   └── sage/          🎯 Full SAGE implementation
├── builder/           🔧 Enhanced with security methods
├── core/
│   ├── agent/         🔧 Enhanced with session support
│   └── protocol/      ✅ Complete
├── internal/          🆕 New wrappers for sage/
│   ├── crypto/        🆕 Crypto wrapper
│   ├── did/           🆕 DID wrapper
│   ├── rfc9421/       🆕 RFC 9421 wrapper
│   ├── session/       🆕 Session wrapper
│   ├── handshake/     🆕 Handshake wrapper
│   ├── hpke/          🆕 HPKE wrapper
│   └── validation/    🆕 Validation wrapper
└── storage/           ✅ Complete
```

### Design Principles

1. **Wrapper Pattern**: Isolate sage/ dependencies in internal packages
2. **Backward Compatibility**: Preserve existing A2A functionality
3. **Interface Segregation**: Small, focused interfaces
4. **Dependency Injection**: All dependencies injected via builder
5. **Testability**: All wrappers fully unit tested

---

## Phase 1: Critical Foundation

### 1.1 Crypto Integration

#### Package Structure

```
sage-adk/internal/crypto/
├── manager.go          # Main crypto manager
├── manager_test.go     # Unit tests
├── keys.go             # Key generation
├── keys_test.go
├── storage.go          # Storage abstraction
├── storage_test.go
└── doc.go              # Package documentation
```

#### API Design

```go
// Package crypto provides cryptographic operations for SAGE-ADK
package crypto

import (
    sagecrypto "github.com/sage-x-project/sage/crypto"
)

// Manager wraps sage/crypto.Manager with ADK-specific functionality
type Manager struct {
    inner sagecrypto.Manager
}

// NewManager creates a new crypto manager
func NewManager() (*Manager, error) {
    inner, err := sagecrypto.NewManager()
    if err != nil {
        return nil, fmt.Errorf("failed to create crypto manager: %w", err)
    }
    return &Manager{inner: inner}, nil
}

// GenerateEd25519 generates an Ed25519 key pair
func (m *Manager) GenerateEd25519() (sagecrypto.KeyPair, error) {
    return m.inner.Generate(sagecrypto.KeyTypeEd25519)
}

// GenerateSecp256k1 generates a Secp256k1 key pair
func (m *Manager) GenerateSecp256k1() (sagecrypto.KeyPair, error) {
    return m.inner.Generate(sagecrypto.KeyTypeSecp256k1)
}

// LoadFromFile loads a key pair from file
func (m *Manager) LoadFromFile(path string) (sagecrypto.KeyPair, error) {
    return m.inner.LoadFromFile(path)
}

// SaveToFile saves a key pair to file
func (m *Manager) SaveToFile(kp sagecrypto.KeyPair, path string) error {
    return m.inner.SaveToFile(kp, path)
}
```

#### Builder Integration

```go
// File: builder/builder.go

type Builder struct {
    // ... existing fields ...
    cryptoManager *crypto.Manager
    keyPair       sagecrypto.KeyPair
}

// WithKeyManager sets the crypto manager
func (b *Builder) WithKeyManager(cm *crypto.Manager) *Builder {
    b.cryptoManager = cm
    return b
}

// WithKeyPair sets the key pair
func (b *Builder) WithKeyPair(kp sagecrypto.KeyPair) *Builder {
    b.keyPair = kp
    return b
}

// WithKeyPath loads a key pair from file
func (b *Builder) WithKeyPath(path string) *Builder {
    if b.cryptoManager == nil {
        b.errors = append(b.errors,
            errors.New("crypto manager required before loading key"))
        return b
    }

    kp, err := b.cryptoManager.LoadFromFile(path)
    if err != nil {
        b.errors = append(b.errors,
            fmt.Errorf("failed to load key from %s: %w", path, err))
        return b
    }

    b.keyPair = kp
    return b
}
```

#### Testing Strategy

```go
// File: internal/crypto/manager_test.go

func TestManager_GenerateEd25519(t *testing.T) {
    m, err := NewManager()
    if err != nil {
        t.Fatalf("NewManager() failed: %v", err)
    }

    kp, err := m.GenerateEd25519()
    if err != nil {
        t.Errorf("GenerateEd25519() error = %v", err)
    }

    if kp.Type() != sagecrypto.KeyTypeEd25519 {
        t.Errorf("key type = %v, want Ed25519", kp.Type())
    }
}

func TestManager_RoundTrip(t *testing.T) {
    tmpDir := t.TempDir()
    keyPath := filepath.Join(tmpDir, "test.pem")

    m, _ := NewManager()

    // Generate
    original, _ := m.GenerateEd25519()

    // Save
    if err := m.SaveToFile(original, keyPath); err != nil {
        t.Fatalf("SaveToFile() failed: %v", err)
    }

    // Load
    loaded, err := m.LoadFromFile(keyPath)
    if err != nil {
        t.Fatalf("LoadFromFile() failed: %v", err)
    }

    // Compare public keys
    if !bytes.Equal(original.PublicKey(), loaded.PublicKey()) {
        t.Error("public keys do not match")
    }
}
```

---

### 1.2 DID Integration

#### Package Structure

```
sage-adk/internal/did/
├── manager.go          # Main DID manager
├── manager_test.go     # Unit tests
├── resolver.go         # DID resolution
├── resolver_test.go
├── cache.go            # DID document cache
├── cache_test.go
└── doc.go              # Package documentation
```

#### API Design

```go
// Package did provides DID management for SAGE-ADK
package did

import (
    sagedid "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage-adk/internal/crypto"
)

// Manager wraps sage/did.Manager
type Manager struct {
    inner sagedid.Manager
    cache *Cache
}

// Config holds DID manager configuration
type Config struct {
    Network         string        // ethereum, solana, kaia
    RPCEndpoint     string        // RPC URL
    ContractAddress string        // Registry contract address
    CacheTTL        time.Duration // Cache TTL
}

// NewManager creates a new DID manager
func NewManager(cfg *Config) (*Manager, error) {
    // Create sage DID manager
    sageConfig := &sagedid.Config{
        Network:         cfg.Network,
        RPCEndpoint:     cfg.RPCEndpoint,
        ContractAddress: cfg.ContractAddress,
    }

    inner, err := sagedid.NewManager(sageConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create DID manager: %w", err)
    }

    cache := NewCache(cfg.CacheTTL)

    return &Manager{
        inner: inner,
        cache: cache,
    }, nil
}

// Resolve resolves a DID to a DID document
func (m *Manager) Resolve(ctx context.Context, did string) (*sagedid.Document, error) {
    // Check cache first
    if doc, ok := m.cache.Get(did); ok {
        return doc, nil
    }

    // Resolve from blockchain
    doc, err := m.inner.Resolve(ctx, did)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve DID %s: %w", did, err)
    }

    // Cache result
    m.cache.Set(did, doc)

    return doc, nil
}

// Register registers a DID on the blockchain
func (m *Manager) Register(ctx context.Context, did string, publicKey []byte) error {
    return m.inner.Register(ctx, did, publicKey)
}

// Update updates a DID document on the blockchain
func (m *Manager) Update(ctx context.Context, did string, doc *sagedid.Document) error {
    // Clear cache
    m.cache.Delete(did)

    return m.inner.Update(ctx, did, doc)
}
```

#### Cache Implementation

```go
// File: internal/did/cache.go

import (
    "sync"
    "time"
)

type cacheEntry struct {
    doc       *sagedid.Document
    expiresAt time.Time
}

type Cache struct {
    entries map[string]*cacheEntry
    ttl     time.Duration
    mu      sync.RWMutex
}

func NewCache(ttl time.Duration) *Cache {
    c := &Cache{
        entries: make(map[string]*cacheEntry),
        ttl:     ttl,
    }

    // Start cleanup goroutine
    go c.cleanup()

    return c
}

func (c *Cache) Get(did string) (*sagedid.Document, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, ok := c.entries[did]
    if !ok {
        return nil, false
    }

    if time.Now().After(entry.expiresAt) {
        return nil, false
    }

    return entry.doc, true
}

func (c *Cache) Set(did string, doc *sagedid.Document) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.entries[did] = &cacheEntry{
        doc:       doc,
        expiresAt: time.Now().Add(c.ttl),
    }
}

func (c *Cache) cleanup() {
    ticker := time.NewTicker(c.ttl / 2)
    defer ticker.Stop()

    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for did, entry := range c.entries {
            if now.After(entry.expiresAt) {
                delete(c.entries, did)
            }
        }
        c.mu.Unlock()
    }
}
```

---

### 1.3 RFC 9421 Integration

#### Package Structure

```
sage-adk/internal/rfc9421/
├── signer.go           # Message signing
├── signer_test.go
├── verifier.go         # Message verification
├── verifier_test.go
└── doc.go              # Package documentation
```

#### API Design

```go
// Package rfc9421 provides RFC 9421 HTTP message signatures
package rfc9421

import (
    "github.com/sage-x-project/sage/core/rfc9421"
    "github.com/sage-x-project/sage-adk/internal/crypto"
)

// Signer signs HTTP messages
type Signer struct {
    service *rfc9421.SignatureService
}

// NewSigner creates a new message signer
func NewSigner(keyPair crypto.KeyPair) (*Signer, error) {
    service, err := rfc9421.NewSignatureService(keyPair)
    if err != nil {
        return nil, fmt.Errorf("failed to create signature service: %w", err)
    }

    return &Signer{service: service}, nil
}

// Sign signs a message
func (s *Signer) Sign(ctx context.Context, msg *types.Message) error {
    // Prepare signature input
    input := &rfc9421.SignatureInput{
        Method:  "POST",
        Path:    "/messages",
        Headers: msg.Headers,
        Body:    msg.Content,
    }

    // Generate signature
    sig, err := s.service.Sign(ctx, input)
    if err != nil {
        return fmt.Errorf("failed to sign message: %w", err)
    }

    // Attach signature to message
    if msg.Security == nil {
        msg.Security = &types.SecurityMetadata{}
    }
    msg.Security.Signature = sig.Signature
    msg.Security.SignatureInput = sig.Input

    return nil
}

// Verifier verifies HTTP messages
type Verifier struct {
    service *rfc9421.VerificationService
}

// NewVerifier creates a new message verifier
func NewVerifier(publicKey []byte) (*Verifier, error) {
    service, err := rfc9421.NewVerificationService(publicKey)
    if err != nil {
        return nil, fmt.Errorf("failed to create verification service: %w", err)
    }

    return &Verifier{service: service}, nil
}

// Verify verifies a message signature
func (v *Verifier) Verify(ctx context.Context, msg *types.Message) error {
    if msg.Security == nil {
        return errors.New("missing security metadata")
    }

    // Prepare verification input
    input := &rfc9421.VerificationInput{
        Method:         "POST",
        Path:           "/messages",
        Headers:        msg.Headers,
        Body:           msg.Content,
        Signature:      msg.Security.Signature,
        SignatureInput: msg.Security.SignatureInput,
    }

    // Verify signature
    if err := v.service.Verify(ctx, input); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    return nil
}
```

---

## Phase 2: High Priority Features

### 2.1 Session Management

#### Package Structure

```
sage-adk/internal/session/
├── manager.go          # Session manager
├── manager_test.go
├── session.go          # Session type
├── session_test.go
├── encryption.go       # Encryption/decryption
├── encryption_test.go
└── doc.go
```

#### API Design

```go
// Package session provides session management for SAGE-ADK
package session

import (
    sagesession "github.com/sage-x-project/sage/session"
)

// Manager manages SAGE sessions
type Manager struct {
    inner    sagesession.Manager
    sessions map[string]*Session
    mu       sync.RWMutex
}

// Session represents a SAGE session
type Session struct {
    ID           string
    RemoteDID    string
    LocalDID     string
    SharedSecret []byte
    CreatedAt    time.Time
    ExpiresAt    time.Time
    Metadata     map[string]interface{}
}

// NewManager creates a new session manager
func NewManager(cryptoManager *crypto.Manager) (*Manager, error) {
    inner, err := sagesession.NewManager(cryptoManager)
    if err != nil {
        return nil, fmt.Errorf("failed to create session manager: %w", err)
    }

    m := &Manager{
        inner:    inner,
        sessions: make(map[string]*Session),
    }

    // Start cleanup goroutine
    go m.cleanup()

    return m, nil
}

// Create creates a new session
func (m *Manager) Create(ctx context.Context, localDID, remoteDID string, sharedSecret []byte) (*Session, error) {
    sess := &Session{
        ID:           uuid.New().String(),
        RemoteDID:    remoteDID,
        LocalDID:     localDID,
        SharedSecret: sharedSecret,
        CreatedAt:    time.Now(),
        ExpiresAt:    time.Now().Add(24 * time.Hour),
        Metadata:     make(map[string]interface{}),
    }

    m.mu.Lock()
    m.sessions[sess.ID] = sess
    m.mu.Unlock()

    return sess, nil
}

// Get retrieves a session by ID
func (m *Manager) Get(sessionID string) (*Session, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    sess, ok := m.sessions[sessionID]
    if !ok {
        return nil, fmt.Errorf("session not found: %s", sessionID)
    }

    if time.Now().After(sess.ExpiresAt) {
        return nil, fmt.Errorf("session expired: %s", sessionID)
    }

    return sess, nil
}

// Encrypt encrypts data using session key
func (m *Manager) Encrypt(ctx context.Context, sessionID string, data []byte) ([]byte, error) {
    sess, err := m.Get(sessionID)
    if err != nil {
        return nil, err
    }

    return m.inner.Encrypt(sess.SharedSecret, data)
}

// Decrypt decrypts data using session key
func (m *Manager) Decrypt(ctx context.Context, sessionID string, ciphertext []byte) ([]byte, error) {
    sess, err := m.Get(sessionID)
    if err != nil {
        return nil, err
    }

    return m.inner.Decrypt(sess.SharedSecret, ciphertext)
}
```

---

### 2.2 Handshake Protocol

#### Package Structure

```
sage-adk/internal/handshake/
├── manager.go          # Handshake manager
├── manager_test.go
├── phases.go           # 4 handshake phases
├── phases_test.go
├── state.go            # State machine
├── state_test.go
└── doc.go
```

#### API Design

```go
// Package handshake implements the SAGE 4-phase handshake protocol
package handshake

import (
    sagehandshake "github.com/sage-x-project/sage/handshake"
)

// Manager manages SAGE handshakes
type Manager struct {
    inner          sagehandshake.Manager
    cryptoManager  *crypto.Manager
    didManager     *did.Manager
    sessionManager *session.Manager
    handshakes     map[string]*State
    mu             sync.RWMutex
}

// State represents handshake state
type State struct {
    ID        string
    Phase     Phase
    LocalDID  string
    RemoteDID string
    Nonce     string
    CreatedAt time.Time
}

// Phase represents handshake phase
type Phase int

const (
    PhaseNone Phase = iota
    PhaseInvite
    PhaseAccept
    PhaseConfirm
    PhaseComplete
)

// NewManager creates a new handshake manager
func NewManager(
    cm *crypto.Manager,
    dm *did.Manager,
    sm *session.Manager,
) (*Manager, error) {
    inner, err := sagehandshake.NewManager(cm, dm, sm)
    if err != nil {
        return nil, fmt.Errorf("failed to create handshake manager: %w", err)
    }

    return &Manager{
        inner:          inner,
        cryptoManager:  cm,
        didManager:     dm,
        sessionManager: sm,
        handshakes:     make(map[string]*State),
    }, nil
}

// InitiateHandshake initiates a handshake (Phase 1: INVITE)
func (m *Manager) InitiateHandshake(ctx context.Context, remoteDID string) (*types.Message, error) {
    // Create handshake state
    state := &State{
        ID:        uuid.New().String(),
        Phase:     PhaseInvite,
        RemoteDID: remoteDID,
        Nonce:     generateNonce(),
        CreatedAt: time.Now(),
    }

    // Generate INVITE message
    msg, err := m.inner.CreateInvite(ctx, remoteDID, state.Nonce)
    if err != nil {
        return nil, fmt.Errorf("failed to create INVITE: %w", err)
    }

    // Store state
    m.mu.Lock()
    m.handshakes[state.ID] = state
    m.mu.Unlock()

    return msg, nil
}

// HandleAccept handles ACCEPT message (Phase 2)
func (m *Manager) HandleAccept(ctx context.Context, msg *types.Message) (*types.Message, error) {
    // Extract handshake ID from message
    handshakeID := msg.Headers["X-Handshake-ID"]

    m.mu.RLock()
    state, ok := m.handshakes[handshakeID]
    m.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("handshake not found: %s", handshakeID)
    }

    if state.Phase != PhaseInvite {
        return nil, fmt.Errorf("invalid phase: %v", state.Phase)
    }

    // Verify ACCEPT message
    if err := m.inner.VerifyAccept(ctx, msg, state.Nonce); err != nil {
        return nil, fmt.Errorf("ACCEPT verification failed: %w", err)
    }

    // Generate CONFIRM message
    confirm, err := m.inner.CreateConfirm(ctx, msg)
    if err != nil {
        return nil, fmt.Errorf("failed to create CONFIRM: %w", err)
    }

    // Update state
    m.mu.Lock()
    state.Phase = PhaseConfirm
    m.mu.Unlock()

    return confirm, nil
}

// HandleComplete handles COMPLETE message (Phase 4)
func (m *Manager) HandleComplete(ctx context.Context, msg *types.Message) error {
    handshakeID := msg.Headers["X-Handshake-ID"]

    m.mu.Lock()
    state, ok := m.handshakes[handshakeID]
    m.mu.Unlock()

    if !ok {
        return fmt.Errorf("handshake not found: %s", handshakeID)
    }

    if state.Phase != PhaseConfirm {
        return fmt.Errorf("invalid phase: %v", state.Phase)
    }

    // Verify COMPLETE message
    if err := m.inner.VerifyComplete(ctx, msg); err != nil {
        return fmt.Errorf("COMPLETE verification failed: %w", err)
    }

    // Create session
    sharedSecret := deriveSharedSecret(msg)
    _, err := m.sessionManager.Create(ctx, state.LocalDID, state.RemoteDID, sharedSecret)
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }

    // Update state
    m.mu.Lock()
    state.Phase = PhaseComplete
    delete(m.handshakes, handshakeID) // Remove completed handshake
    m.mu.Unlock()

    return nil
}
```

---

### 2.5 Complete SAGE Adapter

#### Updated Adapter Implementation

```go
// File: adapters/sage/adapter.go

type Adapter struct {
    config         *Config
    cryptoManager  *crypto.Manager
    didManager     *did.Manager
    sessionManager *session.Manager
    handshakeManager *handshake.Manager
    signer         *rfc9421.Signer
    validator      *validation.Validator
    mu             sync.RWMutex
}

// NewAdapter creates a new SAGE adapter
func NewAdapter(cfg *Config) (*Adapter, error) {
    // Initialize crypto manager
    cm, err := crypto.NewManager()
    if err != nil {
        return nil, fmt.Errorf("failed to create crypto manager: %w", err)
    }

    // Load key pair
    keyPair, err := cm.LoadFromFile(cfg.PrivateKeyPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load key: %w", err)
    }

    // Initialize DID manager
    didConfig := &did.Config{
        Network:         cfg.Network,
        RPCEndpoint:     cfg.RPCEndpoint,
        ContractAddress: cfg.ContractAddress,
        CacheTTL:        cfg.CacheTTL,
    }
    dm, err := did.NewManager(didConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create DID manager: %w", err)
    }

    // Initialize session manager
    sm, err := session.NewManager(cm)
    if err != nil {
        return nil, fmt.Errorf("failed to create session manager: %w", err)
    }

    // Initialize handshake manager
    hm, err := handshake.NewManager(cm, dm, sm)
    if err != nil {
        return nil, fmt.Errorf("failed to create handshake manager: %w", err)
    }

    // Initialize signer
    signer, err := rfc9421.NewSigner(keyPair)
    if err != nil {
        return nil, fmt.Errorf("failed to create signer: %w", err)
    }

    // Initialize validator
    validator := validation.NewValidator()

    return &Adapter{
        config:           cfg,
        cryptoManager:    cm,
        didManager:       dm,
        sessionManager:   sm,
        handshakeManager: hm,
        signer:           signer,
        validator:        validator,
    }, nil
}

// SendMessage sends a message using SAGE protocol
func (a *Adapter) SendMessage(ctx context.Context, msg *types.Message) error {
    a.mu.RLock()
    defer a.mu.RUnlock()

    // Sign message with RFC 9421
    if err := a.signer.Sign(ctx, msg); err != nil {
        return fmt.Errorf("failed to sign message: %w", err)
    }

    // If session exists, encrypt content
    if msg.SessionID != "" {
        encrypted, err := a.sessionManager.Encrypt(ctx, msg.SessionID, []byte(msg.Content))
        if err != nil {
            return fmt.Errorf("failed to encrypt message: %w", err)
        }
        msg.Content = string(encrypted)
        msg.Encrypted = true
    }

    // Send via transport
    // TODO: Implement transport layer

    return nil
}

// ReceiveMessage receives a message using SAGE protocol
func (a *Adapter) ReceiveMessage(ctx context.Context) (*types.Message, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    // Receive from transport
    // TODO: Implement transport layer
    var msg *types.Message

    // Verify message
    if err := a.Verify(ctx, msg); err != nil {
        return nil, fmt.Errorf("message verification failed: %w", err)
    }

    // If encrypted, decrypt content
    if msg.Encrypted && msg.SessionID != "" {
        decrypted, err := a.sessionManager.Decrypt(ctx, msg.SessionID, []byte(msg.Content))
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt message: %w", err)
        }
        msg.Content = string(decrypted)
    }

    return msg, nil
}

// Verify verifies a message
func (a *Adapter) Verify(ctx context.Context, msg *types.Message) error {
    // Basic validation
    if msg.Security == nil {
        return errors.New("missing security metadata")
    }

    // Resolve sender's DID to get public key
    doc, err := a.didManager.Resolve(ctx, msg.From)
    if err != nil {
        return fmt.Errorf("failed to resolve sender DID: %w", err)
    }

    publicKey := extractPublicKey(doc)

    // Create verifier with sender's public key
    verifier, err := rfc9421.NewVerifier(publicKey)
    if err != nil {
        return fmt.Errorf("failed to create verifier: %w", err)
    }

    // Verify RFC 9421 signature
    if err := verifier.Verify(ctx, msg); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }

    // Validate message (nonce, replay, ordering)
    if err := a.validator.Validate(ctx, msg); err != nil {
        return fmt.Errorf("message validation failed: %w", err)
    }

    return nil
}
```

---

## Testing Strategy

### Unit Testing

All packages must have ≥90% test coverage using table-driven tests:

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {"success case", validInput, expectedOutput, false},
        {"error case", invalidInput, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Testing

Create integration tests in `test/integration/sage/`:

```go
//go:build integration

func TestIntegration_CompleteHandshake(t *testing.T) {
    // Setup two agents
    agent1 := setupAgent(t, "did:sage:ethereum:0x001")
    agent2 := setupAgent(t, "did:sage:ethereum:0x002")

    // Agent 1 initiates handshake
    invite, err := agent1.InitiateHandshake(ctx, agent2.DID)
    if err != nil {
        t.Fatalf("InitiateHandshake() failed: %v", err)
    }

    // Agent 2 accepts handshake
    accept, err := agent2.AcceptHandshake(ctx, invite)
    if err != nil {
        t.Fatalf("AcceptHandshake() failed: %v", err)
    }

    // Agent 1 confirms
    confirm, err := agent1.ConfirmHandshake(ctx, accept)
    if err != nil {
        t.Fatalf("ConfirmHandshake() failed: %v", err)
    }

    // Agent 2 completes
    if err := agent2.CompleteHandshake(ctx, confirm); err != nil {
        t.Fatalf("CompleteHandshake() failed: %v", err)
    }

    // Verify session established
    if agent1.SessionID == "" || agent2.SessionID == "" {
        t.Error("session not established")
    }
}
```

---

## Migration Guide

### For Existing sage-adk Users

#### Before (A2A only)
```go
agent, err := adk.NewBuilder().
    WithProtocol(protocol.A2A).
    WithLLM(llmAdapter).
    Build()
```

#### After (with SAGE security)
```go
cryptoManager, _ := crypto.NewManager()
keyPair, _ := cryptoManager.GenerateEd25519()

didManager, _ := did.NewManager(&did.Config{
    Network:     "ethereum",
    RPCEndpoint: "https://eth.example.com",
})

agent, err := adk.NewBuilder().
    WithProtocol(protocol.SAGE).
    WithLLM(llmAdapter).
    WithKeyManager(cryptoManager).
    WithKeyPair(keyPair).
    WithDIDManager(didManager).
    WithDID("did:sage:ethereum:0x123").
    Build()
```

---

## Risk Management

### Breaking Changes
- All new SAGE features are additive
- Existing A2A functionality unchanged
- No breaking API changes in Phase 1-3

### Performance Concerns
- Benchmark all crypto operations
- Optimize hot paths
- Use connection pooling for blockchain calls
- Cache DID documents aggressively

### Security Risks
- Full code review before merge
- Security audit of crypto integration
- Penetration testing of handshake protocol
- Dependency vulnerability scanning

---

**Last Updated**: 2025-10-10
**Next Review**: After Phase 1 completion
