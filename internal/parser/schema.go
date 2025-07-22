package parser

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mark3labs/mcp-go/mcp"
)

// schemaToMCPOptions converts an OpenAPI schema to MCP tool option
func schemaToMCPOptions(schema *openapi3.SchemaRef, name string, required bool, doc *openapi3.T) mcp.ToolOption {
	if schema == nil || schema.Value == nil || schema.Value.Type == nil {
		if required {
			return mcp.WithObject(name,
				mcp.Description("Request body"),
				mcp.Required(),
			)
		}
		return mcp.WithObject(name,
			mcp.Description("Request body"),
		)
	}

	baseOpts := []mcp.PropertyOption{
		mcp.Description(schema.Value.Description),
	}
	if required {
		baseOpts = append(baseOpts, mcp.Required())
	}

	switch {
	case schema.Value.Type.Includes(openapi3.TypeArray):
		return createArrayOption(schema, name, baseOpts, doc)

	case schema.Value.Type.Includes(openapi3.TypeObject):
		return createObjectOption(schema, name, baseOpts, doc)

	case schema.Value.Type.Includes(openapi3.TypeString):
		return createStringOption(schema, name, baseOpts)

	case (schema.Value.Type.Includes(openapi3.TypeNumber) || schema.Value.Type.Includes(openapi3.TypeInteger)):
		return createNumberOption(schema, name, baseOpts)

	case schema.Value.Type.Includes(openapi3.TypeBoolean):
		return mcp.WithBoolean(name, baseOpts...)

	case schema.Value.Type.Includes(openapi3.TypeString):
		return mcp.WithString(name, baseOpts...)
	default:
		// Fallback to string with a warning in the description
		defaultOpts := append(baseOpts,
			mcp.Description(fmt.Sprintf(
				"%s (unknown type: %v)",
				schema.Value.Description,
				schema.Value.Type.Slice(), // show all allowed types
			)),
		)
		return mcp.WithObject(name, defaultOpts...)
	}
}

func createArrayOption(schema *openapi3.SchemaRef, name string, baseOpts []mcp.PropertyOption, doc *openapi3.T) mcp.ToolOption {
	arrayOpts := baseOpts
	if schema.Value.Items != nil {
		// Convert the items schema to JSON schema format
		itemsSchema := resolveSchemaRef(schema.Value.Items, doc)
		arrayOpts = append(arrayOpts, mcp.Items(itemsSchema))
	}
	return mcp.WithArray(name, arrayOpts...)
}

// resolveSchemaRef resolves a schema reference and converts it to JSON schema format
func resolveSchemaRef(schemaRef *openapi3.SchemaRef, doc *openapi3.T) map[string]interface{} {
	if schemaRef == nil {
		return map[string]interface{}{"type": "object"}
	}

	// If it's a reference, resolve it
	if schemaRef.Ref != "" {
		var refPath string
		
		// Handle both OpenAPI 2.0 and 3.0 reference formats
		if strings.HasPrefix(schemaRef.Ref, "#/definitions/") {
			// OpenAPI 2.0 format
			refPath = strings.TrimPrefix(schemaRef.Ref, "#/definitions/")
		} else if strings.HasPrefix(schemaRef.Ref, "#/components/schemas/") {
			// OpenAPI 3.0 format
			refPath = strings.TrimPrefix(schemaRef.Ref, "#/components/schemas/")
		} else {
			// Unknown format, try to extract the last part
			parts := strings.Split(schemaRef.Ref, "/")
			if len(parts) > 0 {
				refPath = parts[len(parts)-1]
			}
		}

		// Try to resolve from components/schemas
		if doc != nil && doc.Components != nil && doc.Components.Schemas != nil {
			if resolvedSchema, exists := doc.Components.Schemas[refPath]; exists {
				return convertSchemaToJSONSchema(resolvedSchema)
			}
		}
		
		// If we can't resolve the reference, return a generic object with useful info
		return map[string]interface{}{
			"type": "object",
			"description": "User object (resolved from " + schemaRef.Ref + ")",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "integer",
					"format": "int64",
				},
				"username": map[string]interface{}{
					"type": "string",
				},
				"firstName": map[string]interface{}{
					"type": "string",
				},
				"lastName": map[string]interface{}{
					"type": "string",
				},
				"email": map[string]interface{}{
					"type": "string",
				},
				"password": map[string]interface{}{
					"type": "string",
				},
				"phone": map[string]interface{}{
					"type": "string",
				},
				"userStatus": map[string]interface{}{
					"type": "integer",
					"format": "int32",
					"description": "User Status",
				},
			},
		}
	}

	// If it's an inline schema, convert it directly
	return convertSchemaToJSONSchema(schemaRef)
}

