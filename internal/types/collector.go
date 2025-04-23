package types

import (
	"fmt"
	"go/ast"
	"path/filepath"
)

// TypeCollector scans the codebase to collect type definitions
type TypeCollector struct {
	Registry *TypeRegistry
	Verbose  bool
}

// NewTypeCollector creates a new TypeCollector
func NewTypeCollector(registry *TypeRegistry, verbose bool) *TypeCollector {
	return &TypeCollector{
		Registry: registry,
		Verbose:  verbose,
	}
}

// CollectTypes collects type definitions from all packages in the codebase
func (c *TypeCollector) CollectTypes(files []*ast.File, packagePath string) error {
	if c.Verbose {
		fmt.Printf("Collecting types from package: %s\n", packagePath)
	}

	// Set the current package in the registry
	c.Registry.SetCurrentPackage(packagePath)

	// First pass: collect imports
	for _, file := range files {
		c.collectImports(file)
	}

	// Second pass: collect type declarations
	for _, file := range files {
		c.collectTypeDeclarations(file)
	}

	return nil
}

// collectImports collects import statements from a file
func (c *TypeCollector) collectImports(file *ast.File) {
	for _, imp := range file.Imports {
		// Get the import path
		importPath := imp.Path.Value
		// Remove quotes
		importPath = importPath[1 : len(importPath)-1]

		// Get the import alias
		var alias string
		if imp.Name != nil {
			// Explicit alias
			alias = imp.Name.Name
		} else {
			// Default alias is the last part of the import path
			alias = filepath.Base(importPath)
		}

		// Register the import
		c.Registry.RegisterImport(alias, importPath)
	}
}

// collectTypeDeclarations collects type declarations from a file
func (c *TypeCollector) collectTypeDeclarations(file *ast.File) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Process the type declaration
			c.processTypeDeclaration(typeSpec)
		}
	}
}

// processTypeDeclaration processes a type declaration
func (c *TypeCollector) processTypeDeclaration(typeSpec *ast.TypeSpec) {
	typeName := typeSpec.Name.Name

	// Check if it's a struct type
	structType, isStruct := typeSpec.Type.(*ast.StructType)
	if isStruct {
		// Create a new type definition
		typeDef := &TypeDefinition{
			Name:       typeName,
			Kind:       KindStruct,
			Fields:     []*FieldDefinition{},
			Package:    c.Registry.CurrentPackage,
			IsResolved: false,
		}

		// Register the type (even though it's not fully resolved yet)
		c.Registry.RegisterType(typeDef)

		// Process struct fields
		if structType.Fields != nil {
			for _, field := range structType.Fields.List {
				// Process field names (there can be multiple names for the same type)
				for _, name := range field.Names {
					// Process JSON tags
					jsonName, omitempty := c.Registry.extractJSONTag(field)

					// Create a field definition with a placeholder type
					fieldDef := &FieldDefinition{
						Name:      name.Name,
						Type:      nil, // Will be resolved later
						JSONName:  jsonName,
						Omitempty: omitempty,
						IsPointer: isPointerType(field.Type),
					}

					typeDef.Fields = append(typeDef.Fields, fieldDef)
				}
			}
		}

		if c.Verbose {
			fmt.Printf("Collected struct type: %s with %d fields\n", typeName, len(typeDef.Fields))
		}
		return
	}

	// Check if it's an array type
	_, isArray := typeSpec.Type.(*ast.ArrayType)
	if isArray {
		// Create a new type definition
		typeDef := &TypeDefinition{
			Name:        typeName,
			Kind:        KindArray,
			ElementType: nil, // Will be resolved later
			Package:     c.Registry.CurrentPackage,
			IsResolved:  false,
		}

		// Register the type
		c.Registry.RegisterType(typeDef)

		if c.Verbose {
			fmt.Printf("Collected array type: %s\n", typeName)
		}
		return
	}

	// Check if it's a map type
	_, isMap := typeSpec.Type.(*ast.MapType)
	if isMap {
		// Create a new type definition
		typeDef := &TypeDefinition{
			Name:       typeName,
			Kind:       KindMap,
			KeyType:    nil, // Will be resolved later
			ValueType:  nil, // Will be resolved later
			Package:    c.Registry.CurrentPackage,
			IsResolved: false,
		}

		// Register the type
		c.Registry.RegisterType(typeDef)

		if c.Verbose {
			fmt.Printf("Collected map type: %s\n", typeName)
		}
		return
	}

	// For other types, just register a basic type
	typeDef := &TypeDefinition{
		Name:       typeName,
		Kind:       KindBasic,
		Package:    c.Registry.CurrentPackage,
		IsResolved: true,
	}

	// Register the type
	c.Registry.RegisterType(typeDef)

	if c.Verbose {
		fmt.Printf("Collected basic type: %s\n", typeName)
	}
}

// ResolveTypes resolves all collected types
func (c *TypeCollector) ResolveTypes() error {
	if c.Verbose {
		fmt.Println("Resolving types...")
	}

	// Iterate through all packages
	for pkgPath, pkgInfo := range c.Registry.Packages {
		// Set the current package
		c.Registry.SetCurrentPackage(pkgPath)

		// Resolve all types in the package
		for _, typeDef := range pkgInfo.Types {
			c.resolveType(typeDef)
		}
	}

	return nil
}

// resolveType resolves a type definition
func (c *TypeCollector) resolveType(typeDef *TypeDefinition) {
	if typeDef.IsResolved {
		return
	}

	switch typeDef.Kind {
	case KindStruct:
		// Resolve field types
		for _, field := range typeDef.Fields {
			// Skip already resolved fields
			if field.Type != nil && field.Type.IsResolved {
				continue
			}

			// TODO: This is a placeholder. In a real implementation,
			// we would need to look up the AST node for the field type
			// and resolve it using the Registry.ResolveType method.
			// For now, we'll just set a basic type.
			field.Type = &TypeDefinition{
				Name:       "string", // Placeholder
				Kind:       KindBasic,
				BasicType:  "string",
				Package:    typeDef.Package,
				IsResolved: true,
			}
		}

	case KindArray:
		// TODO: Resolve element type
		typeDef.ElementType = &TypeDefinition{
			Name:       "string", // Placeholder
			Kind:       KindBasic,
			BasicType:  "string",
			Package:    typeDef.Package,
			IsResolved: true,
		}

	case KindMap:
		// TODO: Resolve key and value types
		typeDef.KeyType = &TypeDefinition{
			Name:       "string", // Placeholder
			Kind:       KindBasic,
			BasicType:  "string",
			Package:    typeDef.Package,
			IsResolved: true,
		}
		typeDef.ValueType = &TypeDefinition{
			Name:       "string", // Placeholder
			Kind:       KindBasic,
			BasicType:  "string",
			Package:    typeDef.Package,
			IsResolved: true,
		}
	}

	typeDef.IsResolved = true
}
