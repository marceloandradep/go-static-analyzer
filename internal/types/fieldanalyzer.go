package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// StructFieldAnalyzer analyzes struct fields to extract detailed type information
type StructFieldAnalyzer struct {
	Registry *TypeRegistry
	Verbose  bool
}

// NewStructFieldAnalyzer creates a new StructFieldAnalyzer
func NewStructFieldAnalyzer(registry *TypeRegistry, verbose bool) *StructFieldAnalyzer {
	return &StructFieldAnalyzer{
		Registry: registry,
		Verbose:  verbose,
	}
}

// AnalyzeStructFields analyzes all struct fields in the registry
func (a *StructFieldAnalyzer) AnalyzeStructFields() error {
	if a.Verbose {
		fmt.Println("Analyzing struct fields...")
	}

	// Iterate through all packages
	for pkgPath, pkgInfo := range a.Registry.Packages {
		// Set the current package
		a.Registry.SetCurrentPackage(pkgPath)

		// Analyze all struct types in the package
		for _, typeDef := range pkgInfo.Types {
			if typeDef.Kind == KindStruct {
				a.analyzeStructType(typeDef)
			}
		}
	}

	return nil
}

// analyzeStructType analyzes a struct type and its fields
func (a *StructFieldAnalyzer) analyzeStructType(typeDef *TypeDefinition) {
	if a.Verbose {
		fmt.Printf("Analyzing struct type: %s.%s\n", typeDef.Package, typeDef.Name)
	}

	// Skip already fully resolved types
	if typeDef.IsResolved {
		return
	}

	// Analyze each field
	for _, field := range typeDef.Fields {
		a.analyzeField(field, typeDef)
	}

	// Mark the type as resolved
	typeDef.IsResolved = true
}

// analyzeField analyzes a struct field
func (a *StructFieldAnalyzer) analyzeField(field *FieldDefinition, parentType *TypeDefinition) {
	if a.Verbose {
		fmt.Printf("  Analyzing field: %s\n", field.Name)
	}

	// Skip already resolved fields
	if field.Type != nil && field.Type.IsResolved {
		return
	}

	// If the field type is nil, try to resolve it
	if field.Type == nil {
		// This would require access to the AST node for the field
		// In a real implementation, we would need to store the AST node with the field
		// For now, we'll just set a placeholder type
		field.Type = &TypeDefinition{
			Name:       "unknown",
			Kind:       KindBasic,
			BasicType:  "unknown",
			Package:    parentType.Package,
			IsResolved: true,
		}
		return
	}

	// If the field type is a struct, analyze its fields recursively
	if field.Type.Kind == KindStruct {
		a.analyzeStructType(field.Type)
	}

	// If the field type is an array, analyze its element type
	if field.Type.Kind == KindArray && field.Type.ElementType != nil {
		if field.Type.ElementType.Kind == KindStruct {
			a.analyzeStructType(field.Type.ElementType)
		}
	}

	// If the field type is a map, analyze its value type
	if field.Type.Kind == KindMap && field.Type.ValueType != nil {
		if field.Type.ValueType.Kind == KindStruct {
			a.analyzeStructType(field.Type.ValueType)
		}
	}

	// If the field type is a pointer, analyze its element type
	if field.Type.Kind == KindPointer && field.Type.ElementType != nil {
		if field.Type.ElementType.Kind == KindStruct {
			a.analyzeStructType(field.Type.ElementType)
		}
	}
}

// EnhanceTypeWithComments enhances type definitions with comments from AST
func (a *StructFieldAnalyzer) EnhanceTypeWithComments(file *ast.File) {
	// Collect all comments in the file
	comments := make(map[token.Pos]*ast.CommentGroup)
	for _, cg := range file.Comments {
		comments[cg.Pos()] = cg
	}

	// Iterate through declarations
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		// Check for type declarations
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Get the type name
			typeName := typeSpec.Name.Name

			// Look up the type in the registry
			typeDef := a.Registry.LookupType(typeName)
			if typeDef == nil {
				continue
			}

			// Check if it's a struct type
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Process struct fields
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					// Skip fields without names
					if len(field.Names) == 0 {
						continue
					}

					fieldName := field.Names[0].Name

					// Find the field in the type definition
					for _, fieldDef := range typeDef.Fields {
						if fieldDef.Name == fieldName {
							// Add comment to field if available
							if field.Doc != nil {
								// Extract comment text
								comment := field.Doc.Text()
								// Clean up comment (remove // or /* */ markers)
								comment = strings.TrimSpace(comment)
								comment = strings.TrimPrefix(comment, "//")
								comment = strings.TrimPrefix(comment, "/*")
								comment = strings.TrimSuffix(comment, "*/")
								comment = strings.TrimSpace(comment)

								// Store comment in field type (we'll need to add a Description field)
								// For now, just log it
								if a.Verbose {
									fmt.Printf("  Field %s comment: %s\n", fieldName, comment)
								}
							}
							break
						}
					}
				}
			}
		}
	}
}

