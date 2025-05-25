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

## 🛠️ Using Auto MCP

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

---

## 📚 Use Cases

## Use Cases

1. **Rapid Prototyping:** Wrap any REST API as an MCP server in seconds—ideal for testing ideas or building AI tools fast.

2. **Bridge Legacy Services:**  Expose legacy or internal systems as MCP endpoints without rewriting them.

3. **Access Any 3rd-Party API from Chat Applications:** Turn any third-party API into an MCP tool, making it accessible to AI assistants like Claude.

4. **Minimal Proxy Tools:** Use auto-mcp to proxy APIs that already handle validation and logic—no wrappers needed.

---

## 🖥️ Running inside Claude Desktop

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


### CLI flags

- `--mode` – override `server.mode` (`stdio` or `sse`).
- `--swagger-file` – path to the OpenAPI document (default: `swagger.json`).
- `--adjustment-file` - mcp-config-builder output filter/change route descriptions


For detailed configureation guidelines, please see [CONFIGURATION.md](docs/CONFIGURATION.md).

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
       ghcr.io/brizzai/auto-mcp:latest \ 
       --swagger-file=/server/swagger.json \ 
       --mode=stdio 
   ```

2. **Run in `sse` mode** :

   ```bash
   docker run \
     -v $(pwd)/swagger.json:/server/swagger.json \
       ghcr.io/brizzai/auto-mcp:latest \
       --swagger-file=/server/swagger.json \
       --mode=sse 
   ```

The bundled `docker-compose.yml` maps port 8080 and persists logs to `./logs`.


## 🤝 Contributing

For detailed contribution guidelines, please see [CONTRIBUTING.md](.github/CONTRIBUTING.md).

---

## 📄 License

Distributed under the Apache License 2.0 License. See [`LICENSE`](LICENSE) for details.
