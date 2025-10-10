# SAGE Transport Layer

SAGE (Secure Agent Guarantee Engine) Transport Layer        Go .

##  

- **4  **: RFC 9421   
- ** **: X25519 ECDH + ChaCha20-Poly1305 AEAD
- ** (Forward Secrecy)**:     
- **  **: Nonce  replay protection
- ** **:    
- ** **: Clock skew   

## 

```
TransportManager
 HandshakeManager    # 4-phase handshake orchestration
    Phase 1: Invitation  (Alice → Bob)
    Phase 2: Request     (Bob → Alice, HPKE encrypted)
    Phase 3: Response    (Alice → Bob, Session key)
    Phase 4: Complete    (Bob → Alice, Acknowledgment)
 SessionManager      # Session lifecycle management
 EncryptionManager   # X25519 + ChaCha20-Poly1305
 SigningManager      # Ed25519 + BLAKE3
```

##  

### 

```bash
go get github.com/sage-x-project/sage-adk/adapters/sage
```

###  

```go
package main

import (
    "context"
    "crypto/ed25519"
    "crypto/rand"
    "fmt"

    "github.com/sage-x-project/sage-adk/adapters/sage"
)

func main() {
    // 1.   
    alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
    bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)

    // 2. Transport Manager 
    alice := sage.NewTransportManager("did:sage:alice", alicePrivateKey, nil)
    bob := sage.NewTransportManager("did:sage:bob", bobPrivateKey, nil)

    ctx := context.Background()

    // 3.  
    invitation, _ := alice.Connect(ctx, "did:sage:bob")
    request, _ := bob.HandleInvitation(ctx, invitation)
    response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
    complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
    alice.HandleComplete(ctx, complete, bobPublicKey)

    // 4.  
    message := map[string]interface{}{
        "type": "greeting",
        "text": "Hello Bob!",
    }

    appMsg, _ := alice.SendMessage(ctx, "did:sage:bob", message)

    bob.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
        var msg map[string]interface{}
        sage.DeserializeMessage(payload, &msg)
        fmt.Printf("Received: %s\n", msg["text"])
        return nil
    })

    bob.ReceiveMessage(ctx, appMsg, alicePublicKey)
}
```

##  

### Phase 1: Invitation (Alice → Bob)

Alice Bob  .

```go
invitation, session, err := alice.Connect(ctx, "did:sage:bob")
```

**Invitation :**
- X25519  
- Nonce (replay )
-   
- 

### Phase 2: Request (Bob → Alice)

Bob   .

```go
request, session, err := bob.HandleInvitation(ctx, invitation)
```

**Request :**
- Bob X25519  
- HPKE   (  )
- Ed25519 
-  ID

### Phase 3: Response (Alice → Bob)

Alice    .

```go
response, err := alice.HandleRequest(ctx, request, bobPublicKey)
```

**Response :**
- ChaCha20-Poly1305   (  )
-   
- Ed25519 

### Phase 4: Complete (Bob → Alice)

Bob  Alice  .

```go
complete, err := bob.HandleResponse(ctx, response, alicePublicKey)
err = alice.HandleComplete(ctx, complete, bobPublicKey)
```

**Complete :**
- Acknowledgment (  )
-  
- Ed25519 

##  

###   (HKDF)

```
  = ECDH(Alice_ephemeral_private, Bob_ephemeral_public)
  = HKDF-SHA256( , salt=nil, info="SAGE-HPKE-v1")
```

###   (ChaCha20-Poly1305)

```
 = ChaCha20-Poly1305.Encrypt(
    key =   (32 bytes),
    nonce =  (12 bytes),
    plaintext = JSON(),
    aad = nil
)
```

###  (Ed25519 + BLAKE3)

```
_ = Base64(BLAKE3(JSON( - Signature )))
 = BLAKE3(_)
 = Ed25519.Sign(, )
```

##  

###  

```go
//  
session, err := tm.GetSession("did:sage:remote")

//   
sessions := tm.ListSessions()

//  
err := tm.Disconnect(ctx, "did:sage:remote")

// Transport Manager 
err := tm.Close()
```

###  

- **Pending**: 
- **Establishing**:   
- **Active**:   
- **Expired**: 
- **Closed**: 

## 

```go
config := sage.DefaultTransportConfig()

// 
config.SessionTTL = 30 * time.Minute      //  
config.MaxClockSkew = 2 * time.Minute     //   
config.HandshakeTimeout = 20 * time.Second //  
config.MaxMessageSize = 5 * 1024 * 1024   //   

tm := sage.NewTransportManager(localDID, privateKey, config)
```

##  

```go
tm.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
    //   
    var msg map[string]interface{}
    if err := sage.DeserializeMessage(payload, &msg); err != nil {
        return err
    }

    //   
    switch msg["type"] {
    case "greeting":
        handleGreeting(msg)
    case "request":
        handleRequest(msg)
    default:
        return fmt.Errorf("unknown message type: %v", msg["type"])
    }

    return nil
})
```

##  

###  

```go
//    
envelope, err := sage.WrapMessage("transaction", payload)

//  
var data map[string]interface{}
err = sage.UnwrapMessage(envelope, &data)
```

### /

```go
// JSON 
bytes, err := sage.SerializeMessage(message)

// JSON 
var message map[string]interface{}
err = sage.DeserializeMessage(bytes, &message)
```

### Base64 

```go
// 
encoded, err := sage.EncodeMessage(message)

// 
var message map[string]interface{}
err = sage.DecodeMessage(encoded, &message)
```

##  

### 

1. ** **:    (HSM, KMS )
2. **Nonce **:  1000,  
3. **Clock Skew**:    
4. ** TTL**:    
5. ** **: DID    

###  

 :
- Man-in-the-Middle (MitM)
- Replay attacks
- Message tampering
- Session hijacking
- Forward secrecy breach

## 

```bash
#  
go test ./adapters/sage -v

#  
go test ./adapters/sage -v -run TestIntegration

# 
go test ./adapters/sage -cover

# 
go test ./adapters/sage -bench=.
```

## 

### 

- Phase 1-4 : ~10ms ()
-  : 1 ( )

###  

- : ~0.1ms (ChaCha20-Poly1305)
- : ~0.2ms (Ed25519)
- : ~0.3ms (Ed25519 + BLAKE3)

### 

- Transport Manager: ~100KB
- Session: ~2KB
- Active handshake: ~1KB

##  

###  

**"signature verification failed"**
- :     
- : DID    

**"session not found"**
- :    
- :  

**"timestamp outside acceptable clock skew"**
- :   
- : NTP   MaxClockSkew 

**"nonce replay detected"**
- :  nonce  
- :    nonce  

## 

LGPL-3.0-or-later

##  

- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [RFC 8032: EdDSA](https://www.rfc-editor.org/rfc/rfc8032.html)
- [BLAKE3 Specification](https://github.com/BLAKE3-team/BLAKE3-specs)

## 

  PR GitHub  :
https://github.com/sage-x-project/agent-develope-kit

## 

- : https://docs.sage-x-project.org
- : https://github.com/sage-x-project/agent-develope-kit/issues
