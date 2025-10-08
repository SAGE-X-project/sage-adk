# Task 8: SAGE Configuration & DID Management - Implementation Plan

**Version**: 1.0
**Date**: 2025-10-07
**Duration**: 2 days

## Analysis: What's Available in @sage/

### 1. Crypto/Key Management (@sage/crypto)

**Available**:
- `crypto.Manager`: Full key management with storage
- `crypto.KeyPair` interface with Ed25519, Secp256k1, X25519 implementations
- `crypto.KeyStorage`: Memory and file-based storage
- PEM and JWK export/import formats
- `crypto.GenerateEd25519KeyPair()`, `crypto.GenerateSecp256k1KeyPair()`

**What We Need**:
- Simple wrapper for sage-adk that uses these existing implementations
- No need to reimplement key generation or storage

### 2. DID Resolution (@sage/did)

**Available**:
- `did.Resolver` interface with full specification
- `did.MultiChainResolver`: Aggregates Ethereum and Solana resolvers
- `did.AgentMetadata`: Complete metadata structure
- `did.AgentDID` type
- `did/ethereum.Resolver`: Full Ethereum blockchain resolution
- Error types: `ErrDIDNotFound`, `ErrInactiveAgent`, etc.

**What We Need**:
- Use existing `did.MultiChainResolver` directly
- Add simple cache wrapper in sage-adk (optional, for Phase 2B simplification)
- For Phase 2B: Can use resolver without blockchain (manual registration for testing)

### 3. Configuration (@sage/config)

**Available**:
- `config.Config`: Main config structure
- `config.BlockchainConfig`: Blockchain settings
- `config.DIDConfig`: DID resolution settings with cache
- `config.KeyStoreConfig`: Key storage settings
- `LoadFromFile()`, `SaveToFile()`: YAML/JSON support

**What We Need**:
- Extend sage-adk's config to include SAGE-specific fields
- Map sage-adk config to sage library config when needed

---

## Implementation Strategy

### Principle: USE EXISTING @sage/ CODE

**DO**:
- Import and use `github.com/sage-x-project/sage/crypto` for all key operations
- Import and use `github.com/sage-x-project/sage/did` for DID resolution
- Import and use `github.com/sage-x-project/sage/config` for configuration structures

**DON'T**:
- Reimplement key generation (use `crypto.GenerateEd25519KeyPair()`)
- Reimplement DID resolution (use `did.MultiChainResolver`)
- Reimplement config loading (use `config.LoadFromFile()`)

---

## Task 8 Revised Implementation Plan

### Part 1: Configuration (adapters/sage/config.go)

**Goal**: Map sage-adk config to sage library config

```go
package sage

import (
    "github.com/sage-x-project/sage/config"
    "github.com/sage-x-project/sage/crypto"
    "github.com/sage-x-project/sage/did"
    adkconfig "github.com/sage-x-project/sage-adk/config"
)

// Config wraps sage library config with ADK-specific extensions
type Config struct {
    // Embed sage library config
    *config.Config

    // ADK-specific fields (if any)
    LocalDID string // The local agent's DID
}

// FromADKConfig converts ADK config to SAGE config
func FromADKConfig(adkCfg *adkconfig.SAGEConfig) (*Config, error) {
    // Map ADK config fields to sage config
    sageCfg := &config.Config{
        Blockchain: &config.BlockchainConfig{
            Network:  adkCfg.Network,
            RPC:      adkCfg.RPCEndpoint,
            Contract: adkCfg.ContractAddress,
        },
        DID: &config.DIDConfig{
            RegistryAddress: adkCfg.ContractAddress,
            Network:         adkCfg.Network,
            CacheTTL:        time.Duration(adkCfg.CacheExpiry) * time.Second,
        },
        KeyStore: &config.KeyStoreConfig{
            Directory: filepath.Dir(adkCfg.PrivateKeyPath),
        },
    }

    return &Config{
        Config:   sageCfg,
        LocalDID: adkCfg.DID,
    }, nil
}
```

**Files**:
- `adapters/sage/config.go` - Config mapping
- `adapters/sage/config_test.go` - Config tests

**What to Test**:
- ADK config → SAGE config conversion
- Validation (required fields)
- Default values

---

### Part 2: Key Management (adapters/sage/keys.go)

**Goal**: Wrapper for sage crypto library

