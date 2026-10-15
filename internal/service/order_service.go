package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"order-tracking-api/internal/model"
	"order-tracking-api/internal/repository"

	"github.com/google/uuid"
)

type OrderService struct {
	repo *repository.OrderRepository
}

func NewOrderService(repo *repository.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrder(
	ctx context.Context,
	req model.CreateOrderRequest,
	idempotencyKey string,
) (*model.Order, bool, error) {
	if err := req.Validate(); err != nil {
		return nil, false, err
	}

	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, false, errors.New("Idempotency-Key header is required")
	}
	if len(idempotencyKey) > 200 {
		return nil, false, errors.New("Idempotency-Key must not exceed 200 characters")
	}

	// Hash เฉพาะข้อมูลใน request ไม่รวม idempotency key
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, false, fmt.Errorf("marshal request: %w", err)
	}

	sum := sha256.Sum256(payload)
	requestHash := hex.EncodeToString(sum[:])

	return s.repo.Create(
		ctx,
		req,
		idempotencyKey,
		requestHash,
	)
}

func (s *OrderService) GetOrder(
	ctx context.Context,
	id uuid.UUID,
) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OrderService) ListOrders(
	ctx context.Context,
	limit int,
	offset int,
) ([]model.Order, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.List(ctx, limit, offset)
}

func (s *OrderService) ConfirmPayment(
	ctx context.Context,
	id uuid.UUID,
) (*model.Order, error) {
	return s.repo.ConfirmPayment(ctx, id)
}

func (s *OrderService) GetTimeline(
	ctx context.Context,
	id uuid.UUID,
) ([]model.OrderTimeline, error) {
	// ตรวจว่ามี order จริงก่อน เพื่อแยกกรณีไม่พบ order
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.repo.GetTimeline(ctx, id)
}

func (s *OrderService) ReportIssue(
	ctx context.Context,
	id uuid.UUID,
	description string,
) (*model.OrderIssue, error) {
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("description is required")
	}
	if len(description) > 2000 {
		return nil, errors.New("description must not exceed 2000 characters")
	}

	return s.repo.CreateIssue(ctx, id, description)
}
