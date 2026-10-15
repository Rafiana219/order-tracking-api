package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	StatusQueued         OrderStatus = "QUEUED"
	StatusInProgress     OrderStatus = "IN_PROGRESS"
	StatusCompleted      OrderStatus = "COMPLETED"
	StatusExpired        OrderStatus = "EXPIRED"
	StatusFailed         OrderStatus = "FAILED"
	StatusRefunded       OrderStatus = "REFUNDED"
)

var (
	ErrInvalidOrder        = errors.New("invalid order")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrOrderNotFound       = errors.New("order not found")
	ErrIdempotencyConflict = errors.New(
		"idempotency key was already used with a different request",
	)
)

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderRequest struct {
	CustomerID string      `json:"customer_id"`
	Items      []OrderItem `json:"items"`
}

func (r CreateOrderRequest) Validate() error {
	if strings.TrimSpace(r.CustomerID) == "" {
		return fmt.Errorf("%w: customer_id is required", ErrInvalidOrder)
	}

	if len(r.Items) == 0 {
		return fmt.Errorf("%w: at least one item is required", ErrInvalidOrder)
	}

	for i, item := range r.Items {
		if strings.TrimSpace(item.ProductID) == "" {
			return fmt.Errorf(
				"%w: items[%d].product_id is required",
				ErrInvalidOrder,
				i,
			)
		}

		if item.Quantity < 1 {
			return fmt.Errorf(
				"%w: items[%d].quantity must be at least 1",
				ErrInvalidOrder,
				i,
			)
		}
	}

	return nil
}

type Order struct {
	ID             uuid.UUID   `json:"id"`
	CustomerID     string      `json:"customer_id"`
	Items          []OrderItem `json:"items"`
	Status         OrderStatus `json:"status"`
	IdempotencyKey string      `json:"-"`
	RequestHash    string      `json:"-"`
	Attempts       int         `json:"attempts"`
	MaxAttempts    int         `json:"max_attempts"`
	NextAttemptAt  *time.Time  `json:"next_attempt_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type OrderTimeline struct {
	ID         int64        `json:"id"`
	OrderID    uuid.UUID    `json:"order_id"`
	FromStatus *OrderStatus `json:"from_status,omitempty"`
	ToStatus   OrderStatus  `json:"to_status"`
	Event      string       `json:"event"`
	Details    string       `json:"details,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type OrderIssue struct {
	ID          int64     `json:"id"`
	OrderID     uuid.UUID `json:"order_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func CanTransition(from, to OrderStatus) bool {
	switch from {
	case StatusPendingPayment:
		return to == StatusQueued || to == StatusExpired

	case StatusQueued:
		return to == StatusInProgress ||
			to == StatusFailed ||
			to == StatusExpired

	case StatusInProgress:
		return to == StatusCompleted || to == StatusFailed

	case StatusFailed:
		return to == StatusQueued || to == StatusRefunded

	case StatusCompleted, StatusExpired, StatusRefunded:
		return false

	default:
		return false
	}
}