```go
package sage

import (
    "github.com/sage-x-project/sage/crypto"
    "github.com/sage-x-project/sage/crypto/formats"
)

// KeyManager wraps sage crypto.Manager
type KeyManager struct {
    manager *crypto.Manager
}

// NewKeyManager creates a key manager using sage crypto library
func NewKeyManager() *KeyManager {
    return &KeyManager{
        manager: crypto.NewManager(),
    }
}

// LoadFromFile loads a key using sage crypto formats
func (km *KeyManager) LoadFromFile(path string, password string) (crypto.KeyPair, error) {
    // Use sage PEM format loader
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    // Use sage's PEM importer
    return km.manager.ImportKeyPair(data, crypto.KeyFormatPEM)
}

// Generate creates a new Ed25519 key pair using sage crypto
func (km *KeyManager) Generate() (crypto.KeyPair, error) {
    return km.manager.GenerateKeyPair(crypto.KeyTypeEd25519)
}

// SaveToFile saves a key using sage PEM format
func (km *KeyManager) SaveToFile(keyPair crypto.KeyPair, path string, password string) error {
    // Use sage's PEM exporter
    data, err := km.manager.ExportKeyPair(keyPair, crypto.KeyFormatPEM)
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0600)
}
```

**Files**:
- `adapters/sage/keys.go` - Key management wrapper
- `adapters/sage/keys_test.go` - Key tests

**What to Test**:
- Generate → Save → Load round-trip
- PEM format compatibility
- File permissions (0600)

---

### Part 3: DID Resolution (adapters/sage/did.go)

**Goal**: Wrapper for sage DID library

```go
package sage

import (
    "context"
    "github.com/sage-x-project/sage/did"
    "github.com/sage-x-project/sage/did/ethereum"
    "crypto/ed25519"
)

// DIDResolver wraps sage did.MultiChainResolver
type DIDResolver struct {
    resolver *did.MultiChainResolver
}

// NewDIDResolver creates a DID resolver using sage library
func NewDIDResolver(cfg *Config) (*DIDResolver, error) {
    resolver := did.NewMultiChainResolver()

    // Add Ethereum resolver if configured
    if cfg.Blockchain != nil && cfg.Blockchain.Network == "ethereum" {
        ethResolver, err := ethereum.NewResolver(cfg.Config)
        if err != nil {
            return nil, err
        }
        resolver.AddResolver(did.ChainEthereum, ethResolver)
    }

    // TODO: Add Solana resolver in Phase 3

    return &DIDResolver{
        resolver: resolver,
    }, nil
}

// Resolve resolves a DID using sage library
func (r *DIDResolver) Resolve(ctx context.Context, didStr string) (*did.AgentMetadata, error) {
    return r.resolver.Resolve(ctx, did.AgentDID(didStr))
}

// ResolvePublicKey gets only the public key (convenience method)
func (r *DIDResolver) ResolvePublicKey(ctx context.Context, didStr string) (ed25519.PublicKey, error) {
    pubKey, err := r.resolver.ResolvePublicKey(ctx, did.AgentDID(didStr))
    if err != nil {
        return nil, err
    }

    // Convert to ed25519.PublicKey
    if ed25519Key, ok := pubKey.(ed25519.PublicKey); ok {
        return ed25519Key, nil
    }

    return nil, fmt.Errorf("public key is not Ed25519")
}
```

**Files**:
- `adapters/sage/did.go` - DID resolver wrapper
- `adapters/sage/did_test.go` - DID tests

**What to Test**:
- Mock resolver for unit tests
- Ethereum resolver integration (with test RPC)
- Cache behavior
- Error handling (DID not found, inactive agent)

---

### Part 4: Integration with Transport (adapters/sage/transport.go)

**Goal**: Update TransportManager to use SAGE config and DID resolver

```go
// Update TransportManager to accept Config and DIDResolver
func NewTransportManagerWithConfig(cfg *Config, keyPair crypto.KeyPair) (*TransportManager, error) {
    // Create DID resolver
    resolver, err := NewDIDResolver(cfg)
    if err != nil {
        return nil, err
    }

    // Extract Ed25519 private key from keyPair
    privateKey := keyPair.Private().(ed25519.PrivateKey)

    // Create transport config
    transportCfg := DefaultTransportConfig()
    if cfg.DID != nil {
        transportCfg.SessionTTL = cfg.DID.CacheTTL
    }

    tm := &TransportManager{
        localDID:          cfg.LocalDID,
        privateKey:        privateKey,
        config:            transportCfg,
        sessionManager:    NewSessionManager(transportCfg.SessionTTL),
        encryptionManager: NewEncryptionManager(),
        signingManager:    NewSigningManager(),
        didResolver:       resolver,
        // ... rest of initialization
    }

    return tm, nil
}

// Update Connect to use DID resolver
func (tm *TransportManager) Connect(ctx context.Context, remoteDID string) (*HandshakeInvitation, error) {
    // Resolve remote DID to get public key
    remotePubKey, err := tm.didResolver.ResolvePublicKey(ctx, remoteDID)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve DID: %w", err)
    }

    // Store public key for later use
    session.RemotePublicKey = remotePubKey

    // Continue with existing handshake logic...
}
```

**Files**:
- Update `adapters/sage/transport.go` - Add config-based constructor
- Update `adapters/sage/transport_test.go` - Test with config

---

### Part 5: Builder Integration (builder/builder.go)

**Goal**: Add SAGE config to builder

