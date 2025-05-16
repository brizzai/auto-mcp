package tests

import (
	"net/http"
	"testing"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/requester"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPAuthManager_ApplyAuth(t *testing.T) {
	tests := []struct {
		name       string
		authType   config.AuthType
		authConfig map[string]string
		req        *http.Request
		wantErr    bool
		checkAuth  func(t *testing.T, req *http.Request)
	}{
		{
			name:       "No Auth",
			authType:   config.AuthTypeNone,
			authConfig: map[string]string{},
			req:        &http.Request{Header: make(http.Header)},
			wantErr:    false,
			checkAuth: func(t *testing.T, req *http.Request) {
				assert.Empty(t, req.Header.Get("Authorization"))
			},
		},
		{
			name:     "Basic Auth",
			authType: config.AuthTypeBasic,
			authConfig: map[string]string{
				"username": "testuser",
				"password": "testpass",
			},
			req:     &http.Request{Header: make(http.Header)},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				username, password, ok := req.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "testuser", username)
				assert.Equal(t, "testpass", password)
			},
		},
		{
			name:     "Bearer Auth",
			authType: config.AuthTypeBearer,
			authConfig: map[string]string{
				"token": "test-token",
			},
			req:     &http.Request{Header: make(http.Header)},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			},
		},
		{
			name:     "API Key Auth",
			authType: config.AuthTypeAPIKey,
			authConfig: map[string]string{
				"key":    "test-key",
				"header": "X-Custom-Key",
			},
			req:     &http.Request{Header: make(http.Header)},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "test-key", req.Header.Get("X-Custom-Key"))
			},
		},
		{
			name:     "OAuth2 Auth",
			authType: config.AuthTypeOAuth2,
			authConfig: map[string]string{
				"token": "oauth-token",
			},
			req:     &http.Request{Header: make(http.Header)},
			wantErr: false,
			checkAuth: func(t *testing.T, req *http.Request) {
				assert.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
			},
		},
		{
			name:       "Invalid Auth Type",
			authType:   "invalid",
			authConfig: map[string]string{},
			req:        &http.Request{Header: make(http.Header)},
			wantErr:    true,
			checkAuth:  func(t *testing.T, req *http.Request) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := requester.NewHTTPAuthManager(&config.EndpointConfig{
				AuthType:   tt.authType,
				AuthConfig: tt.authConfig,
			})

			err := manager.ApplyAuth(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			tt.checkAuth(t, tt.req)
		})
	}
}
