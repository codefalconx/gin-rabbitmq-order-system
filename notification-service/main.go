package main

import (
	"encoding/json"
	"fmt"
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

type Notification struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Type       string    `json:"type"` // email, sms
	Recipient  string    `json:"recipient"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	Status     string    `json:"status"` // pending, sent, failed
	CreatedAt  time.Time `json:"created_at"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
}

var db *gorm.DB

func main() {
	// Initialize database
	var err error
	dsn := "host=localhost user=postgres password=postgres dbname=notification_db port=5439 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	db.AutoMigrate(&Notification{})

	// Initialize RabbitMQ
	mq, err := rabbitmq.NewRabbitMQ("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer mq.Close()

	// Declare and bind queue
	queue, err := mq.DeclareQueue("notification_queue")
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
	r.GET("/notifications", listNotifications)
	r.GET("/notifications/:id", getNotification)

	log.Println("Notification Service running on :8083")
	r.Run(":8083")
}

func consumeOrderEvents(mq *rabbitmq.RabbitMQ) {
	msgs, err := mq.Consume("notification_queue")
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

		// Send email notification
		emailNotification := Notification{
			ID:         uuid.New().String(),
			OrderID:    event.OrderID,
			CustomerID: event.CustomerID,
			Type:       "email",
			Recipient:  event.Email,
			Subject:    "Order Confirmation",
			Body:       formatEmailBody(event),
			Status:     "pending",
			CreatedAt:  time.Now(),
		}

		if err := db.Create(&emailNotification).Error; err != nil {
			log.Printf("Failed to create email notification: %v", err)
		} else {
			sendEmail(emailNotification.ID)
		}

		// Send SMS notification
		smsNotification := Notification{
			ID:         uuid.New().String(),
			OrderID:    event.OrderID,
			CustomerID: event.CustomerID,
			Type:       "sms",
			Recipient:  event.Phone,
			Subject:    "",
			Body:       formatSMSBody(event),
			Status:     "pending",
			CreatedAt:  time.Now(),
		}

		if err := db.Create(&smsNotification).Error; err != nil {
			log.Printf("Failed to create SMS notification: %v", err)
		} else {
			sendSMS(smsNotification.ID)
		}

		msg.Ack(false)
	}
}

func formatEmailBody(event events.OrderCreatedEvent) string {
	body := fmt.Sprintf(
		"Dear Customer,\n\n"+
			"Thank you for your order!\n\n"+
			"Order ID: %s\n"+
			"Total Amount: $%.2f\n\n"+
			"Items:\n",
		event.OrderID,
		event.TotalAmount,
	)

	for _, item := range event.Items {
		body += fmt.Sprintf("- %s (x%d) - $%.2f\n", item.Name, item.Quantity, item.Price)
	}

	body += "\nWe'll notify you when your order is shipped.\n\nBest regards,\nYour Store"
	return body
}

func formatSMSBody(event events.OrderCreatedEvent) string {
	return fmt.Sprintf(
		"Order confirmed! Order ID: %s. Total: $%.2f. Thank you for your purchase!",
		event.OrderID,
		event.TotalAmount,
	)
}

func sendEmail(notificationID string) {
	var notification Notification
	if err := db.First(&notification, "id = ?", notificationID).Error; err != nil {
		log.Printf("Notification not found: %v", err)
		return
	}

	// Simulate email sending
	log.Printf("Sending email to: %s", notification.Recipient)
	log.Printf("Subject: %s", notification.Subject)
	log.Printf("Body: %s", notification.Body)

	// Update status
	now := time.Now()
	notification.Status = "sent"
	notification.SentAt = &now
	db.Save(&notification)

	log.Printf("Email sent successfully: %s", notificationID)
}

func sendSMS(notificationID string) {
	var notification Notification
	if err := db.First(&notification, "id = ?", notificationID).Error; err != nil {
		log.Printf("Notification not found: %v", err)
		return
	}

	// Simulate SMS sending
	log.Printf("Sending SMS to: %s", notification.Recipient)
	log.Printf("Message: %s", notification.Body)

	// Update status
	now := time.Now()
	notification.Status = "sent"
	notification.SentAt = &now
	db.Save(&notification)

	log.Printf("SMS sent successfully: %s", notificationID)
}

func listNotifications(c *gin.Context) {
	var notifications []Notification
	db.Order("created_at desc").Find(&notifications)
	c.JSON(http.StatusOK, notifications)
}

func getNotification(c *gin.Context) {
	id := c.Param("id")
	var notification Notification
	if err := db.First(&notification, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}
	c.JSON(http.StatusOK, notification)
}