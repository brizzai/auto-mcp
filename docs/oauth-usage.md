# MCP OAuth Authentication for Auto-MCP

Auto-MCP implements the Model Context Protocol (MCP) OAuth specification for securing access to MCP servers. This follows the OAuth 2.1 specification with MCP-specific requirements including discovery endpoints, PKCE support, and proper session management.

## Overview

The MCP OAuth implementation provides:

1. **OAuth Discovery Endpoints** - Required by MCP spec for client auto-configuration
2. **PKCE Support** - Proof Key for Code Exchange for enhanced security
3. **Dynamic Client Registration** - Allows MCP clients to register dynamically
4. **Session Management** - Integrated with MCP's session management system
5. **Multiple Provider Support** - GitHub, Google providers

## MCP OAuth Flow

The MCP OAuth flow follows these steps:

1. **Discovery** - Client fetches `/.well-known/oauth-protected-resource` to discover OAuth endpoints
2. **Authorization** - Client redirects to `/oauth/authorize` with PKCE challenge
3. **Authentication** - User authenticates with the OAuth provider
4. **Token Exchange** - Client exchanges authorization code for access token at `/oauth/token`
5. **API Access** - Client uses Bearer token to access MCP endpoints

## Configuration

Add OAuth configuration to your `config.yaml`:

```yaml
oauth:
  enabled: true
  provider: github
  client_id: "<client_id>"
  client_secret: "<cleint_secret>"
  scopes: "openid email profile"
  base_url: "http://localhost:8080"
  # List of allowed origins for CORS (optional)
  allow_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"
```

### Provider Options

#### Internal Provider (Testing/Development)

```yaml
oauth:
  enabled: true
  provider: internal
  base_url: "http://localhost:8080"
```

This provider automatically approves all authorization requests for testing.

#### External OAuth Providers

For production, use external providers:

```yaml
# GitHub
oauth:
  enabled: true
  provider: github
  client_id: "<client_id>"
  client_secret: "<cleint_secret>"
  scopes: "openid email profile"
  base_url: "http://localhost:8080"

# Google
oauth:
  enabled: true
  provider: google
  client_id: "your-client-id.apps.googleusercontent.com"
  client_secret: "your-client-secret"
  scopes: "openid email profile"
  base_url: "http://localhost:8080"
```

### Inject as an environmant

Set the following environment variables to configure OAuth via environment:

```bash
export AUTO_MCP_OAUTH_ENABLED=true
export AUTO_MCP_OAUTH_PROVIDER=github
export AUTO_MCP_OAUTH_CLIENT_ID=your-client-id
export AUTO_MCP_OAUTH_CLIENT_SECRET=your-client-secret
export AUTO_MCP_OAUTH_SCOPES="openid email profile"
export AUTO_MCP_OAUTH_BASE_URL=http://localhost:8080
# Optional overrides:
export AUTO_MCP_OAUTH_HOST=localhost
export AUTO_MCP_OAUTH_PORT=8080
# Optional: comma-separated list of allowed origins for CORS
export AUTO_MCP_OAUTH_ALLOW_ORIGINS="http://localhost:3000,http://localhost:8080"
```

## OAuth Endpoints

### Discovery Endpoints

- `GET /.well-known/oauth-protected-resource` - Protected resource metadata
- `GET /.well-known/oauth-authorization-server` - Authorization server metadata

### OAuth Endpoints

- `GET /oauth/authorize` - Authorization endpoint
- `POST /oauth/token` - Token endpoint
- `POST /oauth/register` - Dynamic client registration

## Authentication Methods

### Bearer Token in Header

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/
```

### Token in Query Parameter (for SSE)

```bash
curl "http://localhost:8080/sse?token=<token>"
```

## Testing with MCP Clients

### Using mcp-remote

Test your OAuth implementation with `mcp-remote`:

> for this to work you need to add the mcp-remote callback url, (http://localhost:17623/oauth/callback)

```bash
npx mcp-remote http://localhost:8080 --transport sse

```

The client will automatically:

1. Discover OAuth endpoints
2. Initiate the OAuth flow
3. Store and use the access token

### Testing OAuth Flow Manually

You can use the [MCP Inspector](https://www.npmjs.com/package/@modelcontextprotocol/inspector) tool to interactively test any step of the OAuth flow. Run the following command:

```bash
npx @modelcontextprotocol/inspector http://localhost:8080
```

This tool allows you to:

- Discover OAuth endpoints
- Initiate and debug the authorization flow
- Exchange tokens
- Inspect responses and headers

It's useful for manual testing, debugging, and learning how the MCP OAuth process works step by step.

### Debug Logging

Enable debug logging to see OAuth flow details:

```yaml
logging:
  level: debug
```

## Differences from Traditional OAuth

MCP OAuth has specific requirements:

1. **No Cookies** - MCP uses Bearer tokens exclusively
2. **Discovery Required** - Clients expect well-known discovery endpoints
3. **PKCE Support** - Enhanced security with code challenges
4. **Session Management** - Integrated with MCP session system
5. **WWW-Authenticate Headers** - Specific format for MCP clients
