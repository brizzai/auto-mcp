# Auto MCP

[![Go Report Card](https://goreportcard.com/badge/github.com/brizzai/auto-mcp)](https://goreportcard.com/report/github.com/brizzai/auto-mcp)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/brizzai/auto-mcp)](https://github.com/brizzai/auto-mcp/releases/latest)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go version](https://img.shields.io/github/go-mod/go-version/brizzai/auto-mcp)](https://golang.org/doc/devel/release.html)
[![Container Registry](https://img.shields.io/badge/container-ghcr.io-blue)](https://github.com/brizzai/auto-mcp/pkgs/container/auto-mcp)
[![Build Status](https://img.shields.io/github/workflow/status/brizzai/auto-mcp/Go)](https://github.com/brizzai/auto-mcp/actions)

Transform any OpenAPI/Swagger definition into a fully-featured **Model Context Protocol (MCP)** server – ready to run locally, inside Claude Desktop, or in the cloud.

The service reads a Swagger (OpenAPI v2) document, generates routes on-the-fly, proxies requests to the upstream endpoint you configure, and exposes them through MCP using either the **STDIO** or **SSE** transport defined in the [MCP specification](https://modelcontextprotocol.io/introduction).

---

## 🤝 Contributing

Issues and pull requests are very welcome. Please follow conventional commits and run `make lint test` before opening a PR.

This project uses pre-commit hooks to ensure code quality. To set up pre-commit:
```bash
pip install pre-commit
pre-commit install
```

For detailed contribution guidelines, please see [CONTRIBUTING.md](.github/CONTRIBUTING.md).

---

## 📄 License

  Distributed under the Apache License 2.0 License. See [`LICENSE`](LICENSE) for details.
