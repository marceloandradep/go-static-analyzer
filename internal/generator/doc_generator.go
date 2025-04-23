package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/user/golang-echo-analyzer/internal/analyzer"
	"github.com/user/golang-echo-analyzer/internal/aws"
	"github.com/user/golang-echo-analyzer/internal/scanner"
	"github.com/user/golang-echo-analyzer/internal/types"
)

// Format constants
const (
	FormatMarkdown = "markdown"
	FormatJSON     = "json"
	FormatOpenAPI  = "openapi"
)

// DocGenerator generates documentation from analysis results
type DocGenerator struct {
	Routes          []scanner.RouteInfo
	Handlers        map[string]*analyzer.HandlerInfo
	Events          []aws.EventInfo
	OutputFile      string
	Format          string
	Verbose         bool
	SchemaGenerator *types.SchemaGenerator
	ResponseTypes   map[string]*types.ResponseInfo
}

// NewDocGenerator creates a new DocGenerator
func NewDocGenerator(outputFile, format string, verbose bool) *DocGenerator {
	return &DocGenerator{
		Routes:        []scanner.RouteInfo{},
		Handlers:      make(map[string]*analyzer.HandlerInfo),
		Events:        []aws.EventInfo{},
		OutputFile:    outputFile,
		Format:        format,
		Verbose:       verbose,
		ResponseTypes: make(map[string]*types.ResponseInfo),
	}
}

// SetData sets the data for the generator
func (g *DocGenerator) SetData(routes []scanner.RouteInfo, handlers map[string]*analyzer.HandlerInfo, events []aws.EventInfo) {
	g.Routes = routes
	g.Handlers = handlers
	g.Events = events
}

// SetSchemaGenerator sets the schema generator
func (g *DocGenerator) SetSchemaGenerator(schemaGenerator *types.SchemaGenerator) {
	g.SchemaGenerator = schemaGenerator
}

// SetResponseTypes sets the response types
func (g *DocGenerator) SetResponseTypes(responseTypes map[string]*types.ResponseInfo) {
	g.ResponseTypes = responseTypes
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
		Routes          []scanner.RouteInfo
		Handlers        map[string]*analyzer.HandlerInfo
		Events          []aws.EventInfo
		ResponseTypes   map[string]*types.ResponseInfo
		SchemaGenerator *types.SchemaGenerator
		GeneratedAt     string
	}{
		Routes:          g.Routes,
		Handlers:        g.Handlers,
		Events:          g.Events,
		ResponseTypes:   g.ResponseTypes,
		SchemaGenerator: g.SchemaGenerator,
		GeneratedAt:     time.Now().Format("January 2, 2006 15:04:05"),
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
	// Create OpenAPI spec
	spec := g.createOpenAPISpec()

	// Convert to JSON
	jsonData, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling OpenAPI spec: %v", err)
	}

	// Write to file
	if err := os.WriteFile(g.OutputFile, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing OpenAPI spec: %v", err)
	}

	return nil
}

// OpenAPISpec represents an OpenAPI specification
type OpenAPISpec struct {
	OpenAPI    string              `json:"openapi"`
	Info       OpenAPIInfo         `json:"info"`
	Servers    []OpenAPIServer     `json:"servers"`
	Paths      map[string]PathItem `json:"paths"`
	Components OpenAPIComponents   `json:"components"`
}

// OpenAPIInfo represents the info section of an OpenAPI specification
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// OpenAPIServer represents a server in an OpenAPI specification
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// PathItem represents a path item in an OpenAPI specification
type PathItem map[string]Operation

// Operation represents an operation in an OpenAPI specification
type Operation struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	OperationID string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Tags        []string            `json:"tags,omitempty"`
}

// Parameter represents a parameter in an OpenAPI specification
type Parameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Schema      interface{} `json:"schema"`
}

// RequestBody represents a request body in an OpenAPI specification
type RequestBody struct {
	Description string                     `json:"description"`
	Content     map[string]MediaTypeObject `json:"content"`
	Required    bool                       `json:"required"`
}

// Response represents a response in an OpenAPI specification
type Response struct {
	Description string                     `json:"description"`
	Content     map[string]MediaTypeObject `json:"content,omitempty"`
}

// MediaTypeObject represents a media type object in an OpenAPI specification
type MediaTypeObject struct {
	Schema interface{} `json:"schema"`
}

// OpenAPIComponents represents the components section of an OpenAPI specification
type OpenAPIComponents struct {
	Schemas map[string]interface{} `json:"schemas"`
}

