package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
)

// MCPOAuth handles OAuth authentication for MCP servers
// Now stateless: only validates JWT or access token with the IDP
// No session or code storage

type MCPOAuth struct {
	config       *config.OAuthConfig
	authProvider AuthProvider
	baseURL      string
}

func NewMCPOAuth(config *config.OAuthConfig, provider AuthProvider) *MCPOAuth {
	baseURL := config.BaseURL
	if baseURL == "" {
		port := config.Port
		if port == 0 {
			port = 3000
		}
		baseURL = fmt.Sprintf("http://%s:%d", config.Host, port)
	}
	return &MCPOAuth{
		config:       config,
		authProvider: provider,
		baseURL:      baseURL,
	}
}

// CORS middleware for MCP
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, MCP-Session-ID")
		w.Header().Set("Access-Control-Expose-Headers", "MCP-Session-ID, WWW-Authenticate")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WrapMuxWithCORS wraps the entire mux with CORS middleware
func WrapMuxWithCORS(mux *http.ServeMux) http.Handler {
	return CORSMiddleware(mux)
}

func (auth *MCPOAuth) HandleProtectedResourceDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	discovery := map[string]interface{}{
		"resource":              auth.baseURL,
		"authorization_servers": []string{auth.baseURL},
		"scopes_supported":      []string{"openid", "profile", "email"},
		"token_types_supported": []string{"Bearer"},
		"resource_metadata_uri": fmt.Sprintf("%s/.well-known/oauth-protected-resource", auth.baseURL),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(discovery); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func (auth *MCPOAuth) HandleAuthorizationServerDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	discovery := map[string]interface{}{
		"issuer":                                auth.baseURL,
		"authorization_endpoint":                fmt.Sprintf("%s/oauth/authorize", auth.baseURL),
		"token_endpoint":                        fmt.Sprintf("%s/oauth/token", auth.baseURL),
		"registration_endpoint":                 fmt.Sprintf("%s/oauth/register", auth.baseURL),
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code"},
		"code_challenge_methods_supported":      []string{"S256"},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(discovery); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

// Authenticate middleware: validates JWT or access token with the IDP
func (auth *MCPOAuth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		logger.Info("Authenticating", zap.String("token", token))

		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "unauthorized",
				"error_description": "Authentication required",
			})
			return
		}
		logger.Info("Validating access token", zap.String("token", token))
		userInfo, err := auth.authProvider.ValidateAccessToken(r.Context(), token)
		logger.Info("Validating access token", zap.Any("userInfo", userInfo), zap.Error(err))
		if err != nil {
			wwwAuthHeader := fmt.Sprintf(`Bearer realm="MCP Server", error="invalid_token", error_description="%s"`, err.Error())
			w.Header().Set("WWW-Authenticate", wwwAuthHeader)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "invalid_token",
				"error_description": err.Error(),
			})
			return
		}
		ctx := context.WithValue(r.Context(), "auth", map[string]interface{}{
			"user_id": userInfo.ID,
			"email":   userInfo.Email,
			"name":    userInfo.Name,
			"token":   token,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthenticate: allows both authenticated and unauthenticated access
func (auth *MCPOAuth) OptionalAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		userInfo, err := auth.authProvider.ValidateAccessToken(r.Context(), token)
		if err != nil {
			logger.Debug("Invalid token provided", zap.Error(err))
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), "auth", map[string]interface{}{
			"user_id": userInfo.ID,
			"email":   userInfo.Email,
			"name":    userInfo.Name,
			"token":   token,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractToken extracts the Bearer token from the request
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	return ""
}

// In-memory code storage for demo (replace with persistent store in production)
var codeStore = make(map[string]struct {
	CodeChallenge       string
	CodeChallengeMethod string
	UserID              string
	ExpiresAt           int64
})

func (auth *MCPOAuth) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	if r.FormValue("grant_type") != "authorization_code" {
		http.Error(w, "unsupported_grant_type", http.StatusBadRequest)
		return
	}
	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	// NOTE: the IdP will do PKCE verification for us; forward everything we got.
	tokenResp, err := auth.authProvider.ExchangeCode(
		r.Context(),
		code,
		r.FormValue("code_verifier"),
		auth.config.RedirectURL,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tokenResp)
}

// sha256SumBase64URL returns the base64url-encoded SHA256 hash
func sha256SumBase64URL(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// In-memory client storage for demo (replace with persistent store in production)
var clientStore = make(map[string]struct {
	ClientName   string
	RedirectURIs []string
	CreatedAt    int64
})

// HandleRegister implements dynamic client registration for internal provider
func (auth *MCPOAuth) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}
	fmt.Println("Register request", req)
	logger.Info("Register request", zap.Any("request", req))
	if req.ClientName == "" {
		http.Error(w, "client_name required", http.StatusBadRequest)
		return
	}
	clientID := generateClientID()
	clientStore[clientID] = struct {
		ClientName   string
		RedirectURIs []string
		CreatedAt    int64
	}{
		ClientName:   req.ClientName,
		RedirectURIs: req.RedirectURIs,
		CreatedAt:    time.Now().Unix(),
	}
	resp := map[string]interface{}{
		"client_id":                  "640007509031-urk4mag682pjrnobkurkrg4veu148mnp.apps.googleusercontent.com",
		"token_endpoint_auth_method": "none",
		"redirect_uris":              []string{auth.config.RedirectURL},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// generateClientID returns a random client_id (demo: timestamp-based)
func generateClientID() string {
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

// HandleAuthCallback handles the OAuth2 callback, returns code and state to the client (no token exchange here)
func (auth *MCPOAuth) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	// For browser-based clients, return code and state as JSON (or render a page that posts them to the backend)
	resp := map[string]interface{}{
		"code":  code,
		"state": state,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleAuthorize creates the IDP authorization URL and redirects to it
func (auth *MCPOAuth) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if auth.authProvider == nil {
		http.Error(w, "No auth provider configured", http.StatusNotImplemented)
		return
	}

	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")

	url := auth.authProvider.GetAuthURL(state, codeChallenge, codeChallengeMethod)
	logger.Info("Redirecting to", zap.String("url", url))
	http.Redirect(w, r, url, http.StatusFound)
}
