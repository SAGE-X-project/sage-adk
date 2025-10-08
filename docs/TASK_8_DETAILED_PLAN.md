# Task 8: SAGE Configuration & DID Management - Detailed Implementation Plan

**Version**: 2.0
**Date**: 2025-10-07
**Status**: Ready to Start
**Duration**: 2 days
**Dependencies**: Task 7 (COMPLETE), Design Documents Review (COMPLETE)

---

## Pre-Implementation Review Summary

### Design Document Analysis Complete

Based on review of all 10 design documents, current implementation status:

**✅ Fully Implemented (7/10)**:
1. Core Types (design-20251007-001510-v1.0.md) - 100%
2. Error Types (design-20251007-003656-v1.0.md) - 100%
3. Protocol Layer (design-20251007-024648-v1.0.md) - 100%
4. A2A Adapter (design-20251007-030000-v1.0.md) - 100%
5. SAGE Adapter Base (design-20251007-033000-v1.0.md) - 100%
6. SAGE Transport (design-20251007-sage-transport-v1.0.md) - 100%
7. Storage Layer (design-20251007-040000-v1.0.md) - 100%

**⚠️ Partially Implemented (2/10)**:
1. Configuration (design-20251007-005132-v1.0.md) - 70% (types done, loader missing)
2. LLM Provider (design-20251007-035000-v1.0.md) - 40% (OpenAI done, others pending)

**❌ Gaps Identified**:
1. Agent advanced features (tools, middleware) - Phase 2C/2D
2. Configuration file loading - **CRITICAL for Task 8**
3. Anthropic/Gemini LLM providers - Phase 2C

### SAGE Library Analysis

**Available in @sage/ (github.com/sage-x-project/sage)**:
- ✅ `crypto.Manager` - Full key management
- ✅ `crypto.KeyPair` - Ed25519, Secp256k1, X25519
- ✅ `crypto.KeyStorage` - Memory and file storage
- ✅ `crypto/formats` - PEM and JWK import/export
- ✅ `did.Resolver` - DID resolution interface
- ✅ `did.MultiChainResolver` - Multi-chain support
- ✅ `did.AgentMetadata` - Complete metadata structure
- ✅ `did/ethereum.Resolver` - Ethereum blockchain resolution
- ✅ `config.Config` - Complete configuration structure
- ✅ `config.LoadFromFile()` - YAML/JSON loading

**Conclusion**: All required functionality exists in @sage/. Task 8 = wrapping + integration.

---

## Task 8 Scope (Revised Based on Review)

### What Task 8 Will Deliver

1. **Configuration Bridge** (`adapters/sage/config.go`):
   - Map ADK's `config.SAGEConfig` → sage library's `config.Config`
   - Validation and defaults

2. **Key Management Wrapper** (`adapters/sage/keys.go`):
   - Simplified API wrapping `crypto.Manager`
   - PEM format import/export
   - File-based key storage

3. **DID Resolution Wrapper** (`adapters/sage/did.go`):
   - Wrap `did.MultiChainResolver`
   - Cache management
   - Type conversion (sage types → ADK types)

4. **Transport Integration** (update `adapters/sage/transport.go`):
   - Add config-based constructor
   - Integrate DID resolver
   - Use key manager for identity

5. **Builder Integration** (update `builder/builder.go`):
   - Add `WithSAGEConfig()` method
   - Wire up SAGE components
   - Validate configuration at build time

### What Task 8 Will NOT Deliver

- ❌ Real blockchain connection (use mock/cache for Phase 2B)
- ❌ Encrypted key storage (use PEM for Phase 2B, encrypt in Phase 3)
- ❌ SAGE Server implementation (Task 9)
- ❌ SAGE Example application (Task 10)
- ❌ Protocol auto-detection (Task 11)

---

## Implementation Plan

### Part 1: Configuration System (4 hours)

#### 1.1 Update ADK Config (`config/config.go`)

**Goal**: Add SAGEConfig to main configuration

