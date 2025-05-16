package server

import (
	"context"
	"fmt"
	"log"
	"os"

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
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(cfg *config.Config, p parser.Parser, requester *requester.HTTPRequester) *MCPServer {
	// Create MCP server
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

	srv.setupTools()
	return srv
}

func (s *MCPServer) setupTools() {
	// Load and parse swagger
	if err := s.parser.Init(s.config.SwaggerFile, s.config.AdjustmentsFile); err != nil {
		// c
		log.Fatalf("Failed to parse swagger file: %v", err)
	}

	// Get tools from parser
	routes := s.parser.GetRouteTools()

	// Add each tool to the MCP server
	for _, route := range routes {
		// Create a new tool with the same configuration
		tool := route.Tool

		logger.Info("Adding tool", zap.String("name", tool.Name))
		executor, err := s.requester.BuildRouteExecutor(route.RouteConfig)
		if err != nil {
			logger.Error("failed to build route function", zap.Error(err))
			continue
		}

		s.mcp.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Convert MCP request parameters to map
			params := make(map[string]interface{})
			for k, v := range request.Params.Arguments {
				params[k] = v
			}
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

func (s *MCPServer) ServeSSE(ctx context.Context) error {
	logger.Info("Starting MCP server via SSE")
	sseServer := mcpserver.NewSSEServer(
		s.mcp,
		mcpserver.WithBaseURL(fmt.Sprintf("http://%s:%d", s.config.Server.Host, s.config.Server.Port)),
	)

	// Create error channel to handle server errors
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
		logger.Info("Server listening", zap.String("address", addr))
		if err := sseServer.Start(addr); err != nil {
			errChan <- err
		}
	}()

	// Wait for either context cancellation or server error
	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal, stopping SSE server...")
		if err := sseServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("error during SSE server shutdown: %w", err)
		}
		return nil
	case err := <-errChan:
		return fmt.Errorf("SSE server error: %w", err)
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
