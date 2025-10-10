# Contributing to SAGE ADK

Thank you for your interest in contributing to SAGE Agent Development Kit! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue with:

- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages

### Suggesting Enhancements

For feature requests or enhancements:

- Check existing issues first to avoid duplicates
- Provide clear use case and rationale
- Describe proposed solution if possible
- Consider backward compatibility

### Pull Requests

1. **Fork the repository**
   ```bash
   git clone https://github.com/sage-x-project/sage-adk.git
   cd sage-adk
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **Make your changes**
   - Follow the coding standards below
   - Add tests for new functionality
   - Update documentation as needed

4. **Run tests and checks**
   ```bash
   make check
   ```

5. **Commit your changes**
   ```bash
   git commit -m 'Add amazing feature'
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test additions/changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks

6. **Push to your fork**
   ```bash
   git push origin feature/amazing-feature
   ```

7. **Open a Pull Request**
   - Provide clear description of changes
   - Reference related issues
   - Ensure CI checks pass

## Development Setup

### Prerequisites

- Go 1.24 or higher
- Git
- Make
- Docker (optional, for testing)

### Setup

```bash
# Clone repository
git clone https://github.com/sage-x-project/sage-adk.git
cd sage-adk

# Install dependencies
make setup

# Copy environment file
cp .env.example .env

# Run tests
make test
```

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint` before submitting

### Code Organization

- Keep packages focused and cohesive
- Minimize dependencies between packages
- Use interfaces for abstraction
- Write clear, self-documenting code

### Documentation

- Add godoc comments to all exported types and functions
- Update README.md for user-facing changes
- Add examples for new features
- Keep documentation in sync with code

### Testing

- Write unit tests for new code
- Aim for >80% code coverage
- Include integration tests where appropriate
- Use table-driven tests for multiple cases

```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"case1", "input1", "output1", false},
        {"case2", "input2", "output2", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Feature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Project Structure

Understand the project layout before contributing:

```
sage-adk/
 core/           # Core functionality
 adapters/       # External integrations
 builder/        # Fluent API builder
 server/         # HTTP/gRPC server
 client/         # Client SDK
 storage/        # Storage backends
 config/         # Configuration management
 examples/       # Example projects
 docs/           # Documentation
```

## Testing Guidelines

### Unit Tests

- Test individual functions/methods
- Mock external dependencies
- Use `t.Parallel()` when safe

### Integration Tests

- Test component interactions
- Use `//go:build integration` tag
- Run with `make test-integration`

### Benchmarks

- Add benchmarks for performance-critical code
- Run with `make bench`

## Documentation

### Code Documentation

- Add godoc comments to all exported items
- Include usage examples
- Document edge cases and limitations

### User Documentation

- Update relevant docs in `docs/` directory
- Add examples to `examples/` if applicable
- Update README.md for major changes

## Review Process

1. **Automated Checks**: CI runs tests, linting, and builds
2. **Code Review**: Maintainers review for:
   - Code quality and style
   - Test coverage
   - Documentation
   - Breaking changes
3. **Approval**: Requires at least one maintainer approval
4. **Merge**: Squash and merge after approval

## Release Process

Maintainers handle releases:

1. Update CHANGELOG.md
2. Create version tag
3. CI builds and publishes artifacts
4. Update documentation

## Questions?

- Open a [Discussion](https://github.com/sage-x-project/sage-adk/discussions)

## License

By contributing, you agree that your contributions will be licensed under the LGPL-3.0 License.

---

Thank you for contributing to SAGE ADK!
