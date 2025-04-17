# Golang Echo Framework Static Analyzer - Architecture Design

## Overview

This document outlines the architecture for a static analysis tool designed to scan Golang repositories using the Echo web framework and AWS SDK. The tool will identify route definitions, analyze request/response patterns, and document AWS SNS/SQS event formats.

## Components

### 1. Code Parser

**Purpose**: Parse Go source files into Abstract Syntax Trees (ASTs) for analysis.

**Implementation**:
- Use Go's standard library `go/parser` and `go/ast` packages
- Parse all Go files in the target repository
- Build a representation of the code structure for analysis

**Inputs**:
- Repository path
- Optional file patterns to include/exclude

**Outputs**:
- AST representations of Go source files
- Package structure information

### 2. Route Definition Scanner

**Purpose**: Identify Echo route definitions in the codebase.

**Implementation**:
- Scan ASTs for method calls on Echo instances (e.g., `e.GET()`, `e.POST()`, etc.)
- Extract route paths and handler function references
- Build a map of routes to handler functions

**Detection Patterns**:
- Direct method calls: `e.GET("/path", handlerFunc)`
- Group method calls: `g := e.Group("/api"); g.GET("/users", getUsers)`
- Router variable patterns: `router := echo.New(); router.GET(...)`

**Outputs**:
- List of routes with HTTP methods, paths, and handler function references

### 3. Handler Analysis Engine

**Purpose**: Analyze handler functions to determine request inputs and response outputs.

**Implementation**:
- For each identified handler, analyze the function body
- Detect Echo Context method calls for request data extraction
- Detect response generation method calls

**Request Input Detection**:
- Path parameters: `c.Param("id")`
- Query parameters: `c.QueryParam("filter")`
- Form values: `c.FormValue("name")`
- Request body binding: `c.Bind(&user)`

**Response Output Detection**:
- String responses: `c.String(http.StatusOK, "text")`
- JSON responses: `c.JSON(http.StatusOK, user)`
- XML responses: `c.XML(http.StatusOK, data)`
- HTML responses: `c.HTML(http.StatusOK, "<html>...</html>")`
- File responses: `c.File("path/to/file")`

**Outputs**:
- For each route, a structured representation of inputs and outputs

### 4. AWS SDK Usage Analyzer

**Purpose**: Identify AWS SDK usage for SNS/SQS and determine event formats.

**Implementation**:
- Scan ASTs for AWS SDK client instantiations
- Detect SNS/SQS client method calls
- Analyze parameters to determine message formats

**Detection Patterns**:
- SNS Publish calls: `snsClient.Publish(...)` or `PublishInput{...}`
- SQS Send calls: `sqsClient.SendMessage(...)` or `SendMessageInput{...}`
- Message attribute definitions and structures

**Outputs**:
- List of SNS topics and SQS queues used
- Message formats and structures for each

### 5. Documentation Generator

**Purpose**: Generate API documentation based on the analysis results.

**Implementation**:
- Combine results from route scanner and handler analyzer
- Format documentation in Markdown, OpenAPI, or custom format
- Include AWS event format documentation

**Outputs**:
- API documentation with endpoints, methods, inputs, and outputs
- AWS event documentation with message formats

## Data Flow

1. User provides repository path
2. Code Parser processes all Go files
3. Route Definition Scanner identifies Echo routes
4. Handler Analysis Engine analyzes each handler function
5. AWS SDK Usage Analyzer identifies SNS/SQS usage
6. Documentation Generator combines results into documentation

## Configuration Options

- Repository path
- Output format (Markdown, OpenAPI, JSON, etc.)
- Analysis depth (basic routes only, detailed handler analysis, etc.)
- File inclusion/exclusion patterns
- Custom Echo instance variable names

## Extension Points

- Support for additional web frameworks
- Support for other AWS services
- Custom documentation templates
- Integration with CI/CD pipelines
