package events

import "time"

// EventType represents the type of event
type EventType string

const (
	OrderCreated EventType = "order.created"
	OrderUpdated EventType = "order.updated"
	OrderCancelled EventType = "order.cancelled"
)

// OrderCreatedEvent represents an order creation event
type OrderCreatedEvent struct {
	EventID     string    `json:"event_id"`
	EventType   EventType `json:"event_type"`
	Timestamp   time.Time `json:"timestamp"`
	OrderID     string    `json:"order_id"`
	CustomerID  string    `json:"customer_id"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	TotalAmount float64   `json:"total_amount"`
	Items       []OrderItem `json:"items"`
}

// OrderItem represents an item in the order
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}