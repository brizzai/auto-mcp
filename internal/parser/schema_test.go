package parser

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestSchemaToMCPOptions(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi3.SchemaRef
		required bool
		check    func(t *testing.T, got mcp.ToolOption)
	}{
		{
			name:     "nil schema",
			schema:   nil,
			required: false,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				assert.Equal(t, "test", tool.Name)
				assert.Empty(t, tool.Description)
			},
		},
		{
			name: "array schema",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"array"},
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
			required: true,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				assert.Equal(t, "test", tool.Name)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "array", prop["type"])
				if len(tool.InputSchema.Required) > 0 {
					assert.Contains(t, tool.InputSchema.Required, "test")
				}
			},
		},
		{
			name: "object schema with properties",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: &openapi3.Types{"object"},
					Properties: map[string]*openapi3.SchemaRef{
						"name": {
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"string"},
							},
						},
					},
					Required: []string{"name"},
				},
			},
			required: true,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				assert.Equal(t, "test", tool.Name)
				assert.Equal(t, "object", tool.InputSchema.Type)

				// Debug: Print the actual structure
				t.Logf("InputSchema: %+v", tool.InputSchema)
				t.Logf("Properties: %+v", tool.InputSchema.Properties)

				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				t.Logf("Test property: %+v", prop)

				// Get the nested properties map
				objPropsVal := prop["properties"]
				t.Logf("objPropsVal: %T, %+v", objPropsVal, objPropsVal)
				objProps, ok := objPropsVal.(map[string]interface{})
				assert.True(t, ok)
				t.Logf("Properties map: %+v", objProps)
				// Check if name exists in properties
				assert.Contains(t, objProps, "name")
				// Check required fields
				reqVal := prop["required"]
				t.Logf("reqVal: %T, %+v", reqVal, reqVal)
				req, ok := reqVal.([]string)
				if !ok {
					reqIface, ok2 := reqVal.([]interface{})
					assert.True(t, ok2)
					for _, r := range reqIface {
						if s, ok := r.(string); ok && s == "name" {
							return
						}
					}
					assert.Fail(t, "'name' not found in required fields")
				} else {
					found := false
					for _, r := range req {
						if r == "name" {
							found = true
						}
					}
					assert.True(t, found)
				}
			},
		},
		{
			name: "string schema with constraints",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"string"},
					MaxLength:   openapi3.Uint64Ptr(100),
					MinLength:   1,
					Pattern:     "^[a-zA-Z]+$",
					Enum:        []interface{}{"option1", "option2"},
					Description: "Test string",
				},
			},
			required: false,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "Test string", prop["description"])
				assert.EqualValues(t, 100, prop["maxLength"])
				assert.EqualValues(t, 1, prop["minLength"])
				assert.Equal(t, "^[a-zA-Z]+$", prop["pattern"])
				assert.ElementsMatch(t, []interface{}{"option1", "option2"}, prop["enum"])
			},
		},
		{
			name: "number schema with constraints",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"number"},
					Max:         openapi3.Float64Ptr(100),
					Min:         openapi3.Float64Ptr(0),
					MultipleOf:  openapi3.Float64Ptr(2),
					Description: "Test number",
				},
			},
			required: true,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "Test number", prop["description"])
				assert.Equal(t, 100.0, prop["maximum"])
				assert.Equal(t, 0.0, prop["minimum"])
				assert.Equal(t, 2.0, prop["multipleOf"])
			},
		},
		{
			name: "boolean schema",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"boolean"},
					Description: "Test boolean",
				},
			},
			required: false,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "Test boolean", prop["description"])
				assert.Equal(t, "boolean", prop["type"])
			},
		},
		{
			name: "unknown type schema",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type:        &openapi3.Types{"unknown"},
					Description: "Test unknown",
				},
			},
			required: false,
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				assert.Empty(t, tool.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schemaToMCPOptions(tt.schema, "test", tt.required, nil)
			tt.check(t, got)
		})
	}
}

