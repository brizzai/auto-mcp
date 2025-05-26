// Package handler provides HTTP request handling for the MCP server.
package handler

import (
	"net/http"

	"github.com/brizzai/auto-mcp/internal/auth"
	"github.com/brizzai/auto-mcp/internal/logger"
)

// Handler manages HTTP request handling and middleware configuration.
type Handler struct {
	auth *auth.Service
}

// NewHandler creates a new HTTP handler.
func NewHandler(auth *auth.Service) *Handler {
	return &Handler{
		auth: auth,
	}
}

// CreateHTTPHandler creates an HTTP handler with the appropriate middleware stack.
// If authentication is enabled, it adds authentication middleware to protected routes.
func (h *Handler) CreateHTTPHandler(mcpHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	// Set up authentication routes and middleware if enabled
	if h.auth != nil {
		h.auth.RegisterRoutes(mux)
		logger.Info("Registered authentication routes")
		mux.Handle("/", h.auth.Authenticate()(mcpHandler))
		logger.Info("Enabled authentication for all routes")
		return h.auth.WrapWithCors(mux)
	} else {
		mux.Handle("/", mcpHandler)
		logger.Info("Running without authentication")
		return mux
	}
}
