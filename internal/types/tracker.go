package types

import (
	"fmt"
	"go/ast"
	"go/token"
)

// VariableInfo represents information about a variable
type VariableInfo struct {
	Name      string
	Type      *TypeDefinition
	IsPointer bool
	Position  token.Position
}

// VariableTracker tracks variable declarations and assignments in functions
type VariableTracker struct {
	Registry    *TypeRegistry
	Variables   map[string]*VariableInfo
	FunctionMap map[string]*TypeDefinition // Maps function names to their return types
	Verbose     bool
}

// NewVariableTracker creates a new VariableTracker
func NewVariableTracker(registry *TypeRegistry, verbose bool) *VariableTracker {
	return &VariableTracker{
		Registry:    registry,
		Variables:   make(map[string]*VariableInfo),
		FunctionMap: make(map[string]*TypeDefinition),
		Verbose:     verbose,
	}
}

// TrackFunction tracks variables in a function
func (t *VariableTracker) TrackFunction(funcDecl *ast.FuncDecl) error {
	if t.Verbose {
		fmt.Printf("Tracking variables in function: %s\n", funcDecl.Name.Name)
	}

	// Clear previous variables
	t.Variables = make(map[string]*VariableInfo)

	// Track function parameters
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := t.Registry.ResolveType(param.Type)
			if paramType == nil {
				continue
			}

			for _, name := range param.Names {
				varInfo := &VariableInfo{
					Name:      name.Name,
					Type:      paramType,
					IsPointer: isPointerType(param.Type),
					Position:  t.Registry.FileSet.Position(name.Pos()),
				}
				t.Variables[name.Name] = varInfo

				if t.Verbose {
					fmt.Printf("  Tracked parameter: %s of type %s\n", name.Name, paramType.Name)
				}
			}
		}
	}

	// Track variables in the function body
	if funcDecl.Body != nil {
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				t.trackAssignment(node)
			case *ast.DeclStmt:
				t.trackDeclaration(node)
			}
			return true
		})
	}

	return nil
}

// trackAssignment tracks variable assignments
func (t *VariableTracker) trackAssignment(stmt *ast.AssignStmt) {
	// Only track := and = assignments
	if stmt.Tok != token.DEFINE && stmt.Tok != token.ASSIGN {
		return
	}

	// Track each variable on the left side
	for i, lhs := range stmt.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			// Get the type from the right side
			var rhsType *TypeDefinition
			if i < len(stmt.Rhs) {
				rhsType = t.resolveExpressionType(stmt.Rhs[i])
			} else if len(stmt.Rhs) == 1 {
				// Multiple assignment from a single value (e.g., a, b := returnsTwoValues())
				rhsType = t.resolveExpressionType(stmt.Rhs[0])
				// TODO: Handle multiple return values properly
			}

			if rhsType == nil {
				continue
			}

			// Create or update variable info
			varInfo := &VariableInfo{
				Name:      ident.Name,
				Type:      rhsType,
				IsPointer: isPointerType(stmt.Rhs[i]),
				Position:  t.Registry.FileSet.Position(ident.Pos()),
			}
			t.Variables[ident.Name] = varInfo

			if t.Verbose {
				fmt.Printf("  Tracked assignment: %s = %s\n", ident.Name, rhsType.Name)
			}
		}
	}
}

// trackDeclaration tracks variable declarations
func (t *VariableTracker) trackDeclaration(stmt *ast.DeclStmt) {
	genDecl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}

	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// Get the type from the value spec
		var varType *TypeDefinition
		if valueSpec.Type != nil {
			varType = t.Registry.ResolveType(valueSpec.Type)
		} else if len(valueSpec.Values) > 0 {
			// Infer type from the first value
			varType = t.resolveExpressionType(valueSpec.Values[0])
		}

		if varType == nil {
			continue
		}

		// Track each variable
		for _, name := range valueSpec.Names {
			varInfo := &VariableInfo{
				Name:      name.Name,
				Type:      varType,
				IsPointer: isPointerType(valueSpec.Type),
				Position:  t.Registry.FileSet.Position(name.Pos()),
			}
			t.Variables[name.Name] = varInfo

			if t.Verbose {
				fmt.Printf("  Tracked declaration: %s of type %s\n", name.Name, varType.Name)
			}
		}
	}
}

