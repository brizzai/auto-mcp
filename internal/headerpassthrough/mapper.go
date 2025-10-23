package headerpassthrough

import (
	"net/http"

	"github.com/brizzai/auto-mcp/internal/config"
	"github.com/brizzai/auto-mcp/internal/logger"
	"go.uber.org/zap"
)

// Mapper handles header mapping and transformation
type Mapper struct {
	mappings map[string]config.HeaderMapping
}

// NewMapper creates a new header mapper
func NewMapper(mappings []config.HeaderMapping) *Mapper {
	mappingMap := make(map[string]config.HeaderMapping)
	for _, mapping := range mappings {
		mappingMap[mapping.McpHeader] = mapping
	}

	return &Mapper{
		mappings: mappingMap,
	}
}

// Map maps and transforms headers according to configuration
func (m *Mapper) Map(headers http.Header) map[string]string {
	result := make(map[string]string)

	for name, values := range headers {
		if len(values) == 0 {
			continue
		}

		value := values[0]
		targetName := name
		transformedValue := value

		// Check if there's a mapping for this header
		if mapping, exists := m.mappings[name]; exists {
			targetName = mapping.RestApiHeader
			transformedValue = m.transform(value, mapping)

			logger.Debug("Mapped header",
				zap.String("from", name),
				zap.String("to", targetName),
				zap.Bool("transformed", mapping.Prefix != "" || mapping.Suffix != ""))
		}

		result[targetName] = transformedValue
	}

	return result
}

// transform applies prefix/suffix transformations
func (m *Mapper) transform(value string, mapping config.HeaderMapping) string {
	result := value
	if mapping.Prefix != "" {
		result = mapping.Prefix + result
	}
	if mapping.Suffix != "" {
		result = result + mapping.Suffix
	}
	return result
}
