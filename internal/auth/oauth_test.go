package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/brizzai/auto-mcp/internal/auth/models"
	"github.com/brizzai/auto-mcp/internal/config"
	"golang.org/x/oauth2"
)

// mockProvider implements providers.Provider for testing
// Only methods needed for Service tests are stubbed

type mockProvider struct{}

func (m *mockProvider) GetAuthURL(state, codeChallenge, codeChallengeMethod, redirectURI string) string {
	return "mock-url"
}
func (m *mockProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error) {
	return &oauth2.Token{}, nil
}
func (m *mockProvider) ValidateToken(ctx context.Context, token *oauth2.Token) (*models.UserInfo, error) {
	return &models.UserInfo{}, nil
}
func (m *mockProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return &oauth2.Token{}, nil
}
func (m *mockProvider) ValidateAccessToken(ctx context.Context, token string) (*models.UserInfo, error) {
	return &models.UserInfo{}, nil
}

func TestNewService(t *testing.T) {
	cfg := &config.OAuthConfig{
		BaseURL: "http://localhost:8080",
	}
	provider := &mockProvider{}
	service, err := NewService(cfg, provider)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if service.config != cfg {
		t.Errorf("expected config to be set")
	}
	if !reflect.DeepEqual(service.authProvider, provider) {
		t.Errorf("expected provider to be set")
	}
	if service.handler == nil {
		t.Errorf("expected handler to be set")
	}
}

func TestRegisterRoutes(t *testing.T) {
	cfg := &config.OAuthConfig{BaseURL: "http://localhost:8080"}
	provider := &mockProvider{}
	service, _ := NewService(cfg, provider)
	mux := http.NewServeMux()
	service.RegisterRoutes(mux)

	routes := []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-authorization-server",
		"/oauth/authorize",
		"/oauth/token",
		"/oauth/register",
		"/oauth/callback",
	}
	for _, route := range routes {
		r, _ := http.NewRequest("GET", route, nil)
		h, pattern := mux.Handler(r)
		if pattern == "" || h == nil {
			t.Errorf("route %s not registered", route)
		}
	}
}

func TestWrapWithCors(t *testing.T) {
	cfg := &config.OAuthConfig{BaseURL: "http://localhost:8080"}
	provider := &mockProvider{}
	service, _ := NewService(cfg, provider)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})
	wrapped := service.WrapWithCors(h)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	wrapped.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Errorf("expected status 204, got %d", rec.Code)
	}
}

func TestGetProvider(t *testing.T) {
	cfg := &config.OAuthConfig{BaseURL: "http://localhost:8080"}
	provider := &mockProvider{}
	service, _ := NewService(cfg, provider)
	if !reflect.DeepEqual(service.GetProvider(), provider) {
		t.Errorf("GetProvider did not return the expected provider")
	}
}
