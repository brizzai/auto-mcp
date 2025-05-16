package parser

import (
	"os"

	"github.com/brizzai/auto-mcp/internal/logger"
	"github.com/brizzai/auto-mcp/internal/models"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Adjuster provides filtering and description overrides based on YAML configuration
type Adjuster struct {
	adjustments *models.MCPAdjustments
}

// NewAdjuster creates a new Adjuster instance
func NewAdjuster() *Adjuster {
	return &Adjuster{
		adjustments: &models.MCPAdjustments{
			Descriptions: []models.RouteDescription{},
			Routes:       []models.RouteSelection{},
		},
	}
}

// Load loads adjustments from a YAML file
func (a *Adjuster) Load(filePath string) error {
	if filePath == "" {
		logger.Info("No adjustments file provided")
		return nil // Return nil if file path is empty
	}

	logger.Info("Loading adjustments from file", zap.String("file", filePath))
	// Check if file exists first
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Error("Adjustments file not found")
		return nil // Return nil if file doesn't exist
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var adjustments models.MCPAdjustments
	err = yaml.Unmarshal(data, &adjustments)
	if err != nil {
		return err
	}

	a.adjustments = &adjustments
	return nil
}

// ExistsInMCP checks if a route with the given method exists in MCP
// Returns true if the route/method IS in the selected routes
func (a *Adjuster) ExistsInMCP(route, method string) bool {
	if a.adjustments == nil || len(a.adjustments.Routes) == 0 {
		return true // No filtering if no adjustments or selected routes, so everything exists
	}

	// Look through all route selections
	for _, selection := range a.adjustments.Routes {
		if selection.Path == route {
			// Check if the method is in the list of selected methods
			for _, m := range selection.Methods {
				if m == method {
					return true
				}
			}
			return false // Path found but method not selected
		}
	}

	return false // Path not found
}

// GetDescription returns the updated description for a route/method if it exists
func (a *Adjuster) GetDescription(route, method, originalDesc string) string {
	if a.adjustments == nil || len(a.adjustments.Descriptions) == 0 {
		return originalDesc // Return original if no adjustments
	}

	// Look through all route descriptions
	for _, desc := range a.adjustments.Descriptions {
		if desc.Path == route {
			// Look through all updates for this route
			for _, update := range desc.Updates {
				if update.Method == method {
					return update.NewDescription
				}
			}
			break // Found the route but no matching method
		}
	}

	return originalDesc
}
