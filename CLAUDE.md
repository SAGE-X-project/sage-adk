# SAGE ADK - Claude AI Context

This file provides context for Claude AI when working with the SAGE ADK (Agent Development Kit) project.

## Project Overview

SAGE ADK is a Go framework for building secure, interoperable AI agents that supports:
- **Dual Protocol**: A2A (Agent-to-Agent) and SAGE (Secure Agent Guarantee Engine)
- **Multiple LLM Providers**: OpenAI, Anthropic, Gemini
- **Flexible Storage**: Memory, Redis, PostgreSQL
- **Production-Ready**: Built-in security, monitoring, and resilience

## Architecture Principles

### SOLID Principles
- **Single Responsibility**: Each package has one clear purpose
- **Open/Closed**: Extensible through interfaces, closed for modification
- **Liskov Substitution**: Implementations are interchangeable
- **Interface Segregation**: Small, focused interfaces
- **Dependency Inversion**: Depend on abstractions, not concrete implementations

### Key Design Patterns
- **Builder Pattern**: Fluent API for agent construction
- **Adapter Pattern**: Protocol and LLM provider abstraction
- **Strategy Pattern**: Protocol selection (A2A/SAGE/Auto)
- **Factory Pattern**: Component creation
- **Middleware Pattern**: Request processing pipeline

## Development Workflow

### 1. Design Phase
- Create design document in `docs/design-{YYYYMMDD-HHMMSS}-v{X.Y}.md`
- Reference existing code in `sage/`, `A2A/`, `sage-a2a-go/`
- Follow SOLID principles and consider extensibility
- Document interfaces, types, and interactions

### 2. TDD Implementation
- Write tests first (`*_test.go`)
- Implement to pass tests
- Refactor for quality
- Target 90%+ test coverage

### 3. Testing
- Run unit tests: `make test`
- Check coverage: `make test-coverage`
- Run all tests including existing ones
- Ensure no bugs or errors

### 4. Commit Process
- Create feature branch from `dev`
- Write commit message in English
- Remove co-author metadata
- Create PR to `dev` branch

## Project Structure

```
sage-adk/
 core/           # Core abstractions (agent, protocol, message)
 adapters/       # External integrations (a2a, sage, llm)
 builder/        # Fluent API builder
 server/         # HTTP/gRPC server
 client/         # Client SDK
 storage/        # Storage backends
 config/         # Configuration management
 security/       # Security features
 pkg/            # Public packages (types, errors, utils)
 internal/       # Private packages
 examples/       # Example projects
 test/           # Integration tests
 docs/           # Documentation
 cmd/            # CLI tools
```

## Key Dependencies

### External Projects (for reference)
- **sage/**: SAGE security framework with DID, signing, handshake
- **A2A/**: A2A protocol specification
- **sage-a2a-go/**: A2A protocol Go implementation

### Go Dependencies
- `github.com/sage-x-project/sage-a2a-go`: A2A protocol
- `github.com/sage-x-project/sage`: SAGE security
- `github.com/sashabaranov/go-openai`: OpenAI API
- `github.com/redis/go-redis/v9`: Redis client
- `github.com/spf13/viper`: Configuration
- `github.com/prometheus/client_golang`: Metrics

## Coding Standards

### File Organization
- One type per file (exceptions: small related types)
- Test files alongside implementation (`*_test.go`)
- Interfaces in separate files when large

### Naming Conventions
- Packages: lowercase, single word
- Types: PascalCase
- Functions/Methods: PascalCase (exported), camelCase (private)
- Constants: PascalCase or SCREAMING_SNAKE_CASE
- Interfaces: Often end with `-er` suffix

### Documentation
- Every exported type/function needs godoc comment
- Package-level doc in `doc.go`
- Complex logic needs inline comments
- Examples in `example_test.go`

### Error Handling
- Use custom error types in `pkg/errors`
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Return errors, don't panic (except initialization)

## Testing Guidelines

### Unit Tests
- Table-driven tests for multiple cases
- Mock external dependencies
- Test edge cases and error paths
- Target 90%+ coverage

### Integration Tests
- In `test/integration/`
- Use build tag: `//go:build integration`
- Test component interactions
- May require external services (Redis, etc.)

### Test Structure
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   Input
        want    Output
        wantErr bool
    }{
        {"success case", validInput, expectedOutput, false},
        {"error case", invalidInput, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Common Tasks

### Run Tests
```bash
make test              # Run all tests
make test-coverage     # With coverage report
make test-integration  # Integration tests only
```

### Build
```bash
make build            # Build binary
make build-all        # Build all components
```

### Development
```bash
make fmt              # Format code
make lint             # Run linters
make check            # fmt + vet + lint + test
```

## Context for Current Work

### Phase 1: Foundation (Current)
Working on core types, errors, and configuration management.

Reference implementations:
- **sage/core/**: RFC 9421, message processing
- **sage/crypto/**: Key management, storage
- **sage/did/**: DID management, resolution
- **sage-a2a-go/protocol/**: A2A types (Message, Task, Agent)
- **sage-a2a-go/taskmanager/**: Task lifecycle management

### Key Considerations
1. **Type System**: Must support both A2A and SAGE protocols
2. **Extensibility**: Easy to add new protocols, LLMs, storage
3. **Performance**: Minimize allocations, use connection pooling
4. **Security**: Validate all inputs, secure defaults
5. **Testability**: Interfaces for all dependencies

## Documentation References

- [Architecture Overview](docs/architecture/overview.md)
- [Protocol Layer](docs/architecture/protocol-layer.md)
- [Development Roadmap](docs/DEVELOPMENT_ROADMAP_v1.0_20251006-235205.md)
- [Task Priority Matrix](docs/TASK_PRIORITY_MATRIX_v1.0_20251006-235205.md)

## Important Notes

- All commit messages must be in English
- Remove co-author metadata from commits
- Always work in feature branches
- Create PRs to `dev` branch, not `main`
- Update design documents and checklists as work progresses
- Test coverage must be ≥90%
- Run full test suite before committing

---

**Last Updated**: 2025-10-06
**Project Version**: 0.1.0-alpha (in development)
