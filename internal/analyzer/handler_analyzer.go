package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/user/golang-echo-analyzer/internal/scanner"
)

// HandlerInfo represents information about a handler function
type HandlerInfo struct {
	Name            string
	Route           scanner.RouteInfo
	RequestInputs   []RequestInput
	ResponseOutputs []ResponseOutput
	Position        token.Position
}

// RequestInput represents an input parameter from a request
type RequestInput struct {
	Type        string // Path, Query, Form, Body, etc.
	Name        string // Parameter name
	DataType    string // Data type if available
	Description string // Description from comments if available
	Required    bool   // Whether the parameter is required
	Position    token.Position
}

// ResponseOutput represents an output returned to the client
type ResponseOutput struct {
	Type        string // JSON, XML, String, HTML, etc.
	StatusCode  int    // HTTP status code
	DataType    string // Data type if available
	Description string // Description from comments if available
	Position    token.Position
}

// HandlerAnalyzer analyzes Echo handler functions to determine inputs and outputs
type HandlerAnalyzer struct {
	FileSet  *token.FileSet
	Handlers map[string]*HandlerInfo
	Verbose  bool
}

// NewHandlerAnalyzer creates a new HandlerAnalyzer
func NewHandlerAnalyzer(fset *token.FileSet, verbose bool) *HandlerAnalyzer {
	return &HandlerAnalyzer{
		FileSet:  fset,
		Handlers: make(map[string]*HandlerInfo),
		Verbose:  verbose,
	}
}

// Analyze analyzes handler functions for request inputs and response outputs
func (a *HandlerAnalyzer) Analyze(files []*ast.File, routes []scanner.RouteInfo) error {
	if a.Verbose {
		fmt.Println("Analyzing handler functions...")
	}

	// First, find all handler function declarations
	handlerFuncs := a.findHandlerFunctions(files)

	// Then, analyze each handler function
	for _, route := range routes {
		if a.Verbose {
			fmt.Printf("  Analyzing handler for route: %s %s\n", route.Method, route.Path)
		}

		// Check if we have the handler function
		handlerFunc, exists := handlerFuncs[route.HandlerName]
		if !exists {
			// Try to analyze the handler directly from the route definition
			// This handles anonymous functions and other cases
			a.analyzeHandlerFromRoute(route)
			continue
		}

		// Create handler info
		handlerInfo := &HandlerInfo{
			Name:            route.HandlerName,
			Route:           route,
			RequestInputs:   []RequestInput{},
			ResponseOutputs: []ResponseOutput{},
			Position:        a.FileSet.Position(handlerFunc.Pos()),
		}

		// Analyze the handler function
		a.analyzeHandlerFunction(handlerFunc, handlerInfo)

		// Store the handler info
		a.Handlers[route.HandlerName] = handlerInfo
	}

	if a.Verbose {
		fmt.Printf("Analyzed %d handlers\n", len(a.Handlers))
	}

	return nil
}

// findHandlerFunctions finds all functions that could be Echo handlers
func (a *HandlerAnalyzer) findHandlerFunctions(files []*ast.File) map[string]*ast.FuncDecl {
	handlerFuncs := make(map[string]*ast.FuncDecl)

	for _, file := range files {
		for _, decl := range file.Decls {
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				// Check if this function has the Echo handler signature
				if a.isEchoHandler(funcDecl) {
					handlerFuncs[funcDecl.Name.Name] = funcDecl
					if a.Verbose {
						fmt.Printf("  Found handler function: %s\n", funcDecl.Name.Name)
					}
				}
			}
		}
	}

	return handlerFuncs
}

// isEchoHandler checks if a function has the Echo handler signature
func (a *HandlerAnalyzer) isEchoHandler(funcDecl *ast.FuncDecl) bool {
	// Echo handlers have the signature: func(c echo.Context) error
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return false
	}

	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 1 {
		return false
	}

	// Check parameter type (should be echo.Context or similar)
	paramType := a.getTypeString(funcDecl.Type.Params.List[0].Type)
	if !strings.Contains(paramType, "Context") {
		return false
	}

	// Check return type (should be error)
	returnType := a.getTypeString(funcDecl.Type.Results.List[0].Type)
	if returnType != "error" {
		return false
	}

	return true
}

// getTypeString returns a string representation of a type
func (a *HandlerAnalyzer) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + a.getTypeString(t.X)
	}
	return "unknown"
}

