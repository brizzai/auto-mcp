package requester

import (
	"net/http"
)

// RouteConfig holds the configuration for a specific route
type RouteConfig struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Description string            `json:"description,omitempty"`
	Headers     map[string]string `json:"headers"`
	Parameters  map[string]string `json:"parameters"`
	// Method specific configurations
	MethodConfig MethodConfig `json:"method_config"`
}

// MethodConfig holds method-specific configurations
type MethodConfig struct {
	// For GET requests
	QueryParams []string `json:"query_params,omitempty"`

	// For multipart/form-data
	FormFields []string `json:"form_fields,omitempty"`

	// For file uploads
	FileUpload *FileUploadConfig `json:"file_upload,omitempty"`
}

// FileUploadConfig holds configuration for file uploads
type FileUploadConfig struct {
	FieldName    string   `json:"field_name"`
	AllowedTypes []string `json:"allowed_types"`
	MaxSize      int64    `json:"max_size"`
}

// RequestResult holds the result of a request
type RequestResult struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	Error      error
}
