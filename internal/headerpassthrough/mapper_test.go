package headerpassthrough

import (
	"net/http"
	"testing"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMapper_Map(t *testing.T) {
	tests := []struct {
		name           string
		mappings       []config.HeaderMapping
		inputHeaders   http.Header
		expectedOutput map[string]string
	}{
		{
			name: "simple mapping without transformation",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-User-Auth",
					RestApiHeader: "Cookie",
				},
			},
			inputHeaders: http.Header{
				"X-User-Auth": []string{"session=abc123"},
			},
			expectedOutput: map[string]string{
				"Cookie": "session=abc123",
			},
		},
		{
			name: "mapping with prefix",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-User-Token",
					RestApiHeader: "Authorization",
					Prefix:        "Bearer ",
				},
			},
			inputHeaders: http.Header{
				"X-User-Token": []string{"token123"},
			},
			expectedOutput: map[string]string{
				"Authorization": "Bearer token123",
			},
		},
		{
			name: "mapping with suffix",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-Api-Key",
					RestApiHeader: "Authorization",
					Suffix:        " key",
				},
			},
			inputHeaders: http.Header{
				"X-Api-Key": []string{"mykey"},
			},
			expectedOutput: map[string]string{
				"Authorization": "mykey key",
			},
		},
		{
			name: "mapping with both prefix and suffix",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-Token",
					RestApiHeader: "X-Custom",
					Prefix:        "[",
					Suffix:        "]",
				},
			},
			inputHeaders: http.Header{
				"X-Token": []string{"value"},
			},
			expectedOutput: map[string]string{
				"X-Custom": "[value]",
			},
		},
		{
			name: "passthrough without mapping",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-Mapped",
					RestApiHeader: "X-Target",
				},
			},
			inputHeaders: http.Header{
				"X-Request-Id": []string{"req123"},
			},
			expectedOutput: map[string]string{
				"X-Request-Id": "req123",
			},
		},
		{
			name: "multiple mappings",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-User-Auth",
					RestApiHeader: "Cookie",
				},
				{
					McpHeader:     "X-User-Token",
					RestApiHeader: "Authorization",
					Prefix:        "Bearer ",
				},
			},
			inputHeaders: http.Header{
				"X-User-Auth":  []string{"session=abc"},
				"X-User-Token": []string{"token123"},
				"X-Request-Id": []string{"req456"},
			},
			expectedOutput: map[string]string{
				"Cookie":        "session=abc",
				"Authorization": "Bearer token123",
				"X-Request-Id":  "req456",
			},
		},
		{
			name:     "empty mappings",
			mappings: []config.HeaderMapping{},
			inputHeaders: http.Header{
				"X-Header": []string{"value"},
			},
			expectedOutput: map[string]string{
				"X-Header": "value",
			},
		},
		{
			name: "empty input headers",
			mappings: []config.HeaderMapping{
				{
					McpHeader:     "X-User-Auth",
					RestApiHeader: "Cookie",
				},
			},
			inputHeaders:   http.Header{},
			expectedOutput: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := NewMapper(tt.mappings)
			output := mapper.Map(tt.inputHeaders)
			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestMapper_Transform(t *testing.T) {
	mapper := NewMapper([]config.HeaderMapping{})

	tests := []struct {
		name     string
		value    string
		mapping  config.HeaderMapping
		expected string
	}{
		{
			name:  "no transformation",
			value: "value",
			mapping: config.HeaderMapping{
				McpHeader:     "X-Test",
				RestApiHeader: "X-Target",
			},
			expected: "value",
		},
		{
			name:  "prefix only",
			value: "value",
			mapping: config.HeaderMapping{
				McpHeader:     "X-Test",
				RestApiHeader: "X-Target",
				Prefix:        "PREFIX-",
			},
			expected: "PREFIX-value",
		},
		{
			name:  "suffix only",
			value: "value",
			mapping: config.HeaderMapping{
				McpHeader:     "X-Test",
				RestApiHeader: "X-Target",
				Suffix:        "-SUFFIX",
			},
			expected: "value-SUFFIX",
		},
		{
			name:  "both prefix and suffix",
			value: "value",
			mapping: config.HeaderMapping{
				McpHeader:     "X-Test",
				RestApiHeader: "X-Target",
				Prefix:        "<<<",
				Suffix:        ">>>",
			},
			expected: "<<<value>>>",
		},
		{
			name:  "empty value",
			value: "",
			mapping: config.HeaderMapping{
				McpHeader:     "X-Test",
				RestApiHeader: "X-Target",
				Prefix:        "pre",
				Suffix:        "post",
			},
			expected: "prepost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.transform(tt.value, tt.mapping)
			assert.Equal(t, tt.expected, result)
		})
	}
}
