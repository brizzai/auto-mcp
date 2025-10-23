package headerpassthrough

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestProcessor_ProcessHeaders(t *testing.T) {
	tests := []struct {
		name            string
		config          *config.HeaderPassthroughConfig
		requestHeaders  http.Header
		expectedHeaders ProcessedHeaders
		expectInContext bool
	}{
		{
			name: "forward whitelisted header with mapping",
			config: &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: []string{"X-User-Auth"},
				HeaderMapping: []config.HeaderMapping{
					{
						McpHeader:     "X-User-Auth",
						RestApiHeader: "Cookie",
					},
				},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
					SensitivePatterns:    []string{},
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders: http.Header{
				"X-User-Auth": []string{"session=abc123"},
			},
			expectedHeaders: ProcessedHeaders{
				"Cookie": "session=abc123",
			},
			expectInContext: true,
		},
		{
			name: "reject non-whitelisted header",
			config: &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: []string{"X-User-Auth"},
				HeaderMapping:  []config.HeaderMapping{},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
					SensitivePatterns:    []string{},
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders: http.Header{
				"X-Malicious": []string{"hack"},
			},
			expectedHeaders: nil,
			expectInContext: false,
		},
		{
			name: "multiple whitelisted headers",
			config: &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: []string{"X-User-Auth", "X-Request-Id"},
				HeaderMapping: []config.HeaderMapping{
					{
						McpHeader:     "X-User-Auth",
						RestApiHeader: "Cookie",
					},
				},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
					SensitivePatterns:    []string{},
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders: http.Header{
				"X-User-Auth":  []string{"session=xyz"},
				"X-Request-Id": []string{"req-123"},
			},
			expectedHeaders: ProcessedHeaders{
				"Cookie":       "session=xyz",
				"X-Request-Id": "req-123",
			},
			expectInContext: true,
		},
		{
			name: "disabled passthrough",
			config: &config.HeaderPassthroughConfig{
				Enabled:        false,
				AllowedHeaders: []string{"X-User-Auth"},
				HeaderMapping:  []config.HeaderMapping{},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders: http.Header{
				"X-User-Auth": []string{"session=abc123"},
			},
			expectedHeaders: nil,
			expectInContext: false,
		},
		{
			name: "header with prefix transformation",
			config: &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: []string{"X-User-Token"},
				HeaderMapping: []config.HeaderMapping{
					{
						McpHeader:     "X-User-Token",
						RestApiHeader: "Authorization",
						Prefix:        "Bearer ",
					},
				},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
					SensitivePatterns:    []string{},
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders: http.Header{
				"X-User-Token": []string{"token123"},
			},
			expectedHeaders: ProcessedHeaders{
				"Authorization": "Bearer token123",
			},
			expectInContext: true,
		},
		{
			name: "sanitize control characters",
			config: &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: []string{"X-Test"},
				HeaderMapping:  []config.HeaderMapping{},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
					SensitivePatterns:    []string{},
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders: http.Header{
				"X-Test": []string{"value\x00with\x00nulls"},
			},
			expectedHeaders: ProcessedHeaders{
				"X-Test": "valuewithnulls",
			},
			expectInContext: true,
		},
		{
			name: "no headers in request",
			config: &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: []string{"X-User-Auth"},
				HeaderMapping:  []config.HeaderMapping{},
				Security: config.HeaderSecurityConfig{
					Sanitize:             true,
					MaxHeaderValueLength: 8192,
				},
				OnError: config.ErrorHandlingWarn,
			},
			requestHeaders:  http.Header{},
			expectedHeaders: nil,
			expectInContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewProcessor(tt.config)
			req := httptest.NewRequest("POST", "/mcp", nil)
			req.Header = tt.requestHeaders

			ctx := processor.ProcessHeaders(context.Background(), req)
			headers := ExtractFromContext(ctx)

			if tt.expectInContext {
				assert.NotNil(t, headers)
				assert.Equal(t, tt.expectedHeaders, headers)
			} else {
				assert.Nil(t, headers)
			}
		})
	}
}

