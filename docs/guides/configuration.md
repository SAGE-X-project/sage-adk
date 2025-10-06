# Configuration Guide

This guide covers all configuration options for SAGE ADK agents, including environment variables, YAML configuration, and programmatic setup.

## Configuration Methods

SAGE ADK supports three configuration methods with the following priority:

1. **Programmatic** (highest priority) - Set directly in code
2. **Environment Variables** - Load from `.env` file or system environment
3. **YAML Configuration File** - Load from `config.yaml`
4. **Defaults** (lowest priority) - Built-in sensible defaults

## Environment Variables

### Core Agent Settings

```bash
# Agent Identity
ADK_AGENT_NAME=my-agent               # Agent identifier
ADK_AGENT_DESCRIPTION="My AI Agent"   # Human-readable description
ADK_AGENT_VERSION=1.0.0               # Agent version

# Server Configuration
ADK_SERVER_HOST=0.0.0.0               # Bind address (0.0.0.0 for all interfaces)
ADK_SERVER_PORT=8080                  # HTTP port
ADK_SERVER_TIMEOUT=30s                # Request timeout
ADK_SERVER_MAX_BODY_SIZE=10MB         # Maximum request body size

# Protocol Selection
ADK_PROTOCOL_MODE=auto                # a2a | sage | auto
```

### A2A Protocol Settings

```bash
# A2A Protocol Version
A2A_PROTOCOL_VERSION=0.2.2            # A2A protocol version

# Storage Backend
A2A_STORAGE_TYPE=redis                # memory | redis | postgres
A2A_STORAGE_TTL=1h                    # Task/message expiration time
A2A_MAX_HISTORY_LENGTH=100            # Maximum message history per context

# Redis Configuration (when A2A_STORAGE_TYPE=redis)
A2A_REDIS_URL=redis://localhost:6379  # Redis connection URL
A2A_REDIS_PASSWORD=                   # Redis password (if required)
A2A_REDIS_DB=0                        # Redis database number
A2A_REDIS_POOL_SIZE=10                # Connection pool size

# PostgreSQL Configuration (when A2A_STORAGE_TYPE=postgres)
A2A_POSTGRES_URL=postgresql://user:pass@localhost:5432/adk
A2A_POSTGRES_MAX_CONNECTIONS=20
A2A_POSTGRES_IDLE_CONNECTIONS=5
```

### SAGE Security Settings

```bash
# SAGE Protocol Enable/Disable
SAGE_ENABLED=true                     # Enable SAGE security features

# Agent DID (Decentralized Identifier)
SAGE_DID=did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678

# Blockchain Network
SAGE_NETWORK=ethereum                 # ethereum | sepolia | kaia | kairos | local

# Ethereum Mainnet
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY
ETHEREUM_CONTRACT_ADDRESS=0x...       # SAGE Registry contract
ETHEREUM_CHAIN_ID=1

# Ethereum Sepolia Testnet
SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
SEPOLIA_CONTRACT_ADDRESS=0x...
SEPOLIA_CHAIN_ID=11155111

# Kaia Mainnet (Cypress)
KAIA_RPC_URL=https://public-en-cypress.klaytn.net
KAIA_CONTRACT_ADDRESS=0x...
KAIA_CHAIN_ID=8217

# Kaia Testnet (Kairos)
KAIROS_RPC_URL=https://public-en-kairos.node.kaia.io
KAIROS_CONTRACT_ADDRESS=0x...
KAIROS_CHAIN_ID=1001

# Local Development (Hardhat)
LOCAL_RPC_URL=http://127.0.0.1:8545
LOCAL_CONTRACT_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
LOCAL_CHAIN_ID=31337

# Private Key for Signing (NEVER commit to git!)
SAGE_PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# Or use mnemonic (NEVER commit to git!)
SAGE_MNEMONIC=test test test test test test test test test test test junk

# Security Features
SAGE_HANDSHAKE_ENABLED=true           # Enable secure handshake protocol
SAGE_ENCRYPTION_ENABLED=true          # Enable message encryption
SAGE_SIGNATURE_REQUIRED=true          # Require signatures on all messages
SAGE_SESSION_MAX_AGE=1h               # Session expiration time
SAGE_SESSION_IDLE_TIMEOUT=10m         # Session idle timeout
SAGE_NONCE_CACHE_SIZE=10000           # Replay protection cache size
SAGE_NONCE_CACHE_TTL=5m               # Nonce expiration time

# DID Cache
SAGE_DID_CACHE_ENABLED=true           # Cache DID resolutions
SAGE_DID_CACHE_TTL=1h                 # Cache entry TTL
SAGE_DID_CACHE_SIZE=1000              # Maximum cache entries
```

