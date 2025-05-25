package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/brizzai/auto-mcp/internal/auth/constants"
	"github.com/brizzai/auto-mcp/internal/auth/providers"
	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/brizzai/auto-mcp/internal/utils"
	"go.uber.org/zap"
)

// Handler handles OAuth-related HTTP requests
type Handler struct {
	baseURL      string
	authProvider providers.Provider
}

// NewHandler creates a new Handler instance
func NewHandler(baseURL string, provider providers.Provider) *Handler {
	return &Handler{
		baseURL:      baseURL,
		authProvider: provider,
	}
}

// HandleProtectedResourceDiscovery handles /.well-known/oauth-protected-resource
func (h *Handler) HandleProtectedResourceDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	discovery := map[string]interface{}{
		"resource":              h.baseURL,
		"authorization_servers": []string{h.baseURL},
		"scopes_supported":      constants.DefaultScopes,
		"token_types_supported": []string{constants.TokenType},
		"resource_metadata_uri": fmt.Sprintf("%s/.well-known/oauth-protected-resource", h.baseURL),
	}

	utils.WriteJSON(w, discovery)
}

// HandleAuthorizationServerDiscovery handles /.well-known/oauth-authorization-server
func (h *Handler) HandleAuthorizationServerDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	discovery := map[string]interface{}{
		"issuer":                                h.baseURL,
		"authorization_endpoint":                fmt.Sprintf("%s/oauth/authorize", h.baseURL),
		"token_endpoint":                        fmt.Sprintf("%s/oauth/token", h.baseURL),
		"registration_endpoint":                 fmt.Sprintf("%s/oauth/register", h.baseURL),
		"token_endpoint_auth_methods_supported": constants.SupportedAuthMethods,
		"scopes_supported":                      constants.DefaultScopes,
		"response_types_supported":              constants.SupportedResponseTypes,
		"response_modes_supported":              constants.SupportedResponseModes,
		"grant_types_supported":                 constants.SupportedGrantTypes,
		"code_challenge_methods_supported":      constants.SupportedPKCEMethods,
	}

	utils.WriteJSON(w, discovery)
}

// HandleToken handles the token endpoint
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.WriteError(w, "invalid_request", "Failed to parse form", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		utils.WriteError(w, "unsupported_grant_type", "Unsupported grant type", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		utils.WriteError(w, "invalid_request", "Code is required", http.StatusBadRequest)
		return
	}

	tokenResp, err := h.authProvider.ExchangeCode(
		r.Context(),
		code,
		r.FormValue("code_verifier"),
		r.FormValue("redirect_uri"),
	)
	if err != nil {
		logger.Error("Failed to exchange code", zap.Error(err))
		utils.WriteError(w, "invalid_grant", err.Error(), http.StatusBadRequest)
		return
	}
	utils.WriteJSON(w, tokenResp)
}

// HandleRegister handles client registration
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientName == "" {
		utils.WriteError(w, "invalid_request", "Client name is required", http.StatusBadRequest)
		return
	}

	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())

	resp := map[string]interface{}{
		"client_id":                  clientID,
		"token_endpoint_auth_method": "none",
		"redirect_uris":              req.RedirectURIs,
	}

	w.WriteHeader(http.StatusCreated)
	utils.WriteJSON(w, resp)
}

// HandleAuthorize handles the authorization endpoint
func (h *Handler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
	redirectURI := r.URL.Query().Get("redirect_uri")

	authURL := h.authProvider.GetAuthURL(state, codeChallenge, codeChallengeMethod, redirectURI)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleAuthCallback handles the OAuth callback
func (h *Handler) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		utils.WriteError(w, "invalid_request", "Code is required", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"code":  code,
		"state": state,
	}

	utils.WriteJSON(w, resp)
}