func TestProcessor_FilterAllowed(t *testing.T) {
	tests := []struct {
		name           string
		allowedHeaders []string
		inputHeaders   http.Header
		expectedOutput http.Header
	}{
		{
			name:           "filter single allowed header",
			allowedHeaders: []string{"X-Allowed"},
			inputHeaders: http.Header{
				"X-Allowed":     []string{"value1"},
				"X-Not-Allowed": []string{"value2"},
			},
			expectedOutput: http.Header{
				"X-Allowed": []string{"value1"},
			},
		},
		{
			name:           "filter multiple allowed headers",
			allowedHeaders: []string{"X-Header-1", "X-Header-2"},
			inputHeaders: http.Header{
				"X-Header-1": []string{"value1"},
				"X-Header-2": []string{"value2"},
				"X-Header-3": []string{"value3"},
			},
			expectedOutput: http.Header{
				"X-Header-1": []string{"value1"},
				"X-Header-2": []string{"value2"},
			},
		},
		{
			name:           "no allowed headers",
			allowedHeaders: []string{},
			inputHeaders: http.Header{
				"X-Header": []string{"value"},
			},
			expectedOutput: http.Header{},
		},
		{
			name:           "allowed header not in request",
			allowedHeaders: []string{"X-Missing"},
			inputHeaders: http.Header{
				"X-Present": []string{"value"},
			},
			expectedOutput: http.Header{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.HeaderPassthroughConfig{
				Enabled:        true,
				AllowedHeaders: tt.allowedHeaders,
			}
			processor := NewProcessor(cfg)
			output := processor.filterAllowed(tt.inputHeaders)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestProcessor_HandleErrors(t *testing.T) {
	tests := []struct {
		name        string
		errorMode   config.ErrorHandlingMode
		errors      []ProcessingError
		expectError bool
	}{
		{
			name:      "warn mode continues with errors",
			errorMode: config.ErrorHandlingWarn,
			errors: []ProcessingError{
				{Header: "X-Test", Reason: "too long"},
			},
			expectError: false,
		},
		{
			name:      "reject mode returns error",
			errorMode: config.ErrorHandlingReject,
			errors: []ProcessingError{
				{Header: "X-Test", Reason: "invalid"},
			},
			expectError: true,
		},
		{
			name:      "fallback mode returns error",
			errorMode: config.ErrorHandlingFallback,
			errors: []ProcessingError{
				{Header: "X-Test", Reason: "sanitization failed"},
			},
			expectError: true,
		},
		{
			name:        "no errors",
			errorMode:   config.ErrorHandlingWarn,
			errors:      []ProcessingError{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.HeaderPassthroughConfig{
				Enabled: true,
				OnError: tt.errorMode,
			}
			processor := NewProcessor(cfg)
			err := processor.handleErrors(tt.errors)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractFromContext(t *testing.T) {
	tests := []struct {
		name     string
		headers  ProcessedHeaders
		expected ProcessedHeaders
	}{
		{
			name: "extract existing headers",
			headers: ProcessedHeaders{
				"Cookie":        "session=abc",
				"Authorization": "Bearer token",
			},
			expected: ProcessedHeaders{
				"Cookie":        "session=abc",
				"Authorization": "Bearer token",
			},
		},
		{
			name:     "extract from empty context",
			headers:  nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.headers != nil {
				ctx = StoreInContext(ctx, tt.headers)
			}

			result := ExtractFromContext(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewProcessor_WithDefaults(t *testing.T) {
	// Test that nil config uses defaults
	processor := NewProcessor(nil)
	assert.NotNil(t, processor)
	assert.NotNil(t, processor.config)
	assert.False(t, processor.config.Enabled)
	assert.Equal(t, config.ErrorHandlingFallback, processor.config.OnError)
}
