# SAGE Transport Layer

SAGE (Secure Agent Guarantee Engine) Transport Layer는 블록체인 기반의 안전한 에이전트 간 통신을 제공하는 Go 구현체입니다.

## 주요 기능

- **4단계 핸드셰이크 프로토콜**: RFC 9421 준수 메시지 서명
- **하이브리드 암호화**: X25519 ECDH + ChaCha20-Poly1305 AEAD
- **전방향 보안(Forward Secrecy)**: 임시 키를 사용한 세션별 보안
- **재생 공격 방지**: Nonce 기반 replay protection
- **세션 관리**: 자동 만료 및 정리
- **타임스탬프 검증**: Clock skew 허용 범위 설정

## 아키텍처

```
TransportManager
├── HandshakeManager    # 4-phase handshake orchestration
│   ├── Phase 1: Invitation  (Alice → Bob)
│   ├── Phase 2: Request     (Bob → Alice, HPKE encrypted)
│   ├── Phase 3: Response    (Alice → Bob, Session key)
│   └── Phase 4: Complete    (Bob → Alice, Acknowledgment)
├── SessionManager      # Session lifecycle management
├── EncryptionManager   # X25519 + ChaCha20-Poly1305
└── SigningManager      # Ed25519 + BLAKE3
```

## 빠른 시작

### 설치

```bash
go get github.com/sage-x-project/sage-adk/adapters/sage
```

### 기본 사용법

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
    // 1. 키 쌍 생성
    alicePublicKey, alicePrivateKey, _ := ed25519.GenerateKey(rand.Reader)
    bobPublicKey, bobPrivateKey, _ := ed25519.GenerateKey(rand.Reader)

    // 2. Transport Manager 생성
    alice := sage.NewTransportManager("did:sage:alice", alicePrivateKey, nil)
    bob := sage.NewTransportManager("did:sage:bob", bobPrivateKey, nil)

    ctx := context.Background()

    // 3. 핸드셰이크 수행
    invitation, _ := alice.Connect(ctx, "did:sage:bob")
    request, _ := bob.HandleInvitation(ctx, invitation)
    response, _ := alice.HandleRequest(ctx, request, bobPublicKey)
    complete, _ := bob.HandleResponse(ctx, response, alicePublicKey)
    alice.HandleComplete(ctx, complete, bobPublicKey)

    // 4. 메시지 송수신
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

## 핸드셰이크 프로토콜

### Phase 1: Invitation (Alice → Bob)

Alice가 Bob에게 연결을 시작합니다.

```go
invitation, session, err := alice.Connect(ctx, "did:sage:bob")
```

**Invitation 내용:**
- X25519 임시 공개키
- Nonce (replay 방지)
- 지원 알고리즘 목록
- 타임스탬프

### Phase 2: Request (Bob → Alice)

Bob이 초대를 수락하고 응답합니다.

```go
request, session, err := bob.HandleInvitation(ctx, invitation)
```

**Request 내용:**
- Bob의 X25519 임시 공개키
- HPKE로 암호화된 페이로드 (공유 비밀 포함)
- Ed25519 서명
- 세션 ID

### Phase 3: Response (Alice → Bob)

Alice가 세션 키를 생성하여 전송합니다.

```go
response, err := alice.HandleRequest(ctx, request, bobPublicKey)
```

**Response 내용:**
- ChaCha20-Poly1305 세션 키 (공유 비밀로 암호화)
- 세션 만료 시간
- Ed25519 서명

### Phase 4: Complete (Bob → Alice)

Bob이 확인하고 Alice가 세션을 활성화합니다.

```go
complete, err := bob.HandleResponse(ctx, response, alicePublicKey)
err = alice.HandleComplete(ctx, complete, bobPublicKey)
```

**Complete 내용:**
- Acknowledgment (세션 키로 암호화)
- 세션 메타데이터
- Ed25519 서명

## 암호화 세부사항

### 키 파생 (HKDF)

```
임시 비밀 = ECDH(Alice_ephemeral_private, Bob_ephemeral_public)
공유 비밀 = HKDF-SHA256(임시 비밀, salt=nil, info="SAGE-HPKE-v1")
```

### 메시지 암호화 (ChaCha20-Poly1305)

```
암호문 = ChaCha20-Poly1305.Encrypt(
    key = 세션 키 (32 bytes),
    nonce = 랜덤 (12 bytes),
    plaintext = JSON(메시지),
    aad = nil
)
```

### 서명 (Ed25519 + BLAKE3)

```
서명_베이스 = Base64(BLAKE3(JSON(메시지 - Signature 필드)))
해시 = BLAKE3(서명_베이스)
서명 = Ed25519.Sign(개인키, 해시)
```

## 세션 관리

### 세션 생명주기

