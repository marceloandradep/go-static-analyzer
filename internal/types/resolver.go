package types

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// PackageResolver handles cross-package type resolution
type PackageResolver struct {
	Registry       *TypeRegistry
	RootPath       string
	ParsedPackages map[string]bool
	Verbose        bool
}

// NewPackageResolver creates a new PackageResolver
func NewPackageResolver(registry *TypeRegistry, rootPath string, verbose bool) *PackageResolver {
	return &PackageResolver{
		Registry:       registry,
		RootPath:       rootPath,
		ParsedPackages: make(map[string]bool),
		Verbose:        verbose,
	}
}

// ResolvePackages resolves types across packages
func (r *PackageResolver) ResolvePackages() error {
	if r.Verbose {
		fmt.Println("Resolving types across packages...")
	}

	// First, build a dependency graph of packages
	dependencies := r.buildPackageDependencies()

	// Then, resolve packages in dependency order
	resolved := make(map[string]bool)
	for pkgPath := range r.Registry.Packages {
		r.resolvePackageDependencies(pkgPath, dependencies, resolved)
	}

	return nil
}

// buildPackageDependencies builds a dependency graph of packages
func (r *PackageResolver) buildPackageDependencies() map[string][]string {
	dependencies := make(map[string][]string)

	// Iterate through all packages
	for pkgPath, pkgInfo := range r.Registry.Packages {
		deps := []string{}

		// Add dependencies from imports
		for _, importPath := range pkgInfo.Imports {
			// Skip standard library imports
			if !strings.Contains(importPath, ".") {
				continue
			}
			deps = append(deps, importPath)
		}

		dependencies[pkgPath] = deps
	}

	return dependencies
}

// resolvePackageDependencies resolves package dependencies recursively
func (r *PackageResolver) resolvePackageDependencies(pkgPath string, dependencies map[string][]string, resolved map[string]bool) {
	// Skip already resolved packages
	if resolved[pkgPath] {
		return
	}

	// Resolve dependencies first
	for _, dep := range dependencies[pkgPath] {
		r.resolvePackageDependencies(dep, dependencies, resolved)
	}

	// Resolve types in this package
	r.resolvePackageTypes(pkgPath)

	// Mark as resolved
	resolved[pkgPath] = true
}

// resolvePackageTypes resolves types in a package
func (r *PackageResolver) resolvePackageTypes(pkgPath string) {
	if r.Verbose {
		fmt.Printf("Resolving types in package: %s\n", pkgPath)
	}

	// Set the current package
	r.Registry.SetCurrentPackage(pkgPath)

	// Get package info
	pkgInfo, exists := r.Registry.Packages[pkgPath]
	if !exists {
		return
	}

	// Resolve each type
	for _, typeDef := range pkgInfo.Types {
		r.resolveType(typeDef)
	}
}

// resolveType resolves a type definition
func (r *PackageResolver) resolveType(typeDef *TypeDefinition) {
	// Skip already resolved types
	if typeDef.IsResolved {
		return
	}

	if r.Verbose {
		fmt.Printf("  Resolving type: %s\n", typeDef.Name)
	}

	switch typeDef.Kind {
	case KindStruct:
		// Resolve field types
		for _, field := range typeDef.Fields {
			if field.Type == nil {
				continue
			}
			r.resolveType(field.Type)
		}

	case KindArray:
		// Resolve element type
		if typeDef.ElementType != nil {
			r.resolveType(typeDef.ElementType)
		}

	case KindMap:
		// Resolve key and value types
		if typeDef.KeyType != nil {
			r.resolveType(typeDef.KeyType)
		}
		if typeDef.ValueType != nil {
			r.resolveType(typeDef.ValueType)
		}

	case KindPointer:
		// Resolve element type
		if typeDef.ElementType != nil {
			r.resolveType(typeDef.ElementType)
		}
	}

	typeDef.IsResolved = true
}

// ScanPackage scans a package for types
func (r *PackageResolver) ScanPackage(packagePath string) error {
	// Skip already parsed packages
	if r.ParsedPackages[packagePath] {
		return nil
	}

	// Mark as parsed
	r.ParsedPackages[packagePath] = true

	if r.Verbose {
		fmt.Printf("Scanning package: %s\n", packagePath)
	}

	// Convert package path to directory path
	dirPath := filepath.Join(r.RootPath, packagePath)

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return fmt.Errorf("package directory does not exist: %s", dirPath)
	}

	// Create a new file set
	fset := token.NewFileSet()

	// Parse package
	pkgs, err := parser.ParseDir(fset, dirPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing package: %v", err)
	}

	// Process each package
	for pkgName, pkg := range pkgs {
		// Skip test packages
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		// Set the current package
		r.Registry.SetCurrentPackage(packagePath)

		// Collect imports
		for _, file := range pkg.Files {
			r.collectImports(file, packagePath)
		}

		// Collect types
		for _, file := range pkg.Files {
			r.collectTypes(file, packagePath)
		}
	}

	return nil
}

