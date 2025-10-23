package headerpassthrough

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
)

// Sanitizer handles header value sanitization and security
type Sanitizer struct {
	config             config.HeaderSecurityConfig
	sensitivePatterns  []*regexp.Regexp
	maxLength          int
	shouldSanitize     bool
}

// NewSanitizer creates a new header sanitizer
func NewSanitizer(cfg config.HeaderSecurityConfig) *Sanitizer {
	patterns := make([]*regexp.Regexp, 0, len(cfg.SensitivePatterns))
	for _, pattern := range cfg.SensitivePatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			patterns = append(patterns, re)
		} else {
			logger.Warn("Invalid sensitive header pattern", zap.String("pattern", pattern), zap.Error(err))
		}
	}

	maxLength := cfg.MaxHeaderValueLength
	if maxLength <= 0 {
		maxLength = 8192 // Default
	}

	return &Sanitizer{
		config:            cfg,
		sensitivePatterns: patterns,
		maxLength:         maxLength,
		shouldSanitize:    cfg.Sanitize,
	}
}

// Sanitize sanitizes header values
func (s *Sanitizer) Sanitize(headers http.Header) (http.Header, []ProcessingError) {
	sanitized := make(http.Header)
	errors := []ProcessingError{}

	for name, values := range headers {
		if len(values) == 0 {
			continue
		}

		original := values[0]
		cleaned := original

		if s.shouldSanitize {
			// Remove control characters
			cleaned = s.removeControlCharacters(cleaned)

			// Enforce max length
			if len(cleaned) > s.maxLength {
				errors = append(errors, ProcessingError{
					Header:    name,
					Reason:    "value too long, truncated",
					Original:  original,
					Sanitized: cleaned[:s.maxLength],
				})
				cleaned = cleaned[:s.maxLength]
			}

			// Trim whitespace
			cleaned = strings.TrimSpace(cleaned)
		}

		sanitized.Set(name, cleaned)

		// Log if sensitive (but don't include value)
		if s.isSensitive(name) {
			logger.Debug("Processing sensitive header",
				zap.String("header", name),
				zap.String("value", "[REDACTED]"))
		} else {
			logger.Debug("Processing header",
				zap.String("header", name),
				zap.String("value", cleaned))
		}
	}

	return sanitized, errors
}

// removeControlCharacters removes control characters from string
func (s *Sanitizer) removeControlCharacters(value string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1 // Remove character
		}
		return r
	}, value)
}

// isSensitive checks if a header name matches sensitive patterns
func (s *Sanitizer) isSensitive(headerName string) bool {
	for _, pattern := range s.sensitivePatterns {
		if pattern.MatchString(headerName) {
			return true
		}
	}
	return false
}