**Changes**:
```go
// Add to config/config.go

// SAGEConfig represents SAGE protocol configuration
type SAGEConfig struct {
    // Identity
    DID             string `json:"did" yaml:"did" env:"SAGE_DID"`
    PrivateKeyPath  string `json:"private_key_path" yaml:"private_key_path" env:"SAGE_PRIVATE_KEY_PATH"`

    // Blockchain
    Network         string `json:"network" yaml:"network" env:"SAGE_NETWORK"`
    RPCEndpoint     string `json:"rpc_endpoint" yaml:"rpc_endpoint" env:"SAGE_RPC_ENDPOINT"`
    ContractAddress string `json:"contract_address" yaml:"contract_address" env:"SAGE_CONTRACT_ADDRESS"`

    // Caching
    CacheExpiry     int    `json:"cache_expiry" yaml:"cache_expiry" env:"SAGE_CACHE_EXPIRY"` // seconds

    // Optional
    KeyPassword     string `json:"-" yaml:"-" env:"SAGE_KEY_PASSWORD"` // Never serialize
}

// Validate validates SAGEConfig
func (c *SAGEConfig) Validate() error {
    if c.DID == "" {
        return errors.ErrMissingConfig.WithMessage("DID is required")
    }
    if c.PrivateKeyPath == "" {
        return errors.ErrMissingConfig.WithMessage("private_key_path is required")
    }
    if c.Network == "" {
        c.Network = "ethereum" // default
    }
    if c.CacheExpiry == 0 {
        c.CacheExpiry = 300 // 5 minutes default
    }
    return nil
}

// Add to Config struct
type Config struct {
    // ... existing fields ...
    SAGE *SAGEConfig `json:"sage,omitempty" yaml:"sage,omitempty"`
}
```

**New File**: `config/sage.go`
```go
package config

import (
    "time"
    sagecfg "github.com/sage-x-project/sage/config"
)

// ToSAGELibraryConfig converts ADK SAGEConfig to sage library Config
func (c *SAGEConfig) ToSAGELibraryConfig() (*sagecfg.Config, error) {
    if err := c.Validate(); err != nil {
        return nil, err
    }

    cfg := &sagecfg.Config{
        Environment: "production",
    }

    // Blockchain config
    if c.RPCEndpoint != "" {
        cfg.Blockchain = &sagecfg.BlockchainConfig{
            Network: c.Network,
            RPC:     c.RPCEndpoint,
            Contract: c.ContractAddress,
        }
    }

    // DID config
    cfg.DID = &sagecfg.DIDConfig{
        RegistryAddress: c.ContractAddress,
        Network:         c.Network,
        CacheTTL:        time.Duration(c.CacheExpiry) * time.Second,
        CacheSize:       100, // default
    }

    // KeyStore config
    if c.PrivateKeyPath != "" {
        cfg.KeyStore = &sagecfg.KeyStoreConfig{
            Directory: filepath.Dir(c.PrivateKeyPath),
            Type:      "file",
        }
    }

    return cfg, nil
}
```

**Tests**: `config/sage_test.go`
- Validate() with valid config
- Validate() with missing required fields
- ToSAGELibraryConfig() conversion
- Default values application

**Time**: 2 hours

---

#### 1.2 SAGE Config Mapping (`adapters/sage/config.go`)

**Goal**: Bridge between ADK and SAGE library configuration

**New File**: `adapters/sage/config.go`
```go
package sage

import (
    "path/filepath"
    "time"

    sagecfg "github.com/sage-x-project/sage/config"
    adkconfig "github.com/sage-x-project/sage-adk/config"
)

// Config wraps sage library config with ADK-specific extensions
type Config struct {
    // Embed sage library config
    *sagecfg.Config

    // ADK-specific fields
    LocalDID        string
    PrivateKeyPath  string
    KeyPassword     string
}

// NewConfigFromADK creates SAGE config from ADK SAGEConfig
func NewConfigFromADK(adkCfg *adkconfig.SAGEConfig) (*Config, error) {
    if adkCfg == nil {
        return nil, errors.ErrMissingConfig.WithMessage("SAGEConfig is required")
    }

    // Convert to sage library config
    sageCfg, err := adkCfg.ToSAGELibraryConfig()
    if err != nil {
        return nil, err
    }

    return &Config{
        Config:         sageCfg,
        LocalDID:       adkCfg.DID,
        PrivateKeyPath: adkCfg.PrivateKeyPath,
        KeyPassword:    adkCfg.KeyPassword,
    }, nil
}

// Validate performs additional validation
func (c *Config) Validate() error {
    if c.LocalDID == "" {
        return errors.ErrMissingConfig.WithMessage("local DID required")
    }
    if c.PrivateKeyPath == "" {
        return errors.ErrMissingConfig.WithMessage("private key path required")
    }
    return nil
}
```