// ExtractJSONTags extracts JSON tags from struct fields in AST
func (a *StructFieldAnalyzer) ExtractJSONTags(file *ast.File) {
	// Iterate through declarations
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		// Check for type declarations
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Get the type name
			typeName := typeSpec.Name.Name

			// Look up the type in the registry
			typeDef := a.Registry.LookupType(typeName)
			if typeDef == nil {
				continue
			}

			// Check if it's a struct type
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Process struct fields
			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					// Skip fields without names
					if len(field.Names) == 0 {
						continue
					}

					fieldName := field.Names[0].Name

					// Skip fields without tags
					if field.Tag == nil {
						continue
					}

					// Extract JSON tag
					tagValue := field.Tag.Value
					// Remove the backticks
					tagValue = strings.Trim(tagValue, "`")

					// Extract the json tag
					jsonTag := ""
					for _, tag := range strings.Split(tagValue, " ") {
						if strings.HasPrefix(tag, "json:") {
							jsonTag = strings.Trim(strings.TrimPrefix(tag, "json:"), "\"")
							break
						}
					}

					if jsonTag == "" {
						continue
					}

					// Parse the JSON tag
					parts := strings.Split(jsonTag, ",")
					jsonName := parts[0]
					omitempty := false
					for _, part := range parts[1:] {
						if part == "omitempty" {
							omitempty = true
							break
						}
					}

					// Find the field in the type definition
					for _, fieldDef := range typeDef.Fields {
						if fieldDef.Name == fieldName {
							// Update JSON name and omitempty flag
							fieldDef.JSONName = jsonName
							fieldDef.Omitempty = omitempty

							if a.Verbose {
								fmt.Printf("  Field %s JSON tag: %s (omitempty: %v)\n", fieldName, jsonName, omitempty)
							}
							break
						}
					}
				}
			}
		}
	}
}

// AnalyzeNestedStructs analyzes nested struct types
func (a *StructFieldAnalyzer) AnalyzeNestedStructs() {
	if a.Verbose {
		fmt.Println("Analyzing nested struct ..")
	}

	// Iterate through all packages
	for pkgPath, pkgInfo := range a.Registry.Packages {
		// Set the current package
		a.Registry.SetCurrentPackage(pkgPath)

		// Analyze all struct types in the package
		for _, typeDef := range pkgInfo.Types {
			if typeDef.Kind == KindStruct {
				a.analyzeNestedStructs(typeDef, make(map[string]bool))
			}
		}
	}
}

// analyzeNestedStructs analyzes nested struct types recursively
func (a *StructFieldAnalyzer) analyzeNestedStructs(typeDef *TypeDefinition, visited map[string]bool) {
	// Avoid infinite recursion
	typeKey := fmt.Sprintf("%s.%s", typeDef.Package, typeDef.Name)
	if visited[typeKey] {
		return
	}
	visited[typeKey] = true

	if a.Verbose {
		fmt.Printf("Analyzing nested structs in: %s\n", typeKey)
	}

	// Analyze each field
	for _, field := range typeDef.Fields {
		if field.Type == nil {
			continue
		}

		// Handle different field types
		switch field.Type.Kind {
		case KindStruct:
			// Recursive analysis of nested struct
			a.analyzeNestedStructs(field.Type, visited)

		case KindArray:
			// Analyze array element type if it's a struct
			if field.Type.ElementType != nil && field.Type.ElementType.Kind == KindStruct {
				a.analyzeNestedStructs(field.Type.ElementType, visited)
			}

		case KindMap:
			// Analyze map value type if it's a struct
			if field.Type.ValueType != nil && field.Type.ValueType.Kind == KindStruct {
				a.analyzeNestedStructs(field.Type.ValueType, visited)
			}

		case KindPointer:
			// Analyze pointer element type if it's a struct
			if field.Type.ElementType != nil && field.Type.ElementType.Kind == KindStruct {
				a.analyzeNestedStructs(field.Type.ElementType, visited)
			}
		}
	}
}
