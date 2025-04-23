package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Profile   *Profile  `json:"profile,omitempty"`
}

// Profile represents a user profile
type Profile struct {
	Bio    string   `json:"bio,omitempty"`
	Skills []string `json:"skills"`
}

// Address represents a physical address
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

// Product represents a product in the system
type Product struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Price       float64           `json:"price"`
	Categories  []string          `json:"categories"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Inventory   *ProductInventory `json:"inventory,omitempty"`
}

// ProductInventory represents inventory information for a product
type ProductInventory struct {
	Quantity  int  `json:"quantity"`
	Available bool `json:"available"`
}

// Order represents a customer order
type Order struct {
	ID              int         `json:"id"`
	UserID          int         `json:"user_id"`
	Items           []OrderItem `json:"items"`
	TotalPrice      float64     `json:"total_price"`
	Status          string      `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	ShippingAddress Address     `json:"shipping_address"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

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

	// Product routes
	e.GET("/products", getProducts)
	e.GET("/products/:id", getProductByID)
	e.POST("/products", createProduct)
	e.PUT("/products/:id", updateProduct)

	// Order routes
	e.GET("/orders", getOrders)
	e.GET("/orders/:id", getOrderByID)
	e.POST("/orders", createOrder)
	e.PUT("/orders/:id/status", updateOrderStatus)

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
	users := []User{
		{
			ID:        1,
			Name:      "John Doe",
			Email:     "john@example.com",
			CreatedAt: time.Now(),
			Profile: &Profile{
				Bio:    "Software Engineer",
				Skills: []string{"Go", "JavaScript", "Docker"},
			},
		},
		{
			ID:        2,
			Name:      "Jane Smith",
			Email:     "jane@example.com",
			CreatedAt: time.Now(),
		},
	}

	return c.JSON(http.StatusOK, users)
}

func getUserByID(c echo.Context) error {
	// Path parameter
	id := c.Param("id")

	// Mock data
	user := User{
		ID:        1,
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
		Profile: &Profile{
			Bio:    "Software Engineer",
			Skills: []string{"Go", "JavaScript", "Docker"},
		},
	}

	return c.JSON(http.StatusOK, user)
}

func createUser(c echo.Context) error {
	// Bind request body
	user := new(User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "InvalidRequest",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Mock response
	user.ID = 123
	user.CreatedAt = time.Now()

	return c.JSON(http.StatusCreated, user)
}

func updateUser(c echo.Context) error {
	// Path parameter
	id := c.Param("id")

	// Bind request body
	user := new(User)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "InvalidRequest",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Set ID from path
	user.ID = 1

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
	products := []Product{
		{
			ID:          1,
			Name:        "Product 1",
			Description: "This is product 1",
			Price:       19.99,
			Categories:  []string{"Electronics", "Gadgets"},
			Attributes: map[string]string{
				"color": "black",
				"size":  "medium",
			},
			Inventory: &ProductInventory{
				Quantity:  100,
				Available: true,
			},
		},
		{
			ID:          2,
			Name:        "Product 2",
			Description: "This is product 2",
			Price:       29.99,
			Categories:  []string{"Home", "Kitchen"},
		},
	}

	return c.JSON(http.StatusOK, products)
}

func getProductByID(c echo.Context) error {
	// Path parameter
	id := c.Param("id")

	// Mock data
	product := Product{
		ID:          1,
		Name:        "Product 1",
		Description: "This is product 1",
		Price:       19.99,
		Categories:  []string{"Electronics", "Gadgets"},
		Attributes: map[string]string{
			"color": "black",
			"size":  "medium",
		},
		Inventory: &ProductInventory{
			Quantity:  100,
			Available: true,
		},
	}

	return c.JSON(http.StatusOK, product)
}

func createProduct(c echo.Context) error {
	// Bind request body
	product := new(Product)
	if err := c.Bind(product); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "InvalidRequest",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Mock response
	product.ID = 123

	// Send SNS notification
	sendProductCreatedEvent(product)

	return c.JSON(http.StatusCreated, product)
}

func updateProduct(c echo.Context) error {
	// Path parameter
	id := c.Param("id")

	// Bind request body
	product := new(Product)
	if err := c.Bind(product); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "InvalidRequest",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Set ID from path
	product.ID = 1

	return c.JSON(http.StatusOK, product)
}

func getOrders(c echo.Context) error {
	// Query parameters
	status := c.QueryParam("status")

	// Mock data
	orders := []Order{
		{
			ID:     1,
			UserID: 1,
			Items: []OrderItem{
				{
					ProductID: 1,
					Quantity:  2,
					Price:     19.99,
				},
				{
					ProductID: 2,
					Quantity:  1,
					Price:     29.99,
				},
			},
			TotalPrice: 69.97,
			Status:     "pending",
			CreatedAt:  time.Now(),
			ShippingAddress: Address{
				Street:  "123 Main St",
				City:    "Anytown",
				State:   "CA",
				ZipCode: "12345",
				Country: "USA",
			},
		},
		{
			ID:     2,
			UserID: 2,
			Items: []OrderItem{
				{
					ProductID: 3,
					Quantity:  1,
					Price:     49.99,
				},
			},
			TotalPrice: 49.99,
			Status:     "shipped",
			CreatedAt:  time.Now(),
			ShippingAddress: Address{
				Street:  "456 Oak Ave",
				City:    "Somewhere",
				State:   "NY",
				ZipCode: "67890",
				Country: "USA",
			},
		},
	}

	return c.JSON(http.StatusOK, orders)
}

func getOrderByID(c echo.Context) error {
	// Path parameter
	id := c.Param("id")

	// Mock data
	order := Order{
		ID:     1,
		UserID: 1,
		Items: []OrderItem{
			{
				ProductID: 1,
				Quantity:  2,
				Price:     19.99,
			},
			{
				ProductID: 2,
				Quantity:  1,
				Price:     29.99,
			},
		},
		TotalPrice: 69.97,
		Status:     "pending",
		CreatedAt:  time.Now(),
		ShippingAddress: Address{
			Street:  "123 Main St",
			City:    "Anytown",
			State:   "CA",
			ZipCode: "12345",
			Country: "USA",
		},
	}

	return c.JSON(http.StatusOK, order)
}

func createOrder(c echo.Context) error {
	// Bind request body
	order := new(Order)
	if err := c.Bind(order); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "InvalidRequest",
			Message: "Invalid request body",
			Code:    400,
		})
	}

	// Mock response
	order.ID = 123
	order.CreatedAt = time.Now()
	order.Status = "pending"

	// Calculate total price
	var totalPrice float64
	for _, item := range order.Items {
		totalPrice += item.Price * float64(item.Quantity)
	}
	order.TotalPrice = totalPrice

	// Send SNS notification
	sendOrderCreatedEvent(order)

	return c.JSON(http.StatusCreated, order)
}

func updateOrderStatus(c echo.Context) error {
	// Path parameter
	id := c.Param("id")

	// Query parameter
	status := c.QueryParam("status")

	// Mock data
	order := Order{
		ID:     1,
		UserID: 1,
		Items: []OrderItem{
			{
				ProductID: 1,
				Quantity:  2,
				Price:     19.99,
			},
			{
				ProductID: 2,
				Quantity:  1,
				Price:     29.99,
			},
		},
		TotalPrice: 69.97,
		Status:     status,
		CreatedAt:  time.Now(),
		ShippingAddress: Address{
			Street:  "123 Main St",
			City:    "Anytown",
			State:   "CA",
			ZipCode: "12345",
			Country: "USA",
		},
	}

	return c.JSON(http.StatusOK, order)
}

// AWS SNS/SQS example
func sendProductCreatedEvent(product *Product) {
	// Create SNS client
	snsClient := sns.New(session.New())

	// Create message
	message, _ := json.Marshal(map[string]interface{}{
		"event":   "product_created",
		"product": product,
	})

	// Publish to SNS topic
	_, err := snsClient.Publish(&sns.PublishInput{
		TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:product-events"),
		Message:  aws.String(string(message)),
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

// Send order created event
func sendOrderCreatedEvent(order *Order) {
	// Create SNS client
	snsClient := sns.New(session.New())

	// Create message
	message, _ := json.Marshal(map[string]interface{}{
		"event": "order_created",
		"order": order,
	})

	// Publish to SNS topic
	_, err := snsClient.Publish(&sns.PublishInput{
		TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:order-events"),
		Message:  aws.String(string(message)),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"event_type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("order_created"),
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
