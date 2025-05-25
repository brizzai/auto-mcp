package auth

import (
	"fmt"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/auth/constants"
	"github.com/brizzai/auto-mcp/internal/auth/handlers"
	"github.com/brizzai/auto-mcp/internal/auth/middleware"
	"github.com/brizzai/auto-mcp/internal/auth/providers"
	"github.com/brizzai/auto-mcp/internal/config"
)

// Service represents the OAuth service
type Service struct {
	config       *config.OAuthConfig
	authProvider providers.OAuthProvider
	handler      *handlers.Handler
}

// NewService creates a new OAuth service
func NewService(cfg *config.OAuthConfig, provider providers.OAuthProvider) (*Service, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		port := cfg.Port
		if port == 0 {
			port = constants.DefaultPort
		}
		baseURL = fmt.Sprintf("http://%s:%d", cfg.Host, port)
	}

	handler := handlers.NewHandler(baseURL, provider)

	return &Service{
		config:       cfg,
		authProvider: provider,
		handler:      handler,
	}, nil
}

// RegisterRoutes registers all OAuth-related routes
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	// Discovery endpoints
	mux.HandleFunc("/.well-known/oauth-protected-resource", s.handler.HandleProtectedResourceDiscovery)
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handler.HandleAuthorizationServerDiscovery)

	// OAuth endpoints
	mux.HandleFunc("/oauth/authorize", s.handler.HandleAuthorize)
	mux.HandleFunc("/oauth/token", s.handler.HandleToken)
	mux.HandleFunc("/oauth/register", s.handler.HandleRegister)
	mux.HandleFunc("/oauth/callback", s.handler.HandleAuthCallback)
}

// WrapWithMiddleware wraps the mux with authentication middleware
func (s *Service) WrapWithMiddleware(handler http.Handler) http.Handler {
	return middleware.CORSWithOrigins(s.config.AllowOrigins)(handler)
}

// Authenticate returns the authentication middleware
func (s *Service) Authenticate() func(http.Handler) http.Handler {
	return middleware.Authenticate(s.authProvider)
}

// OptionalAuthenticate returns the optional authentication middleware
func (s *Service) OptionalAuthenticate() func(http.Handler) http.Handler {
	return middleware.OptionalAuthenticate(s.authProvider)
}

// GetProvider returns the configured auth provider
func (s *Service) GetProvider() providers.OAuthProvider {
	return s.authProvider
}
