# Golang Echo Framework Static Analyzer

A static analysis tool that scans Golang repositories using the Echo web framework and AWS SDK to automatically generate API documentation.

## Features

- Identifies Echo route definitions (GET, POST, PUT, DELETE, etc.)
- Analyzes handler functions to determine request inputs:
  - Path parameters
  - Query parameters
  - Form values
  - Request body bindings
- Analyzes handler functions to determine response outputs:
  - JSON responses
  - XML responses
  - String responses
  - HTML responses
  - File responses
- Identifies AWS SNS/SQS usage and determines message formats
- Generates comprehensive API documentation in Markdown format

## Architecture

The tool consists of five main components:

1. **Code Parser**: Parses Go source files into Abstract Syntax Trees (ASTs)
2. **Route Definition Scanner**: Identifies Echo route definitions in the codebase
3. **Handler Analysis Engine**: Analyzes handler functions for request inputs and response outputs
4. **AWS SDK Usage Analyzer**: Identifies AWS SNS/SQS client usage and message formats
5. **Documentation Generator**: Generates API documentation based on the analysis results

## Usage

```bash
# Build the tool
go build -o echo-analyzer src/main.go

# Run the tool
./echo-analyzer --repo /path/to/your/repo --output api-docs.md
```

### Command Line Options

- `--repo`: Path to the repository to analyze (default: ".")
- `--output`: Output file for the API documentation (default: "api-docs.md")
- `--format`: Output format (markdown, json, openapi) (default: "markdown")
- `--verbose`: Enable verbose output (default: false)

## Example Output

The tool generates documentation that includes:

- List of all endpoints with HTTP methods and paths
- Detailed information about request parameters for each endpoint
- Response information including status codes and data types
- AWS events information including topics/queues and message formats

## Requirements

- Go 1.18 or later
- The repository to analyze must use the Echo framework for routing
- For AWS SDK analysis, the repository must use the AWS SDK for Go

## Development

To contribute to this project:

1. Clone the repository
2. Install dependencies with `go mod tidy`
3. Make your changes
4. Test with the sample application in the `test` directory
5. Submit a pull request

## License

MIT
# go-static-analyzer
