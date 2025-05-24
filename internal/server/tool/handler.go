// Package tool provides tool handling functionality for the MCP server.
package tool

import (
	"context"
	"fmt"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/auth/middleware"
	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/brizzai/auto-mcp/internal/requester"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// Handler manages tool execution and authentication.
type Handler struct {
	auth *bool // nil if auth is disabled, non-nil if enabled
}

// NewHandler creates a new tool handler.
func NewHandler(authEnabled bool) *Handler {
	if authEnabled {
		enabled := true
		return &Handler{auth: &enabled}
	}
	return &Handler{auth: nil}
}

// CreateHandler creates a handler function for a specific tool.
// It handles authentication validation and request execution.
func (h *Handler) CreateHandler(tool *mcp.Tool, executor requester.RouteExecutor) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Validate authentication if enabled
		if h.auth != nil {
			authInfo, ok := ctx.Value(middleware.AuthContextKey).(*middleware.AuthInfo)
			if !ok {
				logger.Error("Failed to get auth info from context",
					zap.String("tool", tool.Name),
					zap.Any("context_keys", ctx.Value(middleware.AuthContextKey)),
				)
				return mcp.NewToolResultError("Unauthorized: No active user info in context"), nil
			}
			logger.Debug("Authenticated tool call",
				zap.String("tool", tool.Name),
				zap.String("user", authInfo.UserID),
			)
		}

		// Execute the tool request
		params := request.GetArguments()
		resp, err := executor(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to execute request for tool %s: %w", tool.Name, err)
		}

		// Handle error responses
		if resp.StatusCode >= http.StatusBadRequest {
			return mcp.NewToolResultError(fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(resp.Body))), nil
		}

		return mcp.NewToolResultText(string(resp.Body)), nil
	}
}