// collectImports collects imports from a file
func (r *PackageResolver) collectImports(file *ast.File, packagePath string) {
	for _, imp := range file.Imports {
		// Get the import path
		importPath := imp.Path.Value
		// Remove quotes
		importPath = importPath[1 : len(importPath)-1]

		// Skip standard library imports
		if !strings.Contains(importPath, ".") {
			continue
		}

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
		r.Registry.RegisterImport(alias, importPath)

		// Scan the imported package
		r.ScanPackage(importPath)
	}
}

// collectTypes collects type declarations from a file
func (r *PackageResolver) collectTypes(file *ast.File, packagePath string) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Process the type declaration
			r.processTypeDeclaration(typeSpec, packagePath)
		}
	}
}

// processTypeDeclaration processes a type declaration
func (r *PackageResolver) processTypeDeclaration(typeSpec *ast.TypeSpec, packagePath string) {
	typeName := typeSpec.Name.Name

	// Check if it's a struct type
	structType, isStruct := typeSpec.Type.(*ast.StructType)
	if isStruct {
		// Create a new type definition
		typeDef := &TypeDefinition{
			Name:       typeName,
			Kind:       KindStruct,
			Fields:     []*FieldDefinition{},
			Package:    packagePath,
			IsResolved: false,
		}

		// Register the type
		r.Registry.RegisterType(typeDef)

		// Process struct fields
		if structType.Fields != nil {
			for _, field := range structType.Fields.List {
				// Process field names (there can be multiple names for the same type)
				for _, name := range field.Names {
					// Process JSON tags
					jsonName, omitempty := r.Registry.extractJSONTag(field)

					// Create a field definition
					fieldDef := &FieldDefinition{
						Name:      name.Name,
						Type:      r.Registry.ResolveType(field.Type),
						JSONName:  jsonName,
						Omitempty: omitempty,
						IsPointer: isPointerType(field.Type),
					}

					typeDef.Fields = append(typeDef.Fields, fieldDef)
				}
			}
		}

		if r.Verbose {
			fmt.Printf("  Collected struct type: %s with %d fields\n", typeName, len(typeDef.Fields))
		}
		return
	}

	// Handle other types similarly to the TypeCollector
	// ...
}

// ResolveImportedTypes resolves types imported from other packages
func (r *PackageResolver) ResolveImportedTypes() error {
	if r.Verbose {
		fmt.Println("Resolving imported types...")
	}

	// Iterate through all packages
	for pkgPath, pkgInfo := range r.Registry.Packages {
		// Set the current package
		r.Registry.SetCurrentPackage(pkgPath)

		// Resolve imported types in this package
		for typeName, typeDef := range pkgInfo.Types {
			if !typeDef.IsResolved {
				r.resolveImportedType(typeDef, pkgPath, typeName)
			}
		}
	}

	return nil
}

// resolveImportedType resolves a type that might be imported from another package
func (r *PackageResolver) resolveImportedType(typeDef *TypeDefinition, pkgPath, typeName string) {
	if r.Verbose {
		fmt.Printf("  Resolving imported type: %s.%s\n", pkgPath, typeName)
	}

	// Skip already resolved types
	if typeDef.IsResolved {
		return
	}

	// Get package info
	pkgInfo, exists := r.Registry.Packages[pkgPath]
	if !exists {
		return
	}

	// Handle different type kinds
	switch typeDef.Kind {
	case KindStruct:
		// Resolve field types
		for _, field := range typeDef.Fields {
			if field.Type == nil {
				// Try to resolve the field type from imports
				fieldType := r.findImportedType(field.Name, pkgInfo.Imports)
				if fieldType != nil {
					field.Type = fieldType
				}
			} else if !field.Type.IsResolved {
				// Recursively resolve the field type
				r.resolveImportedType(field.Type, field.Type.Package, field.Type.Name)
			}
		}

	case KindArray:
		// Resolve element type
		if typeDef.ElementType != nil && !typeDef.ElementType.IsResolved {
			r.resolveImportedType(typeDef.ElementType, typeDef.ElementType.Package, typeDef.ElementType.Name)
		}

	case KindMap:
		// Resolve key and value types
		if typeDef.KeyType != nil && !typeDef.KeyType.IsResolved {
			r.resolveImportedType(typeDef.KeyType, typeDef.KeyType.Package, typeDef.KeyType.Name)
		}
		if typeDef.ValueType != nil && !typeDef.ValueType.IsResolved {
			r.resolveImportedType(typeDef.ValueType, typeDef.ValueType.Package, typeDef.ValueType.Name)
		}

	case KindPointer:
		// Resolve element type
		if typeDef.ElementType != nil && !typeDef.ElementType.IsResolved {
			r.resolveImportedType(typeDef.ElementType, typeDef.ElementType.Package, typeDef.ElementType.Name)
		}
	}

	typeDef.IsResolved = true
}

// findImportedType finds a type imported from another package
func (r *PackageResolver) findImportedType(typeName string, imports map[string]string) *TypeDefinition {
	// Check if it's a qualified name (pkg.Type)
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		pkgAlias := parts[0]
		typeName := parts[1]

		// Look up the package path from the import alias
		if pkgPath, exists := imports[pkgAlias]; exists {
			// Look up the type in the imported package
			if importedPkg, exists := r.Registry.Packages[pkgPath]; exists {
				if typeDef, exists := importedPkg.Types[typeName]; exists {
					return typeDef
				}
			}
		}
	}

	return nil
}
