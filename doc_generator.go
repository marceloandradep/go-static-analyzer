package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/user/golang-echo-analyzer/src/analyzer"
	"github.com/user/golang-echo-analyzer/src/aws"
	"github.com/user/golang-echo-analyzer/src/scanner"
)

// Format constants
const (
	FormatMarkdown = "markdown"
	FormatJSON     = "json"
	FormatOpenAPI  = "openapi"
)

// DocGenerator generates documentation from analysis results
type DocGenerator struct {
	Routes      []scanner.RouteInfo
	Handlers    map[string]*analyzer.HandlerInfo
	Events      []aws.EventInfo
	OutputFile  string
	Format      string
	Verbose     bool
}

// NewDocGenerator creates a new DocGenerator
func NewDocGenerator(outputFile, format string, verbose bool) *DocGenerator {
	return &DocGenerator{
		Routes:     []scanner.RouteInfo{},
		Handlers:   make(map[string]*analyzer.HandlerInfo),
		Events:     []aws.EventInfo{},
		OutputFile: outputFile,
		Format:     format,
		Verbose:    verbose,
	}
}

// SetData sets the data for the generator
func (g *DocGenerator) SetData(routes []scanner.RouteInfo, handlers map[string]*analyzer.HandlerInfo, events []aws.EventInfo) {
	g.Routes = routes
	g.Handlers = handlers
	g.Events = events
}

// Generate generates documentation based on the analysis results
func (g *DocGenerator) Generate() error {
	if g.Verbose {
		fmt.Println("Generating documentation...")
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(g.OutputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	// Generate documentation based on format
	var err error
	switch g.Format {
	case FormatMarkdown:
		err = g.generateMarkdown()
	case FormatJSON:
		err = g.generateJSON()
	case FormatOpenAPI:
		err = g.generateOpenAPI()
	default:
		err = fmt.Errorf("unsupported format: %s", g.Format)
	}

	if err != nil {
		return err
	}

	if g.Verbose {
		fmt.Printf("Documentation generated: %s\n", g.OutputFile)
	}

	return nil
}

// generateMarkdown generates Markdown documentation
func (g *DocGenerator) generateMarkdown() error {
	// Create the template
	tmpl, err := template.New("markdown").Parse(markdownTemplate)
	if err != nil {
		return fmt.Errorf("error creating template: %v", err)
	}

	// Prepare template data
	data := struct {
		Routes      []scanner.RouteInfo
		Handlers    map[string]*analyzer.HandlerInfo
		Events      []aws.EventInfo
		GeneratedAt string
	}{
		Routes:      g.Routes,
		Handlers:    g.Handlers,
		Events:      g.Events,
		GeneratedAt: time.Now().Format("January 2, 2006 15:04:05"),
	}

	// Create output file
	file, err := os.Create(g.OutputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	// Execute the template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}

	return nil
}

// generateJSON generates JSON documentation
func (g *DocGenerator) generateJSON() error {
	// For now, just generate Markdown as a fallback
	// TODO: Implement proper JSON output
	return g.generateMarkdown()
}

// generateOpenAPI generates OpenAPI documentation
func (g *DocGenerator) generateOpenAPI() error {
	// For now, just generate Markdown as a fallback
	// TODO: Implement proper OpenAPI output
	return g.generateMarkdown()
}

// getHandlerForRoute finds the handler info for a route
func (g *DocGenerator) getHandlerForRoute(route scanner.RouteInfo) *analyzer.HandlerInfo {
	// First try direct match by name
	if handler, exists := g.Handlers[route.HandlerName]; exists {
		return handler
	}

	// Try anonymous handler match
	name := fmt.Sprintf("anonymous_%s_%s", route.Method, strings.Replace(route.Path, "/", "_", -1))
	if handler, exists := g.Handlers[name]; exists {
		return handler
	}

	return nil
}

// Markdown template for documentation
const markdownTemplate = `# API Documentation

*Generated at: {{.GeneratedAt}}*

## Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
{{range .Routes}}| {{.Method}} | {{.Path}} | {{.HandlerName}} | |
{{end}}

## Detailed Endpoint Documentation

{{range .Routes}}
### {{.Method}} {{.Path}}

**Handler:** {{.HandlerName}}

{{$handler := index $.Handlers .HandlerName}}
{{if $handler}}
#### Request Parameters

{{if $handler.RequestInputs}}
| Type | Name | Data Type | Required | Description |
|------|------|-----------|----------|-------------|
{{range $handler.RequestInputs}}| {{.Type}} | {{.Name}} | {{.DataType}} | {{.Required}} | {{.Description}} |
{{end}}
{{else}}
*No request parameters*
{{end}}

#### Response

{{if $handler.ResponseOutputs}}
| Type | Status Code | Data Type | Description |
|------|------------|-----------|-------------|
{{range $handler.ResponseOutputs}}| {{.Type}} | {{.StatusCode}} | {{.DataType}} | {{.Description}} |
{{end}}
{{else}}
*No response information available*
{{end}}
{{else}}
*No detailed information available for this endpoint*
{{end}}

{{end}}

## AWS Events

{{if .Events}}
| Service | Operation | Topic/Queue | Message Format |
|---------|-----------|-------------|----------------|
{{range .Events}}| {{.Service}} | {{.Operation}} | {{.TopicOrQueue}} | {{if .MessageFormat.IsStructured}}Structured{{else}}Raw{{end}} |
{{end}}

### Detailed Event Documentation

{{range .Events}}
#### {{.Service}} {{.Operation}} to {{.TopicOrQueue}}

{{if .MessageFormat.IsStructured}}
**Message Fields:**

| Field | Type | Description |
|-------|------|-------------|
{{range .MessageFormat.Fields}}| {{.Name}} | {{.Type}} | {{.Description}} |
{{end}}
{{else if .MessageFormat.RawMessage}}
**Raw Message:**

` + "```" + `
{{.MessageFormat.RawMessage}}
` + "```" + `
{{else}}
*No message format information available*
{{end}}

{{end}}
{{else}}
*No AWS events found*
{{end}}
`
