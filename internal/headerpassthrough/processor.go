package headerpassthrough

import (
	"context"
	"net/http"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
)

// Processor handles header extraction, validation, and transformation
type Processor struct {
	config    *config.HeaderPassthroughConfig
	sanitizer *Sanitizer
	mapper    *Mapper
}

// NewProcessor creates a new header processor
func NewProcessor(cfg *config.HeaderPassthroughConfig) *Processor {
	if cfg == nil {
		cfg = config.DefaultHeaderPassthroughConfig()
	}

	return &Processor{
		config:    cfg,
		sanitizer: NewSanitizer(cfg.Security),
		mapper:    NewMapper(cfg.HeaderMapping),
	}
}

// ProcessHeaders extracts and processes headers from HTTP request
func (p *Processor) ProcessHeaders(ctx context.Context, r *http.Request) context.Context {
	if !p.config.Enabled {
		return ctx
	}

	// 1. Filter allowed headers
	filtered := p.filterAllowed(r.Header)
	if len(filtered) == 0 {
		logger.Debug("No passthrough headers found in request")
		return ctx
	}

	logger.Debug("Processing passthrough headers",
		zap.Int("count", len(filtered)))

	// 2. Sanitize values
	sanitized, sanitizeErrors := p.sanitizer.Sanitize(filtered)

	// 3. Handle errors based on configuration
	if len(sanitizeErrors) > 0 {
		if err := p.handleErrors(sanitizeErrors); err != nil {
			// For reject mode, don't add headers to context
			logger.Error("Header processing rejected",
				zap.Int("error_count", len(sanitizeErrors)))
			return ctx
		}
	}

	// 4. Map and transform
	mapped := p.mapper.Map(sanitized)

	// 5. Store in context
	logger.Info("Applied passthrough headers",
		zap.Int("count", len(mapped)))

	return StoreInContext(ctx, ProcessedHeaders(mapped))
}

// filterAllowed filters headers based on whitelist
func (p *Processor) filterAllowed(headers http.Header) http.Header {
	filtered := make(http.Header)

	for _, allowed := range p.config.AllowedHeaders {
		if value := headers.Get(allowed); value != "" {
			filtered.Set(allowed, value)
		}
	}

	return filtered
}

// handleErrors handles processing errors based on configuration
func (p *Processor) handleErrors(errors []ProcessingError) error {
	switch p.config.OnError {
	case config.ErrorHandlingWarn:
		// Log warnings but continue
		for _, err := range errors {
			logger.Warn("Header processing warning",
				zap.String("header", err.Header),
				zap.String("reason", err.Reason))
		}
		return nil

	case config.ErrorHandlingReject:
		// Reject the request
		for _, err := range errors {
			logger.Error("Header processing error",
				zap.String("header", err.Header),
				zap.String("reason", err.Reason))
		}
		return &ProcessingError{
			Header: "multiple",
			Reason: "header processing failed",
		}

	case config.ErrorHandlingFallback:
		// Log and fall back to static auth
		logger.Warn("Falling back to static auth due to header processing errors",
			zap.Int("error_count", len(errors)))
		return &ProcessingError{
			Header: "fallback",
			Reason: "falling back to static auth",
		}

	default:
		return nil
	}
}

// Error implements error interface for ProcessingError
func (e *ProcessingError) Error() string {
	return e.Reason
}
