# SAGE-Enabled Agent Example

This example demonstrates **low-level SAGE adapter usage** for secure agent-to-agent communication with message signing, encryption, and verification.

## Overview

This example shows how to use the SAGE adapter directly (without the high-level builder API) to:

-  Create SAGE protocol adapters with Ed25519 key pairs
-  Send signed messages over HTTP
-  Receive and verify messages with signature validation
-  Implement nonce-based replay attack protection
-  Validate message timestamps with clock skew tolerance
-  Run agents in interactive or distributed mode

## Architecture

```

                      SAGE Protocol Layer                     

                                                               
           HTTP/JSON                 
    Alice          Bob           
   (Sender)   Signed + Encrypted Msg    (Receiver)       
                                      
                                                            
        1. Add Security Metadata                            
        2. Sign with Ed25519                                
        3. Send via NetworkClient                           
                                                            
                                           4. Verify Signature
                                           5. Check Nonce    
                                           6. Validate Time  
                                                               

```

## Running Modes

### Mode 1: Interactive (Single Process)

Run both Alice and Bob in a single process for demonstration:

```bash
go run main.go interactive
```

This will:
1. Generate Ed25519 key pairs for Alice and Bob
2. Create SAGE adapters for both agents
3. Start Bob's HTTP server on port 18080
4. Send a message from Alice to Bob
5. Verify the message signature and display security metadata

**Output:**
```
 SAGE Interactive Demo - Two agents exchanging secure messages
======================================================================

 Step 1: Generating Ed25519 key pairs for Alice and Bob...
 Alice's public key: a1b2c3d4e5f6g7h8
 Bob's public key: 9i0j1k2l3m4n5o6p

 Step 2: Creating SAGE adapters...
 Alice's adapter created
 Bob's adapter created

 Step 3: Starting Bob's HTTP server on :18080...
 Bob's server is running

 Step 4: Configuring Alice to send messages to Bob...
 Alice configured to send to: http://localhost:18080/sage/message

 Step 5: Alice sending encrypted message to Bob...
 Message sent successfully

 Bob received message from did:sage:alice
 Message signature verified successfully
 Message content: Hello Bob! This is a secure SAGE message from Alice.

 Step 6: Verifying message delivery...
 Message delivered and verified successfully

 Security Metadata:
  Protocol Mode: SAGE
  Agent DID: did:sage:alice
  Timestamp: 2025-10-10T05:45:23Z
  Nonce: MTczMzg4MjcyMz...
  Signature Algorithm: Ed25519
  Signature KeyID: did:sage:alice#key-1
  Signature Length: 64 bytes

 SAGE Interactive Demo completed successfully!
======================================================================
```

### Mode 2: Distributed (Sender + Receiver)

#### Terminal 1: Start Bob (Receiver)

```bash
go run main.go receiver
```

Output:
```
 SAGE Receiver (Bob)
Listening on :18080
 Server started. Press Ctrl+C to stop.
```

#### Terminal 2: Run Alice (Sender)

```bash
go run main.go sender
```

Output:
```
 SAGE Sender (Alice)
 Sending message to: http://localhost:18080/sage/message
 Message sent successfully
```

#### Bob's Terminal:
```
 Received message from did:sage:alice
 Message verified
 Content: Hello from standalone Alice!
```

## Environment Variables

You can customize the agents using environment variables:

```bash
# Alice (Sender)
export ALICE_KEY_PATH="/tmp/alice-key.json"
export BOB_ENDPOINT="http://localhost:18080/sage/message"

# Bob (Receiver)
export BOB_KEY_PATH="/tmp/bob-key.json"
```

See `.env.example` for a complete list of configurable options.

## What This Example Demonstrates

### 1. **Low-Level SAGE Adapter API**
Unlike the `sage-agent` example which uses the high-level builder API, this example shows direct SAGE adapter usage:

```go
// Create adapter
adapter, err := sage.NewAdapter(&config.SAGEConfig{
    DID:            "did:sage:alice",
    Network:        "local",
    PrivateKeyPath: keyPath,
})

// Set endpoint
adapter.SetRemoteEndpoint("http://localhost:18080/sage/message")

// Send message
message := types.NewMessage(types.MessageRoleUser, []types.Part{
    types.NewTextPart("Hello Bob!"),
})
err = adapter.SendMessage(ctx, message)

// Verify received message
err = adapter.Verify(ctx, receivedMessage)
```

