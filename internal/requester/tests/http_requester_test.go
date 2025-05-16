package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/requester"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAuthManager implements the AuthManager interface for testing
type MockAuthManager struct{}

func (m *MockAuthManager) ApplyAuth(req *http.Request) error {
	return nil
}

func TestHTTPRequester(t *testing.T) {
	tests := []struct {
		name           string
		routeConfig    *requester.RouteConfig
		serviceConfig  *config.EndpointConfig
		params         map[string]interface{}
		timeout        time.Duration
		serverResponse func(w http.ResponseWriter, r *http.Request)
		checkResponse  func(t *testing.T, response *requester.Response, err error)
	}{
		{
			name: "Simple GET Request",
			routeConfig: &requester.RouteConfig{
				Path:   "/test",
				Method: "GET",
			},
			serviceConfig: &config.EndpointConfig{
				AuthType:   config.AuthTypeNone,
				AuthConfig: nil,
				Headers:    nil,
			},
			timeout: 30 * time.Second,
			params: map[string]interface{}{
				"param1": "value1",
				"param2": "value2",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/test", r.URL.Path)
				assert.Equal(t, "value1", r.URL.Query().Get("param1"))
				assert.Equal(t, "value2", r.URL.Query().Get("param2"))
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
					t.Errorf("Failed to encode response: %v", err)
				}
			},
			checkResponse: func(t *testing.T, response *requester.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)

				var body map[string]string
				err = json.Unmarshal(response.Body, &body)
				require.NoError(t, err)
				assert.Equal(t, "success", body["status"])
			},
		},
		{
			name: "POST Request with Body",
			routeConfig: &requester.RouteConfig{
				Path:   "/test",
				Method: "POST",
			},
			serviceConfig: &config.EndpointConfig{
				AuthType:   config.AuthTypeNone,
				AuthConfig: nil,
				Headers:    nil,
			},
			timeout: 30 * time.Second,
			params: map[string]interface{}{
				"body": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/test", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&body)
				require.NoError(t, err)
				assert.Equal(t, "value1", body["key1"])
				assert.Equal(t, "value2", body["key2"])

				w.WriteHeader(http.StatusCreated)
				if err := json.NewEncoder(w).Encode(map[string]string{"status": "created"}); err != nil {
					t.Errorf("Failed to encode response: %v", err)
				}
			},
			checkResponse: func(t *testing.T, response *requester.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusCreated, response.StatusCode)

				var body map[string]string
				err = json.Unmarshal(response.Body, &body)
				require.NoError(t, err)
				assert.Equal(t, "created", body["status"])
			},
		},
		{
			name: "Request Timeout",
			routeConfig: &requester.RouteConfig{
				Path:   "/timeout",
				Method: "GET",
			},
			serviceConfig: &config.EndpointConfig{
				AuthType:   config.AuthTypeNone,
				AuthConfig: nil,
				Headers:    nil,
			},
			timeout: 100 * time.Millisecond,
			params:  map[string]interface{}{},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			checkResponse: func(t *testing.T, response *requester.Response, err error) {
				assert.Error(t, err)
				assert.Nil(t, response)
			},
		},
		{
			name: "Request with Headers",
			routeConfig: &requester.RouteConfig{
				Path:   "/headers",
				Method: "GET",
			},
			serviceConfig: &config.EndpointConfig{
				AuthType:   config.AuthTypeNone,
				AuthConfig: nil,
				Headers:    map[string]string{"X-Test-Header": "test-value"},
			},
			timeout: 30 * time.Second,
			params:  map[string]interface{}{},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "test-value", r.Header.Get("X-Test-Header"))
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(map[string]string{"status": "success"}); err != nil {
					t.Errorf("Failed to encode response: %v", err)
				}
			},
			checkResponse: func(t *testing.T, response *requester.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, response.StatusCode)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Set the base URL in the service config
			tt.serviceConfig.BaseURL = server.URL

			// Create the requester
			requester := requester.NewHTTPRequester(requester.HTTPRequesterParams{
				ServiceConfig: tt.serviceConfig,
				AuthManager:   &MockAuthManager{},
			})

			// Set timeout
			requester.SetTimeout(tt.timeout)

			// Build the route executor
			executor, err := requester.BuildRouteExecutor(tt.routeConfig)
			require.NoError(t, err)

			// Execute the request
			resp, err := executor(context.Background(), tt.params)

			// Check the response
			tt.checkResponse(t, resp, err)
		})
	}
}
