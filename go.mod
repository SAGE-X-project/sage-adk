module github.com/sage-x-project/sage-adk

go 1.24.4

toolchain go1.24.7

// Utilities
require (
	github.com/google/uuid v1.6.0
	github.com/sage-x-project/sage v0.0.0
	trpc.group/trpc-go/trpc-a2a-go v0.0.0
)

require (
	filippo.io/edwards25519 v1.0.0-rc.1 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/ethereum/go-ethereum v1.16.1 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.4 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/sashabaranov/go-openai v1.41.2 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/oauth2 v0.26.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
)

// Use local modules for development
replace (
	github.com/sage-x-project/sage => ../../sage
	trpc.group/trpc-go/trpc-a2a-go => ../sage-a2a-go
)
