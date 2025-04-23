package aws

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// EventInfo represents information about an AWS event
type EventInfo struct {
	Service       string         // AWS service (SNS, SQS)
	Operation     string         // Operation (Publish, SendMessage)
	TopicOrQueue  string         // Topic ARN or Queue URL/name
	MessageFormat MessageFormat  // Message format details
	Position      token.Position // Position in source code
}

// MessageFormat represents the format of a message
type MessageFormat struct {
	Fields       []MessageField // Fields in the message
	RawMessage   string         // Raw message if available
	IsStructured bool           // Whether the message is structured
}

// MessageField represents a field in a message
type MessageField struct {
	Name        string // Field name
	Type        string // Field type
	Description string // Description from comments if available
}

// AWSAnalyzer analyzes AWS SDK usage for SNS/SQS
type AWSAnalyzer struct {
	FileSet       *token.FileSet
	Events        []EventInfo
	Verbose       bool
	awsClientVars map[string]string // Maps variable names to AWS service types
}

// NewAWSAnalyzer creates a new AWSAnalyzer
func NewAWSAnalyzer(fset *token.FileSet, verbose bool) *AWSAnalyzer {
	return &AWSAnalyzer{
		FileSet:       fset,
		Events:        []EventInfo{},
		Verbose:       verbose,
		awsClientVars: make(map[string]string),
	}
}

// Analyze analyzes files for AWS SDK usage
func (a *AWSAnalyzer) Analyze(files []*ast.File) error {
	if a.Verbose {
		fmt.Println("Analyzing AWS SDK usage...")
	}

	for _, file := range files {
		// First pass: identify AWS client variables
		a.identifyAWSClients(file)

		// Second pass: find AWS operations
		a.findAWSOperations(file)
	}

	if a.Verbose {
		fmt.Printf("Found %d AWS events\n", len(a.Events))
	}

	return nil
}

// identifyAWSClients finds variables that are AWS service clients
func (a *AWSAnalyzer) identifyAWSClients(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for variable assignments
		if assign, ok := n.(*ast.AssignStmt); ok {
			for i, rhs := range assign.Rhs {
				// Check if right side is a call to an AWS client constructor
				if call, ok := rhs.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok {
							// Check for AWS client creation patterns
							service := a.getAWSService(ident.Name, sel.Sel.Name)
							if service != "" && i < len(assign.Lhs) {
								if lhsIdent, ok := assign.Lhs[i].(*ast.Ident); ok {
									if a.Verbose {
										fmt.Printf("  Found AWS client: %s (%s)\n", lhsIdent.Name, service)
									}
									a.awsClientVars[lhsIdent.Name] = service
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

// getAWSService determines if a function call creates an AWS service client
func (a *AWSAnalyzer) getAWSService(pkgName, funcName string) string {
	// Check for AWS SDK v1 patterns
	if pkgName == "sns" && funcName == "New" {
		return "SNS"
	}
	if pkgName == "sqs" && funcName == "New" {
		return "SQS"
	}

	// Check for AWS SDK v2 patterns
	if pkgName == "sns" && funcName == "NewClient" {
		return "SNS"
	}
	if pkgName == "sqs" && funcName == "NewClient" {
		return "SQS"
	}

	return ""
}

// findAWSOperations finds AWS operations (SNS Publish, SQS SendMessage, etc.)
func (a *AWSAnalyzer) findAWSOperations(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for method calls
		if expr, ok := n.(*ast.CallExpr); ok {
			if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Check if this is a call on an AWS client
					if service, exists := a.awsClientVars[ident.Name]; exists {
						// Check for specific AWS operations
						if operation := a.getAWSOperation(service, sel.Sel.Name); operation != "" {
							// This is an AWS operation
							event := EventInfo{
								Service:   service,
								Operation: operation,
								Position:  a.FileSet.Position(expr.Pos()),
							}

							// Extract topic/queue and message format
							if service == "SNS" {
								a.extractSNSDetails(expr, &event)
							} else if service == "SQS" {
								a.extractSQSDetails(expr, &event)
							}

							a.Events = append(a.Events, event)

							if a.Verbose {
								fmt.Printf("  Found AWS operation: %s %s -> %s\n",
									event.Service, event.Operation, event.TopicOrQueue)
							}
						}
					}
				}
			}
		}
		return true
	})
}

// getAWSOperation determines if a method call is an AWS operation of interest
func (a *AWSAnalyzer) getAWSOperation(service, methodName string) string {
	if service == "SNS" {
		switch methodName {
		case "Publish", "PublishWithContext", "PublishRequest":
			return "Publish"
		}
	} else if service == "SQS" {
		switch methodName {
		case "SendMessage", "SendMessageWithContext", "SendMessageRequest":
			return "SendMessage"
		case "SendMessageBatch", "SendMessageBatchWithContext", "SendMessageBatchRequest":
			return "SendMessageBatch"
		}
	}
	return ""
}

// extractSNSDetails extracts details from an SNS Publish call
func (a *AWSAnalyzer) extractSNSDetails(call *ast.CallExpr, event *EventInfo) {
	// Check for different patterns of SNS Publish calls

	// Pattern 1: Direct args - client.Publish(input)
	if len(call.Args) == 1 {
		if arg, ok := call.Args[0].(*ast.CompositeLit); ok {
			a.extractSNSPublishInput(arg, event)
		}
	}

	// Pattern 2: With context - client.PublishWithContext(ctx, input)
	if len(call.Args) == 2 {
		if arg, ok := call.Args[1].(*ast.CompositeLit); ok {
			a.extractSNSPublishInput(arg, event)
		}
	}
}

// extractSNSPublishInput extracts details from an SNS PublishInput
func (a *AWSAnalyzer) extractSNSPublishInput(lit *ast.CompositeLit, event *EventInfo) {
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok {
				switch key.Name {
				case "TopicArn":
					event.TopicOrQueue = a.extractStringValue(kv.Value)
				case "Message":
					event.MessageFormat.RawMessage = a.extractStringValue(kv.Value)
				case "MessageAttributes":
					a.extractMessageAttributes(kv.Value, &event.MessageFormat)
				}
			}
		}
	}
}