**Tests**: `adapters/sage/config_test.go`
- NewConfigFromADK() with valid config
- NewConfigFromADK() with nil config
- Validate() errors
- Config field mapping correctness

**Time**: 2 hours

---

### Part 2: Key Management (3 hours)

#### 2.1 Key Manager Wrapper (`adapters/sage/keys.go`)

**Goal**: Simplified key management API wrapping sage crypto library

**New File**: `adapters/sage/keys.go`
```go
package sage

import (
    "crypto/ed25519"
    "os"

    "github.com/sage-x-project/sage/crypto"
)

// KeyManager wraps sage crypto.Manager for simplified key operations
type KeyManager struct {
    manager *crypto.Manager
}

// NewKeyManager creates a new key manager
func NewKeyManager() *KeyManager {
    return &KeyManager{
        manager: crypto.NewManager(),
    }
}

// Generate creates a new Ed25519 key pair
func (km *KeyManager) Generate() (crypto.KeyPair, error) {
    return km.manager.GenerateKeyPair(crypto.KeyTypeEd25519)
}

// LoadFromFile loads a key pair from PEM file
func (km *KeyManager) LoadFromFile(path string, password string) (crypto.KeyPair, error) {
    // Read file
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, errors.ErrOperationFailed.
            WithMessage("failed to read key file").
            WithDetails(map[string]interface{}{"path": path, "error": err.Error()})
    }

    // Import using sage PEM importer
    keyPair, err := km.manager.ImportKeyPair(data, crypto.KeyFormatPEM)
    if err != nil {
        return nil, errors.ErrOperationFailed.
            WithMessage("failed to import key from PEM").
            WithDetails(map[string]interface{}{"error": err.Error()})
    }

    return keyPair, nil
}

// SaveToFile saves a key pair to PEM file
func (km *KeyManager) SaveToFile(keyPair crypto.KeyPair, path string, password string) error {
    // Export using sage PEM exporter
    data, err := km.manager.ExportKeyPair(keyPair, crypto.KeyFormatPEM)
    if err != nil {
        return errors.ErrOperationFailed.
            WithMessage("failed to export key to PEM").
            WithDetails(map[string]interface{}{"error": err.Error()})
    }

    // Write file with restrictive permissions
    if err := os.WriteFile(path, data, 0600); err != nil {
        return errors.ErrOperationFailed.
            WithMessage("failed to write key file").
            WithDetails(map[string]interface{}{"path": path, "error": err.Error()})
    }

    return nil
}

// GetPrivateKey extracts Ed25519 private key from KeyPair
func (km *KeyManager) GetPrivateKey(keyPair crypto.KeyPair) (ed25519.PrivateKey, error) {
    privateKey := keyPair.Private()

    ed25519Key, ok := privateKey.(ed25519.PrivateKey)
    if !ok {
        return nil, errors.ErrOperationFailed.
            WithMessage("key pair does not contain Ed25519 private key")
    }

    return ed25519Key, nil
}

// GetPublicKey extracts Ed25519 public key from KeyPair
func (km *KeyManager) GetPublicKey(keyPair crypto.KeyPair) (ed25519.PublicKey, error) {
    publicKey := keyPair.Public()

    ed25519Key, ok := publicKey.(ed25519.PublicKey)
    if !ok {
        return nil, errors.ErrOperationFailed.
            WithMessage("key pair does not contain Ed25519 public key")
    }

    return ed25519Key, nil
}
```