// resolveExpressionType resolves the type of an expression
func (t *VariableTracker) resolveExpressionType(expr ast.Expr) *TypeDefinition {
	switch e := expr.(type) {
	case *ast.Ident:
		// Variable reference
		if varInfo, exists := t.Variables[e.Name]; exists {
			return varInfo.Type
		}
		// It might be a type name
		return t.Registry.LookupType(e.Name)

	case *ast.SelectorExpr:
		// Field access (e.g., user.Name) or package qualified name (e.g., models.User)
		if x, ok := e.X.(*ast.Ident); ok {
			// Check if it's a package qualified name
			qualifiedName := x.Name + "." + e.Sel.Name
			if typeDef := t.Registry.LookupType(qualifiedName); typeDef != nil {
				return typeDef
			}

			// Check if it's a field access
			if varInfo, exists := t.Variables[x.Name]; exists && varInfo.Type.Kind == KindStruct {
				// Find the field in the struct
				for _, field := range varInfo.Type.Fields {
					if field.Name == e.Sel.Name {
						return field.Type
					}
				}
			}
		}

	case *ast.CallExpr:
		// Function call
		return t.resolveFunctionCallType(e)

	case *ast.UnaryExpr:
		// Unary expression (e.g., &user)
		if e.Op == token.AND {
			// Address-of operator
			innerType := t.resolveExpressionType(e.X)
			if innerType != nil {
				return &TypeDefinition{
					Name:        "*" + innerType.Name,
					Kind:        KindPointer,
					ElementType: innerType,
					Package:     innerType.Package,
					IsResolved:  innerType.IsResolved,
				}
			}
		}

	case *ast.CompositeLit:
		// Composite literal (e.g., User{Name: "John"})
		return t.Registry.ResolveType(e.Type)

	case *ast.BasicLit:
		// Basic literal (e.g., "string", 123)
		switch e.Kind {
		case token.STRING:
			return &TypeDefinition{
				Name:       "string",
				Kind:       KindBasic,
				BasicType:  "string",
				Package:    "",
				IsResolved: true,
			}
		case token.INT:
			return &TypeDefinition{
				Name:       "int",
				Kind:       KindBasic,
				BasicType:  "int",
				Package:    "",
				IsResolved: true,
			}
		case token.FLOAT:
			return &TypeDefinition{
				Name:       "float64",
				Kind:       KindBasic,
				BasicType:  "float64",
				Package:    "",
				IsResolved: true,
			}
		case token.CHAR:
			return &TypeDefinition{
				Name:       "rune",
				Kind:       KindBasic,
				BasicType:  "rune",
				Package:    "",
				IsResolved: true,
			}
		}
	}

	return nil
}

// resolveFunctionCallType resolves the return type of a function call
func (t *VariableTracker) resolveFunctionCallType(call *ast.CallExpr) *TypeDefinition {
	// Handle function calls
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// Direct function call
		if returnType, exists := t.FunctionMap[fun.Name]; exists {
			return returnType
		}

	case *ast.SelectorExpr:
		// Method call or function from another package
		if x, ok := fun.X.(*ast.Ident); ok {
			// Check if it's a method call on a variable
			if _, exists := t.Variables[x.Name]; exists {
				// TODO: Look up method in the type's methods
				// For now, return a placeholder
				return &TypeDefinition{
					Name:       "any",
					Kind:       KindBasic,
					BasicType:  "any",
					Package:    "",
					IsResolved: true,
				}
			}

			// Check if it's a function from another package
			funcName := x.Name + "." + fun.Sel.Name
			if returnType, exists := t.FunctionMap[funcName]; exists {
				return returnType
			}
		}
	}

	// If we can't determine the return type, return a placeholder
	return &TypeDefinition{
		Name:       "any",
		Kind:       KindBasic,
		BasicType:  "any",
		Package:    "",
		IsResolved: true,
	}
}

// GetVariableType gets the type of a variable
func (t *VariableTracker) GetVariableType(name string) *TypeDefinition {
	if varInfo, exists := t.Variables[name]; exists {
		return varInfo.Type
	}
	return nil
}

// RegisterFunctionReturnType registers the return type of a function
func (t *VariableTracker) RegisterFunctionReturnType(funcName string, returnType *TypeDefinition) {
	t.FunctionMap[funcName] = returnType
	if t.Verbose {
		fmt.Printf("Registered function return type: %s -> %s\n", funcName, returnType.Name)
	}
}
