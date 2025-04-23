package types

import (
	"fmt"
	"go/ast"
	"go/token"
	"net/http"
	"strconv"
)

// ResponseInfo represents information about a JSON response
type ResponseInfo struct {
	StatusCode int
	Type       *TypeDefinition
	Position   string
}

// ResponseAnalyzer analyzes Echo response methods to extract JSON response formats
type ResponseAnalyzer struct {
	Registry        *TypeRegistry
	VariableTracker *VariableTracker
	Responses       []*ResponseInfo
	Verbose         bool
}

// NewResponseAnalyzer creates a new ResponseAnalyzer
func NewResponseAnalyzer(registry *TypeRegistry, variableTracker *VariableTracker, verbose bool) *ResponseAnalyzer {
	return &ResponseAnalyzer{
		Registry:        registry,
		VariableTracker: variableTracker,
		Responses:       []*ResponseInfo{},
		Verbose:         verbose,
	}
}

// AnalyzeHandler analyzes a handler function for JSON responses
func (a *ResponseAnalyzer) AnalyzeHandler(funcDecl *ast.FuncDecl) error {
	if a.Verbose {
		fmt.Printf("Analyzing handler function: %s for JSON responses\n", funcDecl.Name.Name)
	}

	// Clear previous responses
	a.Responses = []*ResponseInfo{}

	// Analyze the function body
	if funcDecl.Body != nil {
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			// Look for method calls
			if expr, ok := n.(*ast.CallExpr); ok {
				if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						// Check for Echo context methods
						a.checkJSONResponseMethod(ident.Name, sel.Sel.Name, expr)
					}
				}
			}
			return true
		})
	}

	return nil
}

// checkJSONResponseMethod checks if a method call is a JSON response method
func (a *ResponseAnalyzer) checkJSONResponseMethod(objName, methodName string, call *ast.CallExpr) {
	// Common context parameter names
	contextNames := map[string]bool{
		"c": true, "ctx": true, "context": true, "ec": true,
	}

	if !contextNames[objName] {
		return
	}

	// Check for JSON response methods
	isJSONResponse := false
	switch methodName {
	case "JSON", "JSONPretty", "JSONBlob":
		isJSONResponse = true
	}

	if !isJSONResponse {
		return
	}

	// Extract status code and response variable
	var statusCode int = http.StatusOK // Default
	var responseVar ast.Expr

	if len(call.Args) >= 2 {
		// First argument is status code
		statusCode = a.extractStatusCode(call.Args[0])
		// Second argument is response data
		responseVar = call.Args[1]
	}

	// Resolve the type of the response variable
	responseType := a.resolveResponseType(responseVar)
	if responseType == nil {
		if a.Verbose {
			fmt.Printf("  Could not resolve type of response variable\n")
		}
		return
	}

	// Create response info
	responseInfo := &ResponseInfo{
		StatusCode: statusCode,
		Type:       responseType,
		Position:   a.Registry.FileSet.Position(call.Pos()).String(),
	}

	a.Responses = append(a.Responses, responseInfo)

	if a.Verbose {
		fmt.Printf("  Found JSON response: status %d, type %s\n", statusCode, responseType.Name)
	}
}

// extractStatusCode extracts an HTTP status code from an AST expression
func (a *ResponseAnalyzer) extractStatusCode(expr ast.Expr) int {
	// Handle direct integer literals
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.INT {
			code, _ := strconv.Atoi(lit.Value)
			return code
		}
	}

	// Handle http.StatusXXX constants
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "http" {
				switch sel.Sel.Name {
				case "StatusOK":
					return http.StatusOK
				case "StatusCreated":
					return http.StatusCreated
				case "StatusAccepted":
					return http.StatusAccepted
				case "StatusNoContent":
					return http.StatusNoContent
				case "StatusBadRequest":
					return http.StatusBadRequest
				case "StatusUnauthorized":
					return http.StatusUnauthorized
				case "StatusForbidden":
					return http.StatusForbidden
				case "StatusNotFound":
					return http.StatusNotFound
				case "StatusInternalServerError":
					return http.StatusInternalServerError
				}
			}
		}
	}

	return http.StatusOK // Default
}

// resolveResponseType resolves the type of a response variable
func (a *ResponseAnalyzer) resolveResponseType(expr ast.Expr) *TypeDefinition {
	switch e := expr.(type) {
	case *ast.Ident:
		// Variable reference
		return a.VariableTracker.GetVariableType(e.Name)

	case *ast.SelectorExpr:
		// Field access (e.g., user.Profile)
		if x, ok := e.X.(*ast.Ident); ok {
			varType := a.VariableTracker.GetVariableType(x.Name)
			if varType != nil && varType.Kind == KindStruct {
				// Find the field in the struct
				for _, field := range varType.Fields {
					if field.Name == e.Sel.Name {
						return field.Type
					}
				}
			}
		}

	case *ast.CallExpr:
		// Function call (e.g., getUser())
		return a.VariableTracker.resolveFunctionCallType(e)

	case *ast.CompositeLit:
		// Composite literal (e.g., User{Name: "John"})
		return a.Registry.ResolveType(e.Type)

	case *ast.UnaryExpr:
		// Unary expression (e.g., &user)
		if e.Op == token.AND {
			return a.resolveResponseType(e.X)
		}
	}

	return nil
}

// GetResponses returns all analyzed responses
func (a *ResponseAnalyzer) GetResponses() []*ResponseInfo {
	return a.Responses
}
