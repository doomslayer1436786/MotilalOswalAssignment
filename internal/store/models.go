package store

import (
	"time"
)

// Event represents the common structure for all events
type Event struct {
	EventID   string      `json:"eventId"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// UserCreatedData represents the data payload for UserCreated events
type UserCreatedData struct {
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// OrderPlacedData represents the data payload for OrderPlaced events
type OrderPlacedData struct {
	OrderID   string    `json:"orderId"`
	UserID    string    `json:"userId"`
	Total     float64   `json:"total"`
	CreatedAt time.Time `json:"createdAt"`
}

// PaymentSettledData represents the data payload for PaymentSettled events
type PaymentSettledData struct {
	OrderID   string    `json:"orderId"`
	Status    string    `json:"status"`
	Amount    float64   `json:"amount"`
	SettledAt time.Time `json:"settledAt"`
}

// InventoryAdjustedData represents the data payload for InventoryAdjusted events
type InventoryAdjustedData struct {
	SKU        string    `json:"sku"`
	Delta      int       `json:"delta"`
	Reason     string    `json:"reason"`
	AdjustedAt time.Time `json:"adjustedAt"`
}

// Database models
type User struct {
	UserID    string    `json:"userId" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Order struct {
	OrderID   string    `json:"orderId" db:"order_id"`
	UserID    string    `json:"userId" db:"user_id"`
	Total     float64   `json:"total" db:"total"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Payment struct {
	OrderID   string    `json:"orderId" db:"order_id"`
	Status    string    `json:"status" db:"status"`
	Amount    float64   `json:"amount" db:"amount"`
	SettledAt time.Time `json:"settledAt" db:"settled_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Inventory struct {
	SKU            string    `json:"sku" db:"sku"`
	Quantity       int       `json:"quantity" db:"quantity"`
	LastAdjustedAt time.Time `json:"lastAdjustedAt" db:"last_adjusted_at"`
}

// DLQMessage represents a message stored in the dead letter queue
type DLQMessage struct {
	EventID   string      `json:"eventId"`
	Topic     string      `json:"topic"`
	Partition int         `json:"partition"`
	Offset    int64       `json:"offset"`
	Payload   interface{} `json:"payload"`
	Error     string      `json:"error"`
	FailedAt  time.Time   `json:"failedAt"`
}
