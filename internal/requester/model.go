package requester

import (
	"context"
	"io"
	"net/http"
)

// RouteExecutor is a function that can execute a route with params
type RouteExecutor func(ctx context.Context, params map[string]interface{}) (*Response, error)

// Request represents a fully built HTTP request
type Request struct {
	URL         string
	Method      string
	Body        io.Reader
	Headers     map[string]string
	ContentType string
	HttpRequest *http.Request // The actual HTTP request
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Error      error
}