// analyzeHandlerFromRoute analyzes a handler directly from the route definition
func (a *HandlerAnalyzer) analyzeHandlerFromRoute(route scanner.RouteInfo) {
	// Handle anonymous functions
	if funcLit, ok := route.HandlerNode.(*ast.FuncLit); ok {
		handlerInfo := &HandlerInfo{
			Name:            "anonymous",
			Route:           route,
			RequestInputs:   []RequestInput{},
			ResponseOutputs: []ResponseOutput{},
			Position:        a.FileSet.Position(funcLit.Pos()),
		}

		// Analyze the function body
		a.analyzeHandlerBody(funcLit.Body, handlerInfo)

		// Store the handler info with a generated name
		name := fmt.Sprintf("anonymous_%s_%s", route.Method, strings.Replace(route.Path, "/", "_", -1))
		a.Handlers[name] = handlerInfo
	}
}

// analyzeHandlerFunction analyzes a handler function for request inputs and response outputs
func (a *HandlerAnalyzer) analyzeHandlerFunction(funcDecl *ast.FuncDecl, handlerInfo *HandlerInfo) {
	// Get the context parameter name
	var contextParamName string
	if len(funcDecl.Type.Params.List) > 0 {
		if len(funcDecl.Type.Params.List[0].Names) > 0 {
			contextParamName = funcDecl.Type.Params.List[0].Names[0].Name
		}
	}

	if contextParamName == "" {
		contextParamName = "c" // Default context parameter name
	}

	// Analyze the function body
	a.analyzeHandlerBody(funcDecl.Body, handlerInfo)
}

// analyzeHandlerBody analyzes a function body for Echo context method calls
func (a *HandlerAnalyzer) analyzeHandlerBody(body *ast.BlockStmt, handlerInfo *HandlerInfo) {
	if body == nil {
		return
	}

	ast.Inspect(body, func(n ast.Node) bool {
		// Look for method calls on the context parameter
		if expr, ok := n.(*ast.CallExpr); ok {
			if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Check for request input methods
					a.checkRequestInputMethod(ident.Name, sel.Sel.Name, expr, handlerInfo)

					// Check for response output methods
					a.checkResponseOutputMethod(ident.Name, sel.Sel.Name, expr, handlerInfo)
				}
			}
		}
		return true
	})
}

// checkRequestInputMethod checks if a method call is a request input method
func (a *HandlerAnalyzer) checkRequestInputMethod(objName, methodName string, call *ast.CallExpr, handlerInfo *HandlerInfo) {
	// Common context parameter names
	contextNames := map[string]bool{
		"c": true, "ctx": true, "context": true, "ec": true,
	}

	if !contextNames[objName] {
		return
	}

	var inputType, paramName string
	var required bool

	switch methodName {
	case "Param":
		// Path parameter: c.Param("id")
		inputType = "Path"
		required = true
		if len(call.Args) > 0 {
			paramName = a.extractStringLiteral(call.Args[0])
		}
	case "QueryParam":
		// Query parameter: c.QueryParam("filter")
		inputType = "Query"
		required = false
		if len(call.Args) > 0 {
			paramName = a.extractStringLiteral(call.Args[0])
		}
	case "FormValue":
		// Form value: c.FormValue("name")
		inputType = "Form"
		required = false
		if len(call.Args) > 0 {
			paramName = a.extractStringLiteral(call.Args[0])
		}
	case "Bind":
		// Request body binding: c.Bind(&user)
		inputType = "Body"
		required = true
		if len(call.Args) > 0 {
			paramName = a.extractVariableName(call.Args[0])
		}
	}

	if inputType != "" && paramName != "" {
		input := RequestInput{
			Type:     inputType,
			Name:     paramName,
			DataType: "string", // Default type
			Required: required,
			Position: a.FileSet.Position(call.Pos()),
		}

		// Check if this input already exists
		exists := false
		for _, existing := range handlerInfo.RequestInputs {
			if existing.Type == input.Type && existing.Name == input.Name {
				exists = true
				break
			}
		}

		if !exists {
			handlerInfo.RequestInputs = append(handlerInfo.RequestInputs, input)
			if a.Verbose {
				fmt.Printf("    Found request input: %s %s\n", input.Type, input.Name)
			}
		}
	}
}

