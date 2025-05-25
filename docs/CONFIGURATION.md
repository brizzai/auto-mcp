# ⚙️ Configuration

## Configuration Options

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