// extractSQSDetails extracts details from an SQS SendMessage call
func (a *AWSAnalyzer) extractSQSDetails(call *ast.CallExpr, event *EventInfo) {
	// Check for different patterns of SQS SendMessage calls

	// Pattern 1: Direct args - client.SendMessage(input)
	if len(call.Args) == 1 {
		if arg, ok := call.Args[0].(*ast.CompositeLit); ok {
			a.extractSQSSendMessageInput(arg, event)
		}
	}

	// Pattern 2: With context - client.SendMessageWithContext(ctx, input)
	if len(call.Args) == 2 {
		if arg, ok := call.Args[1].(*ast.CompositeLit); ok {
			a.extractSQSSendMessageInput(arg, event)
		}
	}
}

// extractSQSSendMessageInput extracts details from an SQS SendMessageInput
func (a *AWSAnalyzer) extractSQSSendMessageInput(lit *ast.CompositeLit, event *EventInfo) {
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok {
				switch key.Name {
				case "QueueUrl":
					event.TopicOrQueue = a.extractStringValue(kv.Value)
				case "MessageBody":
					event.MessageFormat.RawMessage = a.extractStringValue(kv.Value)
				case "MessageAttributes":
					a.extractMessageAttributes(kv.Value, &event.MessageFormat)
				}
			}
		}
	}
}

// extractMessageAttributes extracts message attributes from an expression
func (a *AWSAnalyzer) extractMessageAttributes(expr ast.Expr, format *MessageFormat) {
	// Handle composite literals (map[string]*MessageAttributeValue{...})
	if lit, ok := expr.(*ast.CompositeLit); ok {
		for _, elt := range lit.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				fieldName := a.extractStringValue(kv.Key)
				fieldType := "string" // Default type

				// Try to determine the actual type
				if valueLit, ok := kv.Value.(*ast.CompositeLit); ok {
					for _, valueElt := range valueLit.Elts {
						if valueKV, ok := valueElt.(*ast.KeyValueExpr); ok {
							if key, ok := valueKV.Key.(*ast.Ident); ok {
								if key.Name == "DataType" {
									fieldType = a.extractStringValue(valueKV.Value)
								}
							}
						}
					}
				}

				format.Fields = append(format.Fields, MessageField{
					Name: fieldName,
					Type: fieldType,
				})

				format.IsStructured = true
			}
		}
	}
}

// extractStringValue extracts a string value from an expression
func (a *AWSAnalyzer) extractStringValue(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind == token.STRING {
			return strings.Trim(v.Value, "\"'`")
		}
	case *ast.Ident:
		return v.Name // Variable name
	}
	return ""
}

// GetEvents returns all found AWS events
func (a *AWSAnalyzer) GetEvents() []EventInfo {
	return a.Events
}
