package tui

import (
	"os"
	"sort"
	"testing"

	"github.com/brizzai/auto-mcp/internal/parser"
	"github.com/brizzai/auto-mcp/internal/requester"
	"github.com/brizzai/auto-mcp/internal/tui/models"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// Helper function to sort routes by path
func sortRoutes(routes []interface{}) {
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].(map[string]interface{})["path"].(string) < routes[j].(map[string]interface{})["path"].(string)
	})
}

// Helper function to sort descriptions by path
func sortDescriptions(descriptions []interface{}) {
	sort.Slice(descriptions, func(i, j int) bool {
		return descriptions[i].(map[string]interface{})["path"].(string) < descriptions[j].(map[string]interface{})["path"].(string)
	})
}

// Helper function to sort methods in routes
func sortMethods(routes []interface{}) {
	for _, route := range routes {
		methods := route.(map[string]interface{})["methods"].([]interface{})
		sort.Slice(methods, func(i, j int) bool {
			return methods[i].(string) < methods[j].(string)
		})
	}
}

func TestExportRoutesToYamlFile(t *testing.T) {
	// Create a temporary file with random name
	tempFile, err := os.CreateTemp("", "test-export-*.yaml")
	assert.NoError(t, err)
	defer func() {
		if closeErr := tempFile.Close(); closeErr != nil {
			t.Errorf("Failed to close temporary file: %v", closeErr)
		}
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Errorf("Failed to remove temporary file: %v", err)
		}
	}()

	// Test cases
	testCases := []struct {
		name     string
		routes   []*models.RouteToolItem
		expected map[string]interface{}
	}{
		{
			name:     "Empty routes list",
			routes:   []*models.RouteToolItem{},
			expected: map[string]interface{}{},
		},
		{
			name:     "Routes with updated descriptions",
			routes:   createRoutesWithUpdatedDescriptions(),
			expected: expectedYamlForUpdatedDescriptions(),
		},
		{
			name:     "Routes marked as removed",
			routes:   createRemovedRoutes(),
			expected: map[string]interface{}{},
		},
		{
			name:     "Routes with both updates and removals",
			routes:   createMixedUpdatesAndRemovals(),
			expected: expectedYamlForMixedUpdatesAndRemovals(),
		},
	}

	// Run all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// When: Export to YAML
			err = ExportRoutesToYamlFile(tc.routes, tempFile.Name())
			assert.NoError(t, err)

			// Then: Read and verify the file content
			actualYaml := readYamlFile(t, tempFile.Name())

			// Sort both expected and actual data for comparison
			if routes, ok := tc.expected["routes"].([]interface{}); ok {
				sortRoutes(routes)
				sortMethods(routes)
			}
			if descriptions, ok := tc.expected["descriptions"].([]interface{}); ok {
				sortDescriptions(descriptions)
			}
			if routes, ok := actualYaml["routes"].([]interface{}); ok {
				sortRoutes(routes)
				sortMethods(routes)
			}
			if descriptions, ok := actualYaml["descriptions"].([]interface{}); ok {
				sortDescriptions(descriptions)
			}

			diff := cmp.Diff(tc.expected, actualYaml, cmpopts.EquateEmpty())
			assert.True(t, diff == "", diff)
		})
	}
}

// readYamlFile reads and parses a YAML file, failing the test if any errors occur
func readYamlFile(t *testing.T, filePath string) map[string]interface{} {
	// Read the file
	yamlData, err := os.ReadFile(filePath)
	assert.NoError(t, err)

	// Parse the YAML
	var result map[string]interface{}
	err = yaml.Unmarshal(yamlData, &result)
	assert.NoError(t, err)

	return result
}

// Test data builders

// routeData represents the data needed to create a RouteToolItem for testing
type routeData struct {
	path           string
	method         string
	description    string
	newDescription string
}

// createRoutesWithUpdatedDescriptions creates routes with updated descriptions
func createRoutesWithUpdatedDescriptions() []*models.RouteToolItem {
	return createRouteItems([]*routeData{
		{path: "/api/products", method: "GET", description: "Get products", newDescription: ""},
		{path: "/api/items", method: "POST", description: "Create item", newDescription: "Updated item creation"},
		{path: "/api/users", method: "GET", description: "Get users", newDescription: "Updated users description"},
	}, []string{})
}

