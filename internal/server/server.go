// Package server provides the core MCP (Model Control Protocol) server implementation.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/brizzai/auto-mcp/internal/auth"
	"github.com/brizzai/auto-mcp/internal/auth/providers"
	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/requester"
	"github.com/brizzai/auto-mcp/internal/server/handler"
	"github.com/brizzai/auto-mcp/internal/server/tool"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// shutdownTimeout is the maximum time to wait for server shutdown
	shutdownTimeout = 5 * time.Second
)

// ErrInvalidOAuthProvider indicates an unsupported OAuth provider was specified
var ErrInvalidOAuthProvider = fmt.Errorf("unsupported OAuth provider")

// Server represents the MCP server instance that handles tool management,
// authentication, and request processing. It supports multiple operation modes
// including SSE, HTTP, and STDIO.
type Server struct {
	config    *config.Config
	parser    parser.Parser
	mcp       *mcpserver.MCPServer
	requester *requester.HTTPRequester
	auth      *auth.Service
	handler   *handler.Handler
	tool      *tool.Handler
}

// NewServer creates a new MCP server instance with the provided configuration.
// It initializes the server with the given parser and requester, and sets up
// authentication if enabled in the configuration.
func NewServer(cfg *config.Config, p parser.Parser, requester *requester.HTTPRequester) *Server {
	if cfg == nil {
		logger.Fatal("Config cannot be nil")
	}
	if p == nil {
		logger.Fatal("Parser cannot be nil")
	}
	if requester == nil {
		logger.Fatal("Requester cannot be nil")
	}

	mcpServer := mcpserver.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
	)

	srv := &Server{
		config:    cfg,
		parser:    p,
		mcp:       mcpServer,
		requester: requester,
	}

	if cfg.OAuth != nil && cfg.OAuth.Enabled {
		if err := srv.setupAuth(); err != nil {
			logger.Fatal("Failed to setup authentication", zap.Error(err))
		}
	}

	// Initialize handlers
	srv.handler = handler.NewHandler(srv.auth)
	srv.tool = tool.NewHandler(srv.auth != nil)

	if err := srv.setupTools(); err != nil {
		logger.Fatal("Failed to setup tools", zap.Error(err))
	}

	return srv
}

func (s *Server) setupAuth() error {
	var provider providers.Provider
	var err error

	switch s.config.OAuth.Provider {
	case "google":
		provider, err = providers.NewGoogleProvider(s.config.OAuth)
	case "github":
		provider = providers.NewGitHubProvider(s.config.OAuth)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidOAuthProvider, s.config.OAuth.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize provider %s: %w", s.config.OAuth.Provider, err)
	}

	authService, err := auth.NewService(s.config.OAuth, provider)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}

	s.auth = authService
	return nil
}

func (s *Server) setupTools() error {
	if err := s.parser.Init(s.config.SwaggerFile, s.config.AdjustmentsFile); err != nil {
		return fmt.Errorf("failed to initialize parser: %w", err)
	}

	routes := s.parser.GetRouteTools()
	for _, route := range routes {
		tool := route.Tool
		executor, err := s.requester.BuildRouteExecutor(route.RouteConfig)
		if err != nil {
			logger.Error("Failed to build route executor", zap.String("tool", tool.Name), zap.Error(err))
			continue
		}

		s.mcp.AddTool(tool, s.tool.CreateHandler(&tool, executor))
	}
	return nil
}

func (s *Server) ServeSSE(ctx context.Context) error {
	logger.Info("Starting SSE server")

	sseServer := mcpserver.NewSSEServer(
		s.mcp,
		mcpserver.WithBaseURL(fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port)),
	)

	return s.serveHTTP(ctx, sseServer, "SSE")
}

func (s *Server) ServeHTTP(ctx context.Context) error {
	logger.Info("Starting HTTP server")
	httpServer := mcpserver.NewStreamableHTTPServer(s.mcp)
	return s.serveHTTP(ctx, httpServer, "HTTP")
}

func (s *Server) serveHTTP(ctx context.Context, handler http.Handler, mode string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: s.handler.CreateHTTPHandler(handler),
	}

	// Channel for server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server",
			zap.String("mode", mode),
			zap.String("address", addr),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		logger.Info("Shutting down server",
			zap.String("mode", mode),
			zap.Duration("timeout", shutdownTimeout),
		)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}
		return nil

	case err := <-errChan:
		return err
	}
}

func (s *Server) ServeSTDIO(ctx context.Context) error {
	logger.Info("Starting STDIO server")
	stdioServer := mcpserver.NewStdioServer(s.mcp)
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}

// Start starts the server in the configured mode (SSE, HTTP, or STDIO).
// It returns an error if the server fails to start or encounters an error
// during operation.
func (s *Server) Start(ctx context.Context) error {
	logger.Info("Starting server",
		zap.String("mode", string(s.config.Server.Mode)),
		zap.String("version", s.config.Server.Version),
	)

	switch s.config.Server.Mode {
	case config.ServerModeSSE:
		return s.ServeSSE(ctx)
	case config.ServerModeHTTP:
		return s.ServeHTTP(ctx)
	case config.ServerModeSTDIO:
		return s.ServeSTDIO(ctx)
	default:
		return fmt.Errorf("unsupported server mode: %s", s.config.Server.Mode)
	}
}

// Module provides the MCP server dependencies
var Module = fx.Module("mcp_server",
	fx.Provide(
		NewServer,
	),
)
