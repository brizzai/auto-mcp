# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

## [0.1.0] - 2025-05-16

### Added
- Core MCP server implementation (STDIO and SSE modes)
- Flexible configuration: CLI flags, environment variables, `.env.dev`, and `config.yaml`
- Swagger/OpenAPI v2 parser and dynamic route generation
- HTTP requester with support for bearer, basic, API key, OAuth2, or no authentication
- Terminal UI (TUI) for interactive endpoint adjustment and config builder
- Adjustment file support for route filtering and description customization
- Docker and distroless container support for secure deployment
- CLI and Docker usage documentation
- Project structure following Go best practices (cmd/, internal/, build/)
- Makefile for build automation
- Logging and error handling modules
- Example and test files for core components
- Container registry and release automation via GoReleaser
- Contribution guidelines and pre-commit hooks

[Unreleased]: https://github.com/brizzai/auto-mcp/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/brizzai/auto-mcp/releases/tag/v0.1.0
