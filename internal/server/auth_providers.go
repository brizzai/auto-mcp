// Package server: Auth providers used by MCP.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// -----------------------------------------------------------------------------
// Common types / interfaces
// -----------------------------------------------------------------------------

type UserInfo struct {
	ID       string
	Email    string
	Name     string
	Picture  string
	Metadata map[string]interface{}
}

// AuthProvider is a pluggable abstraction over concrete IdPs.
//
//   • GetAuthURL is used by the /oauth/authorize handler to send the user to
//     the external consent screen.
//   • ExchangeCode delegates the "code → tokens" step.  The MCP server passes
//     through whatever redirect URI the client used so that public/loop‑back
//     flows continue to work.
//
// The rest of the methods are utility helpers for token validation / refresh.
//
// NOTE:  The extra redirectURI argument was added to support *dynamic*
//        redirect URIs required by desktop‑native LLM clients.
// ---------------------------------------------------------------------------

type AuthProvider interface {
	GetAuthURL(state, codeChallenge, codeChallengeMethod string) string
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error)
	ValidateToken(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error)
	ValidateAccessToken(ctx context.Context, token string) (*UserInfo, error)
}

// -----------------------------------------------------------------------------
// Google (OIDC) implementation
// -----------------------------------------------------------------------------

type GoogleProvider struct {
	OAuth2Config *oauth2.Config
	Verifier     *oidc.IDTokenVerifier
}

func NewGoogleProvider(cfg *config.OAuthConfig) (*GoogleProvider, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, err
	}
	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     google.Endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	return &GoogleProvider{
		OAuth2Config: oauth2Cfg,
		Verifier:     provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
	}, nil
}

func (p *GoogleProvider) GetAuthURL(state, codeChallenge, codeChallengeMethod string) string {
	opts := []oauth2.AuthCodeOption{}
	if codeChallenge != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_challenge", codeChallenge))
		opts = append(opts, oauth2.SetAuthURLParam("code_challenge_method", codeChallengeMethod))
	}
	return p.OAuth2Config.AuthCodeURL(state, opts...)
}

func (p *GoogleProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error) {
	cfg := *p.OAuth2Config // copy
	if redirectURI != "" {
		cfg.RedirectURL = redirectURI
	}
	opts := []oauth2.AuthCodeOption{}
	if codeVerifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}
	return cfg.Exchange(ctx, code, opts...)
}

func (p *GoogleProvider) ValidateToken(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}
	idToken, err := p.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	var claims struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	return &UserInfo{ID: claims.Sub, Email: claims.Email, Name: claims.Name, Picture: claims.Picture}, nil
}

func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return p.OAuth2Config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
}

func (p *GoogleProvider) ValidateAccessToken(ctx context.Context, token string) (*UserInfo, error) {
	// Create a client with the access token
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
		TokenType:   "Bearer",
	}))

	// Call the userinfo endpoint
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		logger.Error("Error calling userinfo endpoint", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed with status %d", resp.StatusCode)
	}

	var userInfo struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:      userInfo.Sub,
		Email:   userInfo.Email,
		Name:    userInfo.Name,
		Picture: userInfo.Picture,
	}, nil
}

type GitHubProvider struct{ OAuth2Config *oauth2.Config }

func NewGitHubProvider(cfg *config.OAuthConfig) *GitHubProvider {
	return &GitHubProvider{
		OAuth2Config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"user:email"},
		},
	}
}

func (p *GitHubProvider) GetAuthURL(state, codeChallenge, codeChallengeMethod string) string {
	opts := []oauth2.AuthCodeOption{}
	if codeChallenge != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_challenge", codeChallenge))
		opts = append(opts, oauth2.SetAuthURLParam("code_challenge_method", codeChallengeMethod))
	}
	return p.OAuth2Config.AuthCodeURL(state, opts...)
}

func (p *GitHubProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error) {
	cfg := *p.OAuth2Config // copy
	logger.Info("Exchanging code", zap.String("code", code), zap.String("code_verifier", codeVerifier), zap.String("redirect_uri", redirectURI))
	if redirectURI != "" {
		cfg.RedirectURL = redirectURI
	}
	opts := []oauth2.AuthCodeOption{}
	if codeVerifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}
	return cfg.Exchange(ctx, code, opts...)
}

func (p *GitHubProvider) ValidateToken(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.OAuth2Config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var gh struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", gh.ID), Email: gh.Email, Name: gh.Name, Picture: gh.AvatarURL, Metadata: map[string]interface{}{"login": gh.Login}}, nil
}

func (p *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return p.OAuth2Config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
}

func (p *GitHubProvider) ValidateAccessToken(ctx context.Context, token string) (*UserInfo, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed with status %d", resp.StatusCode)
	}
	var gh struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, err
	}
	return &UserInfo{ID: fmt.Sprintf("%d", gh.ID), Email: gh.Email, Name: gh.Name, Picture: gh.AvatarURL, Metadata: map[string]interface{}{"login": gh.Login}}, nil
}
