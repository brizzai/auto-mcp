# Auto MCP

[![Go Report Card](https://goreportcard.com/badge/github.com/brizzai/auto-mcp)](https://goreportcard.com/report/github.com/brizzai/auto-mcp)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/brizzai/auto-mcp)](https://github.com/brizzai/auto-mcp/releases/latest)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go version](https://img.shields.io/github/go-mod/go-version/brizzai/auto-mcp)](https://golang.org/doc/devel/release.html)
[![Container Registry](https://img.shields.io/badge/container-ghcr.io-blue)](https://github.com/brizzai/auto-mcp/pkgs/container/auto-mcp)
[![Build Status](https://img.shields.io/github/actions/workflow/status/brizzai/auto-mcp/auto-mcp-tests.yml?branch=master)](https://github.com/brizzai/auto-mcp/actions/workflows/auto-mcp-tests.yml)

Transform any OpenAPI/Swagger definition into a fully-featured **Model Context Protocol (MCP)** server – ready to run locally, inside Claude Desktop, or in the cloud.

The service reads a Swagger (OpenAPI v2) document, generates routes on-the-fly, proxies requests to the upstream endpoint you configure, and exposes them through MCP using either the **STDIO** or **SSE** transport defined in the [MCP specification](https://modelcontextprotocol.io/introduction).

---

## ✨ Why Auto MCP?

- **Zero boiler-plate** – bring your `swagger.json` and start serving.
- **Flexible deployment** – run as a CLI, long-lived daemon, or within Docker/Kubernetes.
- **Two transport modes** –
  - `stdio` (default).
  - `sse` – self-hosted long-running event source.
- **Pluggable auth** – bearer token, basic auth, API keys, OAuth2 or no auth.
- **Runtime configuration** – YAML file, CLI flags, or environment variables (prefixed `AUTO_MCP_`).

---

## 📚 Use Cases

1. **Rapidly expose any REST API to LLMs** – no code generation, perfect for prototyping MCP integrations.
2. **Bridge legacy services** – wrap an existing API and unlock Claude Desktop or any MCP-compliant client.
3. **Ephemeral chat jobs** – spin up `auto-mcp` in `stdio` mode for one-off CLI sessions.
4. **Shared staging environments** – deploy once in `sse` mode and reuse across multiple experiments.

---

## 🛠️ MCP Config Builder

Easily tailor your Swagger/OpenAPI file for optimal MCP integration. The MCP Config Builder lets you:

- **Edit endpoint descriptions** for clearer, more helpful documentation.
- **Filter out unnecessary routes** to streamline your API exposure.
- **Preview and customize** how endpoints appear to LLMs and clients.
- **Generate an adjustment file** (`--adjustment-file`) for use with Auto MCP, applying your customizations automatically.

![MCP Config Builder](docs/mcp-config-builder.gif)

### How it works

1. **Install the MCP Config Builder:**
   ```bash
   go install ./cmd/mcp-config-builder
   ```
   This will build and install the `mcp-config-builder` binary to your `$GOPATH/bin` (usually `~/go/bin`). Make sure this directory is in your `PATH`.
2. **Launch the tool:**
   ```bash
   mcp-config-builder --swagger-file=/path/to/swagger.json
   ```
3. **Interactively review and edit** endpoints in a user-friendly TUI (Terminal User Interface).
4. **Save your adjustments** to a file for future use or sharing.
5. **Run Auto MCP** with your adjustment file to apply your customizations:
   ```bash
   auto-mcp --swagger-file=/path/to/swagger.json --adjustment-file=/path/to/adjustments.json
   ```

**Tip:** Use the adjustment file to keep your API documentation clean and focused, especially when exposing large or legacy APIs to LLMs.

---

## 🚀 Installation

### 🖥️ Running inside Claude Desktop

Add the following snippet to your **Claude Desktop** configuration (⟂ _Settings → MCP Servers_):

```jsonc
{
  "mcpServers": {
    "YourMCP": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-v",
        "/Users/you/path/to/swagger.json:/server/swagger.json",
        "ghcr.io/brizzai/auto-mcp:latest"
        "--swagger-file=/server/swagger.json"
      ]
    }
  },
  "globalShortcut": ""
}
```

Claude will start the container on-demand and connect over STDIO. Replace the host path to `swagger.json` and image tag to suit your setup.

---

## 🏗️ Project Structure

Auto MCP follows the standard Go project layout:

- **`cmd/auto-mcp/`**: Contains the main entry point for the application
  - `main.go`: The main function that serves as the entry point
- **`internal/`**: Private application code not meant to be imported by other projects
  - `config/`: Configuration loading and parsing
  - `logger/`: Application logging setup
  - `parser/`: Swagger/OpenAPI parsing
  - `requester/`: Handles external API requests
  - `server/`: MCP server implementation (STDIO/SSE)
- **`build/`**: Compiled application binaries

To build the project:

```bash
make build   # Binary will be in build/auto-mcp
```

---

## ⚙️ Configuration

Auto MCP accepts configuration via **CLI flags**, **environment variables** (prefix `AUTO_MCP_`), or an optional `config.yaml`. In containerized deployments environment variables are the most convenient.

| Purpose                               | Env variable                          | Example                          |
| ------------------------------------- | ------------------------------------- | -------------------------------- |
| Select transport                      | `AUTO_MCP_SERVER_MODE`                | `stdio` or `sse`                 |
| Bind port (SSE)                       | `AUTO_MCP_SERVER_PORT`                | `8080`                           |
| Upstream base URL                     | `AUTO_MCP_ENDPOINT_BASE_URL`          | `https://petstore.swagger.io/v2` |
| Authentication type                   | `AUTO_MCP_ENDPOINT_AUTH_TYPE`         | `bearer`                         |
| Bearer/OAuth token                    | `AUTO_MCP_ENDPOINT_AUTH_CONFIG_TOKEN` | `123456`                         |
| Extra static header                   | `AUTO_MCP_ENDPOINT_HEADERS_X_CUSTOM`  | `hello`                          |
| Log level                             | `AUTO_MCP_LOGGING_LEVEL`              | `debug`                          |
| Path to swagger file                  | `AUTO_MCP_SWAGGER_FILE`               | `/server/swagger.json`           |
| Path to adjustment file (mcp-builder) | `AUTO_MCP_ADJUSTMENTS_FILE`           | `/server/swagger.json`           |

Underscores replace dots in the YAML path; nested keys keep the hierarchy (e.g., `endpoint.auth_config.token` → `AUTO_MCP_ENDPOINT_AUTH_CONFIG_TOKEN`).

CLI shortcuts:

- `--mode` – overrides the transport.
- `--swagger-file` – absolute or relative path to the OpenAPI document.
- `--adjustment-file` - mcp-config-builder output filter/change route descriptions

### CLI flags

- `--mode` – override `server.mode` (`stdio` or `sse`).
- `--swagger-file` – path to the OpenAPI document (default: `swagger.json`).
- `--adjustment-file` - mcp-config-builder output filter/change route descriptions

### Environment variables

Underscores replace dots and keys are upper-cased. For example, to change the port and log level when using Docker:

```bash
# Unix shell
docker run -e AUTO_MCP_SERVER_PORT=8080 \
           -v $(pwd)/swagger.json:/server/swagger.json \
           -p 8080:8080 auto-mcp:latest --mode=sse --swagger-file=/swagger.json
```

with adjustments

```bash
# Run with adjustment file
docker run --rm -i \
  -v $(pwd)/swagger.json:/server/swagger.json \
  -v $(pwd)/adjustments.json:/server/adjustments.json \
  auto-mcp:latest --mode=stdio \
  --swagger-file=/server/swagger.json \
  --adjustment-file=/server/adjustments.json
```

---

## 🔐 OAuth Support

Auto MCP supports OAuth 2.1 authentication, including PKCE, dynamic client registration, and multiple providers (internal, GitHub, Google). This allows you to secure your MCP server with industry-standard authentication flows.

See the [OAuth Usage Guide](docs/oauth-usage.md) for detailed setup instructions, endpoint descriptions, and testing tips.

---

## 🐳 Running with Docker

1. **Run in `stdio` mode**:

   ```bash
   docker run --rm -i \
     -v $(pwd)/swagger.json:/server/swagger.json \
     auto-mcp:latest --mode=stdio --swagger-file=/server/swagger.json
   ```

2. **Run in `sse` mode** :

   ```bash
   docker-compose up -d  # uses docker-compose.yml
   ```

The bundled `docker-compose.yml` maps port 8080 and persists logs to `./logs`.

### Running Example

The repository includes a ready-to-run example using the Swagger [PetStore](http://petstore.swagger.io/v2) API with Auto MCP:

```bash
# Start the service in SSE mode (runs on port 8080 by default)
docker compose -f examples/petshop/docker-compose.yml up
```

Once running, you can access the MCP SSE endpoint at `http://localhost:8080/sse`.

You can inspect and test your newly created MCP using the MCP Inspector:

```bash
npx @modelcontextprotocol/inspector
```

---

## 🚀 Release & Versioning

Auto MCP uses [GoReleaser](https://goreleaser.com/) to automate builds and publish cross-platform releases. For comprehensive information about the release process, automated GitHub Actions workflows, and available artifacts, see [RELEASE.md](RELEASE.md).

Check the current version with:

```bash
auto-mcp --version
```

---

## 🧩 Extending Auto MCP

- **Add custom middleware** – fork the repo and plug logic inside `internal/server` (e.g., adaptors, caching).
- **Support additional auth types** – edit `internal/config/config.go` and regenerate your image.
- **Upgrade Swagger to OpenAPI v3** – contributions welcome!

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
