package requester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/brizzai/auto-mcp/internal/config"

	"go.uber.org/fx"
)

// HTTPRequestBuilderParams holds the parameters for creating an HTTPRequestBuilder
type HTTPRequestBuilderParams struct {
	fx.In
	EndpointConfig *config.EndpointConfig
	AuthManager    AuthManager
	RouteConfig    *RouteConfig
}

// HTTPRequestBuilder implements the RequestBuilder interface
type HTTPRequestBuilder struct {
	serviceCfg  *config.EndpointConfig
	authMgr     AuthManager
	routeConfig *RouteConfig
}

// NewHTTPRequestBuilder creates a new HTTPRequestBuilder
func NewHTTPRequestBuilder(params HTTPRequestBuilderParams) *HTTPRequestBuilder {
	return &HTTPRequestBuilder{
		serviceCfg:  params.EndpointConfig,
		authMgr:     params.AuthManager,
		routeConfig: params.RouteConfig,
	}
}

// BuildRequest builds a request from a route name and parameters
func (b *HTTPRequestBuilder) BuildRequest(ctx context.Context, params map[string]interface{}) (*Request, error) {
	if b.routeConfig == nil {
		return nil, fmt.Errorf("route config is nil")
	}
	// Build URL
	url := b.buildURL(b.routeConfig.Path, params)

	// Add query parameters for GET requests
	if b.routeConfig.Method == "GET" {
		url = b.addQueryParams(url, params)
	}

	// Create request body
	body, contentType, err := b.createRequestBody(b.routeConfig, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create request body: %w", err)
	}

	// Merge headers
	headers := make(map[string]string)
	for k, v := range b.serviceCfg.Headers {
		headers[k] = v
	}
	for k, v := range b.routeConfig.Headers {
		headers[k] = v
	}

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, b.routeConfig.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	// Apply authentication
	if err := b.authMgr.ApplyAuth(httpReq); err != nil {
		return nil, fmt.Errorf("failed to apply authentication: %w", err)
	}

	return &Request{
		URL:         url,
		Method:      b.routeConfig.Method,
		Body:        body,
		Headers:     headers,
		ContentType: contentType,
		HttpRequest: httpReq,
	}, nil
}

func (b *HTTPRequestBuilder) buildURL(path string, params map[string]interface{}) string {
	url := b.serviceCfg.BaseURL + path

	// Replace path parameters
	for key, value := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		url = strings.ReplaceAll(url, placeholder, fmt.Sprintf("%v", value))
	}

	return url
}

func (b *HTTPRequestBuilder) addQueryParams(baseURL string, params map[string]interface{}) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := u.Query()
	for key, value := range params {
		// Skip body and file parameters
		if key == "body" || key == "file" {
			continue
		}
		q.Set(key, fmt.Sprintf("%v", value))
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func (b *HTTPRequestBuilder) createRequestBody(routeConfig *RouteConfig, params map[string]interface{}) (io.Reader, string, error) {
	switch routeConfig.Method {
	case "GET":
		return nil, "", nil

	case "POST", "PUT", "PATCH":
		// Handle multipart/form-data
		if routeConfig.MethodConfig.FileUpload != nil {
			return b.createMultipartBody(routeConfig, params)
		}

		// Handle regular JSON body
		if body, ok := params["body"]; ok {
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, "", fmt.Errorf("failed to marshal request body: %w", err)
			}
			return bytes.NewBuffer(jsonData), "application/json", nil
		}
		return nil, "", nil

	default:
		// For other methods, just send the params as JSON if not nil
		if params != nil {
			jsonData, err := json.Marshal(params)
			if err != nil {
				return nil, "", fmt.Errorf("failed to marshal request body: %w", err)
			}
			return bytes.NewBuffer(jsonData), "application/json", nil
		}
		return nil, "", nil
	}
}

func (b *HTTPRequestBuilder) createMultipartBody(routeConfig *RouteConfig, params map[string]interface{}) (io.Reader, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file if present
	if file, ok := params[routeConfig.MethodConfig.FileUpload.FieldName].(multipart.File); ok {
		part, err := writer.CreateFormFile(routeConfig.MethodConfig.FileUpload.FieldName, "file")
		if err != nil {
			return nil, "", fmt.Errorf("failed to create form file: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, "", fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// Add other form fields
	for _, field := range routeConfig.MethodConfig.FormFields {
		if value, exists := params[field]; exists {
			if err := writer.WriteField(field, fmt.Sprintf("%v", value)); err != nil {
				return nil, "", fmt.Errorf("failed to write form field: %w", err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}