### LLM Provider Configuration

```bash
# LLM Provider Selection
LLM_PROVIDER=openai                   # openai | anthropic | gemini | custom
LLM_MODEL=gpt-4                       # Model name
LLM_MAX_TOKENS=2048                   # Maximum tokens per response
LLM_TEMPERATURE=0.7                   # Sampling temperature (0.0-1.0)
LLM_TOP_P=1.0                         # Nucleus sampling parameter

# OpenAI Configuration
OPENAI_API_KEY=sk-...                 # OpenAI API key
OPENAI_ORG_ID=org-...                 # Organization ID (optional)
OPENAI_BASE_URL=https://api.openai.com/v1  # Base URL (for proxies)

# Anthropic Configuration
ANTHROPIC_API_KEY=sk-ant-...          # Anthropic API key
ANTHROPIC_VERSION=2023-06-01          # API version

# Google Gemini Configuration
GEMINI_API_KEY=...                    # Gemini API key
GEMINI_PROJECT_ID=...                 # GCP Project ID
GEMINI_LOCATION=us-central1           # GCP region

# LLM Retry & Timeout
LLM_TIMEOUT=30s                       # Request timeout
LLM_MAX_RETRIES=3                     # Maximum retry attempts
LLM_RETRY_DELAY=1s                    # Initial retry delay (exponential backoff)
```

### Logging & Monitoring

```bash
# Logging
LOG_LEVEL=info                        # debug | info | warn | error
LOG_FORMAT=json                       # json | text
LOG_OUTPUT=stdout                     # stdout | stderr | file
LOG_FILE_PATH=/var/log/adk/agent.log  # Log file path (if LOG_OUTPUT=file)

# Metrics
METRICS_ENABLED=true                  # Enable Prometheus metrics
METRICS_PORT=9090                     # Metrics endpoint port
METRICS_PATH=/metrics                 # Metrics endpoint path

# Health Check
HEALTH_CHECK_ENABLED=true             # Enable health check endpoint
HEALTH_CHECK_PATH=/health             # Health check path

# Tracing (OpenTelemetry)
OTEL_ENABLED=false                    # Enable distributed tracing
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_SERVICE_NAME=sage-adk-agent
```

### Security & Authentication

```bash
# CORS Configuration
CORS_ENABLED=true                     # Enable CORS
CORS_ALLOWED_ORIGINS=*                # Allowed origins (comma-separated)
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE
CORS_ALLOWED_HEADERS=Content-Type,Authorization
CORS_MAX_AGE=3600                     # Preflight cache duration (seconds)

# API Authentication
AUTH_ENABLED=false                    # Enable API key authentication
AUTH_TYPE=api_key                     # api_key | jwt | oauth2
AUTH_API_KEY=your-secret-api-key      # Static API key
AUTH_HEADER_NAME=X-API-Key            # Header name for API key

# JWT Authentication
JWT_SECRET=your-jwt-secret            # JWT signing secret
JWT_ISSUER=sage-adk                   # JWT issuer
JWT_AUDIENCE=sage-adk-agent           # JWT audience
JWT_EXPIRY=24h                        # Token expiry

# Rate Limiting
RATE_LIMIT_ENABLED=true               # Enable rate limiting
RATE_LIMIT_REQUESTS=100               # Requests per window
RATE_LIMIT_WINDOW=1m                  # Time window
```

## YAML Configuration File

Create `config.yaml` in your project root:

```yaml
# Agent Configuration
agent:
  name: my-agent
  description: "My AI Agent"
  version: 1.0.0

# Server Configuration
server:
  host: 0.0.0.0
  port: 8080
  timeout: 30s
  max_body_size: 10MB
  cors:
    enabled: true
    allowed_origins:
      - "*"

# Protocol Configuration
protocol:
  mode: auto  # a2a | sage | auto

# A2A Configuration
a2a:
  protocol_version: 0.2.2
  storage:
    type: redis
    ttl: 1h
    max_history_length: 100
  redis:
    url: redis://localhost:6379
    password: ""
    db: 0
    pool_size: 10

# SAGE Configuration
sage:
  enabled: true
  did: did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678
  network: ethereum

  blockchain:
    ethereum:
      rpc_url: https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
      contract_address: 0x...
      chain_id: 1
    kaia:
      rpc_url: https://public-en-cypress.klaytn.net
      contract_address: 0x...
      chain_id: 8217

  security:
    handshake_enabled: true
    encryption_enabled: true
    signature_required: true

  session:
    max_age: 1h
    idle_timeout: 10m

  cache:
    did_cache_enabled: true
    did_cache_ttl: 1h
    did_cache_size: 1000

# LLM Configuration
llm:
  provider: openai
  model: gpt-4
  max_tokens: 2048
  temperature: 0.7

  openai:
    api_key: sk-...
    org_id: org-...

  retry:
    max_retries: 3
    timeout: 30s

# Logging Configuration
logging:
  level: info
  format: json
  output: stdout

# Metrics Configuration
metrics:
  enabled: true
  port: 9090
  path: /metrics

# Health Check Configuration
health:
  enabled: true
  path: /health
```