// checkResponseOutputMethod checks if a method call is a response output method
func (a *HandlerAnalyzer) checkResponseOutputMethod(objName, methodName string, call *ast.CallExpr, handlerInfo *HandlerInfo) {
	// Common context parameter names
	contextNames := map[string]bool{
		"c": true, "ctx": true, "context": true, "ec": true,
	}

	if !contextNames[objName] {
		return
	}

	var outputType string
	var statusCode int = 200 // Default status code

	switch methodName {
	case "String":
		// String response: c.String(http.StatusOK, "Hello")
		outputType = "String"
	case "JSON":
		// JSON response: c.JSON(http.StatusOK, user)
		outputType = "JSON"
	case "XML":
		// XML response: c.XML(http.StatusOK, data)
		outputType = "XML"
	case "HTML":
		// HTML response: c.HTML(http.StatusOK, "<html>...</html>")
		outputType = "HTML"
	case "File":
		// File response: c.File("path/to/file")
		outputType = "File"
	case "Blob":
		// Blob response: c.Blob(http.StatusOK, "application/octet-stream", data)
		outputType = "Blob"
	case "Stream":
		// Stream response: c.Stream(http.StatusOK, "application/octet-stream", reader)
		outputType = "Stream"
	case "NoContent":
		// No content response: c.NoContent(http.StatusNoContent)
		outputType = "NoContent"
	case "Redirect":
		// Redirect response: c.Redirect(http.StatusFound, "/new-url")
		outputType = "Redirect"
	}

	if outputType != "" {
		// Try to extract status code from first argument
		if len(call.Args) > 0 {
			statusCode = a.extractStatusCode(call.Args[0])
		}

		output := ResponseOutput{
			Type:       outputType,
			StatusCode: statusCode,
			DataType:   "unknown", // Default type
			Position:   a.FileSet.Position(call.Pos()),
		}

		// Try to determine data type for JSON/XML responses
		if (outputType == "JSON" || outputType == "XML") && len(call.Args) > 1 {
			output.DataType = a.extractDataType(call.Args[1])
		}

		handlerInfo.ResponseOutputs = append(handlerInfo.ResponseOutputs, output)
		if a.Verbose {
			fmt.Printf("    Found response output: %s (status %d)\n", output.Type, output.StatusCode)
		}
	}
}

// extractStringLiteral extracts a string literal from an AST expression
func (a *HandlerAnalyzer) extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.STRING {
			// Remove quotes
			return strings.Trim(lit.Value, "\"'`")
		}
	}
	return ""
}

// extractVariableName extracts a variable name from an AST expression
func (a *HandlerAnalyzer) extractVariableName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.UnaryExpr:
		// Handle address-of operator (&user)
		if v.Op == token.AND {
			if ident, ok := v.X.(*ast.Ident); ok {
				return ident.Name
			}
		}
	}
	return "unknown"
}

// extractStatusCode extracts an HTTP status code from an AST expression
func (a *HandlerAnalyzer) extractStatusCode(expr ast.Expr) int {
	// Handle direct integer literals
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.INT {
			var code int
			fmt.Sscanf(lit.Value, "%d", &code)
			return code
		}
	}

	// Handle http.StatusXXX constants
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "http" {
				switch sel.Sel.Name {
				case "StatusOK":
					return 200
				case "StatusCreated":
					return 201
				case "StatusAccepted":
					return 202
				case "StatusNoContent":
					return 204
				case "StatusBadRequest":
					return 400
				case "StatusUnauthorized":
					return 401
				case "StatusForbidden":
					return 403
				case "StatusNotFound":
					return 404
				case "StatusInternalServerError":
					return 500
				}
			}
		}
	}

	return 200 // Default to 200 OK
}

// extractDataType extracts the data type from an AST expression
func (a *HandlerAnalyzer) extractDataType(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		if x, ok := v.X.(*ast.Ident); ok {
			return x.Name + "." + v.Sel.Name
		}
	case *ast.UnaryExpr:
		// Handle address-of operator (&user)
		if v.Op == token.AND {
			return a.extractDataType(v.X)
		}
	case *ast.CompositeLit:
		// Handle composite literals (e.g., map[string]interface{}{...})
		return a.extractDataType(v.Type)
	case *ast.MapType:
		// Handle map types
		keyType := a.extractDataType(v.Key)
		valueType := a.extractDataType(v.Value)
		return fmt.Sprintf("map[%s]%s", keyType, valueType)
	case *ast.ArrayType:
		// Handle array types
		elemType := a.extractDataType(v.Elt)
		return fmt.Sprintf("[]%s", elemType)
	case *ast.StarExpr:
		// Handle pointer types
		return "*" + a.extractDataType(v.X)
	}
	return "unknown"
}

// GetHandlers returns all analyzed handlers
func (a *HandlerAnalyzer) GetHandlers() map[string]*HandlerInfo {
	return a.Handlers
}
