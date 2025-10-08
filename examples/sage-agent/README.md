# SAGE Protocol Agent

A secure AI chatbot agent using SAGE ADK with blockchain-based identity verification and the SAGE protocol.

## Features

- **SAGE Protocol**: Blockchain-secured agent communication
- **DID (Decentralized Identity)**: Verified agent identity on-chain
- **Secure Handshake**: Encrypted and signed messages
- **OpenAI Integration**: AI-powered conversational agent
- **Configurable Networks**: Supports Ethereum, Sepolia, Kaia, and more
- **Graceful Shutdown**: Clean lifecycle management

## Prerequisites

- Go 1.21 or later
- OpenAI API key
- Blockchain RPC endpoint (e.g., Alchemy, Infura)
- SAGE contract deployed on target network
- Ed25519 private key for agent identity

## Setup

### 1. Generate Agent Keys

First, generate an Ed25519 key pair for your agent:

```bash
# Create keys directory
mkdir -p keys

# Generate key using sage-cli or openssl
# Option 1: Using OpenSSL
openssl genpkey -algorithm ED25519 -out keys/agent.pem

# Option 2: Using sage KeyManager (Go)
# See examples/key-generation/main.go
```

### 2. Register Agent DID

Register your agent's DID on the blockchain:

```bash
# Deploy or interact with SAGE contract
# This will map your DID to your public key on-chain
# Example: did:sage:sepolia:0x123456789abcdef
```

### 3. Configure Environment

Set the required environment variables:

```bash
# OpenAI API Key (required)
export OPENAI_API_KEY="sk-your-openai-api-key"

# SAGE Configuration (all optional with defaults)
export SAGE_NETWORK="sepolia"                                    # Network: ethereum, sepolia, kaia, etc.
export SAGE_DID="did:sage:sepolia:0x123456789abcdef"            # Your agent's DID
export SAGE_RPC_ENDPOINT="https://eth-sepolia.g.alchemy.com/v2/YOUR-API-KEY"
export SAGE_CONTRACT_ADDRESS="0x0000000000000000000000000000000000000000"
export SAGE_PRIVATE_KEY_PATH="./keys/agent.pem"                 # Path to your Ed25519 key
```

### 4. Run the Agent

```bash
go run -tags examples main.go
```

The agent will start listening on `http://localhost:8080`.

**Note:** The `-tags examples` flag is required because example files are excluded from normal builds.

## Configuration

### Supported Networks

The SAGE protocol supports multiple blockchain networks:

| Network | Chain ID | Environment | Description |
|---------|----------|-------------|-------------|
| `ethereum` / `mainnet` | 1 | Production | Ethereum Mainnet |
| `sepolia` | 11155111 | Testnet | Ethereum Sepolia Testnet |
| `goerli` | 5 | Testnet | Ethereum Goerli Testnet (deprecated) |
| `kaia` / `cypress` | 8217 | Production | Kaia Mainnet |
| `kairos` / `kaia-testnet` | 1001 | Testnet | Kaia Testnet |
| `local` / `localhost` | 31337 | Development | Local Hardhat/Anvil |

### Configuration Options

```go
&config.SAGEConfig{
    Enabled:         true,                  // Enable SAGE protocol
    Network:         "sepolia",             // Blockchain network
    DID:             "did:sage:...",        // Agent's DID
    RPCEndpoint:     "https://...",         // Blockchain RPC endpoint
    ContractAddress: "0x...",               // SAGE contract address
    PrivateKeyPath:  "./keys/agent.pem",   // Ed25519 private key path
    CacheEnabled:    true,                  // Enable DID resolution cache
    CacheTTL:        1 * time.Hour,         // Cache TTL
}
```

## Architecture

```
┌─────────────────┐
│   User/Client   │
│  (with DID)     │
└────────┬────────┘
         │ SAGE Protocol
         │ (Signed & Encrypted)
         ↓
┌─────────────────────────┐
│    SAGE Agent           │
│    (Port 8080)          │
├─────────────────────────┤
│  1. Verify Signature    │
│  2. Resolve DID         │
│  3. Decrypt Message     │
│  4. Process with LLM    │
│  5. Sign Response       │
│  6. Encrypt Response    │
└─────────────────────────┘
         │
         ↓
┌─────────────────────────┐
│  Blockchain Network     │
│  - DID Resolution       │
│  - Public Key Registry  │
└─────────────────────────┘
```

### Key Components

