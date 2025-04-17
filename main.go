package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/user/golang-echo-analyzer/src/analyzer"
	"github.com/user/golang-echo-analyzer/src/aws"
	"github.com/user/golang-echo-analyzer/src/generator"
	"github.com/user/golang-echo-analyzer/src/parser"
	"github.com/user/golang-echo-analyzer/src/scanner"
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

	// 2. Scan for Echo route definitions
	fmt.Println("Step 2: Scanning for Echo route definitions...")
	routeScanner := scanner.NewRouteScanner(codeParser.FileSet, verbose)
	if err := routeScanner.Scan(codeParser.GetAllFiles()); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning for routes: %v\n", err)
		os.Exit(1)
	}
	routes := routeScanner.GetRoutes()
	fmt.Printf("  Found %d routes.\n", len(routes))

	// 3. Analyze handler functions
	fmt.Println("Step 3: Analyzing handler functions...")
	handlerAnalyzer := analyzer.NewHandlerAnalyzer(codeParser.FileSet, verbose)
	if err := handlerAnalyzer.Analyze(codeParser.GetAllFiles(), routes); err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing handlers: %v\n", err)
		os.Exit(1)
	}
	handlers := handlerAnalyzer.GetHandlers()
	fmt.Printf("  Analyzed %d handlers.\n", len(handlers))

	// 4. Scan for AWS SDK usage
	fmt.Println("Step 4: Analyzing AWS SDK usage...")
	awsAnalyzer := aws.NewAWSAnalyzer(codeParser.FileSet, verbose)
	if err := awsAnalyzer.Analyze(codeParser.GetAllFiles()); err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing AWS SDK usage: %v\n", err)
		os.Exit(1)
	}
	events := awsAnalyzer.GetEvents()
	fmt.Printf("  Found %d AWS events.\n", len(events))

	// 5. Generate documentation
	fmt.Println("Step 5: Generating documentation...")
	docGenerator := generator.NewDocGenerator(outputFile, outputFormat, verbose)
	docGenerator.SetData(routes, handlers, events)
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
	fmt.Println(bold(cyan("│ ")) + " from Echo routes and AWS SDK usage          " + bold(cyan("│")))
	fmt.Println(bold(cyan("└─────────────────────────────────────────────┘")))
	fmt.Println()
}