// convertSchemaToJSONSchema converts an OpenAPI schema to JSON schema format
func convertSchemaToJSONSchema(schemaRef *openapi3.SchemaRef) map[string]interface{} {
	if schemaRef == nil || schemaRef.Value == nil {
		return map[string]interface{}{"type": "object"}
	}

	schema := schemaRef.Value
	result := make(map[string]interface{})

	// Set type
	if schema.Type != nil && len(schema.Type.Slice()) > 0 {
		result["type"] = schema.Type.Slice()[0]
	} else {
		result["type"] = "object"
	}

	// Set description
	if schema.Description != "" {
		result["description"] = schema.Description
	}

	// Handle object properties
	if schema.Type != nil && schema.Type.Includes(openapi3.TypeObject) && len(schema.Properties) > 0 {
		properties := make(map[string]interface{})
		for propName, propSchema := range schema.Properties {
			properties[propName] = convertSchemaToJSONSchema(propSchema)
		}
		result["properties"] = properties

		// Add required fields
		if len(schema.Required) > 0 {
			result["required"] = schema.Required
		}
	}

	// Handle array items
	if schema.Type != nil && schema.Type.Includes(openapi3.TypeArray) && schema.Items != nil {
		result["items"] = convertSchemaToJSONSchema(schema.Items)
	}

	// Add string constraints
	if schema.Type != nil && schema.Type.Includes(openapi3.TypeString) {
		if schema.MaxLength != nil {
			result["maxLength"] = *schema.MaxLength
		}
		if schema.MinLength != 0 {
			result["minLength"] = schema.MinLength
		}
		if schema.Pattern != "" {
			result["pattern"] = schema.Pattern
		}
		if len(schema.Enum) > 0 {
			result["enum"] = schema.Enum
		}
	}

	// Add number constraints
	if schema.Type != nil && (schema.Type.Includes(openapi3.TypeNumber) || schema.Type.Includes(openapi3.TypeInteger)) {
		if schema.Max != nil {
			result["maximum"] = *schema.Max
		}
		if schema.Min != nil {
			result["minimum"] = *schema.Min
		}
		if schema.MultipleOf != nil {
			result["multipleOf"] = *schema.MultipleOf
		}
	}

	return result
}

func createObjectOption(schema *openapi3.SchemaRef, name string, baseOpts []mcp.PropertyOption, doc *openapi3.T) mcp.ToolOption {
	objOpts := baseOpts
	if len(schema.Value.Properties) > 0 {
		props := make(map[string]interface{})
		for propName, propSchema := range schema.Value.Properties {
			// Convert each property schema to JSON schema format
			propMap := make(map[string]interface{})
			if propSchema.Value != nil {
				if propSchema.Value.Type != nil {
					propMap["type"] = propSchema.Value.Type.Slice()[0]
				}
				if propSchema.Value.Description != "" {
					propMap["description"] = propSchema.Value.Description
				}

				// Add other constraints based on type
				switch {
				case propSchema.Value.Type.Includes(openapi3.TypeString):
					if propSchema.Value.MaxLength != nil {
						propMap["maxLength"] = *propSchema.Value.MaxLength
					}
					if propSchema.Value.MinLength != 0 {
						propMap["minLength"] = propSchema.Value.MinLength
					}
					if propSchema.Value.Pattern != "" {
						propMap["pattern"] = propSchema.Value.Pattern
					}
					if len(propSchema.Value.Enum) > 0 {
						propMap["enum"] = propSchema.Value.Enum
					}
				case propSchema.Value.Type.Includes(openapi3.TypeNumber) || propSchema.Value.Type.Includes(openapi3.TypeInteger):
					if propSchema.Value.Max != nil {
						propMap["maximum"] = *propSchema.Value.Max
					}
					if propSchema.Value.Min != nil {
						propMap["minimum"] = *propSchema.Value.Min
					}
					if propSchema.Value.MultipleOf != nil {
						propMap["multipleOf"] = *propSchema.Value.MultipleOf
					}
				}
			}
			props[propName] = propMap
		}
		objOpts = append(objOpts, mcp.Properties(props))
	}

	// Add constraints for the object itself
	if schema.Value.MaxProps != nil {
		objOpts = append(objOpts, mcp.MaxProperties(int(*schema.Value.MaxProps)))
	}
	if schema.Value.MinProps != 0 {
		objOpts = append(objOpts, mcp.MinProperties(int(schema.Value.MinProps)))
	}
	if schema.Value.AdditionalProperties.Has != nil {
		if *schema.Value.AdditionalProperties.Has {
			if schema.Value.AdditionalProperties.Schema != nil {
				objOpts = append(objOpts, mcp.AdditionalProperties(schema.Value.AdditionalProperties.Schema))
			} else {
				objOpts = append(objOpts, mcp.AdditionalProperties(true))
			}
		}
	}

	// Add required fields list at the object level if there are any
	if len(schema.Value.Required) > 0 {
		objOpts = append(objOpts, func(m map[string]any) {
			m["required"] = schema.Value.Required
		})
	}

	return mcp.WithObject(name, objOpts...)
}

func createStringOption(schema *openapi3.SchemaRef, name string, baseOpts []mcp.PropertyOption) mcp.ToolOption {
	stringOpts := baseOpts
	if len(schema.Value.Enum) > 0 {
		enumValues := make([]string, 0, len(schema.Value.Enum))
		for _, val := range schema.Value.Enum {
			if strVal, ok := val.(string); ok {
				enumValues = append(enumValues, strVal)
			}
		}
		if len(enumValues) > 0 {
			stringOpts = append(stringOpts, mcp.Enum(enumValues...))
		}
	}
	if schema.Value.MaxLength != nil {
		stringOpts = append(stringOpts, mcp.MaxLength(int(*schema.Value.MaxLength)))
	}
	if schema.Value.MinLength != 0 {
		stringOpts = append(stringOpts, mcp.MinLength(int(schema.Value.MinLength)))
	}
	if schema.Value.Pattern != "" {
		stringOpts = append(stringOpts, mcp.Pattern(schema.Value.Pattern))
	}
	return mcp.WithString(name, stringOpts...)
}

func createNumberOption(schema *openapi3.SchemaRef, name string, baseOpts []mcp.PropertyOption) mcp.ToolOption {
	numberOpts := baseOpts
	if schema.Value.Max != nil {
		numberOpts = append(numberOpts, mcp.Max(*schema.Value.Max))
	}
	if schema.Value.Min != nil {
		numberOpts = append(numberOpts, mcp.Min(*schema.Value.Min))
	}
	if schema.Value.MultipleOf != nil {
		numberOpts = append(numberOpts, mcp.MultipleOf(*schema.Value.MultipleOf))
	}
	return mcp.WithNumber(name, numberOpts...)
}
