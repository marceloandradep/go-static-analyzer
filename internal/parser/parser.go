package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// CodeParser is responsible for parsing Go source files into ASTs
type CodeParser struct {
	RootPath string
	FileSet  *token.FileSet
	Packages map[string]*ast.Package
	Verbose  bool
}

// NewCodeParser creates a new CodeParser instance
func NewCodeParser(rootPath string, verbose bool) *CodeParser {
	return &CodeParser{
		RootPath: rootPath,
		FileSet:  token.NewFileSet(),
		Packages: make(map[string]*ast.Package),
		Verbose:  verbose,
	}
}

// Parse parses all Go files in the repository
func (p *CodeParser) Parse() error {
	if p.Verbose {
		fmt.Println("Parsing Go files in repository...")
	}

	err := filepath.Walk(p.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() {
			// Skip hidden directories and vendor directory
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Skip test files if desired
		if strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		if p.Verbose {
			fmt.Printf("  Parsing file: %s\n", path)
		}

		// Parse the file
		file, err := parser.ParseFile(p.FileSet, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("error parsing file %s: %v", path, err)
		}

		// Get the package name
		pkgName := file.Name.Name
		pkg, exists := p.Packages[pkgName]
		if !exists {
			pkg = &ast.Package{
				Name:  pkgName,
				Files: make(map[string]*ast.File),
			}
			p.Packages[pkgName] = pkg
		}

		// Add the file to the package
		pkg.Files[path] = file

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking repository: %v", err)
	}

	if p.Verbose {
		fmt.Printf("Parsed %d packages\n", len(p.Packages))
		for pkgName, pkg := range p.Packages {
			fmt.Printf("  Package %s: %d files\n", pkgName, len(pkg.Files))
		}
	}

	return nil
}

// GetAllFiles returns all parsed files across all packages
func (p *CodeParser) GetAllFiles() []*ast.File {
	var files []*ast.File
	for _, pkg := range p.Packages {
		for _, file := range pkg.Files {
			files = append(files, file)
		}
	}
	return files
}

// GetFilePosition returns the position information for a given node
func (p *CodeParser) GetFilePosition(pos token.Pos) token.Position {
	return p.FileSet.Position(pos)
}
