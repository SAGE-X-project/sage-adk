# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2025-10-11

### Added
- **Enterprise Features Complete**
  - gRPC server support with A2A protocol integration
  - Response caching system with LRU eviction
  - Distributed tracing with OpenTelemetry and Jaeger
  - Rate limiting with token bucket and sliding window algorithms
  - Multi-tenant support with per-tenant isolation
- **Observability Enhancements**
  - Comprehensive test coverage for tracing package (72.7%)
  - Comprehensive test coverage for cache package (62.1%)
  - Full test suite for cmd/adk CLI tool (39.3%)
- **Builder API Improvements**
  - Added WithDescription() and WithVersion() methods
  - Enhanced builder pattern for better agent construction
- **Examples**
  - Multi-agent chat system example
  - Multi-tenant architecture example
  - Rate limiting demonstration
  - All examples build successfully

### Changed
- Migrated examples to new builder API pattern
- Updated multi-agent-chat and multi-tenant examples to use builder pattern
- Improved error handling in CLI commands

### Fixed
- Build failures in multi-agent-chat and multi-tenant examples
- Git tracking of binary files (added to .gitignore)
- Build constraint examples now compile with -tags flag
- OpenTelemetry test API usage in tracing tests

## [0.1.0] - TBD

### Added
- Initial alpha release
- Basic agent functionality
- A2A protocol support
- SAGE security features
- LLM integrations
- Documentation

---

## Release Notes

### Version 0.1.0 (Alpha)

This is the initial alpha release of SAGE ADK. The framework provides:

**Core Features:**
- Agent builder with fluent API
- A2A protocol support via sage-a2a-go
- SAGE security protocol integration
- Multi-LLM support (OpenAI, Anthropic, Gemini)
- Flexible storage backends
- Production-ready features (metrics, health checks, logging)

**Known Limitations:**
- API may change before 1.0.0 release
- Limited test coverage in some areas
- Documentation in progress

**Breaking Changes:**
- N/A (initial release)

**Migration Guide:**
- N/A (initial release)

---

[Unreleased]: https://github.com/sage-x-project/sage-adk/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/sage-x-project/sage-adk/releases/tag/v1.2.0
[0.1.0]: https://github.com/sage-x-project/sage-adk/releases/tag/v0.1.0
