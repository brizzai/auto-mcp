package headerpassthrough

import (
	"net/http"
	"testing"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestSanitizer_Sanitize(t *testing.T) {
	tests := []struct {
		name           string
		config         config.HeaderSecurityConfig
		inputHeaders   http.Header
		expectedOutput http.Header
		expectErrors   bool
	}{
		{
			name: "sanitize control characters",
			config: config.HeaderSecurityConfig{
				Sanitize:             true,
				MaxHeaderValueLength: 1000,
				SensitivePatterns:    []string{},
			},
			inputHeaders: http.Header{
				"X-Test": []string{"value\x00with\x1Fcontrol\x7Fchars"},
			},
			expectedOutput: http.Header{
				"X-Test": []string{"valuewithcontrolchars"},
			},
			expectErrors: false,
		},
		{
			name: "truncate long values",
			config: config.HeaderSecurityConfig{
				Sanitize:             true,
				MaxHeaderValueLength: 10,
				SensitivePatterns:    []string{},
			},
			inputHeaders: http.Header{
				"X-Long": []string{"this is a very long header value"},
			},
			expectedOutput: http.Header{
				"X-Long": []string{"this is a"}, // Trimmed after truncation
			},
			expectErrors: true,
		},
		{
			name: "trim whitespace",
			config: config.HeaderSecurityConfig{
				Sanitize:             true,
				MaxHeaderValueLength: 1000,
				SensitivePatterns:    []string{},
			},
			inputHeaders: http.Header{
				"X-Trim": []string{"  spaces around  "},
			},
			expectedOutput: http.Header{
				"X-Trim": []string{"spaces around"},
			},
			expectErrors: false,
		},
		{
			name: "no sanitization when disabled",
			config: config.HeaderSecurityConfig{
				Sanitize:             false,
				MaxHeaderValueLength: 10,
				SensitivePatterns:    []string{},
			},
			inputHeaders: http.Header{
				"X-Test": []string{"  value with spaces  "},
			},
			expectedOutput: http.Header{
				"X-Test": []string{"  value with spaces  "},
			},
			expectErrors: false,
		},
		{
			name: "multiple headers",
			config: config.HeaderSecurityConfig{
				Sanitize:             true,
				MaxHeaderValueLength: 1000,
				SensitivePatterns:    []string{},
			},
			inputHeaders: http.Header{
				"X-Header-1": []string{"value1"},
				"X-Header-2": []string{"value2\x00"},
				"X-Header-3": []string{"  value3  "},
			},
			expectedOutput: http.Header{
				"X-Header-1": []string{"value1"},
				"X-Header-2": []string{"value2"},
				"X-Header-3": []string{"value3"},
			},
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitizer := NewSanitizer(tt.config)
			output, errors := sanitizer.Sanitize(tt.inputHeaders)

			assert.Equal(t, tt.expectedOutput, output)
			if tt.expectErrors {
				assert.NotEmpty(t, errors)
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

func TestSanitizer_IsSensitive(t *testing.T) {
	tests := []struct {
		name       string
		patterns   []string
		headerName string
		expected   bool
	}{
		{
			name:       "matches authorization pattern",
			patterns:   []string{".*[Aa]uth.*"},
			headerName: "Authorization",
			expected:   true,
		},
		{
			name:       "matches cookie pattern case insensitive",
			patterns:   []string{".*[Cc]ookie.*"},
			headerName: "Cookie",
			expected:   true,
		},
		{
			name:       "matches token pattern",
			patterns:   []string{".*[Tt]oken.*"},
			headerName: "X-Access-Token",
			expected:   true,
		},
		{
			name:       "does not match non-sensitive",
			patterns:   []string{".*[Aa]uth.*", ".*[Cc]ookie.*"},
			headerName: "X-Request-Id",
			expected:   false,
		},
		{
			name:       "empty patterns",
			patterns:   []string{},
			headerName: "Authorization",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.HeaderSecurityConfig{
				SensitivePatterns:    tt.patterns,
				Sanitize:             true,
				MaxHeaderValueLength: 1000,
			}
			sanitizer := NewSanitizer(cfg)
			result := sanitizer.isSensitive(tt.headerName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizer_RemoveControlCharacters(t *testing.T) {
	sanitizer := NewSanitizer(config.HeaderSecurityConfig{
		Sanitize:             true,
		MaxHeaderValueLength: 1000,
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text\x00with\x00nulls", "textwithnulls"},
		{"text\nwith\nnewlines", "textwithnewlines"},
		{"text\twith\ttabs", "textwithtabs"},
		{"text\x1Bwith\x1Bescape", "textwithescape"},
		{"\x7Fdelete\x7Fchars", "deletechars"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizer.removeControlCharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