**Tests**: `adapters/sage/keys_test.go`
- Generate() creates valid Ed25519 key
- SaveToFile() creates file with 0600 permissions
- LoadFromFile() reads key correctly
- Generate → Save → Load round-trip
- GetPrivateKey() extracts correct key
- GetPublicKey() extracts correct key
- Error handling (invalid path, corrupted file)

**Time**: 3 hours

---

### Part 3: DID Resolution (4 hours)

#### 3.1 DID Resolver Wrapper (`adapters/sage/did.go`)

**Goal**: Wrap sage DID library with simplified API and caching

**New File**: `adapters/sage/did.go`
```go
package sage

import (
    "context"
    "crypto/ed25519"
    "fmt"
    "sync"
    "time"

    sagedid "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage/did/ethereum"
)

// DIDResolver wraps sage did.MultiChainResolver
type DIDResolver struct {
    resolver    *sagedid.MultiChainResolver
    cache       map[string]*cachedMetadata
    cacheMu     sync.RWMutex
    cacheTTL    time.Duration
}

type cachedMetadata struct {
    metadata  *sagedid.AgentMetadata
    cachedAt  time.Time
}

// NewDIDResolver creates a new DID resolver
func NewDIDResolver(cfg *Config) (*DIDResolver, error) {
    resolver := sagedid.NewMultiChainResolver()

    // Add Ethereum resolver if configured
    if cfg.Blockchain != nil && cfg.Blockchain.Network == "ethereum" {
        ethResolver, err := ethereum.NewResolver(cfg.Config)
        if err != nil {
            return nil, errors.ErrOperationFailed.
                WithMessage("failed to create Ethereum resolver").
                WithDetails(map[string]interface{}{"error": err.Error()})
        }
        resolver.AddResolver(sagedid.ChainEthereum, ethResolver)
    }

    // TODO Phase 3: Add Solana resolver

    cacheTTL := 5 * time.Minute
    if cfg.DID != nil && cfg.DID.CacheTTL > 0 {
        cacheTTL = cfg.DID.CacheTTL
    }

    return &DIDResolver{
        resolver: resolver,
        cache:    make(map[string]*cachedMetadata),
        cacheTTL: cacheTTL,
    }, nil
}

// Resolve resolves a DID to agent metadata
func (r *DIDResolver) Resolve(ctx context.Context, didStr string) (*sagedid.AgentMetadata, error) {
    // Check cache first
    r.cacheMu.RLock()
    cached, exists := r.cache[didStr]
    r.cacheMu.RUnlock()

    if exists && time.Since(cached.cachedAt) < r.cacheTTL {
        return cached.metadata, nil
    }

    // Resolve from blockchain
    metadata, err := r.resolver.Resolve(ctx, sagedid.AgentDID(didStr))
    if err != nil {
        return nil, errors.ErrOperationFailed.
            WithMessage("DID resolution failed").
            WithDetails(map[string]interface{}{"did": didStr, "error": err.Error()})
    }

    // Verify agent is active
    if !metadata.IsActive {
        return nil, errors.ErrOperationFailed.
            WithMessage("agent is not active").
            WithDetails(map[string]interface{}{"did": didStr})
    }

    // Cache result
    r.cacheMu.Lock()
    r.cache[didStr] = &cachedMetadata{
        metadata: metadata,
        cachedAt: time.Now(),
    }
    r.cacheMu.Unlock()

    return metadata, nil
}

// ResolvePublicKey resolves a DID and extracts the public key
func (r *DIDResolver) ResolvePublicKey(ctx context.Context, didStr string) (ed25519.PublicKey, error) {
    metadata, err := r.Resolve(ctx, didStr)
    if err != nil {
        return nil, err
    }

    // Extract Ed25519 public key
    pubKey, ok := metadata.PublicKey.(ed25519.PublicKey)
    if !ok {
        return nil, errors.ErrOperationFailed.
            WithMessage("public key is not Ed25519 type").
            WithDetails(map[string]interface{}{"did": didStr})
    }

    return pubKey, nil
}

// Register manually registers a DID in cache (for testing/development)
func (r *DIDResolver) Register(didStr string, publicKey ed25519.PublicKey) {
    r.cacheMu.Lock()
    defer r.cacheMu.Unlock()

    r.cache[didStr] = &cachedMetadata{
        metadata: &sagedid.AgentMetadata{
            DID:       sagedid.AgentDID(didStr),
            PublicKey: publicKey,
            IsActive:  true,
        },
        cachedAt: time.Now(),
    }
}

// ClearCache clears the DID cache
func (r *DIDResolver) ClearCache() {
    r.cacheMu.Lock()
    defer r.cacheMu.Unlock()
    r.cache = make(map[string]*cachedMetadata)
}
```

