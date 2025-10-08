# Key Generation Tool

A utility for generating Ed25519 key pairs for SAGE agents.

## Overview

This tool generates cryptographic key pairs used for:
- **Agent Identity**: Uniquely identifies your agent
- **Message Signing**: Proves message authenticity
- **DID Registration**: Links your agent to blockchain identity
- **Secure Communication**: Enables encrypted message exchange

## Usage

### Basic Usage

Generate a key with default settings (PEM format):

```bash
go run -tags examples main.go
```

This creates `./keys/agent.pem` with restrictive permissions (0600).

### Custom Output Path

Specify where to save the key:

```bash
go run -tags examples main.go -output ./my-keys/agent.pem
```

### JWK Format

Generate key in JWK (JSON Web Key) format:

```bash
go run -tags examples main.go -format jwk -output ./keys/agent.jwk
```

### Show Public Key

Display the public key after generation:

```bash
go run -tags examples main.go -show-public
```

Output:
```
📋 Public Key (base64):
MCowBQYDK2VwAyEAXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX=

📋 Public Key (hex):
5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c5c
```

### All Options

```bash
go run -tags examples main.go \
  -output ./keys/production.pem \
  -format pem \
  -show-public
```

## Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-output` | `./keys/agent.pem` | Output path for the private key |
| `-format` | `pem` | Key format: `pem` or `jwk` |
| `-show-public` | `false` | Display public key after generation |

## Key Formats

### PEM Format (Default)

```pem
-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIGhoYWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0
-----END PRIVATE KEY-----
```

**Characteristics:**
- Standard format for cryptographic keys
- Base64-encoded ASN.1 structure
- Widely supported across tools
- Human-readable with clear boundaries

### JWK Format

```json
{
  "kty": "OKP",
  "crv": "Ed25519",
  "x": "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
  "d": "YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"
}
```

**Characteristics:**
- JSON-based format
- Native to web standards
- Easy to parse programmatically
- Compact representation

## Security Best Practices

### File Permissions

The tool automatically sets restrictive permissions (0600):
```bash
# Only owner can read/write
-rw------- 1 user user 119 Oct  7 12:00 agent.pem
```

Verify permissions:
```bash
ls -l keys/agent.pem
```

### Key Storage

**Development:**
- Store in `./keys/` directory (gitignored)
- Use separate keys per developer
- Rotate keys periodically

**Production:**
- Use hardware security module (HSM)
- Implement key rotation policy
- Maintain secure offline backups
- Use different keys per environment

### Environment Variables

Never hardcode keys in code. Use environment variables:

```bash
export SAGE_PRIVATE_KEY_PATH=./keys/production.pem
```

Add to `.gitignore`:
```
keys/
*.pem
*.jwk
.env
```

## Integration with SAGE Agent

After generating a key, use it with your SAGE agent:

```go
package main

import (
    "github.com/sage-x-project/sage-adk/builder"
    "github.com/sage-x-project/sage-adk/config"
)

func main() {
    sageConfig := &config.SAGEConfig{
        Enabled:         true,
        Network:         "sepolia",
        DID:             "did:sage:sepolia:0x123...",
        RPCEndpoint:     "https://eth-sepolia.example.com",
        ContractAddress: "0xABC...",
        PrivateKeyPath:  "./keys/agent.pem", // ← Generated key
        CacheTTL:        1 * time.Hour,
    }

    agent := builder.FromSAGEConfig(sageConfig).Build()
    agent.Start(":8080")
}
```

## DID Registration Workflow

1. **Generate Key Pair**:
   ```bash
   go run -tags examples main.go -show-public
   ```

2. **Extract Public Key**:
   Copy the hex output from `-show-public`

3. **Register DID**:
   Interact with SAGE contract to register:
   ```solidity
   // Pseudo-code
   contract.registerDID(
       did: "did:sage:sepolia:0x123...",
       publicKey: "0x5c5c5c..." // From step 2
   )
   ```

4. **Verify Registration**:
   ```bash
   # Query blockchain
   cast call $CONTRACT "getPublicKey(string)" "did:sage:sepolia:0x123..."
   ```

5. **Use in Agent**:
   Set `SAGE_PRIVATE_KEY_PATH` and start agent

## Key Rotation

To rotate keys:

1. **Generate new key**:
   ```bash
   go run -tags examples main.go -output ./keys/agent-v2.pem
   ```

2. **Update DID mapping** on blockchain

3. **Deploy updated agent** with new key

4. **Revoke old key** in contract

5. **Securely delete old key**:
   ```bash
   shred -vfz -n 10 ./keys/agent-v1.pem
   ```

## Troubleshooting

### "Permission denied"

Ensure output directory is writable:
```bash
mkdir -p keys
chmod 755 keys
```

### "Failed to generate key"

Check Go version (requires 1.21+):
```bash
go version
```

### "File already exists"

Use `-output` to specify different path or confirm overwrite when prompted.

## Examples

### Generate Multiple Keys

```bash
# Development key
go run -tags examples main.go -output ./keys/dev.pem

# Staging key
go run -tags examples main.go -output ./keys/staging.pem

# Production key
go run -tags examples main.go -output ./keys/prod.pem
```

### Batch Generation Script

```bash
#!/bin/bash
for env in dev staging prod; do
  go run -tags examples main.go -output "./keys/${env}.pem"
  echo "Generated key for: $env"
done
```

### Extract Public Key from Existing Key

```go
package main

import (
    "fmt"
    "log"
    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    km := sage.NewKeyManager()
    keyPair, _ := km.LoadFromFile("./keys/agent.pem")
    publicKey, _ := km.ExtractEd25519PublicKey(keyPair)
    fmt.Printf("Public Key: %x\n", publicKey)
}
```

## Related Documentation

- **SAGE Agent Example**: `../sage-agent/` - Using keys with SAGE protocol
- **DID Registration**: Coming soon
- **Key Management Best Practices**: See SAGE documentation

## Algorithm Details

**Key Type**: Ed25519
**Curve**: Curve25519 (Edwards form)
**Key Size**: 32 bytes (256 bits)
**Security Level**: ~128-bit security
**Signature Size**: 64 bytes

Ed25519 provides:
- Fast signature generation
- Fast signature verification
- Small key size
- Strong security guarantees
- Deterministic signatures

## License

LGPL-3.0-or-later
