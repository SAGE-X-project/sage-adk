# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure
- Core agent abstraction layer
- A2A protocol adapter (sage-a2a-go integration)
- SAGE protocol adapter (sage security integration)
- LLM provider abstraction (OpenAI, Anthropic, Gemini)
- Fluent API builder pattern
- HTTP server implementation
- Storage backends (Memory, Redis, PostgreSQL)
- Configuration management (YAML, ENV)
- Protocol switching logic (A2A/SAGE/Auto)
- Comprehensive documentation
- Example projects
- Docker support
- Makefile for common tasks

### Changed
- N/A (initial release)

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- SAGE protocol integration for secure communication
- DID-based identity verification
- RFC 9421 message signatures
- End-to-end encryption support

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

[Unreleased]: https://github.com/sage-x-project/sage-adk/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/sage-x-project/sage-adk/releases/tag/v0.1.0