**Tests**: `adapters/sage/did_test.go`
- NewDIDResolver() with valid config
- NewDIDResolver() with Ethereum config
- Resolve() with cached DID
- Resolve() with uncached DID (mock)
- ResolvePublicKey() extracts key correctly
- Register() for manual DID registration
- Cache expiration behavior
- Inactive agent error handling
- ClearCache() works

**Time**: 4 hours

---

### Part 4: Transport Integration (3 hours)

#### 4.1 Update Transport Manager (`adapters/sage/transport.go`)

**Goal**: Add config-based constructor and DID integration

**Changes to existing file**:
```go
// Add new constructor
func NewTransportManagerWithConfig(cfg *Config) (*TransportManager, error) {
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    // Load key pair
    keyManager := NewKeyManager()
    keyPair, err := keyManager.LoadFromFile(cfg.PrivateKeyPath, cfg.KeyPassword)
    if err != nil {
        return nil, fmt.Errorf("failed to load key: %w", err)
    }

    // Extract Ed25519 private key
    privateKey, err := keyManager.GetPrivateKey(keyPair)
    if err != nil {
        return nil, err
    }

    // Create DID resolver
    didResolver, err := NewDIDResolver(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create DID resolver: %w", err)
    }

    // Create transport config from SAGE config
    transportCfg := DefaultTransportConfig()
    if cfg.DID != nil && cfg.DID.CacheTTL > 0 {
        transportCfg.SessionTTL = cfg.DID.CacheTTL
    }

    return &TransportManager{
        localDID:          cfg.LocalDID,
        privateKey:        privateKey,
        config:            transportCfg,
        sessionManager:    NewSessionManager(transportCfg.SessionTTL),
        encryptionManager: NewEncryptionManager(),
        signingManager:    NewSigningManager(),
        didResolver:       didResolver,
        activeHandshakes:  make(map[string]*HandshakeState),
        messageHandler:    nil,
    }, nil
}

// Update Connect to use DID resolver if available
func (tm *TransportManager) Connect(ctx context.Context, remoteDID string) (*HandshakeInvitation, error) {
    // Check for existing connection
    if session, err := tm.sessionManager.GetByDID(remoteDID); err == nil && session.IsActive() {
        return nil, errors.ErrOperationFailed.WithMessage("already connected to " + remoteDID)
    }

    // Resolve remote DID if resolver available
    if tm.didResolver != nil {
        remotePubKey, err := tm.didResolver.ResolvePublicKey(ctx, remoteDID)
        if err != nil {
            return nil, fmt.Errorf("failed to resolve remote DID: %w", err)
        }
        // Store public key for verification (will use in HandleRequest)
        // For now, log it
        _ = remotePubKey
    }

    // Continue with existing handshake logic
    invitation, err := tm.handshakeManager.InitiateHandshake(ctx, remoteDID)
    if err != nil {
        return nil, err
    }

    // Store handshake state
    tm.activeHandshakes[remoteDID] = &HandshakeState{
        Phase:      PhaseInvitation,
        Session:    session,
        StartedAt:  time.Now(),
    }

    return invitation, nil
}
```

**Add field to TransportManager struct**:
```go
type TransportManager struct {
    // ... existing fields ...
    didResolver       *DIDResolver // NEW
}
```

**Tests**: Update `adapters/sage/transport_test.go`
- NewTransportManagerWithConfig() with valid config
- NewTransportManagerWithConfig() with invalid key path
- Connect() uses DID resolver
- DID resolution errors handled properly

**Time**: 3 hours

---

### Part 5: Builder Integration (3 hours)

#### 5.1 Update Builder (`builder/builder.go`)

**Goal**: Add SAGE configuration support

