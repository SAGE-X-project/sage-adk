# SAGE ADK Project Structure

## Directory Layout

```
sage-adk/
 core/                          # Core ADK functionality
    agent/                     # Agent abstraction layer
       base.go               # BaseAgent interface
       lifecycle.go          # Agent lifecycle management
       registry.go           # Agent registry
       agent_test.go
    protocol/                  # Protocol integration layer
       a2a.go                # A2A protocol adapter
       sage.go               # SAGE protocol adapter
       selector.go           # Protocol selection logic
       protocol_test.go
    message/                   # Message processing
        router.go             # Message routing
        processor.go          # Message processor
        middleware.go         # Middleware chain
        message_test.go

 adapters/                      # External integrations
    a2a/                      # A2A protocol integration
       client.go             # sage-a2a-go client wrapper
       server.go             # sage-a2a-go server wrapper
       taskmanager.go       # TaskManager integration
       converter.go          # Type conversions
       a2a_test.go
    sage/                     # SAGE protocol integration
       security.go           # SAGE security features
       did.go                # DID management
       handshake.go          # Handshake protocol
       verifier.go           # RFC 9421 signature verification
       session.go            # Session management
       sage_test.go
    llm/                      # LLM integrations
        provider.go           # LLM Provider interface
        openai.go             # OpenAI API
        anthropic.go          # Anthropic Claude
        gemini.go             # Google Gemini
        config.go             # LLM configuration
        llm_test.go

 builder/                       # Agent builder pattern
    builder.go                # Fluent API builder
    options.go                # Agent options
    validator.go              # Configuration validator
    templates/                # Pre-defined templates
        simple.go             # Simple agent template
        conversational.go     # Conversational agent
        orchestrator.go       # Orchestrator agent

 server/                        # Server implementation
    http.go                   # HTTP server
    grpc.go                   # gRPC server (optional)
    router.go                 # HTTP router
    middleware/               # Server middleware
       auth.go              # Authentication
       logging.go           # Request logging
       metrics.go           # Prometheus metrics
       cors.go              # CORS support
       ratelimit.go         # Rate limiting
    handlers/                 # HTTP handlers
        a2a_handler.go       # A2A protocol handler
        health_handler.go    # Health check
        metrics_handler.go   # Metrics endpoint

 client/                        # Client SDK
    client.go                 # Unified client
    discovery.go              # Agent discovery
    connection.go             # Connection management
    client_test.go

 storage/                       # Storage abstraction
    interface.go              # Storage interface
    memory.go                 # In-memory storage
    redis.go                  # Redis storage
    postgres.go               # PostgreSQL storage
    storage_test.go

 config/                        # Configuration management
    config.go                 # Configuration struct
    loader.go                 # YAML/ENV loader
    validator.go              # Configuration validator
    defaults.go               # Default values
    config_test.go

 security/                      # Security features
    protocol_switch.go        # Protocol switching logic
    signature.go              # Message signing
    validation.go             # Signature validation
    did_cache.go              # DID resolution cache
    security_test.go

 examples/                      # Example projects
    simple-agent/             # Basic A2A agent
       main.go
       .env.example
       README.md
    sage-enabled-agent/       # SAGE security agent
       main.go
       config.yaml
       .env.example
       README.md
    multi-llm-agent/          # Multiple LLM providers
       main.go
       handlers.go
       README.md
    orchestrator/             # Multi-agent orchestrator
        main.go
        agents/
        README.md

 cmd/                           # CLI tools
    adk/                      # Main CLI
       main.go
    commands/                 # CLI commands
        init.go               # Initialize project
        generate.go           # Code generation
        serve.go              # Run agent
        register.go           # DID registration
        validate.go           # Config validation

 pkg/                           # Public packages
    types/                    # Common types
       message.go
       task.go
       agent.go
       security.go
    errors/                   # Error definitions
       errors.go
       codes.go
    utils/                    # Utilities
        id.go                 # ID generation
        crypto.go             # Crypto helpers
        http.go               # HTTP helpers

 internal/                      # Private packages
    codec/                    # Encoding/Decoding
       json.go
       protobuf.go
    transport/                # Transport logic
        http.go
        grpc.go

 docs/                          # Documentation
    README.md                 # Documentation home
    overview.md               # ADK overview
    architecture/             # Architecture docs
       overview.md
       protocol-layer.md
       message-flow.md
       security-model.md
    guides/                   # User guides
       quick-start.md
       installation.md
       configuration.md
       building-agents.md
       a2a-protocol.md
       sage-integration.md
       llm-providers.md
       storage-backends.md
       multi-agent-systems.md
    api/                      # API reference
       agent.md
       message.md
       protocol.md
       llm.md
       storage.md
    examples/                 # Example documentation
        simple-agent.md
        sage-agent.md
        multi-llm-agent.md
        orchestrator-agent.md
        production-deployment.md

 scripts/                       # Build and deployment scripts
    setup.sh                  # Initial setup
    build.sh                  # Build binaries
    test.sh                   # Run tests
    deploy.sh                 # Deployment script
    generate-docs.sh          # Generate documentation

 test/                          # Integration tests
    integration/              # Integration tests
       a2a_test.go
       sage_test.go
       e2e_test.go
    fixtures/                 # Test fixtures
    mocks/                    # Mock implementations

 go.mod                         # Go module definition
 go.sum                         # Dependency checksums
 Makefile                       # Build automation
 Dockerfile                     # Docker image
 docker-compose.yml             # Docker compose for development
 .env.example                   # Environment variables template
 config.yaml.example            # YAML config template
 .gitignore                     # Git ignore rules
 LICENSE                        # LGPL-3.0 license
 README.md                      # Project README
 CONTRIBUTING.md                # Contribution guidelines
 CHANGELOG.md                   # Version history
```

