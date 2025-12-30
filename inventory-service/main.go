package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/codefalconx/order-system/shared/events"
	"github.com/codefalconx/order-system/shared/rabbitmq"
)

type Product struct {
	ID       string `gorm:"primaryKey" json:"id"`
	Name     string `json:"name"`
	Stock    int    `json:"stock"`
	Reserved int    `json:"reserved"`
}

type StockHistory struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Action    string `json:"action"`
}

var db *gorm.DB

func main() {
	// Initialize database
	var err error
	dsn := "host=localhost user=postgres password=postgres dbname=inventory_db port=5437 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	db.AutoMigrate(&Product{}, &StockHistory{})

	// Seed initial products
	seedProducts()

	// Initialize RabbitMQ
	mq, err := rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer mq.Close()

	// Declare and bind queue
	queue, err := mq.DeclareQueue("inventory_queue")
	if err != nil {
		log.Fatal("Failed to declare queue:", err)
	}

	err = mq.BindQueue(queue.Name, "order_events", "")
	if err != nil {
		log.Fatal("Failed to bind queue:", err)
	}

	// Start consuming messages
	go consumeOrderEvents(mq)

	// Initialize Gin
	r := gin.Default()

	// Routes
	r.GET("/products", listProducts)
	r.GET("/products/:id", getProduct)
	r.POST("/products", createProduct)
	r.PUT("/products/:id/stock", updateStock)

	log.Println("Inventory Service running on :8081")
	r.Run(":8081")
}

func seedProducts() {
	products := []Product{
		{ID: "prod-1", Name: "Laptop", Stock: 50, Reserved: 0},
		{ID: "prod-2", Name: "Mouse", Stock: 200, Reserved: 0},
		{ID: "prod-3", Name: "Keyboard", Stock: 150, Reserved: 0},
	}

	for _, p := range products {
		db.FirstOrCreate(&p, Product{ID: p.ID})
	}
}

func consumeOrderEvents(mq *rabbitmq.RabbitMQ) {
	msgs, err := mq.Consume("inventory_queue")
	if err != nil {
		log.Fatal("Failed to consume messages:", err)
	}

	log.Println("Waiting for order events...")

	for msg := range msgs {
		var event events.OrderCreatedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			msg.Nack(false, false)
			continue
		}

		log.Printf("Processing order event: %s", event.OrderID)

		// Reserve stock for each item
		tx := db.Begin()
		success := true

		for _, item := range event.Items {
			var product Product
			if err := tx.First(&product, "id = ?", item.ProductID).Error; err != nil {
				log.Printf("Product not found: %s", item.ProductID)
				success = false
				break
			}

			if product.Stock-product.Reserved < item.Quantity {
				log.Printf("Insufficient stock for product: %s", item.ProductID)
				success = false
				break
			}

			// Update reserved stock
			product.Reserved += item.Quantity
			tx.Save(&product)

			// Record history
			history := StockHistory{
				OrderID:   event.OrderID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Action:    "reserved",
			}
			tx.Create(&history)
		}

		if success {
			tx.Commit()
			log.Printf("Successfully reserved stock for order: %s", event.OrderID)
			msg.Ack(false)
		} else {
			tx.Rollback()
			log.Printf("Failed to reserve stock for order: %s", event.OrderID)
			msg.Nack(false, true)
		}
	}
}

func listProducts(c *gin.Context) {
	var products []Product
	db.Find(&products)
	c.JSON(http.StatusOK, products)
}

func getProduct(c *gin.Context) {
	id := c.Param("id")
	var product Product
	if err := db.First(&product, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	c.JSON(http.StatusOK, product)
}

func createProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := db.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, product)
}

func updateStock(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Stock int `json:"stock" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var product Product
	if err := db.First(&product, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	product.Stock = req.Stock
	db.Save(&product)

	c.JSON(http.StatusOK, product)
}