package tests

import (
	"context"
	"net/http"
	"testing"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/requester"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuthManager struct {
	applyAuthFunc func(*http.Request) error
}

func (m *mockAuthManager) ApplyAuth(req *http.Request) error {
	return m.applyAuthFunc(req)
}

func TestHTTPRequestBuilder_BuildRequest(t *testing.T) {
	tests := []struct {
		name         string
		route        string
		params       map[string]interface{}
		config       *config.EndpointConfig
		authManager  requester.AuthManager
		routeConfig  *requester.RouteConfig
		wantErr      bool
		checkRequest func(t *testing.T, req *requester.Request)
	}{
		{
			name:  "Simple GET Request",
			route: "test-route",
			params: map[string]interface{}{
				"query": "test",
			},
			config: &config.EndpointConfig{
				BaseURL: "http://api.example.com",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
			routeConfig: &requester.RouteConfig{
				Method: "GET",
				Path:   "/test-route",
			},
			authManager: &mockAuthManager{
				applyAuthFunc: func(req *http.Request) error {
					req.Header.Set("Authorization", "Bearer test-token")
					return nil
				},
			},
			wantErr: false,
			checkRequest: func(t *testing.T, req *requester.Request) {
				assert.Equal(t, "http://api.example.com/test-route?query=test", req.HttpRequest.URL.String())
				assert.Equal(t, "GET", req.HttpRequest.Method)
				assert.Equal(t, "application/json", req.HttpRequest.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer test-token", req.HttpRequest.Header.Get("Authorization"))
			},
		},
		{
			name:  "POST Request with Body",
			route: "create-resource",
			params: map[string]interface{}{
				"body": map[string]interface{}{
					"name": "test",
				},
			},
			config: &config.EndpointConfig{
				BaseURL: "http://api.example.com",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
			routeConfig: &requester.RouteConfig{
				Method: "POST",
				Path:   "/create-resource",
			},
			authManager: &mockAuthManager{
				applyAuthFunc: func(req *http.Request) error {
					return nil
				},
			},
			wantErr: false,
			checkRequest: func(t *testing.T, req *requester.Request) {
				assert.Equal(t, "http://api.example.com/create-resource", req.HttpRequest.URL.String())
				assert.Equal(t, "POST", req.HttpRequest.Method)
				assert.Equal(t, "application/json", req.HttpRequest.Header.Get("Content-Type"))
			},
		},
		{
			name:   "Invalid Route",
			route:  "invalid-route",
			params: map[string]interface{}{},
			config: &config.EndpointConfig{
				BaseURL: "http://api.example.com",
			},
			routeConfig: nil,
			authManager: &mockAuthManager{
				applyAuthFunc: func(req *http.Request) error {
					return nil
				},
			},
			wantErr:      true,
			checkRequest: func(t *testing.T, req *requester.Request) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := requester.NewHTTPRequestBuilder(requester.HTTPRequestBuilderParams{
				EndpointConfig: tt.config,
				AuthManager:    tt.authManager,
				RouteConfig:    tt.routeConfig,
			})

			req, err := builder.BuildRequest(context.Background(), tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.checkRequest(t, req)
		})
	}
}
