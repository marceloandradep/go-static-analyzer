package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Simple Echo application for testing the analyzer
func main() {
	// Create a new Echo instance
	e := echo.New()

	// Routes
	e.GET("/", helloWorld)
	e.GET("/users", getUsers)
	e.GET("/users/:id", getUserByID)
	e.POST("/users", createUser)
	e.PUT("/users/:id", updateUser)
	e.DELETE("/users/:id", deleteUser)

	// Group routes
	api := e.Group("/api")
	{
		api.GET("/products", getProducts)
		api.GET("/products/:id", getProductByID)
		api.POST("/products", createProduct)
	}

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}

// Handler functions
func helloWorld(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func getUsers(c echo.Context) error {
	// Query parameters
	limit := c.QueryParam("limit")
	offset := c.QueryParam("offset")
	
	// Mock data
	users := []map[string]interface{}{
		{"id": 1, "name": "John Doe", "email": "john@example.com"},
		{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
	}
	
	return c.JSON(http.StatusOK, users)
}

func getUserByID(c echo.Context) error {
	// Path parameter
	id := c.Param("id")
	
	// Mock data
	user := map[string]interface{}{
		"id":    id,
		"name":  "John Doe",
		"email": "john@example.com",
	}
	
	return c.JSON(http.StatusOK, user)
}

func createUser(c echo.Context) error {
	// Bind request body
	user := new(User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	
	// Mock response
	user.ID = 123
	
	return c.JSON(http.StatusCreated, user)
}

func updateUser(c echo.Context) error {
	// Path parameter
	id := c.Param("id")
	
	// Bind request body
	user := new(User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	
	// Set ID from path
	user.ID = id
	
	return c.JSON(http.StatusOK, user)
}

func deleteUser(c echo.Context) error {
	// Path parameter
	id := c.Param("id")
	
	return c.NoContent(http.StatusNoContent)
}

func getProducts(c echo.Context) error {
	// Query parameters
	category := c.QueryParam("category")
	
	// Mock data
	products := []map[string]interface{}{
		{"id": 1, "name": "Product 1", "price": 19.99},
		{"id": 2, "name": "Product 2", "price": 29.99},
	}
	
	return c.JSON(http.StatusOK, products)
}

func getProductByID(c echo.Context) error {
	// Path parameter
	id := c.Param("id")
	
	// Mock data
	product := map[string]interface{}{
		"id":    id,
		"name":  "Product 1",
		"price": 19.99,
	}
	
	return c.JSON(http.StatusOK, product)
}

func createProduct(c echo.Context) error {
	// Bind request body
	product := new(Product)
	if err := c.Bind(product); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}
	
	// Mock response
	product.ID = 123
	
	// Send SNS notification
	sendProductCreatedEvent(product)
	
	return c.JSON(http.StatusCreated, product)
}

// AWS SNS/SQS example
func sendProductCreatedEvent(product *Product) {
	// Create SNS client
	snsClient := sns.New(session.New())
	
	// Create message
	message := fmt.Sprintf(`{"event":"product_created","product":{"id":%d,"name":"%s","price":%f}}`, 
		product.ID, product.Name, product.Price)
	
	// Publish to SNS topic
	_, err := snsClient.Publish(&sns.PublishInput{
		TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:product-events"),
		Message:  aws.String(message),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("product_created"),
			},
		},
	})
	
	if err != nil {
		fmt.Println("Error publishing to SNS:", err)
	}
}

// Send message to SQS
func sendToQueue(message string) {
	// Create SQS client
	sqsClient := sqs.New(session.New())
	
	// Send message to SQS queue
	_, err := sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String("https://sqs.us-east-1.amazonaws.com/123456789012/product-queue"),
		MessageBody: aws.String(message),
		MessageAttributes: map[string]*sqs.MessageAttributeValue{
			"source": {
				DataType:    aws.String("String"),
				StringValue: aws.String("product-service"),
			},
		},
	})
	
	if err != nil {
		fmt.Println("Error sending to SQS:", err)
	}
}

// Data models
type User struct {
	ID    interface{} `json:"id,omitempty"`
	Name  string      `json:"name" validate:"required"`
	Email string      `json:"email" validate:"required,email"`
}

type Product struct {
	ID    interface{} `json:"id,omitempty"`
	Name  string      `json:"name" validate:"required"`
	Price float64     `json:"price" validate:"required,gt=0"`
}
