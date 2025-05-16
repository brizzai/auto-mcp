package requester

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/brizzai/auto-mcp/internal/config"

	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// HTTPRequester handles both request building and execution
type HTTPRequester struct {
	client     *http.Client
	serviceCfg *config.EndpointConfig
	authMgr    AuthManager
}

type HTTPRequesterParams struct {
	fx.In

	ServiceConfig *config.EndpointConfig
	AuthManager   AuthManager
}

// NewHTTPRequester creates a new HTTPRequester with default configuration
func NewHTTPRequester(params HTTPRequesterParams) *HTTPRequester {
	return &HTTPRequester{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		serviceCfg: params.ServiceConfig,
		authMgr:    params.AuthManager,
	}
}

// SetTimeout sets the timeout for the HTTP client
func (r *HTTPRequester) SetTimeout(timeout time.Duration) {
	r.client.Timeout = timeout
}

// BuildRouteExecutor creates a function that can execute requests for a specific route
func (r *HTTPRequester) BuildRouteExecutor(config *RouteConfig) (RouteExecutor, error) {
	builder := &HTTPRequestBuilder{
		serviceCfg:  r.serviceCfg,
		authMgr:     r.authMgr,
		routeConfig: config,
	}

	// Return a function that builds and executes the request
	return func(ctx context.Context, params map[string]interface{}) (*Response, error) {
		// Build request
		req, err := builder.BuildRequest(ctx, params)
		if err != nil {
			return nil, err
		}
		logger.Info("request route", zap.Any("request", req.URL))

		// CR if u pass the context to BuildRequest, u dont need this
		// Update the context of the HTTP request
		if ctx != nil && req.HttpRequest != nil {
			req.HttpRequest = req.HttpRequest.WithContext(ctx)
		}

		// Execute request
		resp, err := r.execute(req)
		if err != nil {
			logger.Error("failed to execute request", zap.Error(err))
			return nil, err
		}

		return resp, nil
	}, nil
}

// execute performs the actual HTTP request execution
func (r *HTTPRequester) execute(req *Request) (*Response, error) {
	// Use the pre-built HTTP request
	httpReq := req.HttpRequest

	// Execute request
	resp, err := r.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		// CR: Its importnat to read the whole body
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close response body: %w", closeErr)
		}
	}()

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       bodyBytes,
		Headers:    resp.Header,
	}, nil
}
