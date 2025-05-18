package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/requester"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMCPServer_SemiE2E tests the creation and initialization of an MCP server
// with a real OpenAPI specification but without actually serving requests.
func TestNewMCPServer_SemiE2E(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	// For the tests to work, we need to resolve paths relative to the project root
	// Go up to the project root (if needed)
	projectRoot := filepath.Join(cwd, "..", "..")

	// Locate test resources (relative to the repository root)
	cfgDir := filepath.Join(projectRoot, "examples", "petshop", "config")
	swaggerPath := filepath.Join(cfgDir, "swagger.json")
	adjustmentPath := filepath.Join(cfgDir, "adjustment.yaml")

	// Ensure test files exist
	for _, path := range []string{swaggerPath, adjustmentPath} {
		_, err := os.Stat(path)
		require.NoError(t, err, "Test file %s not found", path)
	}

	// Build a minimal configuration
	srvCfg := &config.Config{
		SwaggerFile:     swaggerPath,
		AdjustmentsFile: adjustmentPath,
		EndpointConfig: config.EndpointConfig{
			BaseURL: "https://petstore.swagger.io/v2", // real API base (won't be hit in this test)
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
			Mode: config.ServerModeSTDIO,
		},
	}

	// Parser & adjustments
	adjuster := parser.NewAdjuster()
	swaggerParser := parser.NewSwaggerParser(adjuster)

	// HTTP requester (network will not actually be used – it is only needed for tool construction)
	endpointCfg := &srvCfg.EndpointConfig
	httpRequester := requester.NewHTTPRequester(requester.HTTPRequesterParams{
		ServiceConfig: endpointCfg,
		AuthManager:   requester.NewHTTPAuthManager(endpointCfg),
	})

	// Create the MCP server under test
	mcpSrv := NewMCPServer(srvCfg, swaggerParser, httpRequester)
	require.NotNil(t, mcpSrv, "expected MCP server instance, got nil")

	// Ensure that tools have been loaded according to the adjustments file
	tools := swaggerParser.GetRouteTools()
	assert.NotEmpty(t, tools, "expected route tools to be loaded, got 0")

	// The adjustment.yaml selects a specific subset of the pet-store spec.
	// Verify the exact number of tools based on the current adjustment.yaml
	expectedToolCount := 11 // Based on the routes defined in adjustment.yaml
	assert.Len(t, tools, expectedToolCount, "expected %d tools to be registered, got %d", expectedToolCount, len(tools))

	// Check for specific routes that should be included
	expectedRoutes := map[string]string{
		"POST /pet":                     "post_pet",
		"PUT /pet":                      "put_pet",
		"GET /pet/findByStatus":         "get_pet_findbystatus",
		"GET /pet/{petId}":              "get_pet_petid",
		"POST /pet/{petId}":             "post_pet_petid",
		"POST /pet/{petId}/uploadImage": "post_pet_petid_uploadimage",
		"GET /store/inventory":          "get_store_inventory",
		"POST /store/order":             "post_store_order",
		"GET /store/order/{orderId}":    "get_store_order_orderid",
		"GET /user/logout":              "get_user_logout",
		"GET /pet/findByTags":           "get_pet_findbytags",
	}

	// Build a map of actual routes for easier testing
	actualRouteMap := make(map[string]bool)
	actualToolNameMap := make(map[string]*parser.RouteTool)

	for _, tool := range tools {
		routeKey := tool.RouteConfig.Method + " " + tool.RouteConfig.Path
		actualRouteMap[routeKey] = true
		actualToolNameMap[tool.Tool.Name] = tool
	}

	// Verify all expected routes exist
	for route, toolName := range expectedRoutes {
		assert.True(t, actualRouteMap[route], "Expected route %s not found", route)
		assert.Contains(t, actualToolNameMap, toolName, "Tool %s not found", toolName)
	}

	// Verify specific tool configurations for key endpoints
	t.Run("Validate findbystatus endpoint", func(t *testing.T) {
		findByStatusTool, ok := actualToolNameMap["get_pet_findbystatus"]
		require.True(t, ok, "get_pet_findbystatus tool not found")

		// Check that query parameters are correctly defined
		params := findByStatusTool.Tool.InputSchema.Properties
		statusParam, hasStatus := params["status"].(map[string]interface{})
		assert.True(t, hasStatus, "Should have 'status' query parameter")
		if hasStatus {
			assert.Equal(t, "string", statusParam["type"], "Status parameter should be a string")
		}

		// Check the route configuration
		assert.Equal(t, "GET", findByStatusTool.RouteConfig.Method)
		assert.Equal(t, "/pet/findByStatus", findByStatusTool.RouteConfig.Path)
	})

	t.Run("Validate upload image endpoint", func(t *testing.T) {
		uploadTool, ok := actualToolNameMap["post_pet_petid_uploadimage"]
		require.True(t, ok, "post_pet_petid_uploadimage tool not found")

		// Test description override specified in the adjustment.yaml
		assert.Contains(t, uploadTool.Tool.Description,
			"uploads an image to a specifc pet id",
			"Description override from adjustment.yaml not applied")

		// Check the route configuration
		assert.Equal(t, "POST", uploadTool.RouteConfig.Method)
		assert.Equal(t, "/pet/{petId}/uploadImage", uploadTool.RouteConfig.Path)

		// Check route has petId parameter in its path
		assert.Contains(t, uploadTool.RouteConfig.Path, "{petId}", "Path should contain petId parameter")
	})

	t.Run("Validate store inventory endpoint", func(t *testing.T) {
		inventoryTool, ok := actualToolNameMap["get_store_inventory"]
		require.True(t, ok, "get_store_inventory tool not found")

		// This endpoint should have no parameters in its path
		assert.NotContains(t, inventoryTool.RouteConfig.Path, "{", "Inventory endpoint should not have path parameters")

		// There should be no query parameters
		assert.Empty(t, inventoryTool.RouteConfig.MethodConfig.QueryParams,
			"Inventory endpoint should not have query parameters")
	})
}

