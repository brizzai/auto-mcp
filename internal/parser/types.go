package parser

import (
	"io"

	"github.com/brizzai/auto-mcp/internal/requester"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
)

// RouteTool combines a route configuration with its corresponding MCP tool
type RouteTool struct {
	RouteConfig *requester.RouteConfig
	Tool        mcp.Tool
}

// Parser handles parsing of Swagger/OpenAPI specifications
type Parser interface {
	// Init parses a Swagger/OpenAPI specification from a file
	Init(openAPISpec string, adjustmentsFile string) error
	// ParseReader parses a Swagger/OpenAPI specification from a reader
	ParseReader(reader io.Reader) error
	// GetRouteTools returns the parsed route tools
	GetRouteTools() []*RouteTool
}

// SwaggerParser parses Swagger specifications and generates route configurations
type SwaggerParser struct {
	doc        *openapi3.T
	routeTools []*RouteTool
	adjuster   *Adjuster
}
