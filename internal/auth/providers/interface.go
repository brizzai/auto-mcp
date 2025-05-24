package providers

import (
	"context"

	"github.com/brizzai/auto-mcp/internal/auth/models"
	"golang.org/x/oauth2"
)

// Provider defines the interface that all OAuth providers must implement
type Provider interface {
	// GetAuthURL returns the authorization URL for the provider
	GetAuthURL(state, codeChallenge, codeChallengeMethod string) string

	// ExchangeCode exchanges an authorization code for tokens
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error)

	// ValidateToken validates an OAuth token and returns user info
	ValidateToken(ctx context.Context, token *oauth2.Token) (*models.UserInfo, error)

	// RefreshToken refreshes an OAuth token
	RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error)

	// ValidateAccessToken validates a raw access token and returns user info
	ValidateAccessToken(ctx context.Context, token string) (*models.UserInfo, error)
}