### 2. **Message Signing (RFC 9421)**
Messages are signed using Ed25519 keys according to RFC 9421 HTTP Message Signatures:

```go
// Security metadata is automatically added
msg.Security = &types.SecurityMetadata{
    Mode:      types.ProtocolModeSAGE,
    AgentDID:  "did:sage:alice",
    Nonce:     "MTczMzg4MjcyMz...",
    Timestamp: time.Now(),
    Signature: &types.SignatureData{
        Algorithm: "Ed25519",
        KeyID:     "did:sage:alice#key-1",
        Signature: []byte{...},
    },
}
```

### 3. **Network Layer (HTTP Transport)**
Messages are transmitted over HTTP using the NetworkClient:

```go
// POST http://localhost:18080/sage/message
// Headers:
//   Content-Type: application/json
//   X-SAGE-Protocol-Mode: SAGE
//   X-SAGE-Agent-DID: did:sage:alice
//
// Body: JSON-serialized message
```

### 4. **Security Validation Pipeline**
The receiver validates messages through multiple security checks:

1. **Signature Verification**: Ed25519 signature validation
2. **Nonce Check**: Replay attack protection
3. **Timestamp Validation**: Clock skew tolerance (5 minutes)
4. **Protocol Mode Check**: Ensures SAGE protocol

```go
// Verification process
if err := adapter.Verify(ctx, msg); err != nil {
    // One of the security checks failed
    return err
}
// Message is verified and safe to process
```

## Key Differences from `sage-agent` Example

| Feature | sage-agent | sage-enabled-agent |
|---------|------------|-------------------|
| **API Level** | High-level builder API | Low-level adapter API |
| **LLM Integration** |  Uses OpenAI |  No LLM (pure transport) |
| **Use Case** | Production chatbot | Transport protocol demo |
| **Complexity** | Simple (5 lines) | Detailed (shows internals) |
| **Server** | Built-in HTTP server | Manual NetworkServer setup |
| **Learning Goal** | Quick start | Understanding SAGE protocol |

## Code Structure

```
sage-enabled-agent/
 main.go              # Main entry point with 3 modes
 README.md            # This file
 .env.example         # Environment variable template
```

## Security Features

### 1. **Ed25519 Signatures**
- Fast elliptic curve signatures
- 64-byte signature size
- Immune to timing attacks

### 2. **Nonce-Based Replay Protection**
- Cryptographically random nonces (16 bytes + timestamp)
- In-memory nonce cache (10,000 max entries)
- Prevents replay attacks

### 3. **Timestamp Validation**
- RFC 3339 format timestamps
- 5-minute clock skew tolerance
- Prevents old/future message attacks

### 4. **TLS Support** (Production Ready)
In production, enable HTTPS:

```go
server := sage.NewNetworkServer(":18080", handler)
server.StartTLS(certFile, keyFile)
```

## Testing

Build and run the example:

```bash
# Interactive mode
go run -tags examples main.go interactive

# Distributed mode
go run -tags examples main.go receiver  # Terminal 1
go run -tags examples main.go sender    # Terminal 2
```

## Related Examples

- **[sage-agent](../sage-agent/)** - High-level SAGE agent with LLM integration
- **[simple-agent](../simple-agent/)** - Basic agent without SAGE security
- **[key-generation](../key-generation/)** - Generate Ed25519 keys for SAGE

## Documentation

- [SAGE Adapter Implementation](../../adapters/sage/adapter.go)
- [Network Layer](../../adapters/sage/network.go)
- [RFC 9421 Signing](../../adapters/sage/signing.go)
- [Integration Tests](../../adapters/sage/integration_test.go)

## Troubleshooting

### Port Already in Use
```
Error: bind: address already in use
```

**Solution**: Change the port in receiver mode:
```bash
# Bob uses port 18081 instead
sed -i 's/:18080/:18081/g' main.go
```

### Signature Verification Failed
```
Error: signature verification failed
```

**Cause**: Key mismatch or message tampering

**Solution**: Ensure both agents use the same key paths and keys haven't been modified.

### Message Not Received
```
Error: message was not received
```

**Solution**: Check that Bob's server started before Alice sends the message:
```go
time.Sleep(100 * time.Millisecond) // Wait for server
```

## License

LGPL-3.0-or-later

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development guidelines.
