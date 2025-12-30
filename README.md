# Order Management Microservices System

A complete event-driven microservices architecture built with Go, Gin, RabbitMQ, and PostgreSQL. This system demonstrates how to build scalable, decoupled services that communicate through message queues.

![Architecture](https://img.shields.io/badge/Architecture-Microservices-blue)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![RabbitMQ](https://img.shields.io/badge/RabbitMQ-FF6600?logo=rabbitmq&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?logo=postgresql&logoColor=white)

## 📋 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Prerequisites](#-prerequisites)
- [Installation](#-installation)
- [Usage](#-usage)
- [API Endpoints](#-api-endpoints)
- [Project Structure](#-project-structure)
- [How It Works](#-how-it-works)
- [Testing](#-testing)
- [Monitoring](#-monitoring)
- [Troubleshooting](#-troubleshooting)
- [Contributing](#-contributing)

## ✨ Features

- **Event-Driven Architecture**: Decoupled services communicating via RabbitMQ
- **Microservices Pattern**: Four independent services with their own databases
- **Database Per Service**: Each service has its own PostgreSQL database
- **Fanout Exchange**: Broadcast events to multiple consumers simultaneously
- **RESTful APIs**: Clean REST endpoints using Gin framework
- **Transaction Management**: ACID compliance with GORM transactions
- **Manual Acknowledgment**: Reliable message processing
- **Error Handling**: Comprehensive error handling and logging
- **Docker Support**: Easy setup with Docker Compose

## 🏗️ Architecture

```
┌─────────────────┐
│  Order Service  │ (Port 8080)
│   (order_db)    │
└────────┬────────┘
         │ Publishes "Order Created"
         ▼
┌─────────────────────────────┐
│   RabbitMQ Exchange         │
│   (order_events - fanout)   │
└──┬──────────┬──────────┬────┘
   │          │          │
   ▼          ▼          ▼
┌──────┐  ┌──────┐  ┌─────────┐
│Inv Q │  │Bill Q│  │Notif Q  │
└──┬───┘  └──┬───┘  └────┬────┘
   │         │           │
   ▼         ▼           ▼
┌─────────┐ ┌─────────┐ ┌──────────┐
│Inventory│ │ Billing │ │Notification│
│ Service │ │ Service │ │  Service  │
│  :8081  │ │  :8082  │ │   :8083   │
│inventory│ │billing  │ │notification│
│   _db   │ │  _db    │ │    _db    │
└─────────┘ └─────────┘ └───────────┘
```

### Services

1. **Order Service** (Port 8080)
    - Creates and manages customer orders
    - Publishes order events to RabbitMQ
    - Database: `order_db` (Port 5436)

2. **Inventory Service** (Port 8081)
    - Manages product stock levels
    - Reserves inventory when orders are created
    - Tracks stock history
    - Database: `inventory_db` (Port 5437)

3. **Billing Service** (Port 8082)
    - Generates invoices for orders
    - Processes payments
    - Tracks payment history
    - Database: `billing_db` (Port 5438)

4. **Notification Service** (Port 8083)
    - Sends email confirmations
    - Sends SMS alerts
    - Logs all notifications
    - Database: `notification_db` (Port 5439)

## 📦 Prerequisites

Before you begin, ensure you have the following installed:

- **Go** 1.21 or higher ([Download](https://golang.org/dl/))
- **Docker** and **Docker Compose** ([Download](https://docs.docker.com/get-docker/))
- **Git** ([Download](https://git-scm.com/downloads))
- **curl** or **Postman** (for testing APIs)

## 🚀 Installation

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/order-microservices.git
cd order-microservices
```

### 2. Project Structure Setup

Create the following directory structure:

```
order-system/
├── docker-compose.yml
├── shared/
│   ├── events/
│   │   └── events.go
│   └── rabbitmq/
│       └── rabbitmq.go
├── order-service/
│   └── main.go
├── inventory-service/
│   └── main.go
├── billing-service/
│   └── main.go
└── notification-service/
    └── main.go
```

### 3. Initialize Go Modules

In each service directory, initialize Go modules:

```bash
# For shared module
cd shared
go mod init github.com/yourusername/order-system/shared
go mod tidy

# For order-service
cd ../order-service
go mod init github.com/yourusername/order-system/order-service
go get -u github.com/gin-gonic/gin
go get -u gorm.io/gorm
go get -u gorm.io/driver/postgres
go get -u github.com/google/uuid
go get -u github.com/rabbitmq/amqp091-go
go mod tidy

# Repeat for other services (inventory-service, billing-service, notification-service)
```

**Note**: Replace `github.com/yourusername/order-system` with your actual module path in all `main.go` files.

### 4. Start Infrastructure

```bash
# Start RabbitMQ and PostgreSQL databases
docker-compose up -d

# Check if containers are running
docker ps
```

Expected output:
```
CONTAINER ID   IMAGE              STATUS          PORTS
abc123def456   rabbitmq:3-mgmt   Up 2 minutes    0.0.0.0:5672->5672/tcp, 0.0.0.0:15672->15672/tcp
def456ghi789   postgres:15        Up 2 minutes    0.0.0.0:5432->5432/tcp
...
```

### 5. Verify RabbitMQ

Open browser and navigate to: `http://localhost:15672`

- **Username**: `guest`
- **Password**: `guest`

## 💻 Usage

### Start All Services

Open **4 separate terminal windows** and run:

**Terminal 1 - Order Service:**
```bash
cd order-service
go run main.go
```

**Terminal 2 - Inventory Service:**
```bash
cd inventory-service
go run main.go
```

**Terminal 3 - Billing Service:**
```bash
cd billing-service
go run main.go
```

**Terminal 4 - Notification Service:**
```bash
cd notification-service
go run main.go
```

You should see output like:
```
Order Service running on :8080
Inventory Service running on :8081
Billing Service running on :8082
Notification Service running on :8083
Waiting for order events...
```

## 📡 API Endpoints

### Order Service (http://localhost:8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/orders` | Create a new order |
| GET | `/orders` | List all orders |
| GET | `/orders/:id` | Get order details |

### Inventory Service (http://localhost:8081)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/products` | List all products |
| GET | `/products/:id` | Get product details |
| POST | `/products` | Create a new product |
| PUT | `/products/:id/stock` | Update product stock |

### Billing Service (http://localhost:8082)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/invoices` | List all invoices |
| GET | `/invoices/:id` | Get invoice details |
| POST | `/invoices/:id/pay` | Process payment |

### Notification Service (http://localhost:8083)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/notifications` | List all notifications |
| GET | `/notifications/:id` | Get notification details |

## 📁 Project Structure

```
order-system/
├── docker-compose.yml          # Infrastructure setup
├── README.md                   # This file
│
├── shared/                     # Shared libraries
│   ├── events/
│   │   └── events.go          # Event definitions
│   └── rabbitmq/
│       └── rabbitmq.go        # RabbitMQ client
│
├── order-service/
│   ├── main.go                # Order service implementation
│   └── go.mod
│
├── inventory-service/
│   ├── main.go                # Inventory service implementation
│   └── go.mod
│
├── billing-service/
│   ├── main.go                # Billing service implementation
│   └── go.mod
│
└── notification-service/
    ├── main.go                # Notification service implementation
    └── go.mod
```

## 🔄 How It Works

### Event Flow

1. **Client** sends POST request to Order Service with order details
2. **Order Service**:
    - Validates the request
    - Saves order to `order_db`
    - Publishes `OrderCreatedEvent` to RabbitMQ `order_events` exchange
3. **RabbitMQ** broadcasts the event to all bound queues (fanout pattern)
4. **Services consume events in parallel**:
    - **Inventory Service**: Reserves stock for ordered items
    - **Billing Service**: Creates an invoice for the order
    - **Notification Service**: Sends email and SMS notifications
5. Each service updates its own database independently

### Message Flow Example

```
Order Created → RabbitMQ Exchange → 3 Queues → 3 Services → 3 Databases
```

## 🧪 Testing

### 1. Create an Order

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-123",
    "email": "customer@example.com",
    "phone": "+1234567890",
    "items": [
      {
        "product_id": "prod-1",
        "name": "Laptop",
        "quantity": 2,
        "price": 999.99
      },
      {
        "product_id": "prod-2",
        "name": "Mouse",
        "quantity": 1,
        "price": 29.99
      }
    ]
  }'
```

**Expected Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust-123",
  "email": "customer@example.com",
  "phone": "+1234567890",
  "total_amount": 2029.97,
  "status": "pending",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### 2. Verify Order Created

```bash
curl http://localhost:8080/orders
```

### 3. Check Inventory Was Reserved

```bash
curl http://localhost:8081/products
```

Look for the `reserved` field - it should be updated.

### 4. Check Invoice Was Created

```bash
curl http://localhost:8082/invoices
```

### 5. Check Notifications Were Sent

```bash
curl http://localhost:8083/notifications
```

You should see both email and SMS notifications with status "sent".

### 6. Process Payment

```bash
# Get invoice_id from step 4
curl -X POST http://localhost:8082/invoices/{invoice_id}/pay \
  -H "Content-Type: application/json" \
  -d '{
    "payment_method": "credit_card"
  }'
```

### Complete Test Script

```bash
#!/bin/bash

echo "Creating order..."
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-123",
    "email": "customer@example.com",
    "phone": "+628112345678",
    "items": [
      {"product_id": "prod-1", "name": "Laptop", "quantity": 1, "price": 999.99}
    ]
  }')

echo "Order created: $ORDER_RESPONSE"

sleep 2

echo -e "\n\nChecking products..."
curl -s http://localhost:8081/products | jq '.'

echo -e "\n\nChecking invoices..."
curl -s http://localhost:8082/invoices | jq '.'

echo -e "\n\nChecking notifications..."
curl -s http://localhost:8083/notifications | jq '.'
```

## 📊 Monitoring

### RabbitMQ Management UI

Access: `http://localhost:15672`

Monitor:
- Exchange `order_events`
- Queues: `inventory_queue`, `billing_queue`, `notification_queue`
- Message rates and delivery status

### Service Logs

Each service outputs logs to the console:

```
Order Service:
[GIN] 2024/01/15 - 10:30:00 | 201 | POST /orders
Published message to exchange: order_events

Inventory Service:
Processing order event: 550e8400-e29b-41d4-a716-446655440000
Successfully reserved stock for order: 550e8400-...

Billing Service:
Invoice created: 7c9e6679-7425-40de-944b-e07fc1f90ae7

Notification Service:
Sending email to: customer@example.com
Email sent successfully: 9b3e6679-...
```

### Database Access

Connect to databases using any PostgreSQL client:

```bash
# Order database
psql -h localhost -p 5436 -U postgres -d order_db

# Inventory database
psql -h localhost -p 5437 -U postgres -d inventory_db

# Billing database
psql -h localhost -p 5438 -U postgres -d billing_db

# Notification database
psql -h localhost -p 5439 -U postgres -d notification_db
```

Password: `postgres`

## 🔧 Troubleshooting

### Services Can't Connect to RabbitMQ

**Problem**: Connection refused to RabbitMQ

**Solution**:
```bash
# Check RabbitMQ logs
docker logs rabbitmq

# Wait for RabbitMQ to be fully ready (look for "Server startup complete")
# Restart your services
```

### Database Connection Errors

**Problem**: Can't connect to PostgreSQL

**Solution**:
```bash
# Check if databases are running
docker ps | grep postgres

# Check database logs
docker logs order-db

# Verify connection string in code matches docker-compose.yml
```

### Messages Not Being Consumed

**Problem**: Events published but not consumed

**Solution**:
1. Check RabbitMQ Management UI (`http://localhost:15672`)
2. Verify exchange `order_events` exists
3. Verify queues are bound to exchange
4. Check service logs for errors
5. Ensure consumers are running

### Port Already in Use

**Problem**: `bind: address already in use`

**Solution**:
```bash
# Find process using the port (example: port 8080)
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or change port in service code
```

## 🛠️ Development

### Adding a New Service

1. Create new service directory
2. Implement service with RabbitMQ consumer
3. Add database in `docker-compose.yml`
4. Bind queue to `order_events` exchange
5. Update README

### Adding New Event Types

1. Add event struct to `shared/events/events.go`
2. Update publishers in Order Service
3. Update consumers in relevant services

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Authors

- Your Name - [@codefalconx]

## 🙏 Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [GORM](https://gorm.io/)
- [RabbitMQ](https://www.rabbitmq.com/)
- [PostgreSQL](https://www.postgresql.org/)

## 📧 Contact

For questions or support, please open an issue or contact: codefalconx@gmail.com

---

**Happy Coding! 🚀**