## Package Descriptions

### Core Packages

#### `core/agent`
Provides the core agent abstraction and lifecycle management.

**Key Types**:
- `Agent`: Main agent interface
- `BaseAgent`: Default agent implementation
- `AgentRegistry`: Multi-agent registry

#### `core/protocol`
Protocol abstraction layer for A2A and SAGE.

**Key Types**:
- `Protocol`: Protocol interface
- `ProtocolSelector`: Protocol selection logic
- `ProtocolMode`: A2A, SAGE, or Auto

#### `core/message`
Message routing and processing.

**Key Types**:
- `Router`: Message router
- `Processor`: Message processor
- `Middleware`: Middleware chain

### Adapter Packages

#### `adapters/a2a`
Wraps `sage-a2a-go` library for A2A protocol support.

**Dependencies**:
- `github.com/sage-x-project/sage-a2a-go`

#### `adapters/sage`
Integrates SAGE security library.

**Dependencies**:
- `github.com/sage-x-project/sage`

#### `adapters/llm`
LLM provider integrations (OpenAI, Anthropic, Gemini).

**Dependencies**:
- `github.com/sashabaranov/go-openai`
- `github.com/anthropics/anthropic-sdk-go`
- `google.golang.org/api`

### Builder Package

#### `builder`
Fluent API for building agents.

**Key Types**:
- `Builder`: Agent builder
- `Options`: Configuration options
- `Validator`: Configuration validator

### Server Package

#### `server`
HTTP/gRPC server implementation.

**Key Types**:
- `Server`: Main server
- `Router`: HTTP router
- Middleware: Auth, logging, metrics, CORS, rate limiting

### Storage Package

#### `storage`
Pluggable storage backends.

**Implementations**:
- Memory (in-process)
- Redis (production)
- PostgreSQL (enterprise)

### Configuration Package

#### `config`
Configuration loading and validation.

**Key Types**:
- `Config`: Configuration struct
- `Loader`: YAML/ENV loader
- `Validator`: Configuration validator

## File Naming Conventions

- **Implementation files**: `{feature}.go` (e.g., `agent.go`, `router.go`)
- **Test files**: `{feature}_test.go` (e.g., `agent_test.go`)
- **Interface files**: `interface.go` (in each package)
- **Mock files**: `mock_{feature}.go` (e.g., `mock_llm.go`)
- **Example files**: `example_test.go` (runnable examples)

## Import Paths

```go
import (
    // Core
    "github.com/sage-x-project/sage-adk/core/agent"
    "github.com/sage-x-project/sage-adk/core/protocol"
    "github.com/sage-x-project/sage-adk/core/message"

    // Adapters
    "github.com/sage-x-project/sage-adk/adapters/a2a"
    "github.com/sage-x-project/sage-adk/adapters/sage"
    "github.com/sage-x-project/sage-adk/adapters/llm"

    // Builder
    "github.com/sage-x-project/sage-adk/builder"

    // Server
    "github.com/sage-x-project/sage-adk/server"
    "github.com/sage-x-project/sage-adk/server/middleware"

    // Storage
    "github.com/sage-x-project/sage-adk/storage"

    // Config
    "github.com/sage-x-project/sage-adk/config"

    // Public packages
    "github.com/sage-x-project/sage-adk/pkg/types"
    "github.com/sage-x-project/sage-adk/pkg/errors"
    "github.com/sage-x-project/sage-adk/pkg/utils"
)
```

## Build Targets

```makefile
# Makefile targets
make build          # Build all binaries
make test           # Run all tests
make test-coverage  # Generate coverage report
make lint           # Run linters
make fmt            # Format code
make docs           # Generate documentation
make docker         # Build Docker image
make clean          # Clean build artifacts
```

## Development Workflow

1. **Initialize**: `make setup` - Install dependencies
2. **Develop**: Write code, tests
3. **Test**: `make test` - Run tests
4. **Lint**: `make lint` - Check code quality
5. **Build**: `make build` - Build binaries
6. **Document**: Update docs in `docs/`
7. **Commit**: Follow conventional commits

## Versioning

SAGE ADK follows [Semantic Versioning](https://semver.org/):

- **Major**: Breaking changes
- **Minor**: New features (backward compatible)
- **Patch**: Bug fixes

Current version: `v0.1.0` (alpha)

---

[← Documentation Home](README.md)
