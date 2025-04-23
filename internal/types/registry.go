package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// TypeKind represents the kind of a type
type TypeKind int

const (
	KindStruct TypeKind = iota
	KindArray
	KindMap
	KindBasic
	KindPointer
)

// TypeDefinition represents a Go type definition
type TypeDefinition struct {
	Name        string
	Kind        TypeKind
	Fields      []*FieldDefinition // For structs
	ElementType *TypeDefinition    // For arrays and pointers
	KeyType     *TypeDefinition    // For maps
	ValueType   *TypeDefinition    // For maps
	Package     string             // Package path
	BasicType   string             // For basic types (string, int, etc.)
	IsResolved  bool               // Whether the type has been fully resolved
}

// FieldDefinition represents a field in a struct
type FieldDefinition struct {
	Name      string
	Type      *TypeDefinition
	JSONName  string
	Omitempty bool
	IsPointer bool
}

// PackageInfo represents information about a package
type PackageInfo struct {
	// Map of type name to type definition
	Types map[string]*TypeDefinition

	// Map of import alias to package path
	Imports map[string]string
}

// TypeRegistry is a central repository for storing and retrieving type information
type TypeRegistry struct {
	// Map of package path to package info
	Packages map[string]*PackageInfo

	// Current package being analyzed
	CurrentPackage string

	// FileSet for position information
	FileSet *token.FileSet

	// Verbose mode
	Verbose bool
}

// NewTypeRegistry creates a new TypeRegistry
func NewTypeRegistry(fset *token.FileSet, verbose bool) *TypeRegistry {
	return &TypeRegistry{
		Packages:      make(map[string]*PackageInfo),
		CurrentPackage: "",
		FileSet:       fset,
		Verbose:       verbose,
	}
}

// RegisterPackage registers a package with the registry
func (r *TypeRegistry) RegisterPackage(packagePath string) *PackageInfo {
	if _, exists := r.Packages[packagePath]; !exists {
		r.Packages[packagePath] = &PackageInfo{
			Types:   make(map[string]*TypeDefinition),
			Imports: make(map[string]string),
		}
		if r.Verbose {
			fmt.Printf("Registered package: %s\n", packagePath)
		}
	}
	return r.Packages[packagePath]
}

// SetCurrentPackage sets the current package being analyzed
func (r *TypeRegistry) SetCurrentPackage(packagePath string) {
	r.CurrentPackage = packagePath
	r.RegisterPackage(packagePath)
}

// RegisterImport registers an import with the current package
func (r *TypeRegistry) RegisterImport(alias, packagePath string) {
	pkg := r.RegisterPackage(r.CurrentPackage)
	pkg.Imports[alias] = packagePath
	if r.Verbose {
		fmt.Printf("Registered import: %s -> %s in package %s\n", alias, packagePath, r.CurrentPackage)
	}
}

// RegisterType registers a type with the current package
func (r *TypeRegistry) RegisterType(typeDef *TypeDefinition) {
	pkg := r.RegisterPackage(r.CurrentPackage)
	pkg.Types[typeDef.Name] = typeDef
	if r.Verbose {
		fmt.Printf("Registered type: %s in package %s\n", typeDef.Name, r.CurrentPackage)
	}
}

// LookupType looks up a type by name in the current package
func (r *TypeRegistry) LookupType(name string) *TypeDefinition {
	// Check if it's a qualified name (pkg.Type)
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)
		pkgAlias := parts[0]
		typeName := parts[1]

		// Look up the package path from the import alias
		pkg := r.RegisterPackage(r.CurrentPackage)
		if pkgPath, exists := pkg.Imports[pkgAlias]; exists {
			// Look up the type in the imported package
			if importedPkg, exists := r.Packages[pkgPath]; exists {
				if typeDef, exists := importedPkg.Types[typeName]; exists {
					return typeDef
				}
			}
		}
		return nil
	}

	// Look up in the current package
	pkg := r.RegisterPackage(r.CurrentPackage)
	if typeDef, exists := pkg.Types[name]; exists {
		return typeDef
	}

	return nil
}