1. **FromSAGEConfig Builder**: Convenience method for SAGE-enabled agents
   ```go
   agent := builder.FromSAGEConfig(sageConfig).
       WithLLM(provider).
       Build()
   ```

2. **Automatic Protocol Selection**: `FromSAGEConfig()` automatically sets `ProtocolSAGE`

3. **Blockchain Integration**:
   - DID resolution via smart contract
   - On-chain public key verification
   - Network-specific chain ID mapping

4. **Security Features**:
   - Ed25519 signature verification
   - X25519 key exchange for encryption
   - Message authenticity and confidentiality

## Usage

### Sending Messages to SAGE Agent

To communicate with a SAGE agent, you need:
1. Your own DID registered on-chain
2. Your Ed25519 private key
3. The agent's DID

```go
package main

import (
    "context"
    "log"

    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    // Load your keys
    km := sage.NewKeyManager()
    keyPair, _ := km.LoadFromFile("./keys/client.pem")
    privateKey, _ := km.ExtractEd25519PrivateKey(keyPair)

    // Create SAGE config
    cfg := &sage.Config{
        LocalDID: "did:sage:sepolia:0xYourDID",
        // ... other config ...
    }

    // Create transport manager
    tm, _ := sage.NewTransportManagerFromConfig(cfg, km)

    // Create secure session with agent
    session, _ := tm.CreateSession(
        context.Background(),
        "did:sage:sepolia:0xAgentDID", // Target agent DID
    )

    // Send encrypted message
    response, _ := session.Send([]byte("Hello, SAGE agent!"))
    log.Printf("Response: %s", response)
}
```

## Security Considerations

### Key Management
- Store private keys securely (use hardware wallets in production)
- Never commit keys to version control
- Use environment-specific keys (dev/staging/prod)
- Rotate keys periodically

### DID Registration
- Ensure DID is registered on-chain before starting agent
- Verify public key matches your private key
- Monitor for unauthorized DID updates

### Network Security
- Use HTTPS for RPC endpoints
- Validate contract addresses
- Monitor for suspicious message patterns
- Implement rate limiting

## Error Handling

The agent handles various error scenarios:

| Error | Handling |
|-------|----------|
| Missing API key | Fatal error at startup |
| Invalid SAGE config | Fatal error at startup |
| Missing/invalid private key | Fatal error at startup |
| DID resolution failure | Log error, reject message |
| Signature verification failure | Reject message |
| Decryption failure | Reject message |
| LLM failure | Log error, return error to client |

## Troubleshooting

### "Failed to load private key"
- Verify `SAGE_PRIVATE_KEY_PATH` points to valid Ed25519 PEM file
- Check file permissions (should be readable)
- Ensure key format is correct (PEM or JWK)

### "DID resolution failed"
- Verify DID is registered on-chain
- Check RPC endpoint connectivity
- Confirm contract address is correct
- Ensure network name matches chain ID

### "Signature verification failed"
- Ensure sender's DID is registered on-chain
- Verify sender is using correct private key
- Check for clock skew between systems

## Example: Config from YAML

You can also load SAGE configuration from a YAML file:

```yaml
# config.yaml
sage:
  enabled: true
  network: sepolia
  did: did:sage:sepolia:0x123456789abcdef
  rpc_endpoint: https://eth-sepolia.g.alchemy.com/v2/YOUR-API-KEY
  contract_address: "0x0000000000000000000000000000000000000000"
  private_key_path: ./keys/agent.pem
  cache_enabled: true
  cache_ttl: 1h

llm:
  provider: openai
  api_key: ${OPENAI_API_KEY}
  model: gpt-3.5-turbo
  max_tokens: 2000
  temperature: 0.7
```

Load and use:

```go
cfg, err := config.LoadFromFile("config.yaml")
if err != nil {
    log.Fatal(err)
}

agent := builder.FromSAGEConfig(&cfg.SAGE).
    WithLLM(provider).
    Build()
```

## Next Steps

- Implement multi-agent communication
- Add conversation history with blockchain audit trail
- Implement streaming responses
- Add custom tools/functions
- Deploy to production environment
- Monitor agent performance and security

## Related Examples

- **simple-agent**: Basic A2A protocol agent without SAGE
- **key-generation**: Generate and manage Ed25519 keys
- **did-registration**: Register DIDs on blockchain
- **multi-agent**: Multiple SAGE agents communicating

## License

LGPL-3.0-or-later
