package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/brizzai/auto-mcp/internal/requester"
	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// Package parser implements OpenAPI specification parsing functionality
// for converting OpenAPI/Swagger definitions into MCP tools.

// NewSwaggerParser creates a new SwaggerParser instance
func NewSwaggerParser(adjuster *Adjuster) *SwaggerParser {
	return &SwaggerParser{
		routeTools: make([]*RouteTool, 0),
		adjuster:   adjuster,
	}
}

// GetRouteTools returns the parsed route tools
func (p *SwaggerParser) GetRouteTools() []*RouteTool {
	return p.routeTools
}

// generateTool creates an MCP tool from a route configuration
func (p *SwaggerParser) generateTool(route *requester.RouteConfig) mcp.Tool {
	// Create a tool name from the path and method
	path := strings.TrimPrefix(route.Path, "/") // Remove leading slash
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "{", "")
	path = strings.ReplaceAll(path, "}", "")
	toolName := strings.ToLower(fmt.Sprintf("%s_%s", route.Method, path))

	// Create tool options
	opts := []mcp.ToolOption{
		mcp.WithDescription(fmt.Sprintf("%s %s \n %s", route.Method, route.Path, route.Description)),
	}

	// Add path parameters
	pathParams := extractPathParams(route.Path)
	for _, param := range pathParams {
		opts = append(opts, mcp.WithString(param,
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Path parameter: %s", param)),
		))
	}

	// Add query parameters
	if route.MethodConfig.QueryParams != nil {
		for _, param := range route.MethodConfig.QueryParams {
			opts = append(opts, mcp.WithString(param,
				mcp.Description(fmt.Sprintf("Query parameter: %s", param)),
			))
		}
	}

	// Add form fields
	if route.MethodConfig.FormFields != nil {
		for _, field := range route.MethodConfig.FormFields {
			opts = append(opts, mcp.WithString(field,
				mcp.Description(fmt.Sprintf("Form field: %s", field)),
			))
		}
	}

	// Add file upload configuration
	if route.MethodConfig.FileUpload != nil {
		opts = append(opts, mcp.WithString("file",
			mcp.Required(),
			mcp.Description("File to upload"),
		))
	}

	// Add body parameter if it's a POST/PUT/PATCH request
	if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
		p.addBodyParameter(route, &opts)
	}

	// Create and return the tool
	return mcp.NewTool(toolName, opts...)
}

// addBodyParameter adds body parameters to the tool options
func (p *SwaggerParser) addBodyParameter(route *requester.RouteConfig, opts *[]mcp.ToolOption) {
	// Find the operation for this route
	pathItem := p.doc.Paths.Find(route.Path)
	if pathItem == nil {
		logger.Debug("No path item found", zap.String("path", route.Path))
		return
	}

	var operation *openapi3.Operation
	switch route.Method {
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "PATCH":
		operation = pathItem.Patch
	}
	if operation == nil {
		logger.Debug("No operation found",
			zap.String("path", route.Path),
			zap.String("method", route.Method))
		return
	}

	// Find the request body
	schema, required := getFirstBodySchema(operation)
	if schema != nil {
		bodyOpt := schemaToMCPOptions(schema, "body", required, p.doc)
		*opts = append(*opts, bodyOpt)
	}
}

func getFirstBodySchema(operation *openapi3.Operation) (*openapi3.SchemaRef, bool) {
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		content := operation.RequestBody.Value.Content

		// If there's no content, return nil
		if len(content) == 0 {
			return nil, false
		}

		// If there's only one content type, return its schema
		if len(content) == 1 {
			for _, mediaType := range content {
				return mediaType.Schema, operation.RequestBody.Value.Required
			}
		}

		// If there are multiple content types, merge their schemas
		mergedSchema := &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type:       &openapi3.Types{"object"},
				Properties: make(openapi3.Schemas),
			},
		}

		// Merge all schemas
		for _, mediaType := range content {
			if mediaType.Schema != nil && mediaType.Schema.Value != nil {
				for propName, propSchema := range mediaType.Schema.Value.Properties {
					mergedSchema.Value.Properties[propName] = propSchema
				}
			}
		}

		return mergedSchema, operation.RequestBody.Value.Required
	}
	return nil, false
}

// extractPathParams extracts path parameters from a URL path
func extractPathParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			param := strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
			params = append(params, param)
		}
	}
	return params
}