```go
// Add to builder/builder.go
func (b *Builder) WithSAGEConfig(cfg *config.SAGEConfig) *Builder {
    if err := cfg.Validate(); err != nil {
        b.errors = append(b.errors, err)
        return b
    }

    b.sageConfig = cfg
    b.protocol = protocol.ProtocolSAGE
    return b
}

// In Build() method, initialize SAGE transport if configured
if b.protocol == protocol.ProtocolSAGE && b.sageConfig != nil {
    // Convert ADK config to SAGE config
    sageCfg, err := sage.FromADKConfig(b.sageConfig)
    if err != nil {
        return nil, err
    }

    // Load key pair
    keyManager := sage.NewKeyManager()
    keyPair, err := keyManager.LoadFromFile(b.sageConfig.PrivateKeyPath, "")
    if err != nil {
        return nil, err
    }

    // Create transport manager
    transport, err := sage.NewTransportManagerWithConfig(sageCfg, keyPair)
    if err != nil {
        return nil, err
    }

    // Use transport for agent communication
    // ...
}
```

---

## Missing Features Analysis

### ❓ Questions for User: What's Missing in @sage/?

Based on analysis, **@sage/ library is complete** for our needs:

1. **Key Management**: ✅ Complete (`crypto.Manager`, multiple formats)
2. **DID Resolution**: ✅ Complete (`did.MultiChainResolver`, Ethereum + Solana)
3. **Configuration**: ✅ Complete (`config.Config` with all needed fields)
4. **Storage**: ✅ Complete (Memory and file-based key storage)

**Conclusion**: We don't need to implement anything in @sage/. Everything is already there!

### What We DO Need in @sage-adk/:

1. **Config Mapping** (`adapters/sage/config.go`):
   - Convert ADK's simple config to sage's detailed config
   - Validation specific to ADK use cases

2. **Convenience Wrappers** (`adapters/sage/keys.go`, `adapters/sage/did.go`):
   - Simpler API for ADK users
   - Hide sage library complexity
   - Type conversions (e.g., sage types → ADK types)

3. **Integration Code** (`adapters/sage/transport.go`, `builder/builder.go`):
   - Connect SAGE components to ADK builder
   - Wire up DID resolution with transport layer
   - Make it work seamlessly with Builder API

---

## Implementation Order

### Day 1: Core Integration

1. **Morning**: Config mapping (`adapters/sage/config.go`)
   - Map ADK config → SAGE config
   - Tests with validation

2. **Afternoon**: Key management wrapper (`adapters/sage/keys.go`)
   - Wrap `crypto.Manager`
   - Generate/Load/Save tests
   - Integration with existing transport

### Day 2: DID Resolution & Builder

1. **Morning**: DID resolver wrapper (`adapters/sage/did.go`)
   - Wrap `did.MultiChainResolver`
   - Mock resolver for tests
   - Integration with transport

2. **Afternoon**: Builder integration (`builder/builder.go`)
   - `WithSAGEConfig()` method
   - Wire up all components
   - End-to-end test: Builder → Agent with SAGE

---

## Success Criteria

```go
// This should work at the end of Task 8:

// 1. Generate and save keys
km := sage.NewKeyManager()
keyPair, _ := km.Generate()
km.SaveToFile(keyPair, "keys/agent.key", "")

// 2. Create config
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

// 4. Agent can resolve DIDs and establish secure connections
// (Full E2E in Task 9: SAGE Server)
```

---

## Test Plan

### Unit Tests
- [ ] Config conversion: ADK → SAGE
- [ ] Config validation
- [ ] Key generation
- [ ] Key save/load round-trip
- [ ] DID resolver (mock)
- [ ] Builder.WithSAGEConfig()

### Integration Tests
- [ ] Load real PEM key file
- [ ] Config from YAML file
- [ ] Builder creates agent with SAGE transport
- [ ] DID resolution with test Ethereum RPC

### Test Coverage Target
- ≥ 85% for new code
- All public APIs tested

---

## Dependencies

### External Libraries (Already in @sage/)
- `github.com/sage-x-project/sage/crypto` - Key management
- `github.com/sage-x-project/sage/did` - DID resolution
- `github.com/sage-x-project/sage/config` - Configuration
- `github.com/ethereum/go-ethereum` - Ethereum client (via sage/did/ethereum)

### Internal Dependencies
- Task 7: SAGE Transport Layer (COMPLETE)
- ADK config system (EXISTS)
- ADK builder (EXISTS)

---

## Notes

1. **No Blockchain Required for Testing**: Use mock resolver or manual DID registration
2. **Phase 2B Simplification**: Full blockchain integration in Phase 3
3. **Reuse Over Rewrite**: Maximize use of existing @sage/ code
4. **Minimal Wrappers**: Only add wrappers when absolutely necessary for ADK integration

---

**Ready to Start**: All dependencies analyzed, plan is clear, no missing features in @sage/
**Next Step**: Begin implementation (Day 1 Morning: Config mapping)
