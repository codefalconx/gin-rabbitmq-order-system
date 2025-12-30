package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/codefalconx/order-system/shared/events"
	"github.com/codefalconx/order-system/shared/rabbitmq"
)

type Order struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	CustomerID  string    `json:"customer_id"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderItem struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type CreateOrderRequest struct {
	CustomerID string                `json:"customer_id" binding:"required"`
	Email      string                `json:"email" binding:"required,email"`
	Phone      string                `json:"phone" binding:"required"`
	Items      []events.OrderItem    `json:"items" binding:"required,min=1"`
}

var (
	db  *gorm.DB
	mq  *rabbitmq.RabbitMQ
)

func main() {
	// Initialize database
	var err error
	dsn := "host=localhost user=postgres password=postgres dbname=order_db port=5436 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	db.AutoMigrate(&Order{}, &OrderItem{})

	// Initialize RabbitMQ
	mq, err = rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer mq.Close()

	// Declare exchange
	err = mq.DeclareExchange("order_events", "fanout")
	if err != nil {
		log.Fatal("Failed to declare exchange:", err)
	}

	// Initialize Gin
	r := gin.Default()

	// Routes
	r.POST("/orders", createOrder)
	r.GET("/orders/:id", getOrder)
	r.GET("/orders", listOrders)

	log.Println("Order Service running on :8080")
	r.Run(":8080")
}

func createOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate total amount
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	// Create order
	order := Order{
		ID:          uuid.New().String(),
		CustomerID:  req.CustomerID,
		Email:       req.Email,
		Phone:       req.Phone,
		TotalAmount: totalAmount,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Create order items
	for _, item := range req.Items {
		orderItem := OrderItem{
			OrderID:   order.ID,
			ProductID: item.ProductID,
			Name:      item.Name,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
		db.Create(&orderItem)
	}

	// Publish event
	event := events.OrderCreatedEvent{
		EventID:     uuid.New().String(),
		EventType:   events.OrderCreated,
		Timestamp:   time.Now(),
		OrderID:     order.ID,
		CustomerID:  order.CustomerID,
		Email:       order.Email,
		Phone:       order.Phone,
		TotalAmount: order.TotalAmount,
		Items:       req.Items,
	}

	if err := mq.Publish("order_events", "", event); err != nil {
		log.Printf("Failed to publish event: %v", err)
	}

	c.JSON(http.StatusCreated, order)
}

func getOrder(c *gin.Context) {
	id := c.Param("id")
	var order Order
	if err := db.First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var items []OrderItem
	db.Where("order_id = ?", id).Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"order": order,
		"items": items,
	})
}

func listOrders(c *gin.Context) {
	var orders []Order
	db.Order("created_at desc").Find(&orders)
	c.JSON(http.StatusOK, orders)
}