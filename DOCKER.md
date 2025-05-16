# Docker Setup for Auto MCP

This guide explains how to build, configure, and run the Auto MCP server using Docker for both development and production environments.

---

## 1. Building the Docker Image

Build the Docker image from the project root:

```bash
docker build -t auto-mcp -f Dockerfile .
```

---

## 2. Configuration Overview

Auto MCP supports flexible configuration, prioritized as follows:

1. **Command-line flags**
2. **Environment variables** (prefixed with `AUTO_MCP_`)
3. **`.env` file** (for development)
4. **`config.yaml` file**

### Key Configuration Options

- `mode`: Server mode (`stdio` or `sse`)
- `swagger-file`: Path to the Swagger/OpenAPI file (default: `swagger.json`)
- `adjustments-file`: Path to the adjustments file (default: none)

#### Example Environment Variables

```bash
# Server mode (stdio|sse)
AUTO_MCP_SERVER_MODE=stdio
# Swagger file path
AUTO_MCP_SWAGGER_FILE=swagger.json
# Adjustments file path
AUTO_MCP_ADJUSTMENTS_FILE=adjustments.yaml
```

#### Development Environment

For local development, create a `.env.dev` file:

```bash
# Development configuration
AUTO_MCP_SWAGGER_FILE=example_swagger.json
AUTO_MCP_ADJUSTMENTS_FILE=example_adjustments.yaml
```

---

## 3. Adjustments File

The adjustments file (YAML) allows you to:

- **Filter** which API routes are enabled in the MCP server
- **Customize** descriptions for specific routes and methods

**Example (`adjustments.yaml`):**

```yaml
# Custom descriptions for routes
update_descriptions:
  /api/v1/users/{id}:
    GET: "Get user by ID with customized description"
    PUT: "Update user with customized description"

# Only these routes will be enabled
selected_routes:
  /api/v1/users:
    - GET
    - POST
  /api/v1/users/{id}:
    - GET
    - PUT
    - DELETE
```

---

## 4. Running the Container

### a. STDIO Mode (Default)

```bash
docker run -i auto-mcp
```

### b. SSE Mode (HTTP Server)

```bash
docker run -p 8080:8080 auto-mcp --mode=sse
```

### c. Custom Configuration Examples

```bash
# Using environment variables
docker run -e AUTO_MCP_SWAGGER_FILE=custom.json auto-mcp

# Using a custom adjustments file
docker run -e AUTO_MCP_ADJUSTMENTS_FILE=adjustments.yaml auto-mcp

# Using a custom config file
docker run -v $(pwd)/config.yaml:/server/config.yaml auto-mcp

# Using custom swagger and adjustments files
docker run -v $(pwd)/swagger.json:/server/swagger.json \
           -v $(pwd)/adjustments.yaml:/server/adjustments.yaml auto-mcp
```

---

## 5. Dockerfile Details

The Dockerfile uses a **multi-stage build** for security and efficiency:

### Build Stage
- **Base:** `golang:1.24.2-alpine`
- Installs `git` for Go dependencies
- Builds the application with:
  - CGO **disabled**
  - Build and module caching enabled

### Runtime Stage
- **Base:** `distroless/base-debian12` (minimal attack surface)
- Copies only the built binary and default config
- Runs as a non-root user
- Defaults to STDIO mode

---

## 6. Security Considerations

- Uses a distroless runtime image for minimal attack surface
- No shell or package manager in the runtime image
- Runs as a non-root user
- Only necessary files are included in the final image

---

## 7. Development Tips

### Local Development with Custom Swagger

```bash
# Create .env.dev
echo "AUTO_MCP_SWAGGER_FILE=example_swagger.json" > .env.dev

# Run with development config
go run cmd/auto-mcp/main.go
```

### Testing Different Configurations

```bash
# Test SSE mode
go run cmd/auto-mcp/main.go --mode=sse

# Test with custom swagger
go run cmd/auto-mcp/main.go --swagger-file=custom.json

# Test with adjustments file
go run cmd/auto-mcp/main.go --adjustments-file=adjustments.yaml
```

---

## 8. Troubleshooting

- **File permissions:** Ensure mounted files (e.g., `swagger.json`, `adjustments.yaml`) are readable by the container.
- **Port conflicts:** If using SSE mode, ensure port 8080 is available.
- **Configuration precedence:** Command-line flags override environment variables and config files.

---

For more details, see the project [README.md](./README.md) or contact the maintainers.
