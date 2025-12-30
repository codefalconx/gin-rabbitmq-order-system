module github.com/codefalconx/order-service

go 1.22

require (
    github.com/codefalconx/order-system/shared v0.0.0
    github.com/gin-gonic/gin v1.9.1
    gorm.io/driver/postgres v1.5.4
    gorm.io/gorm v1.25.5
    github.com/google/uuid v1.4.0
    github.com/rabbitmq/amqp091-go v1.9.0
)

replace github.com/codefalconx/order-system/shared => ../shared