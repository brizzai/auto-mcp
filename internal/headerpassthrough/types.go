package headerpassthrough

import (
	"context"
)

// ProcessedHeaders holds the headers ready to be forwarded to REST API
type ProcessedHeaders map[string]string

// contextKey is a custom type for storing headers in context
type contextKey string

const passthroughHeadersKey contextKey = "passthrough-headers"

// ProcessingError represents an error during header processing
type ProcessingError struct {
	Header    string
	Reason    string
	Original  string
	Sanitized string
}

// StoreInContext stores processed headers in the context
func StoreInContext(ctx context.Context, headers ProcessedHeaders) context.Context {
	return context.WithValue(ctx, passthroughHeadersKey, headers)
}

// ExtractFromContext retrieves processed headers from context
func ExtractFromContext(ctx context.Context) ProcessedHeaders {
	if headers, ok := ctx.Value(passthroughHeadersKey).(ProcessedHeaders); ok {
		return headers
	}
	return nil
}
