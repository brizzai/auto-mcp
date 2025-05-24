package server

import (
	"context"
	"fmt"
	"log"
	"maps"
	"net/http"
	"os"
	"time"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/requester"

	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MCPServer struct {
	config    *config.Config
	parser    parser.Parser
	mcp       *mcpserver.MCPServer
	requester *requester.HTTPRequester
	auth      *MCPOAuth
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(cfg *config.Config, p parser.Parser, requester *requester.HTTPRequester) *MCPServer {
	// Create MCP server with session capabilities
	mcpServer := mcpserver.NewMCPServer(
		"Auto MCP",
		"1.0.0",
	)

	srv := &MCPServer{
		config:    cfg,
		parser:    p,
		mcp:       mcpServer,
		requester: requester,
	}

	// Initialize auth if OAuth is enabled
	if cfg.OAuth != nil && cfg.OAuth.Enabled {
		var provider AuthProvider
		switch cfg.OAuth.Provider {
		case "google":
			prov, err := NewGoogleProvider(cfg.OAuth)
			if err != nil {
				log.Fatalf("Failed to initialize GoogleProvider: %v", err)
			}
			provider = prov
		case "github":
			provider = NewGitHubProvider(cfg.OAuth)
		default:
			log.Fatalf("Unknown OAuth provider: %s", cfg.OAuth.Provider)
		}
		srv.auth = NewMCPOAuth(cfg.OAuth, provider)
	}

	srv.setupTools()
	return srv
}

func (s *MCPServer) setupTools() {
	// Load and parse swagger
	if err := s.parser.Init(s.config.SwaggerFile, s.config.AdjustmentsFile); err != nil {
		log.Fatalf("Failed to parse swagger file: %v", err)
	}

	// Get tools from parser
	routes := s.parser.GetRouteTools()

	// Add each tool to the MCP server
	for _, route := range routes {
		tool := route.Tool
		logger.Info("Adding tool", zap.String("name", tool.Name))
		executor, err := s.requester.BuildRouteExecutor(route.RouteConfig)
		if err != nil {
			logger.Error("failed to build route function", zap.Error(err))
			continue
		}
		s.mcp.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Get auth info from context if auth is enabled
			if s.auth != nil {
				// Check if we have auth info in context
				authInfo, ok := ctx.Value("auth").(map[string]interface{})
				if !ok {
					return mcp.NewToolResultError("Unauthorized: No active user info in context"), nil
				}
				logger.Debug("Tool called by authenticated user",
					zap.String("tool", tool.Name),
					zap.String("user_id", authInfo["user_id"].(string)),
				)
			}
			// Convert MCP request parameters to map
			params := make(map[string]interface{})
			maps.Copy(params, request.GetArguments())
			// Execute request using requester
			resp, err := executor(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to execute request: %w", err)
			}
			// Return response as tool result
			if resp.StatusCode >= 400 {
				errMessage := fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(resp.Body))
				logger.Error("HTTP Error", zap.String("error", errMessage))
				return mcp.NewToolResultError(errMessage), nil
			} else {
				return mcp.NewToolResultText(string(resp.Body)), nil
			}
		})
	}
}

// LoggingMiddleware logs information about each incoming request
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(rw, r)

		// Log request details
		duration := time.Since(start)
		logger.Info("HTTP Request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration", duration),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

