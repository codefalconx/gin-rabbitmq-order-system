package main

import (
	"encoding/json"
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

type Invoice struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	OrderID     string    `json:"order_id"`
	CustomerID  string    `json:"customer_id"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"` // pending, paid, failed
	CreatedAt   time.Time `json:"created_at"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

type Payment struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	InvoiceID     string    `json:"invoice_id"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

var db *gorm.DB

func main() {
	// Initialize database
	var err error
	dsn := "host=localhost user=postgres password=postgres dbname=billing_db port=5438 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	db.AutoMigrate(&Invoice{}, &Payment{})

	// Initialize RabbitMQ
	mq, err := rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer mq.Close()

	// Declare and bind queue
	queue, err := mq.DeclareQueue("billing_queue")
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
	r.GET("/invoices", listInvoices)
	r.GET("/invoices/:id", getInvoice)
	r.POST("/invoices/:id/pay", processPayment)

	log.Println("Billing Service running on :8082")
	r.Run(":8082")
}

func consumeOrderEvents(mq *rabbitmq.RabbitMQ) {
	msgs, err := mq.Consume("billing_queue")
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

		// Create invoice
		invoice := Invoice{
			ID:         uuid.New().String(),
			OrderID:    event.OrderID,
			CustomerID: event.CustomerID,
			Amount:     event.TotalAmount,
			Status:     "pending",
			CreatedAt:  time.Now(),
		}

		if err := db.Create(&invoice).Error; err != nil {
			log.Printf("Failed to create invoice: %v", err)
			msg.Nack(false, true)
			continue
		}

		log.Printf("Invoice created: %s for order: %s", invoice.ID, event.OrderID)
		msg.Ack(false)
	}
}

func listInvoices(c *gin.Context) {
	var invoices []Invoice
	db.Order("created_at desc").Find(&invoices)
	c.JSON(http.StatusOK, invoices)
}

func getInvoice(c *gin.Context) {
	id := c.Param("id")
	var invoice Invoice
	if err := db.First(&invoice, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}

	var payments []Payment
	db.Where("invoice_id = ?", id).Find(&payments)

	c.JSON(http.StatusOK, gin.H{
		"invoice":  invoice,
		"payments": payments,
	})
}

func processPayment(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		PaymentMethod string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var invoice Invoice
	if err := db.First(&invoice, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}

	if invoice.Status == "paid" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invoice already paid"})
		return
	}

	// Simulate payment processing
	payment := Payment{
		ID:            uuid.New().String(),
		InvoiceID:     invoice.ID,
		Amount:        invoice.Amount,
		PaymentMethod: req.PaymentMethod,
		Status:        "completed",
		CreatedAt:     time.Now(),
	}

	tx := db.Begin()
	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment processing failed"})
		return
	}

	now := time.Now()
	invoice.Status = "paid"
	invoice.PaidAt = &now
	if err := tx.Save(&invoice).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update invoice"})
		return
	}

	tx.Commit()

	log.Printf("Payment processed for invoice: %s", invoice.ID)
	c.JSON(http.StatusOK, gin.H{
		"invoice": invoice,
		"payment": payment,
	})
}