// ResolveType resolves a type expression to a TypeDefinition
func (r *TypeRegistry) ResolveType(expr ast.Expr) *TypeDefinition {
	if expr == nil {
		return nil
	}

	switch t := expr.(type) {
	case *ast.Ident:
		// Basic type or type defined in the current package
		if isBasicType(t.Name) {
			return &TypeDefinition{
				Name:       t.Name,
				Kind:       KindBasic,
				BasicType:  t.Name,
				Package:    r.CurrentPackage,
				IsResolved: true,
			}
		}
		return r.LookupType(t.Name)

	case *ast.SelectorExpr:
		// Type from another package (pkg.Type)
		if x, ok := t.X.(*ast.Ident); ok {
			qualifiedName := x.Name + "." + t.Sel.Name
			return r.LookupType(qualifiedName)
		}

	case *ast.ArrayType:
		// Array type ([]Type)
		elemType := r.ResolveType(t.Elt)
		if elemType != nil {
			return &TypeDefinition{
				Name:        fmt.Sprintf("[]%s", elemType.Name),
				Kind:        KindArray,
				ElementType: elemType,
				Package:     r.CurrentPackage,
				IsResolved:  elemType.IsResolved,
			}
		}

	case *ast.MapType:
		// Map type (map[KeyType]ValueType)
		keyType := r.ResolveType(t.Key)
		valueType := r.ResolveType(t.Value)
		if keyType != nil && valueType != nil {
			return &TypeDefinition{
				Name:       fmt.Sprintf("map[%s]%s", keyType.Name, valueType.Name),
				Kind:       KindMap,
				KeyType:    keyType,
				ValueType:  valueType,
				Package:    r.CurrentPackage,
				IsResolved: keyType.IsResolved && valueType.IsResolved,
			}
		}

	case *ast.StarExpr:
		// Pointer type (*Type)
		elemType := r.ResolveType(t.X)
		if elemType != nil {
			return &TypeDefinition{
				Name:        fmt.Sprintf("*%s", elemType.Name),
				Kind:        KindPointer,
				ElementType: elemType,
				Package:     r.CurrentPackage,
				IsResolved:  elemType.IsResolved,
			}
		}

	case *ast.StructType:
		// Anonymous struct type
		structDef := &TypeDefinition{
			Name:       "anonymous",
			Kind:       KindStruct,
			Fields:     []*FieldDefinition{},
			Package:    r.CurrentPackage,
			IsResolved: true,
		}

		// Process struct fields
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldType := r.ResolveType(field.Type)
				if fieldType == nil {
					structDef.IsResolved = false
					continue
				}

				// Process field names (there can be multiple names for the same type)
				for _, name := range field.Names {
					// Process JSON tags
					jsonName, omitempty := r.extractJSONTag(field)

					fieldDef := &FieldDefinition{
						Name:      name.Name,
						Type:      fieldType,
						JSONName:  jsonName,
						Omitempty: omitempty,
						IsPointer: isPointerType(field.Type),
					}

					structDef.Fields = append(structDef.Fields, fieldDef)
				}
			}
		}

		return structDef
	}

	return nil
}

// extractJSONTag extracts the JSON tag from a struct field
func (r *TypeRegistry) extractJSONTag(field *ast.Field) (string, bool) {
	if field.Tag == nil {
		return "", false
	}

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
		return "", false
	}

	// Check for omitempty
	parts := strings.Split(jsonTag, ",")
	jsonName := parts[0]
	omitempty := false
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitempty = true
			break
		}
	}

	// If the JSON name is "-", the field is not exported to JSON
	if jsonName == "-" {
		return "", true
	}

	return jsonName, omitempty
}

// isBasicType checks if a type name is a basic Go type
func isBasicType(name string) bool {
	basicTypes := map[string]bool{
		"bool":       true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"float32":    true,
		"float64":    true,
		"complex64":  true,
		"complex128": true,
		"string":     true,
		"byte":       true,
		"rune":       true,
		"error":      true,
	}
	return basicTypes[name]
}

// isPointerType checks if a type is a pointer
func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}