// responseWriter is a custom ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and passes it to the underlying ResponseWriter
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// createHTTPHandler creates a generic HTTP handler that works for both SSE and HTTP
func (s *MCPServer) createHTTPHandler(mcpHandler http.Handler, isSSE bool) http.Handler {
	mux := http.NewServeMux()

	if s.auth != nil {
		// Always public endpoints (no auth)
		mux.Handle("/.well-known/oauth-protected-resource", LoggingMiddleware(http.HandlerFunc(s.auth.HandleProtectedResourceDiscovery)))
		mux.Handle("/.well-known/oauth-authorization-server", LoggingMiddleware(http.HandlerFunc(s.auth.HandleAuthorizationServerDiscovery)))
		mux.Handle("/oauth/register", LoggingMiddleware(http.HandlerFunc(s.auth.HandleRegister)))
		mux.Handle("/oauth/token", LoggingMiddleware(http.HandlerFunc(s.auth.HandleToken)))
		mux.Handle("/oauth/callback", LoggingMiddleware(http.HandlerFunc(s.auth.HandleAuthCallback)))
		mux.Handle("/auth/callback", LoggingMiddleware(http.HandlerFunc(s.auth.HandleAuthCallback)))
		mux.Handle("/oauth/authorize", LoggingMiddleware(http.HandlerFunc(s.auth.HandleAuthorize)))
		// Protected endpoints
		requireAuth := s.config.OAuth.Enabled
		if requireAuth {
			mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
				LoggingMiddleware(s.auth.Authenticate(mcpHandler)).ServeHTTP(w, r)
			})
			mux.Handle("/", LoggingMiddleware(s.auth.Authenticate(mcpHandler)))
		} else {
			mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
				LoggingMiddleware(s.auth.OptionalAuthenticate(mcpHandler)).ServeHTTP(w, r)
			})
			mux.Handle("/", LoggingMiddleware(s.auth.OptionalAuthenticate(mcpHandler)))
		}
	} else {
		mux.Handle("/", LoggingMiddleware(mcpHandler))
	}

	return WrapMuxWithCORS(mux)
}

func (s *MCPServer) ServeSSE(ctx context.Context) error {
	logger.Info("Starting MCP server via SSE", zap.String("mode", "sse"))

	sseServer := mcpserver.NewSSEServer(
		s.mcp,
		mcpserver.WithBaseURL(fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port)),
	)

	// Create HTTP handler with auth if enabled
	handler := s.createHTTPHandler(sseServer, true)

	// Start server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	logger.Info("Server listening", zap.String("address", addr))

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Handle graceful shutdown
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic in SSE server goroutine", zap.Any("error", r))
			}
		}()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("SSE server ListenAndServe error", zap.Error(err))
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, stopping SSE server...", zap.String("reason", ctx.Err().Error()))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			logger.Error("Error during SSE server shutdown", zap.Error(err))
		}
		logger.Info("SSE server stopped")
		return err
	case err := <-errChan:
		logger.Error("SSE server error", zap.Error(err))
		return fmt.Errorf("SSE server error: %w", err)
	}
}

func (s *MCPServer) ServeHTTP(ctx context.Context) error {
	logger.Info("Starting MCP server via HTTP", zap.String("mode", "http"))

	httpServer := mcpserver.NewStreamableHTTPServer(s.mcp)

	// Create HTTP handler with auth if enabled
	handler := s.createHTTPHandler(httpServer, false)

	// Start server
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	logger.Info("Server listening", zap.String("address", addr))

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Handle graceful shutdown
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic in HTTP server goroutine", zap.Any("error", r))
			}
		}()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server ListenAndServe error", zap.Error(err))
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, stopping HTTP server...", zap.String("reason", ctx.Err().Error()))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			logger.Error("Error during HTTP server shutdown", zap.Error(err))
		}
		logger.Info("HTTP server stopped")
		return err
	case err := <-errChan:
		logger.Error("HTTP server error", zap.Error(err))
		return fmt.Errorf("HTTP server error: %w", err)
	}
}

// ServeSTDIO starts the MCP server using standard I/O (default)
func (s *MCPServer) ServeSTDIO(ctx context.Context) error {
	logger.Info("Starting MCP server via STDIO")
	stdioServer := mcpserver.NewStdioServer(s.mcp)
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}

// Start starts the MCP server based on the configured server mode
func (s *MCPServer) Start(ctx context.Context) error {
	switch s.config.Server.Mode {
	case config.ServerModeSSE:
		return s.ServeSSE(ctx)
	case config.ServerModeHTTP:
		return s.ServeHTTP(ctx)
	case config.ServerModeSTDIO:
		fallthrough
	default:
		return s.ServeSTDIO(ctx)
	}
}

// Module provides the MCP server dependencies
var Module = fx.Module("mcp_server",
	fx.Provide(
		NewMCPServer,
	),
)
