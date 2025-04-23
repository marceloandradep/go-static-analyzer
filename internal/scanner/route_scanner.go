package scanner

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// RouteInfo represents information about an Echo route
type RouteInfo struct {
	Method      string      // HTTP method (GET, POST, etc.)
	Path        string      // Route path
	HandlerName string      // Name of the handler function
	HandlerNode ast.Node    // AST node of the handler function
	Position    token.Position // Position in source code
}

// RouteScanner scans AST for Echo route definitions
type RouteScanner struct {
	FileSet     *token.FileSet
	Routes      []RouteInfo
	Verbose     bool
	echoVarNames map[string]bool // Tracks variables that might be Echo instances
}

// NewRouteScanner creates a new RouteScanner
func NewRouteScanner(fset *token.FileSet, verbose bool) *RouteScanner {
	return &RouteScanner{
		FileSet:     fset,
		Routes:      []RouteInfo{},
		Verbose:     verbose,
		echoVarNames: map[string]bool{
			"e":      true,
			"echo":   true,
			"router": true,
			"app":    true,
			"server": true,
		},
	}
}

// Scan scans all files for Echo route definitions
func (s *RouteScanner) Scan(files []*ast.File) error {
	if s.Verbose {
		fmt.Println("Scanning for Echo route definitions...")
	}

	for _, file := range files {
		// First pass: identify Echo instance variables
		s.identifyEchoInstances(file)
		
		// Second pass: find route definitions
		s.findRouteDefinitions(file)
	}

	if s.Verbose {
		fmt.Printf("Found %d routes\n", len(s.Routes))
	}

	return nil
}

// identifyEchoInstances finds variables that might be Echo instances
func (s *RouteScanner) identifyEchoInstances(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for variable assignments
		if assign, ok := n.(*ast.AssignStmt); ok {
			for i, rhs := range assign.Rhs {
				// Check if right side is a call to echo.New() or similar
				if call, ok := rhs.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok {
							if ident.Name == "echo" && sel.Sel.Name == "New" {
								// This is a call to echo.New()
								if i < len(assign.Lhs) {
									if lhsIdent, ok := assign.Lhs[i].(*ast.Ident); ok {
										if s.Verbose {
											fmt.Printf("  Found Echo instance: %s\n", lhsIdent.Name)
										}
										s.echoVarNames[lhsIdent.Name] = true
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

// findRouteDefinitions finds Echo route definitions
func (s *RouteScanner) findRouteDefinitions(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for method calls
		if expr, ok := n.(*ast.CallExpr); ok {
			if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Check if this is a call on an Echo instance
					if s.echoVarNames[ident.Name] {
						// Check if this is a route definition method
						method := s.getHTTPMethod(sel.Sel.Name)
						if method != "" && len(expr.Args) >= 2 {
							// This is a route definition
							path := s.extractStringLiteral(expr.Args[0])
							handlerInfo := s.extractHandlerInfo(expr.Args[1])
							
							if path != "" {
								route := RouteInfo{
									Method:      method,
									Path:        path,
									HandlerName: handlerInfo,
									HandlerNode: expr.Args[1],
									Position:    s.FileSet.Position(expr.Pos()),
								}
								s.Routes = append(s.Routes, route)
								
								if s.Verbose {
									fmt.Printf("  Found route: %s %s -> %s\n", method, path, handlerInfo)
								}
							}
						}
						
						// Check for group definitions
						if sel.Sel.Name == "Group" && len(expr.Args) >= 1 {
							prefix := s.extractStringLiteral(expr.Args[0])
							if prefix != "" {
								// Track the group variable for subsequent route definitions
								if assign, ok := n.(*ast.AssignStmt); ok {
									for i, rhs := range assign.Rhs {
										if rhs == expr && i < len(assign.Lhs) {
											if lhsIdent, ok := assign.Lhs[i].(*ast.Ident); ok {
												if s.Verbose {
													fmt.Printf("  Found Echo group: %s with prefix %s\n", lhsIdent.Name, prefix)
												}
												s.echoVarNames[lhsIdent.Name] = true
												// TODO: Track the prefix for this group
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})
}

// getHTTPMethod returns the HTTP method for an Echo method name
func (s *RouteScanner) getHTTPMethod(methodName string) string {
	switch methodName {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD":
		return methodName
	case "Any":
		return "ANY"
	default:
		return ""
	}
}

// extractStringLiteral extracts a string literal from an AST expression
func (s *RouteScanner) extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.STRING {
			// Remove quotes
			return strings.Trim(lit.Value, "\"'`")
		}
	}
	return ""
}

// extractHandlerInfo extracts information about a handler function
func (s *RouteScanner) extractHandlerInfo(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		// Direct function name
		return v.Name
	case *ast.SelectorExpr:
		// Package.Function
		if x, ok := v.X.(*ast.Ident); ok {
			return x.Name + "." + v.Sel.Name
		}
	case *ast.FuncLit:
		// Anonymous function
		return "anonymous"
	}
	return "unknown"
}

// GetRoutes returns all found routes
func (s *RouteScanner) GetRoutes() []RouteInfo {
	return s.Routes
}
