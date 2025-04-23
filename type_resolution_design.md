# Type Resolution System Design

## Overview

This document outlines the design for a type resolution system to enhance the static analyzer tool for Golang Echo applications. The system will analyze variable types in handler functions, particularly focusing on JSON response formats, to generate more detailed API documentation.

## Requirements

1. **Type Resolution Depth**:
   - Handle direct struct fields with their JSON tags
   - Include nested structs and their fields
   - Support arrays and maps
   - Interfaces not required

2. **Package Scope**:
   - Only resolve types within the codebase (no standard library or third-party dependencies)

3. **Documentation Format**:
   - Generate JSON Schema format compatible with OpenAPI

4. **Special Cases**:
   - No need to handle custom JSON marshalers, conditional fields, or generics

## System Components

### 1. Type Registry

**Purpose**: Central repository for storing and retrieving type information.

**Functionality**:
- Store mapping of package paths to their types
- Store mapping of type names to their definitions
- Provide lookup methods for resolving types across packages
- Cache resolved types to improve performance

**Data Structures**:
```go
type TypeRegistry struct {
    // Map of package path to package info
    Packages map[string]*PackageInfo
    
    // Current package being analyzed
    CurrentPackage string
}

type PackageInfo struct {
    // Map of type name to type definition
    Types map[string]*TypeDefinition
    
    // Map of import alias to package path
    Imports map[string]string
}

type TypeDefinition struct {
    Name       string
    Kind       TypeKind // Struct, Array, Map, etc.
    Fields     []*FieldDefinition // For structs
    ElementType *TypeDefinition   // For arrays
    KeyType    *TypeDefinition    // For maps
    ValueType  *TypeDefinition    // For maps
    Package    string             // Package path
}

type FieldDefinition struct {
    Name      string
    Type      *TypeDefinition
    JSONName  string
    Omitempty bool
    IsPointer bool
}

type TypeKind int
const (
    KindStruct TypeKind = iota
    KindArray
    KindMap
    KindBasic
    KindPointer
)
```

### 2. Type Collector

**Purpose**: Scan the codebase to collect type definitions.

**Functionality**:
- Parse all Go files in the codebase
- Extract struct definitions and their fields
- Extract JSON struct tags
- Build a map of type names to their definitions
- Track import statements to resolve cross-package types

**Process**:
1. Scan all packages in the codebase
2. For each package, collect import statements
3. For each file, collect type declarations
4. For struct types, collect field definitions and tags
5. Store all collected information in the Type Registry

### 3. Variable Tracker

**Purpose**: Track variable declarations and assignments in handler functions.

**Functionality**:
- Identify variable declarations in handler functions
- Track variable assignments and reassignments
- Resolve variable types using the Type Registry
- Handle function calls and their return types

**Process**:
1. Scan handler function bodies
2. Identify variable declarations and their initial types
3. Track assignments to update variable types
4. For function calls, resolve the return type
5. Build a symbol table mapping variable names to their types

### 4. Response Analyzer

**Purpose**: Analyze Echo response methods to extract JSON response formats.

**Functionality**:
- Identify calls to `c.JSON()`, `c.JSONPretty()`, etc.
- Extract the variable or expression being returned
- Resolve the type of the response using the Variable Tracker
- Generate JSON schema for the response type

**Process**:
1. Scan for Echo context method calls
2. For JSON response methods, extract the response variable
3. Resolve the variable's type using the Variable Tracker
4. Generate a JSON schema for the resolved type
5. Associate the schema with the corresponding route

### 5. Schema Generator

**Purpose**: Generate JSON Schema from Go type definitions.

**Functionality**:
- Convert Go struct types to JSON Schema objects
- Handle nested structs as nested schema objects
- Convert arrays to JSON Schema array types
- Convert maps to JSON Schema object types with additionalProperties
- Process JSON struct tags to determine property names and required fields

**Process**:
1. For each type, determine the appropriate JSON Schema type
2. For structs, create a properties object with each field
3. For arrays, create an items schema for the element type
4. For maps, create an additionalProperties schema for the value type
5. Use JSON tags to determine property names and required status

## Integration with Existing Components

### Enhanced Handler Analyzer

The existing `HandlerAnalyzer` will be extended to:
- Use the Type Registry to resolve types
- Use the Variable Tracker to track variables in handler functions
- Use the Response Analyzer to extract JSON response formats
- Store the generated JSON schemas with the handler information

### Enhanced Documentation Generator

The existing `DocGenerator` will be extended to:
- Include JSON Schema information in the generated documentation
- Format the schemas according to OpenAPI specification
- Generate example JSON responses based on the schemas

## Implementation Strategy

1. **Phase 1**: Implement the Type Registry and Type Collector
   - Build the core data structures for storing type information
   - Implement the scanning logic to collect type definitions
   - Test with simple struct types

2. **Phase 2**: Implement the Variable Tracker
   - Build the symbol table for tracking variables
   - Implement the logic for resolving variable types
   - Test with simple variable declarations and assignments

3. **Phase 3**: Implement the Response Analyzer
   - Extend the handler analyzer to identify JSON responses
   - Implement the logic for extracting response variables
   - Test with simple JSON responses

4. **Phase 4**: Implement the Schema Generator
   - Build the logic for converting Go types to JSON Schema
   - Handle nested structs, arrays, and maps
   - Test with complex type structures

5. **Phase 5**: Integrate with Existing Components
   - Extend the handler analyzer to use the new components
   - Extend the documentation generator to include JSON schemas
   - Test with real-world Echo applications

## Challenges and Considerations

1. **Type Resolution Complexity**:
   - Handling complex nested types may require recursive algorithms
   - Circular type references need special handling to avoid infinite loops

2. **Cross-Package Resolution**:
   - Resolving types across packages requires careful tracking of imports
   - Package aliases need to be considered when resolving types

3. **Performance**:
   - Analyzing large codebases may be resource-intensive
   - Caching resolved types can improve performance

4. **Accuracy**:
   - Static analysis has inherent limitations in determining runtime types
   - Some complex patterns may not be resolvable through static analysis

## Example Usage

```go
// Example handler function
func getUserByID(c echo.Context) error {
    id := c.Param("id")
    
    // Get user from database
    user := getUserFromDB(id)
    
    // Return JSON response
    return c.JSON(http.StatusOK, user)
}

// Example type definition
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    Profile   *Profile  `json:"profile,omitempty"`
}

type Profile struct {
    Bio    string   `json:"bio,omitempty"`
    Skills []string `json:"skills"`
}
```

Expected JSON Schema output:
```json
{
  "type": "object",
  "properties": {
    "id": { "type": "integer" },
    "name": { "type": "string" },
    "email": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" },
    "profile": {
      "type": "object",
      "properties": {
        "bio": { "type": "string" },
        "skills": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": ["skills"]
    }
  },
  "required": ["id", "name", "created_at"]
}
```
