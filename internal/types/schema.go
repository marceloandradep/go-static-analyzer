package types

import (
	"encoding/json"
	"fmt"
)

// JSONSchemaType represents a JSON Schema type
type JSONSchemaType string

const (
	JSONSchemaTypeObject  JSONSchemaType = "object"
	JSONSchemaTypeArray   JSONSchemaType = "array"
	JSONSchemaTypeString  JSONSchemaType = "string"
	JSONSchemaTypeNumber  JSONSchemaType = "number"
	JSONSchemaTypeInteger JSONSchemaType = "integer"
	JSONSchemaTypeBoolean JSONSchemaType = "boolean"
	JSONSchemaTypeNull    JSONSchemaType = "null"
)

// JSONSchemaFormat represents a JSON Schema format
type JSONSchemaFormat string

const (
	JSONSchemaFormatDateTime JSONSchemaFormat = "date-time"
	JSONSchemaFormatEmail    JSONSchemaFormat = "email"
	JSONSchemaFormatURI      JSONSchemaFormat = "uri"
)

// JSONSchemaProperty represents a property in a JSON Schema
type JSONSchemaProperty struct {
	Type                 JSONSchemaType                 `json:"type,omitempty"`
	Format               JSONSchemaFormat               `json:"format,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Items                *JSONSchema                    `json:"items,omitempty"`
	Properties           map[string]*JSONSchemaProperty `json:"properties,omitempty"`
	Required             []string                       `json:"required,omitempty"`
	Ref                  string                         `json:"$ref,omitempty"`
	AdditionalProperties *JSONSchemaProperty            `json:"additionalProperties,omitempty"`
}

// JSONSchema represents a JSON Schema
type JSONSchema struct {
	Type                 JSONSchemaType                 `json:"type,omitempty"`
	Format               JSONSchemaFormat               `json:"format,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Items                *JSONSchema                    `json:"items,omitempty"`
	Properties           map[string]*JSONSchemaProperty `json:"properties,omitempty"`
	Required             []string                       `json:"required,omitempty"`
	AdditionalProperties *JSONSchemaProperty            `json:"additionalProperties,omitempty"`
}

// SchemaGenerator generates JSON Schema from Go type definitions
type SchemaGenerator struct {
	Registry *TypeRegistry
	Schemas  map[string]*JSONSchema
	Verbose  bool
}

// NewSchemaGenerator creates a new SchemaGenerator
func NewSchemaGenerator(registry *TypeRegistry, verbose bool) *SchemaGenerator {
	return &SchemaGenerator{
		Registry: registry,
		Schemas:  make(map[string]*JSONSchema),
		Verbose:  verbose,
	}
}

// GenerateSchema generates a JSON Schema for a type definition
func (g *SchemaGenerator) GenerateSchema(typeDef *TypeDefinition) *JSONSchema {
	if typeDef == nil {
		return nil
	}

	// Check if we've already generated a schema for this type
	schemaKey := fmt.Sprintf("%s.%s", typeDef.Package, typeDef.Name)
	if schema, exists := g.Schemas[schemaKey]; exists {
		return schema
	}

	// Create a new schema based on the type kind
	var schema *JSONSchema
	switch typeDef.Kind {
	case KindStruct:
		schema = g.generateStructSchema(typeDef)
	case KindArray:
		schema = g.generateArraySchema(typeDef)
	case KindMap:
		schema = g.generateMapSchema(typeDef)
	case KindBasic:
		schema = g.generateBasicSchema(typeDef)
	case KindPointer:
		// For pointers, generate schema for the element type
		if typeDef.ElementType != nil {
			schema = g.GenerateSchema(typeDef.ElementType)
		}
	}

	// Store the schema for future reference
	if schema != nil {
		g.Schemas[schemaKey] = schema
	}

	return schema
}

