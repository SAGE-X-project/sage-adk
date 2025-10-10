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
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.23.2
	github.com/redis/go-redis/v9 v9.7.0
	github.com/sashabaranov/go-openai v1.41.2
	github.com/spf13/cobra v1.8.1
	golang.org/x/crypto v0.43.0
	gopkg.in/yaml.v3 v3.0.1
	lukechampine.com/blake3 v1.4.1
)

require (
	filippo.io/edwards25519 v1.0.0-rc.1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.20.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/consensys/gnark-crypto v0.18.0 // indirect
	github.com/crate-crypto/go-eth-kzg v1.3.0 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.0 // indirect
	github.com/ethereum/go-ethereum v1.16.1 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/lestrrat-go/blackmagic v1.0.2 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.4 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/supranational/blst v0.3.14 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

// Use local modules for development
replace (
	github.com/sage-x-project/sage => ../../sage
	trpc.group/trpc-go/trpc-a2a-go => ../sage-a2a-go
)