**Changes**:
```go
// Add field to Builder struct
type Builder struct {
    // ... existing fields ...
    sageConfig *config.SAGEConfig
}

// Add method
func (b *Builder) WithSAGEConfig(cfg *config.SAGEConfig) *Builder {
    if cfg == nil {
        b.errors = append(b.errors, errors.ErrMissingConfig.WithMessage("SAGEConfig cannot be nil"))
        return b
    }

    if err := cfg.Validate(); err != nil {
        b.errors = append(b.errors, err)
        return b
    }

    b.sageConfig = cfg
    b.protocol = protocol.ProtocolSAGE
    return b
}

// Update Build() method to initialize SAGE transport
func (b *Builder) Build() (agent.Agent, error) {
    // ... existing validation ...

    // Initialize SAGE if configured
    var sageTransport *sage.TransportManager
    if b.protocol == protocol.ProtocolSAGE && b.sageConfig != nil {
        // Convert ADK config to SAGE config
        sageCfg, err := sage.NewConfigFromADK(b.sageConfig)
        if err != nil {
            return nil, fmt.Errorf("invalid SAGE config: %w", err)
        }

        // Create transport manager
        sageTransport, err = sage.NewTransportManagerWithConfig(sageCfg)
        if err != nil {
            return nil, fmt.Errorf("failed to create SAGE transport: %w", err)
        }

        // TODO Task 9: Create SAGE server and wire up
        // For now, just validate that transport was created
    }

    // ... rest of Build() logic ...
}
```

**Tests**: Update `builder/builder_test.go`
- WithSAGEConfig() with valid config
- WithSAGEConfig() with nil config
- WithSAGEConfig() with invalid config
- Build() creates SAGE transport
- Build() fails with missing key file

**Time**: 3 hours

---

## Test Plan

### Unit Tests

**Config** (config/sage_test.go):
- [ ] SAGEConfig.Validate() - all fields
- [ ] SAGEConfig.Validate() - missing required fields
- [ ] SAGEConfig.ToSAGELibraryConfig() - field mapping
- [ ] Default values applied correctly

**SAGE Config** (adapters/sage/config_test.go):
- [ ] NewConfigFromADK() - valid config
- [ ] NewConfigFromADK() - nil config
- [ ] Config.Validate() - all validations

**Key Manager** (adapters/sage/keys_test.go):
- [ ] Generate() - creates Ed25519 key
- [ ] SaveToFile() - file permissions 0600
- [ ] LoadFromFile() - reads key correctly
- [ ] Round-trip: Generate → Save → Load
- [ ] GetPrivateKey() - correct extraction
- [ ] GetPublicKey() - correct extraction
- [ ] Errors: invalid path, corrupted file

**DID Resolver** (adapters/sage/did_test.go):
- [ ] NewDIDResolver() - creates resolver
- [ ] Resolve() - cache hit
- [ ] Resolve() - cache miss (mock)
- [ ] ResolvePublicKey() - extracts key
- [ ] Register() - manual registration
- [ ] Cache expiration
- [ ] Inactive agent error
- [ ] ClearCache()

**Transport** (adapters/sage/transport_test.go - update):
- [ ] NewTransportManagerWithConfig() - success
- [ ] NewTransportManagerWithConfig() - invalid config
- [ ] Connect() - uses DID resolver

**Builder** (builder/builder_test.go - update):
- [ ] WithSAGEConfig() - valid
- [ ] WithSAGEConfig() - nil
- [ ] WithSAGEConfig() - invalid
- [ ] Build() - creates SAGE transport

### Integration Tests

**End-to-End** (adapters/sage/integration_sage_config_test.go - new file):
```go
func TestIntegration_SAGEConfigToTransport(t *testing.T) {
    // 1. Generate key
    km := sage.NewKeyManager()
    keyPair, _ := km.Generate()

    // 2. Save key
    tmpFile := filepath.Join(t.TempDir(), "test.key")
    km.SaveToFile(keyPair, tmpFile, "")

    // 3. Create config
    adkCfg := &config.SAGEConfig{
        DID:            "did:sage:test:alice",
        PrivateKeyPath: tmpFile,
        Network:        "ethereum",
        CacheExpiry:    60,
    }

    // 4. Build agent
    agent := builder.NewAgent("test").
        WithSAGEConfig(adkCfg).
        WithLLM(llm.Mock()).
        OnMessage(func(ctx context.Context, msg agent.MessageContext) error {
            return nil
        }).
        Build()

    // 5. Verify agent created successfully
    assert.NotNil(t, agent)
}
```