// generateStructSchema generates a JSON Schema for a struct type
func (g *SchemaGenerator) generateStructSchema(typeDef *TypeDefinition) *JSONSchema {
	schema := &JSONSchema{
		Type:       JSONSchemaTypeObject,
		Properties: make(map[string]*JSONSchemaProperty),
		Required:   []string{},
	}

	// Process struct fields
	for _, field := range typeDef.Fields {
		// Skip fields without a type
		if field.Type == nil {
			continue
		}

		// Determine JSON field name
		jsonName := field.Name
		if field.JSONName != "" {
			jsonName = field.JSONName
		}

		// Generate schema for the field type
		fieldSchema := g.GenerateSchema(field.Type)
		if fieldSchema == nil {
			continue
		}

		// Create property from field schema
		property := &JSONSchemaProperty{
			Type:                 fieldSchema.Type,
			Format:               fieldSchema.Format,
			Description:          fieldSchema.Description,
			Items:                fieldSchema.Items,
			Properties:           fieldSchema.Properties,
			Required:             fieldSchema.Required,
			AdditionalProperties: fieldSchema.AdditionalProperties,
		}

		// Add property to schema
		schema.Properties[jsonName] = property

		// Add to required fields if not omitempty
		if !field.Omitempty {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema
}

// generateArraySchema generates a JSON Schema for an array type
func (g *SchemaGenerator) generateArraySchema(typeDef *TypeDefinition) *JSONSchema {
	schema := &JSONSchema{
		Type: JSONSchemaTypeArray,
	}

	// Generate schema for the element type
	if typeDef.ElementType != nil {
		elemSchema := g.GenerateSchema(typeDef.ElementType)
		if elemSchema != nil {
			schema.Items = elemSchema
		}
	}

	return schema
}

// generateMapSchema generates a JSON Schema for a map type
func (g *SchemaGenerator) generateMapSchema(typeDef *TypeDefinition) *JSONSchema {
	schema := &JSONSchema{
		Type: JSONSchemaTypeObject,
	}

	// Generate schema for the value type
	if typeDef.ValueType != nil {
		valueSchema := g.GenerateSchema(typeDef.ValueType)
		if valueSchema != nil {
			schema.AdditionalProperties = &JSONSchemaProperty{
				Type:                 valueSchema.Type,
				Format:               valueSchema.Format,
				Description:          valueSchema.Description,
				Items:                valueSchema.Items,
				Properties:           valueSchema.Properties,
				Required:             valueSchema.Required,
				AdditionalProperties: valueSchema.AdditionalProperties,
			}
		}
	}

	return schema
}

// generateBasicSchema generates a JSON Schema for a basic type
func (g *SchemaGenerator) generateBasicSchema(typeDef *TypeDefinition) *JSONSchema {
	schema := &JSONSchema{}

	// Map Go basic types to JSON Schema types
	switch typeDef.BasicType {
	case "string":
		schema.Type = JSONSchemaTypeString
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		schema.Type = JSONSchemaTypeInteger
	case "float32", "float64":
		schema.Type = JSONSchemaTypeNumber
	case "bool":
		schema.Type = JSONSchemaTypeBoolean
	case "time.Time":
		schema.Type = JSONSchemaTypeString
		schema.Format = JSONSchemaFormatDateTime
	default:
		// Default to string for unknown types
		schema.Type = JSONSchemaTypeString
	}

	return schema
}

// GenerateSchemaString generates a JSON Schema string for a type definition
func (g *SchemaGenerator) GenerateSchemaString(typeDef *TypeDefinition) (string, error) {
	schema := g.GenerateSchema(typeDef)
	if schema == nil {
		return "", fmt.Errorf("failed to generate schema for type %s", typeDef.Name)
	}

	// Convert schema to JSON
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}

	return string(schemaBytes), nil
}

// GenerateExampleJSON generates an example JSON string for a type definition
func (g *SchemaGenerator) GenerateExampleJSON(typeDef *TypeDefinition) (string, error) {
	example := g.generateExample(typeDef)
	if example == nil {
		return "", fmt.Errorf("failed to generate example for type %s", typeDef.Name)
	}

	// Convert example to JSON
	exampleBytes, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return "", err
	}

	return string(exampleBytes), nil
}

// generateExample generates an example value for a type definition
func (g *SchemaGenerator) generateExample(typeDef *TypeDefinition) interface{} {
	if typeDef == nil {
		return nil
	}

	switch typeDef.Kind {
	case KindStruct:
		return g.generateStructExample(typeDef)
	case KindArray:
		return g.generateArrayExample(typeDef)
	case KindMap:
		return g.generateMapExample(typeDef)
	case KindBasic:
		return g.generateBasicExample(typeDef)
	case KindPointer:
		// For pointers, generate example for the element type
		if typeDef.ElementType != nil {
			return g.generateExample(typeDef.ElementType)
		}
	}

	return nil
}

// generateStructExample generates an example for a struct type
func (g *SchemaGenerator) generateStructExample(typeDef *TypeDefinition) interface{} {
	example := make(map[string]interface{})

	// Generate example for each field
	for _, field := range typeDef.Fields {
		// Skip fields without a type
		if field.Type == nil {
			continue
		}

		// Determine JSON field name
		jsonName := field.Name
		if field.JSONName != "" {
			jsonName = field.JSONName
		}

		// Skip omitempty fields for simplicity
		if field.Omitempty {
			continue
		}

		// Generate example for the field
		fieldExample := g.generateExample(field.Type)
		if fieldExample != nil {
			example[jsonName] = fieldExample
		}
	}

	return example
}

// generateArrayExample generates an example for an array type
func (g *SchemaGenerator) generateArrayExample(typeDef *TypeDefinition) interface{} {
	// Generate a single example element
	if typeDef.ElementType != nil {
		elemExample := g.generateExample(typeDef.ElementType)
		if elemExample != nil {
			return []interface{}{elemExample}
		}
	}

	return []interface{}{}
}

// generateMapExample generates an example for a map type
func (g *SchemaGenerator) generateMapExample(typeDef *TypeDefinition) interface{} {
	example := make(map[string]interface{})

	// Generate a single example value
	if typeDef.ValueType != nil {
		valueExample := g.generateExample(typeDef.ValueType)
		if valueExample != nil {
			example["key"] = valueExample
		}
	}

	return example
}

// generateBasicExample generates an example for a basic type
func (g *SchemaGenerator) generateBasicExample(typeDef *TypeDefinition) interface{} {
	// Generate example based on the basic type
	switch typeDef.BasicType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return 0
	case "float32", "float64":
		return 0.0
	case "bool":
		return false
	case "time.Time":
		return "2025-04-23T01:27:02Z"
	default:
		return "unknown"
	}
}