```go
// 세션 조회
session, err := tm.GetSession("did:sage:remote")

// 모든 세션 조회
sessions := tm.ListSessions()

// 연결 종료
err := tm.Disconnect(ctx, "did:sage:remote")

// Transport Manager 종료
err := tm.Close()
```

### 세션 상태

- **Pending**: 생성됨
- **Establishing**: 핸드셰이크 진행 중
- **Active**: 메시지 송수신 가능
- **Expired**: 만료됨
- **Closed**: 종료됨

## 설정

```go
config := sage.DefaultTransportConfig()

// 커스터마이징
config.SessionTTL = 30 * time.Minute      // 세션 수명
config.MaxClockSkew = 2 * time.Minute     // 타임스탬프 허용 오차
config.HandshakeTimeout = 20 * time.Second // 핸드셰이크 제한시간
config.MaxMessageSize = 5 * 1024 * 1024   // 최대 메시지 크기

tm := sage.NewTransportManager(localDID, privateKey, config)
```

## 메시지 핸들러

```go
tm.SetMessageHandler(func(ctx context.Context, fromDID string, payload []byte) error {
    // 메시지 처리 로직
    var msg map[string]interface{}
    if err := sage.DeserializeMessage(payload, &msg); err != nil {
        return err
    }

    // 메시지 타입별 처리
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

## 메시지 유틸리티

### 메시지 래핑

```go
// 타입이 지정된 메시지 생성
envelope, err := sage.WrapMessage("transaction", payload)

// 메시지 언래핑
var data map[string]interface{}
err = sage.UnwrapMessage(envelope, &data)
```

### 직렬화/역직렬화

```go
// JSON 직렬화
bytes, err := sage.SerializeMessage(message)

// JSON 역직렬화
var message map[string]interface{}
err = sage.DeserializeMessage(bytes, &message)
```

### Base64 인코딩

```go
// 인코딩
encoded, err := sage.EncodeMessage(message)

// 디코딩
var message map[string]interface{}
err = sage.DecodeMessage(encoded, &message)
```

## 보안 고려사항

### 권장사항

1. **키 관리**: 개인키를 안전하게 저장 (HSM, KMS 권장)
2. **Nonce 캐시**: 기본 1000개, 필요시 증가
3. **Clock Skew**: 네트워크 환경에 맞게 조정
4. **세션 TTL**: 사용 패턴에 맞게 설정
5. **공개키 검증**: DID 문서에서 공개키 검증 필수

### 위협 모델

보호하는 공격:
- Man-in-the-Middle (MitM)
- Replay attacks
- Message tampering
- Session hijacking
- Forward secrecy breach

## 테스트

```bash
# 단위 테스트
go test ./adapters/sage -v

# 통합 테스트
go test ./adapters/sage -v -run TestIntegration

# 커버리지
go test ./adapters/sage -cover

# 벤치마크
go test ./adapters/sage -bench=.
```

## 성능

### 핸드셰이크

- Phase 1-4 완료: ~10ms (로컬)
- 세션 설정: 1회 (재사용 가능)

### 메시지 처리

- 암호화: ~0.1ms (ChaCha20-Poly1305)
- 서명: ~0.2ms (Ed25519)
- 검증: ~0.3ms (Ed25519 + BLAKE3)

### 메모리

- Transport Manager: ~100KB
- Session: ~2KB
- Active handshake: ~1KB

## 문제 해결

### 일반적인 오류

**"signature verification failed"**
- 원인: 잘못된 공개키 또는 메시지 변조
- 해결: DID 문서에서 올바른 공개키 확인

**"session not found"**
- 원인: 세션이 만료되었거나 존재하지 않음
- 해결: 핸드셰이크 재수행

**"timestamp outside acceptable clock skew"**
- 원인: 시스템 시간 불일치
- 해결: NTP 동기화 또는 MaxClockSkew 증가

**"nonce replay detected"**
- 원인: 동일한 nonce 재사용 시도
- 해결: 각 메시지마다 새로운 nonce 생성 확인

## 라이센스

LGPL-3.0-or-later

## 참고 문서

- [RFC 9421: HTTP Message Signatures](https://www.rfc-editor.org/rfc/rfc9421.html)
- [RFC 9180: HPKE](https://www.rfc-editor.org/rfc/rfc9180.html)
- [RFC 8032: EdDSA](https://www.rfc-editor.org/rfc/rfc8032.html)
- [BLAKE3 Specification](https://github.com/BLAKE3-team/BLAKE3-specs)

## 기여

이슈 및 PR은 GitHub 저장소에서 환영합니다:
https://github.com/sage-x-project/agent-develope-kit

## 지원

- 문서: https://docs.sage-x-project.org
- 이슈: https://github.com/sage-x-project/agent-develope-kit/issues
