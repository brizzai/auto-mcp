# MCP OAuth Authentication for Auto-MCP

Auto-MCP implements the Model Context Protocol (MCP) OAuth specification for securing access to MCP servers. This follows the OAuth 2.1 specification with MCP-specific requirements including discovery endpoints, PKCE support, and proper session management.

## Overview

The MCP OAuth implementation provides:

1. **OAuth Discovery Endpoints** - Required by MCP spec for client auto-configuration
2. **PKCE Support** - Proof Key for Code Exchange for enhanced security
3. **Dynamic Client Registration** - Allows MCP clients to register dynamically
4. **Session Management** - Integrated with MCP's session management system
5. **Multiple Provider Support** - Internal, OAuth2, GitHub, Google providers

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
  provider: internal # For testing/development
  base_url: "http://localhost:3000"
  secure_cookies: false
```

### Provider Options

#### Internal Provider (Testing/Development)

```yaml
oauth:
  enabled: true
  provider: internal
  base_url: "http://localhost:3000"
```

This provider automatically approves all authorization requests for testing.

#### External OAuth Providers

For production, use external providers:

```yaml
# GitHub
oauth:
  enabled: true
  provider: github
  client_id: "your-github-client-id"
  client_secret: "your-github-client-secret"
  redirect_url: "http://localhost:3000/auth/callback"
  scopes: "read:user user:email"

# Google
oauth:
  enabled: true
  provider: google
  client_id: "your-client-id.apps.googleusercontent.com"
  client_secret: "your-client-secret"
  redirect_url: "http://localhost:3000/auth/callback"
  scopes: "openid email profile"

# Generic OAuth2
oauth:
  enabled: true
  provider: oauth2
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  redirect_url: "http://localhost:3000/auth/callback"
  auth_url: "https://provider.com/oauth/authorize"
  token_url: "https://provider.com/oauth/token"
  user_info_url: "https://provider.com/api/user"
  scopes: "custom scopes"
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
curl -H "Authorization: Bearer <token>" http://localhost:3000/
```

### Token in Query Parameter (for SSE)

```bash
curl "http://localhost:3000/sse?token=<token>"
```

## PKCE (Proof Key for Code Exchange)

The implementation supports PKCE for enhanced security:

```javascript
// Generate code verifier and challenge
const codeVerifier = generateRandomString(128);
const codeChallenge = base64url(sha256(codeVerifier));

// Authorization request
const authUrl = `/oauth/authorize?client_id=${clientId}&redirect_uri=${redirectUri}&state=${state}&code_challenge=${codeChallenge}&code_challenge_method=S256`;

// Token exchange
const tokenResponse = await fetch("/oauth/token", {
  method: "POST",
  headers: { "Content-Type": "application/x-www-form-urlencoded" },
  body: new URLSearchParams({
    grant_type: "authorization_code",
    code: authorizationCode,
    code_verifier: codeVerifier,
    client_id: clientId,
    redirect_uri: redirectUri,
  }),
});
```

## Accessing Authentication in Tools

Tools can access authentication information from the request context:

```go
func (s *MCPServer) setupTools() {
    s.mcp.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Authentication info is available in context
        if authInfo, ok := ctx.Value("auth").(map[string]interface{}); ok {
            userID := authInfo["user_id"].(string)
            token := authInfo["token"].(string)

            // Use auth info for API calls
            resp, err := makeAuthenticatedAPICall(token, request.GetArguments())
            // ...
        }

        // Tool implementation
    })
}
```

## Testing with MCP Clients

### Using mcp-remote

Test your OAuth implementation with `mcp-remote`:

```bash
npx mcp-remote http://localhost:3000 --transport sse
```

The client will automatically:

1. Discover OAuth endpoints
2. Initiate the OAuth flow
3. Store and use the access token

### Testing OAuth Flow Manually

1. **Get Discovery Info**:

```bash
curl http://localhost:3000/.well-known/oauth-protected-resource
```

2. **Authorize** (in browser):

```
http://localhost:3000/oauth/authorize?client_id=test&redirect_uri=http://localhost:8080/callback&state=random-state&code_challenge=challenge&code_challenge_method=plain
```

3. **Exchange Code for Token**:

```bash
curl -X POST http://localhost:3000/oauth/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=authorization_code&code=<code>&code_verifier=challenge&client_id=test&redirect_uri=http://localhost:8080/callback"
```

## Security Considerations

1. **HTTPS in Production** - Always use HTTPS in production. Set `secure_cookies: true`.
2. **PKCE Required** - The implementation enforces PKCE when code_challenge is provided.
3. **Token Expiry** - Tokens expire after 1 hour by default.
4. **Session Cleanup** - Expired sessions are automatically cleaned up.

## Troubleshooting

### Common Issues

1. **"WWW-Authenticate header missing"** - The client expects proper OAuth discovery endpoints
2. **"Invalid code_verifier"** - PKCE validation failed, ensure proper challenge/verifier generation
3. **"Token expired"** - Implement token refresh or re-authenticate
4. **Discovery endpoint 404** - Ensure OAuth is enabled in config

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
