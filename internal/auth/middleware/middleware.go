package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/brizzai/auto-mcp/internal/auth/constants"
	"github.com/brizzai/auto-mcp/internal/auth/providers"
	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
)

// AuthContext is the key type for the context
type authContextKey string

const (
	// AuthContextKey is used to store auth info in the request context
	AuthContextKey authContextKey = "auth"
)

// AuthInfo represents the authentication information stored in context
type AuthInfo struct {
	UserID string
	Email  string
	Name   string
	Token  string
}

// Authenticate middleware validates JWT or access token with the IDP
func Authenticate(provider providers.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logger.Info("Authenticate middleware", zap.Any("next", next))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Info("Authenticate middleware request",
				zap.String("method", r.Method),
				zap.String("url", r.URL.String()),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
			)
			token := extractToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
				return
			}

			userInfo, err := provider.ValidateAccessToken(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid_token", err.Error())
				return
			}
			logger.Info("Authenticate middleware userInfo", zap.Any("userInfo", userInfo))

			ctx := context.WithValue(r.Context(), AuthContextKey, &AuthInfo{
				UserID: userInfo.ID,
				Email:  userInfo.Email,
				Name:   userInfo.Name,
				Token:  token,
			})

			logger.Info("Authenticate middleware ctx", zap.Any("ctx", ctx))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthenticate allows both authenticated and unauthenticated access
func OptionalAuthenticate(provider providers.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			userInfo, err := provider.ValidateAccessToken(r.Context(), token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), AuthContextKey, &AuthInfo{
				UserID: userInfo.ID,
				Email:  userInfo.Email,
				Name:   userInfo.Name,
				Token:  token,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS middleware for MCP
func CORS(next http.Handler) http.Handler {
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

// extractToken extracts the Bearer token from the request
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get(constants.AuthHeaderName)
	if strings.HasPrefix(authHeader, constants.AuthHeaderPrefix) {
		return strings.TrimPrefix(authHeader, constants.AuthHeaderPrefix)
	}
	return r.URL.Query().Get(constants.TokenQueryParam)
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="MCP Server", error="%s", error_description="%s"`, code, message))
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": message,
	})
}
