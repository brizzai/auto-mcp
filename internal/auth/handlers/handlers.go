package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/brizzai/auto-mcp/internal/auth/constants"
	"github.com/brizzai/auto-mcp/internal/auth/providers"
	"github.com/brizzai/auto-mcp/internal/logger"
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

	writeJSON(w, discovery)
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

	writeJSON(w, discovery)
}

// HandleToken handles the token endpoint
func (h *Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, "invalid_request", "Failed to parse form", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		writeError(w, "unsupported_grant_type", "Unsupported grant type", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		writeError(w, "invalid_request", "Code is required", http.StatusBadRequest)
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
		writeError(w, "invalid_grant", err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, tokenResp)
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
		writeError(w, "invalid_request", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientName == "" {
		writeError(w, "invalid_request", "Client name is required", http.StatusBadRequest)
		return
	}

	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())

	resp := map[string]interface{}{
		"client_id":                  clientID,
		"token_endpoint_auth_method": "none",
		"redirect_uris":              req.RedirectURIs,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
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

	authURL := h.authProvider.GetAuthURL(state, codeChallenge, codeChallengeMethod)
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
		writeError(w, "invalid_request", "Code is required", http.StatusBadRequest)
		return
	}

	resp := map[string]interface{}{
		"code":  code,
		"state": state,
	}

	writeJSON(w, resp)
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error("Failed to encode JSON response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": message,
	}); err != nil {
		logger.Error("Failed to encode error response", zap.Error(err))
	}
}
