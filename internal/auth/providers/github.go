package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/auth/constants"
	"github.com/brizzai/auto-mcp/internal/auth/models"
	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubProvider struct {
	oauth2Config *oauth2.Config
}

func NewGitHubProvider(cfg *config.OAuthConfig) *GitHubProvider {
	return &GitHubProvider{
		oauth2Config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     github.Endpoint,
			Scopes:       cfg.Scopes,
		},
	}
}

func (p *GitHubProvider) GetAuthURL(state, codeChallenge, codeChallengeMethod, redirectURI string) string {
	opts := []oauth2.AuthCodeOption{}
	if redirectURI != "" {
		opts = append(opts, oauth2.SetAuthURLParam("redirect_uri", redirectURI))
	}
	if codeChallenge != "" {
		opts = append(opts,
			oauth2.SetAuthURLParam("code_challenge", codeChallenge),
			oauth2.SetAuthURLParam("code_challenge_method", codeChallengeMethod),
		)
	}
	return p.oauth2Config.AuthCodeURL(state, opts...)
}

func (p *GitHubProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth2.Token, error) {
	cfg := *p.oauth2Config // copy
	if redirectURI != "" {
		cfg.RedirectURL = redirectURI
	}

	opts := []oauth2.AuthCodeOption{}
	if codeVerifier != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	}

	return cfg.Exchange(ctx, code, opts...)
}

func (p *GitHubProvider) ValidateToken(ctx context.Context, token *oauth2.Token) (*models.UserInfo, error) {
	client := p.oauth2Config.Client(ctx, token)
	return p.getUserInfo(client)
}

func (p *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	return p.oauth2Config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	}).Token()
}

func (p *GitHubProvider) ValidateAccessToken(ctx context.Context, token string) (*models.UserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
		TokenType:   constants.TokenType,
	}))
	return p.getUserInfo(client)
}

func (p *GitHubProvider) getUserInfo(client *http.Client) (*models.UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("Failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info request failed with status %d", resp.StatusCode)
	}

	var gh struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gh); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &models.UserInfo{
		ID:      fmt.Sprintf("%d", gh.ID),
		Email:   gh.Email,
		Name:    gh.Name,
		Picture: gh.AvatarURL,
		Metadata: map[string]interface{}{
			"login": gh.Login,
		},
	}, nil
}