// expectedYamlForUpdatedDescriptions returns the expected YAML for the updated descriptions test case
func expectedYamlForUpdatedDescriptions() map[string]interface{} {
	return map[string]interface{}{
		"descriptions": []interface{}{
			map[string]interface{}{
				"path": "/api/items",
				"updates": []interface{}{
					map[string]interface{}{
						"method":          "POST",
						"new_description": "Updated item creation",
					},
				},
			},
			map[string]interface{}{
				"path": "/api/users",
				"updates": []interface{}{
					map[string]interface{}{
						"method":          "GET",
						"new_description": "Updated users description",
					},
				},
			},
		},
		"routes": []interface{}{
			map[string]interface{}{
				"path":    "/api/products",
				"methods": []interface{}{"GET"},
			},
			map[string]interface{}{
				"path":    "/api/items",
				"methods": []interface{}{"POST"},
			},
			map[string]interface{}{
				"path":    "/api/users",
				"methods": []interface{}{"GET"},
			},
		},
	}
}

// createRemovedRoutes creates routes marked as removed
func createRemovedRoutes() []*models.RouteToolItem {
	return createRouteItems([]*routeData{
		{path: "/api/users", method: "DELETE", description: "Delete user", newDescription: ""},
		{path: "/api/settings", method: "PUT", description: "Update settings", newDescription: ""},
	}, []string{"/api/users:DELETE", "/api/settings:PUT"})
}

// createMixedUpdatesAndRemovals creates routes with both updates and removals
func createMixedUpdatesAndRemovals() []*models.RouteToolItem {
	return createRouteItems([]*routeData{
		{path: "/api/users", method: "GET", description: "Get users", newDescription: "Updated users description"},
		{path: "/api/users", method: "DELETE", description: "Delete user", newDescription: ""},
		{path: "/api/items", method: "POST", description: "Create item", newDescription: "Updated item creation"},
	}, []string{"/api/users:DELETE"})
}

// expectedYamlForMixedUpdatesAndRemovals returns the expected YAML for mixed updates and removals
func expectedYamlForMixedUpdatesAndRemovals() map[string]interface{} {
	return map[string]interface{}{
		"descriptions": []interface{}{
			map[string]interface{}{
				"path": "/api/users",
				"updates": []interface{}{
					map[string]interface{}{
						"method":          "GET",
						"new_description": "Updated users description",
					},
				},
			},
			map[string]interface{}{
				"path": "/api/items",
				"updates": []interface{}{
					map[string]interface{}{
						"method":          "POST",
						"new_description": "Updated item creation",
					},
				},
			},
		},
		"routes": []interface{}{
			map[string]interface{}{
				"path":    "/api/users",
				"methods": []interface{}{"GET"},
			},
			map[string]interface{}{
				"path":    "/api/items",
				"methods": []interface{}{"POST"},
			},
		},
	}
}

// createRouteItems creates test RouteToolItem objects
func createRouteItems(routes []*routeData, removedRoutes []string) []*models.RouteToolItem {
	items := make([]*models.RouteToolItem, 0, len(routes))

	for _, r := range routes {
		// Create route configuration
		routeConfig := &requester.RouteConfig{
			Path:        r.path,
			Method:      r.method,
			Description: r.description,
			Headers:     map[string]string{},
			Parameters:  map[string]string{},
		}

		// Create tool with route config
		tool := &parser.RouteTool{
			RouteConfig: routeConfig,
			Tool:        mcp.Tool{},
		}

		// Create route item and apply modifications
		item := models.RouteToolItem{
			Tool: tool,
		}

		// Apply description update if provided
		if r.newDescription != "" {
			item = item.UpdatedDescription(r.newDescription)
		}

		// Mark as removed if in the removal list
		routeKey := r.path + ":" + r.method
		for _, removedRoute := range removedRoutes {
			if removedRoute == routeKey {
				item = item.ToggleRemoved()
				break
			}
		}

		items = append(items, &item)
	}

	return items
}