## Programmatic Configuration

### Basic Configuration

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
    "github.com/sage-x-project/sage-adk/adapters/sage"
    "github.com/sage-x-project/sage-adk/storage"
)

func main() {
    agent := adk.NewAgent("my-agent").
        // Protocol
        WithProtocol(adk.ProtocolAuto).

        // LLM
        WithLLM(llm.OpenAI(llm.OpenAIOptions{
            APIKey:      "sk-...",
            Model:       "gpt-4",
            MaxTokens:   2048,
            Temperature: 0.7,
        })).

        // Storage
        WithStorage(storage.Redis(storage.RedisOptions{
            URL:      "redis://localhost:6379",
            Password: "",
            DB:       0,
        })).

        // SAGE Security
        WithSAGE(sage.Options{
            DID:     "did:sage:ethereum:0x...",
            Network: sage.NetworkEthereum,
            RPC:     "https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY",
            ContractAddress: "0x...",
            PrivateKey: privateKey,
        }).

        // Build
        Build()

    agent.Start(":8080")
}
```

### Loading from Environment

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/adapters/llm"
    "github.com/sage-x-project/sage-adk/adapters/sage"
    "github.com/sage-x-project/sage-adk/config"
)

func main() {
    // Load configuration from environment
    cfg, err := config.LoadFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    agent := adk.NewAgent(cfg.Agent.Name).
        WithConfig(cfg).
        WithLLM(llm.FromConfig(cfg.LLM)).
        WithSAGE(sage.FromConfig(cfg.SAGE)).
        Build()

    agent.Start(cfg.Server.Address())
}
```

### Loading from YAML File

```go
package main

import (
    "github.com/sage-x-project/sage-adk/adk"
    "github.com/sage-x-project/sage-adk/config"
)

func main() {
    // Load configuration from YAML
    cfg, err := config.LoadFromFile("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    agent := adk.NewAgent(cfg.Agent.Name).
        WithConfig(cfg).
        Build()

    agent.Start(cfg.Server.Address())
}
```

## Environment File Example

Create `.env` file:

```bash
# .env - Development Configuration

# Agent
ADK_AGENT_NAME=dev-agent
ADK_AGENT_DESCRIPTION="Development Agent"
ADK_SERVER_PORT=8080

# Protocol
ADK_PROTOCOL_MODE=a2a

# A2A
A2A_STORAGE_TYPE=memory

# LLM
LLM_PROVIDER=openai
LLM_MODEL=gpt-3.5-turbo
OPENAI_API_KEY=sk-...

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
```

Create `.env.production`:

```bash
# .env.production - Production Configuration

# Agent
ADK_AGENT_NAME=prod-agent
ADK_AGENT_DESCRIPTION="Production Agent"
ADK_SERVER_PORT=8080

# Protocol
ADK_PROTOCOL_MODE=auto

# A2A
A2A_STORAGE_TYPE=redis
A2A_REDIS_URL=redis://redis:6379

# SAGE
SAGE_ENABLED=true
SAGE_DID=did:sage:ethereum:0x...
SAGE_NETWORK=ethereum
ETHEREUM_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
ETHEREUM_CONTRACT_ADDRESS=0x...

# LLM
LLM_PROVIDER=openai
LLM_MODEL=gpt-4
OPENAI_API_KEY=sk-...

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Metrics
METRICS_ENABLED=true
HEALTH_CHECK_ENABLED=true
```

## Configuration Validation

ADK validates configuration at build time:

```go
agent, err := adk.NewAgent("my-agent").
    WithLLM(llm.OpenAI()).
    Build()

if err != nil {
    // Configuration validation errors
    log.Fatal(err)
}
```

Common validation errors:

- Missing required fields (e.g., LLM API key)
- Invalid protocol mode
- Invalid network configuration
- Invalid DID format
- Missing blockchain RPC URL

## Best Practices

1. **Never commit secrets**: Use `.gitignore` for `.env` files
2. **Use environment-specific files**: `.env.development`, `.env.production`
3. **Provide defaults**: Use YAML for defaults, override with env vars
4. **Validate early**: Validate configuration at startup
5. **Document options**: Keep this guide updated with new options

---

[← Quick Start](quick-start.md) | [Building Agents →](building-agents.md)
