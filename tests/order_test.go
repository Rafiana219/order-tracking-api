package tests

import (
	"testing"

	"order-tracking-api/internal/model"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from model.OrderStatus
		to   model.OrderStatus
		want bool
	}{
		{
			name: "pending payment to queued",
			from: model.StatusPendingPayment,
			to:   model.StatusQueued,
			want: true,
		},
		{
			name: "pending payment to expired",
			from: model.StatusPendingPayment,
			to:   model.StatusExpired,
			want: true,
		},
		{
			name: "queued to in progress",
			from: model.StatusQueued,
			to:   model.StatusInProgress,
			want: true,
		},
		{
			name: "queued to failed",
			from: model.StatusQueued,
			to:   model.StatusFailed,
			want: true,
		},
		{
			name: "in progress to completed",
			from: model.StatusInProgress,
			to:   model.StatusCompleted,
			want: true,
		},
		{
			name: "in progress to failed",
			from: model.StatusInProgress,
			to:   model.StatusFailed,
			want: true,
		},
		{
			name: "failed to queued",
			from: model.StatusFailed,
			to:   model.StatusQueued,
			want: true,
		},
		{
			name: "failed to refunded",
			from: model.StatusFailed,
			to:   model.StatusRefunded,
			want: true,
		},
		{
			name: "completed cannot return to queued",
			from: model.StatusCompleted,
			to:   model.StatusQueued,
			want: false,
		},
		{
			name: "expired cannot return to pending payment",
			from: model.StatusExpired,
			to:   model.StatusPendingPayment,
			want: false,
		},
		{
			name: "refunded cannot return to queued",
			from: model.StatusRefunded,
			to:   model.StatusQueued,
			want: false,
		},
		{
			name: "unknown status cannot transition",
			from: model.OrderStatus("UNKNOWN"),
			to:   model.StatusQueued,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.CanTransition(tt.from, tt.to)

			if got != tt.want {
				t.Errorf(
					"CanTransition(%q, %q) = %v; want %v",
					tt.from,
					tt.to,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCreateOrderRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request model.CreateOrderRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: model.CreateOrderRequest{
				CustomerID: "customer-001",
				Items: []model.OrderItem{
					{
						ProductID: "product-001",
						Quantity:  2,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing customer ID",
			request: model.CreateOrderRequest{
				Items: []model.OrderItem{
					{
						ProductID: "product-001",
						Quantity:  1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty items",
			request: model.CreateOrderRequest{
				CustomerID: "customer-001",
				Items:      []model.OrderItem{},
			},
			wantErr: true,
		},
		{
			name: "missing product ID",
			request: model.CreateOrderRequest{
				CustomerID: "customer-001",
				Items: []model.OrderItem{
					{
						ProductID: "",
						Quantity:  1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "zero quantity",
			request: model.CreateOrderRequest{
				CustomerID: "customer-001",
				Items: []model.OrderItem{
					{
						ProductID: "product-001",
						Quantity:  0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative quantity",
			request: model.CreateOrderRequest{
				CustomerID: "customer-001",
				Items: []model.OrderItem{
					{
						ProductID: "product-001",
						Quantity:  -1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "whitespace customer ID",
			request: model.CreateOrderRequest{
				CustomerID: "   ",
				Items: []model.OrderItem{
					{
						ProductID: "product-001",
						Quantity:  1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "whitespace product ID",
			request: model.CreateOrderRequest{
				CustomerID: "customer-001",
				Items: []model.OrderItem{
					{
						ProductID: "   ",
						Quantity:  1,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"Validate() error = %v; wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
