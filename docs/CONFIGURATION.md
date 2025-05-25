# ⚙️ Configuration

## Configuration Options

Auto MCP accepts configuration via **CLI flags**, **environment variables** (prefix `AUTO_MCP_`), or an optional `config.yaml`. In containerized deployments environment variables are the most convenient.

| Purpose                               | Env variable                          | Example                          |
| ------------------------------------- | ------------------------------------- | -------------------------------- |
| Select transport                      | `AUTO_MCP_SERVER_MODE`                | `stdio` or `http` or `sse`       |
| Bind port (SSE)                       | `AUTO_MCP_SERVER_PORT`                | `8080`                           |
| Upstream base URL                     | `AUTO_MCP_ENDPOINT_BASE_URL`          | `https://petstore.swagger.io/v2` |
| Authentication type                   | `AUTO_MCP_ENDPOINT_AUTH_TYPE`         | `bearer`                         |
| Bearer/OAuth token                    | `AUTO_MCP_ENDPOINT_AUTH_CONFIG_TOKEN` | `123456`                         |
| Extra static header                   | `AUTO_MCP_ENDPOINT_HEADERS_X_CUSTOM`  | `hello`                          |
| Log level                             | `AUTO_MCP_LOGGING_LEVEL`              | `debug`                          |
| Path to swagger file                  | `AUTO_MCP_SWAGGER_FILE`               | `/server/swagger.json`           |
| Path to adjustment file (mcp-builder) | `AUTO_MCP_ADJUSTMENTS_FILE`           | `/server/swagger.json`           |
| Enable OAuth                          | `AUTO_MCP_OAUTH_ENABLED`              | `true`                           |
| OAuth provider                        | `AUTO_MCP_OAUTH_PROVIDER`             | `github` / `google`              |
| OAuth client ID                       | `AUTO_MCP_OAUTH_CLIENT_ID`            | `your-client-id`                 |
| OAuth client secret                   | `AUTO_MCP_OAUTH_CLIENT_SECRET`        | `your-client-secret`             |
| OAuth scopes                          | `AUTO_MCP_OAUTH_SCOPES`               | `openid email profile`           |
| OAuth base URL                        | `AUTO_MCP_OAUTH_BASE_URL`             | `http://localhost:8080/oauth`    |
| OAuth host (optional)                 | `AUTO_MCP_OAUTH_HOST`                 | `localhost`                      |
| OAuth port (optional)                 | `AUTO_MCP_OAUTH_PORT`                 | `8080`                           |
| Server name (display)                 | `AUTO_MCP_SERVER_NAME`                | `Auto MCP`                       |
| Server version (display)              | `AUTO_MCP_SERVER_VERSION`             | `1.0.0`                          |

Underscores replace dots in the YAML path; nested keys keep the hierarchy (e.g., `endpoint.auth_config.token` → `AUTO_MCP_ENDPOINT_AUTH_CONFIG_TOKEN`).

CLI shortcuts:

- `--mode` – overrides the transport.
- `--swagger-file` – absolute or relative path to the OpenAPI document.
- `--adjustment-file` - mcp-config-builder output filter/change route descriptions.

---

## Environment Variables

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

## 📝 Mounting and Overwriting `config.yaml` (Full Configuration)

Instead of passing individual CLI flags or environment variables, you can mount a complete `config.yaml` into the container to control all aspects of Auto MCP—including the Swagger and adjustment files, server mode, logging, authentication, and OAuth settings.

**Recommended for production or reproducible deployments.**

### Example Directory Structure

Suppose you have the following files in a directory (e.g., `examples/petshop/config`):

- `config.yaml` (references the other files by their paths)
- `swagger.json` (your OpenAPI/Swagger definition)
- `adjustment.yaml` (optional, for endpoint adjustments)

Your `config.yaml` should include references like:

```yaml
swagger_file: "/config/swagger.json"
adjustments_file: "/config/adjustment.yaml"
```

### Mounting the Entire Config Directory in Docker

```bash
docker run --rm -i \
  -v $(pwd)/examples/petshop/config:/config \
  ghcr.io/brizzai/auto-mcp:latest
```

- The container will use `/config/config.yaml` for all configuration.
- The `swagger_file` and `adjustments_file` paths in your config should match the mount locations.
- No need to pass `--swagger-file` or `--adjustment-file` flags if set in `config.yaml`.

This approach keeps all related files together and is ideal for local development or sharing example setups.

---

## Example config.yaml

```yaml
server:
  mode: http # Server mode: http, stdio, or sse
  port: 8080 # Port to bind (for http/sse)
  host: "0.0.0.0" # Host to bind
  timeout: 30s # Request timeout (e.g., 30s, 1m)
  name: "Auto MCP" # Server display name
  version: "1.0.0" # Server version string

logging:
  level: "info" # Log level: debug, info, warn, error
  format: "json" # Log format: json or console
  color: true # Enable color in logs (console only)
  disable_stacktrace: false # Disable stacktraces in logs
  output_path: "logs/auto-mcp.log" # Log file path
  append_to_file: true # Append to log file if true
  disable_console: false # Disable console logging if true

endpoint:
  base_url: "https://petstore.swagger.io/v2" # Upstream API base URL
  auth_type: "none" # Auth type: none, basic, bearer, api_key, oauth2
  # auth_config:           # (optional) Auth config map, e.g. {token: "..."}
  # headers:               # (optional) Extra headers map, e.g. {X-Api-Key: "..."}

oauth:
  enabled: false # Enable OAuth2 authentication
  provider: github # OAuth provider (github, google, etc.)
  client_id: "" # OAuth client ID
  client_secret: "" # OAuth client secret
  scopes: "" # OAuth scopes (space-separated)
  base_url: "" # OAuth base URL (optional, usually auto-set)
  allow_origins: [] # List of allowed CORS origins

swagger_file: "/config/swagger.json" # Path to OpenAPI/Swagger file
adjustments_file: "/config/adjustment.yaml" # Path to adjustments file
```