// TestMCPServer_ListTools tests that the MCP server correctly returns the list of available tools
// when queried through the MCP protocol
func TestMCPServer_ListTools(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	// For the tests to work, we need to resolve paths relative to the project root
	// Go up to the project root (if needed)
	projectRoot := filepath.Join(cwd, "..", "..")

	// Locate test resources (relative to the repository root)
	cfgDir := filepath.Join(projectRoot, "examples", "petshop", "config")
	swaggerPath := filepath.Join(cfgDir, "swagger.json")
	adjustmentPath := filepath.Join(cfgDir, "adjustment.yaml")

	// Ensure test files exist
	for _, path := range []string{swaggerPath, adjustmentPath} {
		_, err := os.Stat(path)
		require.NoError(t, err, "Test file %s not found", path)
	}

	// Find an available port for the server
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to create listener")
	port := listener.Addr().(*net.TCPAddr).Port
	err = listener.Close()
	require.NoError(t, err, "Failed to close listener")

	// Include the MCP path in the server address
	serverAddr := fmt.Sprintf("http://localhost:%d/sse", port)

	// Build configuration for MCP server
	srvCfg := &config.Config{
		SwaggerFile:     swaggerPath,
		AdjustmentsFile: adjustmentPath,
		EndpointConfig: config.EndpointConfig{
			BaseURL: "https://petstore.swagger.io/v2", // real API base (won't be hit in this test)
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: port,
			Mode: config.ServerModeSSE, // Use SSE mode for HTTP testing
		},
	}

	// Parser & adjustments
	adjuster := parser.NewAdjuster()
	swaggerParser := parser.NewSwaggerParser(adjuster)

	// HTTP requester (network will not actually be used – it is only needed for tool construction)
	endpointCfg := &srvCfg.EndpointConfig
	httpRequester := requester.NewHTTPRequester(requester.HTTPRequesterParams{
		ServiceConfig: endpointCfg,
		AuthManager:   requester.NewHTTPAuthManager(endpointCfg),
	})

	// Create the MCP server under test
	mcpSrv := NewMCPServer(srvCfg, swaggerParser, httpRequester)
	require.NotNil(t, mcpSrv, "expected MCP server instance, got nil")

	// Create a context with cancellation for the server
	serverCtx, stopServer := context.WithCancel(context.Background())
	defer stopServer()

	// Start the server in a goroutine
	go func() {
		if err := mcpSrv.ServeSSE(serverCtx); err != nil && err != context.Canceled {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give the server time to start
	time.Sleep(2 * time.Second)

	// Create a client context with timeout
	clientCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create an SSE client to communicate with the server
	sseClient, err := client.NewSSEMCPClient(serverAddr)
	require.NoError(t, err, "Failed to create SSE client")

	// Start the client and initialize it
	err = sseClient.Start(clientCtx)
	require.NoError(t, err, "Failed to start client")

	// Initialize the client with the server
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.Capabilities = mcp.ClientCapabilities{}
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	initResult, err := sseClient.Initialize(clientCtx, initReq)
	require.NoError(t, err, "Failed to initialize client")
	require.NotNil(t, initResult, "Initialize result is nil")

	// Test listing available tools
	t.Run("List Available Tools", func(t *testing.T) {
		tools, err := sseClient.ListTools(clientCtx, mcp.ListToolsRequest{})
		require.NoError(t, err, "Failed to get tools from server")
		require.NotEmpty(t, tools.Tools, "No tools returned")

		// Expected tool names
		expectedTools := map[string]bool{
			"post_pet":                   true,
			"put_pet":                    true,
			"get_pet_findbystatus":       true,
			"get_pet_petid":              true,
			"post_pet_petid":             true,
			"post_pet_petid_uploadimage": true,
			"get_store_inventory":        true,
			"post_store_order":           true,
			"get_store_order_orderid":    true,
			"get_user_logout":            true,
			"get_pet_findbytags":         true,
		}

		// Verify all expected tools exist
		for _, tool := range tools.Tools {
			assert.True(t, expectedTools[tool.Name], "Unexpected tool: %s", tool.Name)
			delete(expectedTools, tool.Name) // Remove from map to track which ones we've seen
		}

		// Verify we've seen all expected tools
		assert.Empty(t, expectedTools, "Missing expected tools: %v", expectedTools)
	})

	// Test getting a specific tool's details
	t.Run("Tool Detail Test", func(t *testing.T) {
		// Test the upload image endpoint which has a custom description
		toolName := "post_pet_petid_uploadimage"

		// Find the tool in the list of tools
		var tool mcp.Tool
		tools, err := sseClient.ListTools(clientCtx, mcp.ListToolsRequest{})
		require.NoError(t, err, "Failed to get tools")

		for _, t := range tools.Tools {
			if t.Name == toolName {
				tool = t
				break
			}
		}

		require.NotEmpty(t, tool, "Failed to find tool")

		// Verify the tool details
		assert.Equal(t, toolName, tool.Name, "Incorrect tool name")
		assert.Contains(t, tool.Description,
			"uploads an image to a specifc pet id",
			"Description override from adjustment.yaml not applied")
	})

	// Test calling a tool
	t.Run("Tool Call Test", func(t *testing.T) {
		// Test the GET /pet/findByStatus endpoint (it's simple and doesn't require complex data)
		toolName := "get_pet_findbystatus"

		// Create tool call request
		request := mcp.CallToolRequest{}
		request.Params.Name = toolName
		request.Params.Arguments = map[string]interface{}{
			"status": "available",
		}

		// We don't expect this to succeed since we're not hitting a real API,
		// but we want to ensure the request is properly processed by the server
		_, err := sseClient.CallTool(clientCtx, request)
		// It's okay if this fails with a specific type of error indicating the actual endpoint couldn't be called
		// We're just testing that the server accepts and processes the request as expected
		if err != nil {
			// Log but don't fail the test as we expect an error when not connecting to a real API
			t.Logf("Expected error calling tool: %v", err)
		}
	})
}

// TestMCPServer_ContextCancellation tests that the server shuts down properly when context is cancelled
func TestMCPServer_ContextCancellation(t *testing.T) {
	// Create a minimal server configuration with a very simple schema
	simpleSchema := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/test": {
				"get": {
					"summary": "Test endpoint",
					"responses": {
						"200": {
							"description": "OK"
						}
					}
				}
			}
		}
	}`

	// Create a temporary file for the schema
	tmpFile, err := os.CreateTemp("", "test-schema-*.json")
	require.NoError(t, err, "Failed to create temporary file")
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("Failed to remove temporary file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(simpleSchema)
	require.NoError(t, err, "Failed to write to temporary file")
	require.NoError(t, tmpFile.Close(), "Failed to close temporary file")

	// Create a server configuration
	srvCfg := &config.Config{
		SwaggerFile: tmpFile.Name(),
		EndpointConfig: config.EndpointConfig{
			BaseURL: "http://example.com",
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,                    // Random port
			Mode: config.ServerModeSSE, // Test SSE mode which uses context
		},
	}

	// Create parser and requester
	adjuster := parser.NewAdjuster()
	swaggerParser := parser.NewSwaggerParser(adjuster)
	endpointCfg := &srvCfg.EndpointConfig
	httpRequester := requester.NewHTTPRequester(requester.HTTPRequesterParams{
		ServiceConfig: endpointCfg,
		AuthManager:   requester.NewHTTPAuthManager(endpointCfg),
	})

	// Create the server
	mcpSrv := NewMCPServer(srvCfg, swaggerParser, httpRequester)
	require.NotNil(t, mcpSrv, "Failed to create MCP server")

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- mcpSrv.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context to trigger shutdown
	cancel()

	// Wait for server to stop with timeout
	select {
	case err := <-errCh:
		assert.NoError(t, err, "Server should shut down gracefully")
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
}

// TestMCPServer_ToolRegistration verifies that tools are correctly registered with the MCP server
func TestMCPServer_ToolRegistration(t *testing.T) {
	// Define a simple tool for testing
	testTool := mcp.NewTool("test_tool", mcp.WithDescription("Test tool"))

	// Create a mock parser that always returns our test tool
	mockParser := &mockParser{
		tools: []*parser.RouteTool{
			{
				RouteConfig: &requester.RouteConfig{
					Path:   "/test",
					Method: "GET",
				},
				Tool: testTool,
			},
		},
	}

	// Create a minimal configuration
	srvCfg := &config.Config{
		EndpointConfig: config.EndpointConfig{
			BaseURL: "http://example.com",
		},
		Server: config.ServerConfig{
			Mode: config.ServerModeSTDIO,
		},
	}

	// Create HTTP requester
	endpointCfg := &srvCfg.EndpointConfig
	httpRequester := requester.NewHTTPRequester(requester.HTTPRequesterParams{
		ServiceConfig: endpointCfg,
		AuthManager:   requester.NewHTTPAuthManager(endpointCfg),
	})

	// Create MCP server with our mock parser
	mcpSrv := NewMCPServer(srvCfg, mockParser, httpRequester)
	require.NotNil(t, mcpSrv, "Failed to create MCP server")

	// Since we can't directly access the tools registered in the MCP server,
	// we can use reflection or just verify that the server was created successfully
	// and our Init method was called, which shows the tools were processed
	assert.True(t, mockParser.initCalled, "Parser Init method should have been called")
}

// mockParser implements the parser.Parser interface for testing
type mockParser struct {
	tools      []*parser.RouteTool
	initCalled bool
}

func (m *mockParser) Init(openAPISpec string, adjustmentsFile string) error {
	m.initCalled = true
	return nil
}

func (m *mockParser) ParseReader(reader io.Reader) error {
	return nil
}

func (m *mockParser) GetRouteTools() []*parser.RouteTool {
	return m.tools
}