func TestCreateArrayOption(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi3.SchemaRef
		baseOpts []mcp.PropertyOption
		check    func(t *testing.T, got mcp.ToolOption)
	}{
		{
			name: "array with items",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"string"},
						},
					},
				},
			},
			baseOpts: []mcp.PropertyOption{mcp.Description("Test array")},
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				assert.Equal(t, "test", tool.Name)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "array", prop["type"])
				assert.Equal(t, "Test array", prop["description"])
				if items, ok := prop["items"].(map[string]interface{}); ok {
					assert.Equal(t, "string", items["type"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createArrayOption(tt.schema, "test", tt.baseOpts)
			tt.check(t, got)
		})
	}
}

func TestCreateObjectOption(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi3.SchemaRef
		baseOpts []mcp.PropertyOption
		check    func(t *testing.T, got mcp.ToolOption)
	}{
		{
			name: "object with properties and constraints",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Properties: map[string]*openapi3.SchemaRef{
						"name": {
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"string"},
							},
						},
					},
					Required: []string{"name"},
					MaxProps: openapi3.Uint64Ptr(10),
					MinProps: 1,
					AdditionalProperties: openapi3.AdditionalProperties{
						Has: openapi3.BoolPtr(true),
					},
				},
			},
			baseOpts: []mcp.PropertyOption{mcp.Description("Test object")},
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				assert.Equal(t, "test", tool.Name)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "object", prop["type"])
				assert.Equal(t, "Test object", prop["description"])

				props, ok := prop["properties"].(map[string]interface{})
				assert.True(t, ok)
				assert.Contains(t, props, "name")

				required, ok := prop["required"].([]string)
				if !ok {
					reqIface, ok := prop["required"].([]interface{})
					assert.True(t, ok)
					found := false
					for _, r := range reqIface {
						if s, ok := r.(string); ok && s == "name" {
							found = true
							break
						}
					}
					assert.True(t, found, "'name' not found in required fields")
				} else {
					assert.Contains(t, required, "name")
				}

				assert.EqualValues(t, 10, prop["maxProperties"])
				assert.EqualValues(t, 1, prop["minProperties"])
				assert.Equal(t, true, prop["additionalProperties"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createObjectOption(tt.schema, "test", tt.baseOpts, nil)
			tt.check(t, got)
		})
	}
}

func TestCreateStringOption(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi3.SchemaRef
		baseOpts []mcp.PropertyOption
		check    func(t *testing.T, got mcp.ToolOption)
	}{
		{
			name: "string with all constraints",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					MaxLength:   openapi3.Uint64Ptr(100),
					MinLength:   1,
					Pattern:     "^[a-zA-Z]+$",
					Enum:        []interface{}{"option1", "option2"},
					Description: "Test string",
				},
			},
			baseOpts: []mcp.PropertyOption{mcp.Description("Test string")},
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "Test string", prop["description"])
				assert.EqualValues(t, 100, prop["maxLength"])
				assert.EqualValues(t, 1, prop["minLength"])
				assert.Equal(t, "^[a-zA-Z]+$", prop["pattern"])
				assert.ElementsMatch(t, []interface{}{"option1", "option2"}, prop["enum"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createStringOption(tt.schema, "test", tt.baseOpts)
			tt.check(t, got)
		})
	}
}

func TestCreateNumberOption(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi3.SchemaRef
		baseOpts []mcp.PropertyOption
		check    func(t *testing.T, got mcp.ToolOption)
	}{
		{
			name: "number with all constraints",
			schema: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Max:         openapi3.Float64Ptr(100),
					Min:         openapi3.Float64Ptr(0),
					MultipleOf:  openapi3.Float64Ptr(2),
					Description: "Test number",
				},
			},
			baseOpts: []mcp.PropertyOption{mcp.Description("Test number")},
			check: func(t *testing.T, got mcp.ToolOption) {
				tool := mcp.NewTool("test", got)
				prop, ok := tool.InputSchema.Properties["test"].(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, "Test number", prop["description"])
				assert.Equal(t, 100.0, prop["maximum"])
				assert.Equal(t, 0.0, prop["minimum"])
				assert.Equal(t, 2.0, prop["multipleOf"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createNumberOption(tt.schema, "test", tt.baseOpts)
			tt.check(t, got)
		})
	}
}