// createOpenAPISpec creates an OpenAPI specification
func (g *DocGenerator) createOpenAPISpec() OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       "API Documentation",
			Description: "Generated by Echo Framework Static Analyzer",
			Version:     "1.0.0",
		},
		Servers: []OpenAPIServer{
			{
				URL:         "/",
				Description: "Default server",
			},
		},
		Paths: make(map[string]PathItem),
		Components: OpenAPIComponents{
			Schemas: make(map[string]interface{}),
		},
	}

	// Add paths
	for _, route := range g.Routes {
		path := route.Path
		method := strings.ToLower(route.Method)

		// Create path item if it doesn't exist
		if _, exists := spec.Paths[path]; !exists {
			spec.Paths[path] = make(PathItem)
		}

		// Create operation
		operation := Operation{
			Summary:     fmt.Sprintf("%s %s", route.Method, route.Path),
			Description: fmt.Sprintf("Handler: %s", route.HandlerName),
			OperationID: fmt.Sprintf("%s_%s", method, strings.Replace(path, "/", "_", -1)),
			Parameters:  []Parameter{},
			Responses:   make(map[string]Response),
		}

		// Get handler info
		handler := g.getHandlerForRoute(route)
		if handler != nil {
			// Add parameters
			for _, input := range handler.RequestInputs {
				param := Parameter{
					Name:        input.Name,
					Description: input.Description,
					Required:    input.Required,
				}

				// Set parameter location
				switch input.Type {
				case "Path":
					param.In = "path"
					param.Required = true
				case "Query":
					param.In = "query"
				case "Header":
					param.In = "header"
				case "Cookie":
					param.In = "cookie"
				}

				// Set schema
				param.Schema = map[string]string{
					"type": "string", // Default
				}

				// Add parameter
				operation.Parameters = append(operation.Parameters, param)
			}

			// Add request body if needed
			for _, input := range handler.RequestInputs {
				if input.Type == "Body" {
					// Check if we have a schema for this type
					var schema interface{} = map[string]string{
						"type": "object", // Default
					}

					// Add request body
					operation.RequestBody = &RequestBody{
						Description: "Request body",
						Content: map[string]MediaTypeObject{
							"application/json": {
								Schema: schema,
							},
						},
						Required: true,
					}
					break
				}
			}

			// Add responses
			for _, output := range handler.ResponseOutputs {
				statusCode := fmt.Sprintf("%d", output.StatusCode)
				response := Response{
					Description: fmt.Sprintf("%d response", output.StatusCode),
				}

				// Add content if it's a JSON response
				if output.Type == "JSON" {
					// Check if we have a schema for this response
					responseKey := fmt.Sprintf("%s_%s", route.HandlerName, statusCode)
					if responseInfo, exists := g.ResponseTypes[responseKey]; exists && responseInfo.Type != nil {
						// Generate JSON schema
						if g.SchemaGenerator != nil {
							schema := g.SchemaGenerator.GenerateSchema(responseInfo.Type)
							if schema != nil {
								// Add schema to components
								schemaName := fmt.Sprintf("%s_%s_Response", route.HandlerName, statusCode)
								spec.Components.Schemas[schemaName] = schema

								// Reference the schema
								response.Content = map[string]MediaTypeObject{
									"application/json": {
										Schema: map[string]string{
											"$ref": fmt.Sprintf("#/components/schemas/%s", schemaName),
										},
									},
								}
							}
						}
					} else {
						// Default schema
						response.Content = map[string]MediaTypeObject{
							"application/json": {
								Schema: map[string]string{
									"type": "object",
								},
							},
						}
					}
				}

				// Add response
				operation.Responses[statusCode] = response
			}

			// Add default response if no responses are defined
			if len(operation.Responses) == 0 {
				operation.Responses["200"] = Response{
					Description: "200 response",
				}
			}
		}

		// Add operation to path
		spec.Paths[path][method] = operation
	}

	return spec
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

{{$responseKey := printf "%s_%d" $handler.Name 200}}
{{$responseInfo := index $.ResponseTypes $responseKey}}
{{if $responseInfo}}
{{if $responseInfo.Type}}
{{if $.SchemaGenerator}}
**JSON Schema:**

` + "```json" + `
{{$schema := $.SchemaGenerator.GenerateSchemaString $responseInfo.Type}}
{{$schema}}
` + "```" + `

**Example Response:**

` + "```json" + `
{{$example := $.SchemaGenerator.GenerateExampleJSON $responseInfo.Type}}
{{$example}}
` + "```" + `
{{end}}
{{end}}
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