// detectAndParseOpenAPI attempts to parse data as either OpenAPI 2.0 or 3.0
func (p *SwaggerParser) detectAndParseOpenAPI(data []byte) error {
	// First try to unmarshal as a generic JSON to catch invalid JSON early
	var jsonObj map[string]interface{}
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		return fmt.Errorf("invalid JSON in OpenAPI spec: %w", err)
	}

	// Check for version fields
	swaggerVersion, hasSwagger := jsonObj["swagger"]
	openapiVersion, hasOpenAPI := jsonObj["openapi"]

	if !hasSwagger && !hasOpenAPI {
		return fmt.Errorf("document is missing 'swagger' or 'openapi' version field")
	}

	// Try to unmarshal as OpenAPI 2.0
	if hasSwagger {
		convertedDoc, err := p.convertOpenAPI2to3(data, swaggerVersion)
		if err != nil {
			return err
		}
		p.doc = convertedDoc
		return nil
	}

	// Try to parse as OpenAPI 3.0
	if hasOpenAPI {
		if ver, ok := openapiVersion.(string); !ok || !strings.HasPrefix(ver, "3.") {
			return fmt.Errorf("unsupported OpenAPI version: %v", openapiVersion)
		}
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		logger.Error("Failed to parse OpenAPI 3.0 spec", zap.Error(err))
		return fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	if doc == nil {
		return fmt.Errorf("failed to parse OpenAPI spec: document is empty")
	}

	logger.Info("Successfully parsed OpenAPI 3.0 spec")
	p.doc = doc
	return nil
}

// convertOpenAPI2to3 converts an OpenAPI 2.0 specification to OpenAPI 3.0
func (p *SwaggerParser) convertOpenAPI2to3(data []byte, swaggerVersion interface{}) (*openapi3.T, error) {
	var swagger2Doc openapi2.T
	if err := json.Unmarshal(data, &swagger2Doc); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI 2.0 spec: %w", err)
	}

	if swagger2Doc.Swagger != "2.0" {
		return nil, fmt.Errorf("unsupported Swagger version: %s", swaggerVersion)
	}

	logger.Info("Detected OpenAPI 2.0 spec, converting to OpenAPI 3.0")
	convertedDoc, err := openapi2conv.ToV3(&swagger2Doc)
	if err != nil {
		logger.Error("Failed to convert OpenAPI 2.0 to 3.0", zap.Error(err))
		return nil, fmt.Errorf("failed to convert OpenAPI 2.0 to 3.0: %w", err)
	}

	logger.Info("Successfully converted OpenAPI 2.0 to 3.0")
	return convertedDoc, nil
}

// Init parses a Swagger/OpenAPI specification from a file
func (p *SwaggerParser) Init(openAPISpec string, adjustmentsFile string) error {
	data, err := os.ReadFile(openAPISpec)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %w", err)
	}
	if adjustmentsFile != "" {
		err = p.adjuster.Load(adjustmentsFile)
	}
	if err != nil {
		return fmt.Errorf("failed to load adjustments file: %w", err)
	}

	if err := p.detectAndParseOpenAPI(data); err != nil {
		return err
	}

	return p.processOperations()
}

// ParseReader parses a Swagger/OpenAPI specification from a reader
func (p *SwaggerParser) ParseReader(reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read swagger spec: %w", err)
	}

	if err := p.detectAndParseOpenAPI(data); err != nil {
		return err
	}

	return p.processOperations()
}

// processOperations iterates through paths and operations in the spec
func (p *SwaggerParser) processOperations() error {
	for path, pathItem := range p.doc.Paths.Map() {
		httpMethods := []struct {
			Method    string
			Operation *openapi3.Operation
		}{
			{"GET", pathItem.Get},
			{"POST", pathItem.Post},
			{"PUT", pathItem.Put},
			{"DELETE", pathItem.Delete},
			{"PATCH", pathItem.Patch},
		}

		for _, httpMethod := range httpMethods {
			if httpMethod.Operation != nil {
				routeConfig := p.createRouteConfig(path, httpMethod.Method, httpMethod.Operation)
				if p.adjuster.ExistsInMCP(routeConfig.Path, routeConfig.Method) {
					tool := p.generateTool(routeConfig)
					p.routeTools = append(p.routeTools, &RouteTool{
						RouteConfig: routeConfig,
						Tool:        tool,
					})
				}
			}
		}
	}

	return nil
}

// createRouteConfig creates a route configuration from a path and operation
func (p *SwaggerParser) createRouteConfig(path, method string, operation *openapi3.Operation) *requester.RouteConfig {
	routeConfig := &requester.RouteConfig{
		Path:   path,
		Method: method,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
	var desc string
	// Add operation description if available
	if operation.Description != "" {
		desc = operation.Description
	} else if operation.Summary != "" {
		// Fallback to summary if description is not available
		desc = operation.Summary
	}
	routeConfig.Description = p.adjuster.GetDescription(routeConfig.Path, routeConfig.Method, desc)

	// Add operation-specific headers
	if operation.Responses != nil {
		// Get the first response's content type
		for _, response := range operation.Responses.Map() {
			if response.Value != nil && response.Value.Content != nil {
				for contentType := range response.Value.Content {
					routeConfig.Headers["Accept"] = contentType
					break
				}
				break
			}
		}
	}

	// Add operation-specific configuration
	routeConfig.MethodConfig = requester.MethodConfig{
		QueryParams: make([]string, 0),
	}

	// Add query parameters
	for _, param := range operation.Parameters {
		if param.Value != nil && param.Value.In == "query" {
			routeConfig.MethodConfig.QueryParams = append(routeConfig.MethodConfig.QueryParams, param.Value.Name)
		}
	}

	return routeConfig
}