### Test Coverage Target

- Config: ≥ 90%
- Key Manager: ≥ 90%
- DID Resolver: ≥ 85%
- Transport updates: ≥ 85%
- Builder updates: ≥ 85%

---

## Success Criteria

At the end of Task 8, this code should work:

```go
// 1. Generate and save key
km := sage.NewKeyManager()
keyPair, _ := km.Generate()
km.SaveToFile(keyPair, "keys/agent.key", "")

// 2. Create ADK config
sageConfig := &config.SAGEConfig{
    DID:             "did:sage:ethereum:0x123",
    PrivateKeyPath:  "keys/agent.key",
    Network:         "ethereum",
    RPCEndpoint:     "http://localhost:8545",
    ContractAddress: "0xABC...",
    CacheExpiry:     300,
}

// 3. Build agent with SAGE
agent := builder.NewAgent("secure-agent").
    WithSAGEConfig(sageConfig).
    WithLLM(llm.OpenAI()).
    OnMessage(handleMessage).
    Build()

// 4. Agent has SAGE transport initialized
// (Full E2E messaging in Task 9)
```

---

## Implementation Schedule

### Day 1 (8 hours)

**Morning (4 hours)**:
- 08:00-10:00: Part 1.1 - Update ADK Config (2h)
- 10:00-12:00: Part 1.2 - SAGE Config Mapping (2h)

**Afternoon (4 hours)**:
- 13:00-16:00: Part 2.1 - Key Manager Wrapper (3h)
- 16:00-17:00: Write tests for Part 1 & 2 (1h)

**End of Day 1**: Config + Key Manager complete with tests

### Day 2 (8 hours)

**Morning (4 hours)**:
- 08:00-12:00: Part 3.1 - DID Resolver Wrapper (4h)

**Afternoon (4 hours)**:
- 13:00-16:00: Part 4.1 - Transport Integration (3h)
- 16:00-17:00: Part 5.1 - Builder Integration (1h)

**Evening (2 hours)**:
- 17:00-19:00: Integration tests + verification (2h)

**End of Day 2**: All components integrated, tests passing

---

## Dependencies

### External Libraries
- ✅ `github.com/sage-x-project/sage/crypto` - Key management
- ✅ `github.com/sage-x-project/sage/did` - DID resolution
- ✅ `github.com/sage-x-project/sage/config` - Configuration
- ✅ `github.com/sage-x-project/sage/did/ethereum` - Ethereum resolver

### Internal Dependencies
- ✅ Task 7: SAGE Transport Layer (COMPLETE)
- ✅ ADK config system (EXISTS, needs SAGE extension)
- ✅ ADK builder (EXISTS, needs SAGE method)
- ✅ Error types (EXISTS)

---

## Risk Assessment

### Low Risk
- Config mapping - straightforward struct conversion
- Key manager wrapper - simple API over existing code
- Tests - standard patterns

### Medium Risk
- DID resolver caching - need careful thread safety
- PEM format compatibility - must match sage library

### Mitigation
- Use sage library's own PEM format (guaranteed compatibility)
- Test caching thoroughly with concurrent access
- Mock DID resolver for unit tests

---

## Post-Task 8 Checklist

- [ ] All unit tests passing (≥85% coverage)
- [ ] Integration test passing
- [ ] `WithSAGEConfig()` works in builder
- [ ] Agent can be created with SAGE config
- [ ] Key generation/save/load round-trip works
- [ ] DID resolver initializes correctly
- [ ] Code reviewed and documented
- [ ] Commit with message: "feat(sage): implement SAGE configuration and DID management (Task 8)"

---

**Ready to Start**: ✅
**Next Step**: Begin Day 1 Morning - Part 1.1 (Update ADK Config)
