package main

import (
	"flag"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/user/golang-echo-analyzer/internal/analyzer"
	"github.com/user/golang-echo-analyzer/internal/aws"
	"github.com/user/golang-echo-analyzer/internal/generator"
	"github.com/user/golang-echo-analyzer/internal/parser"
	"github.com/user/golang-echo-analyzer/internal/scanner"
	"github.com/user/golang-echo-analyzer/internal/types"
)

// Command line flags
var (
	repoPath     string
	outputFile   string
	outputFormat string
	verbose      bool
)

func init() {
	flag.StringVar(&repoPath, "repo", ".", "Path to the repository to analyze")
	flag.StringVar(&outputFile, "output", "api-docs.md", "Output file for the API documentation")
	flag.StringVar(&outputFormat, "format", "markdown", "Output format (markdown, json, openapi)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.Parse()
}

func main() {
	// Validate repository path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving repository path: %v\n", err)
		os.Exit(1)
	}

	// Check if the path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Repository path does not exist: %s\n", absPath)
		os.Exit(1)
	}

	// Print banner
	printBanner()

	// Print configuration
	fmt.Println("Configuration:")
	fmt.Printf("  Repository path: %s\n", absPath)
	fmt.Printf("  Output file: %s\n", outputFile)
	fmt.Printf("  Output format: %s\n", outputFormat)
	fmt.Printf("  Verbose mode: %v\n", verbose)
	fmt.Println()

	// 1. Parse Go source files
	fmt.Println("Step 1: Parsing Go source files...")
	codeParser := parser.NewCodeParser(absPath, verbose)
	if err := codeParser.Parse(); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing repository: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  Parsing completed successfully.")

	// 2. Initialize type registry and collector
	fmt.Println("Step 2: Initializing type resolution system...")
	typeRegistry := types.NewTypeRegistry(codeParser.FileSet, verbose)
	typeCollector := types.NewTypeCollector(typeRegistry, verbose)

	// Collect types from all packages
	for pkgPath, pkg := range codeParser.Packages {
		files := make([]*ast.File, 0, len(pkg.Files))
		for _, file := range pkg.Files {
			files = append(files, file)
		}
		if err := typeCollector.CollectTypes(files, pkgPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error collecting types from package %s: %v\n", pkgPath, err)
		}
	}

	// Resolve types
	if err := typeCollector.ResolveTypes(); err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving types: %v\n", err)
	}

	// 3. Initialize package resolver
	packageResolver := types.NewPackageResolver(typeRegistry, absPath, verbose)
	if err := packageResolver.ResolvePackages(); err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving packages: %v\n", err)
	}

	// 4. Initialize struct field analyzer
	fieldAnalyzer := types.NewStructFieldAnalyzer(typeRegistry, verbose)
	if err := fieldAnalyzer.AnalyzeStructFields(); err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing struct fields: %v\n", err)
	}

	// Analyze nested structs
	fieldAnalyzer.AnalyzeNestedStructs()

	fmt.Println("  Type resolution system initialized successfully.")

	// 5. Scan for Echo route definitions
	fmt.Println("Step 3: Scanning for Echo route definitions...")
	routeScanner := scanner.NewRouteScanner(codeParser.FileSet, verbose)
	if err := routeScanner.Scan(codeParser.GetAllFiles()); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for routes: %v\n", err)
		os.Exit(1)
	}
	routes := routeScanner.GetRoutes()
	fmt.Printf("  Found %d routes.\n", len(routes))

	// 6. Analyze handler functions
	fmt.Println("Step 4: Analyzing handler functions...")
	handlerAnalyzer := analyzer.NewHandlerAnalyzer(codeParser.FileSet, verbose)
	if err := handlerAnalyzer.Analyze(codeParser.GetAllFiles(), routes); err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing handlers: %v\n", err)
		os.Exit(1)
	}
	handlers := handlerAnalyzer.GetHandlers()
	fmt.Printf("  Analyzed %d handlers.\n", len(handlers))

	// 7. Analyze response types
	fmt.Println("Step 5: Analyzing response types...")
	responseTypes := make(map[string]*types.ResponseInfo)

	// For each handler function
	for handlerName, _ := range handlers {
		// Initialize variable tracker
		variableTracker := types.NewVariableTracker(typeRegistry, verbose)

		// Find the handler function in the AST
		for _, file := range codeParser.GetAllFiles() {
			for _, decl := range file.Decls {
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					if funcDecl.Name.Name == handlerName {
						// Track variables in the function
						if err := variableTracker.TrackFunction(funcDecl); err != nil {
							fmt.Fprintf(os.Stderr, "Error tracking variables in handler %s: %v\n", handlerName, err)
							continue
						}

						// Analyze responses
						responseAnalyzer := types.NewResponseAnalyzer(typeRegistry, variableTracker, verbose)
						if err := responseAnalyzer.AnalyzeHandler(funcDecl); err != nil {
							fmt.Fprintf(os.Stderr, "Error analyzing responses in handler %s: %v\n", handlerName, err)
							continue
						}

						// Store response types
						for _, response := range responseAnalyzer.GetResponses() {
							responseKey := fmt.Sprintf("%s_%d", handlerName, response.StatusCode)
							responseTypes[responseKey] = response
						}
					}
				}
			}
		}
	}

	fmt.Printf("  Analyzed %d response types.\n", len(responseTypes))

	// 8. Scan for AWS SDK usage
	fmt.Println("Step 6: Analyzing AWS SDK usage...")
	awsAnalyzer := aws.NewAWSAnalyzer(codeParser.FileSet, verbose)
	if err := awsAnalyzer.Analyze(codeParser.GetAllFiles()); err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing AWS SDK usage: %v\n", err)
		os.Exit(1)
	}
	events := awsAnalyzer.GetEvents()
	fmt.Printf("  Found %d AWS events.\n", len(events))

	// 9. Generate documentation
	fmt.Println("Step 7: Generating documentation...")

	// Initialize schema generator
	schemaGenerator := types.NewSchemaGenerator(typeRegistry, verbose)

	// Initialize documentation generator
	docGenerator := generator.NewDocGenerator(outputFile, outputFormat, verbose)
	docGenerator.SetData(routes, handlers, events)
	docGenerator.SetSchemaGenerator(schemaGenerator)
	docGenerator.SetResponseTypes(responseTypes)

	if err := docGenerator.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating documentation: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Documentation generated: %s\n", outputFile)

	fmt.Println("\nAnalysis completed successfully!")
}

// printBanner prints a fancy banner for the tool
func printBanner() {
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println()
	fmt.Println(bold(cyan("┌─────────────────────────────────────────────┐")))
	fmt.Println(bold(cyan("│ ")) + bold(green(" Echo Framework Static Analyzer ")) + bold(cyan("            │")))
	fmt.Println(bold(cyan("│ ")) + "                                             " + bold(cyan("│")))
	fmt.Println(bold(cyan("│ ")) + " Automatically generate API documentation    " + bold(cyan("│")))
	fmt.Println(bold(cyan("│ ")) + " with detailed JSON response schemas         " + bold(cyan("│")))
	fmt.Println(bold(cyan("└─────────────────────────────────────────────┘")))
	fmt.Println()
}
