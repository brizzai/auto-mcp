package constants

const (
	// DefaultPort is the default port for the auth server
	DefaultPort = 3000

	// TokenType for Bearer authentication
	TokenType = "Bearer"

	// AuthHeaderName is the name of the Authorization header
	AuthHeaderName = "Authorization"

	// AuthHeaderPrefix is the prefix for the Authorization header value
	AuthHeaderPrefix = "Bearer "

	// TokenQueryParam is the query parameter name for token
	TokenQueryParam = "token"
)

// OAuth scopes
var DefaultScopes = []string{"openid", "profile", "email"}

// Response types and modes
var (
	SupportedResponseTypes = []string{"code"}
	SupportedResponseModes = []string{"query"}
	SupportedGrantTypes    = []string{"authorization_code"}
	SupportedAuthMethods   = []string{"none"}
)

// PKCE methods
var SupportedPKCEMethods = []string{"S256"}
