package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"order-tracking-api/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(
	ctx context.Context,
	req model.CreateOrderRequest,
	idempotencyKey string,
	requestHash string,
) (*model.Order, bool, error) {
	itemsJSON, err := json.Marshal(req.Items)
	if err != nil {
		return nil, false, fmt.Errorf("marshal order items: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin create order: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	order := &model.Order{}

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			customer_id,
			items,
			status,
			idempotency_key,
			request_hash
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING
			id,
			customer_id,
			items,
			status,
			idempotency_key,
			request_hash,
			attempts,
			max_attempts,
			next_attempt_at,
			created_at,
			updated_at
	`,
		req.CustomerID,
		itemsJSON,
		model.StatusPendingPayment,
		idempotencyKey,
		requestHash,
	).Scan(
		&order.ID,
		&order.CustomerID,
		&itemsJSON,
		&order.Status,
		&order.IdempotencyKey,
		&order.RequestHash,
		&order.Attempts,
		&order.MaxAttempts,
		&order.NextAttemptAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		var existingHash string

		err = tx.QueryRow(ctx, `
			SELECT request_hash
			FROM orders
			WHERE idempotency_key = $1
		`, idempotencyKey).Scan(&existingHash)
		if err != nil {
			return nil, false, fmt.Errorf(
				"load existing idempotency record: %w", err,
			)
		}

		if existingHash != requestHash {
			return nil, false, model.ErrIdempotencyConflict
		}

		err = tx.QueryRow(ctx, `
			SELECT
				id,
				customer_id,
				items,
				status,
				idempotency_key,
				request_hash,
				attempts,
				max_attempts,
				next_attempt_at,
				created_at,
				updated_at
			FROM orders
			WHERE idempotency_key = $1
		`, idempotencyKey).Scan(
			&order.ID,
			&order.CustomerID,
			&itemsJSON,
			&order.Status,
			&order.IdempotencyKey,
			&order.RequestHash,
			&order.Attempts,
			&order.MaxAttempts,
			&order.NextAttemptAt,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, false, fmt.Errorf(
				"load existing order: %w", err,
			)
		}

		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, false, fmt.Errorf("decode order items: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, false, fmt.Errorf("commit existing order: %w", err)
		}

		return order, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("insert order: %w", err)
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, false, fmt.Errorf("decode order items: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_timeline (
			order_id,
			from_status,
			to_status,
			event,
			details
		)
		VALUES ($1, NULL, $2, $3, $4)
	`,
		order.ID,
		order.Status,
		"ORDER_CREATED",
		"Order created",
	)
	if err != nil {
		return nil, false, fmt.Errorf("insert initial timeline: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit create order: %w", err)
	}

	return order, true, nil
}

func (r *OrderRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.Order, error) {
	order := &model.Order{}
	var itemsJSON []byte

	err := r.db.QueryRow(ctx, `
		SELECT
			id,
			customer_id,
			items,
			status,
			idempotency_key,
			request_hash,
			attempts,
			max_attempts,
			next_attempt_at,
			created_at,
			updated_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&order.ID,
		&order.CustomerID,
		&itemsJSON,
		&order.Status,
		&order.IdempotencyKey,
		&order.RequestHash,
		&order.Attempts,
		&order.MaxAttempts,
		&order.NextAttemptAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("decode order items: %w", err)
	}

	return order, nil
}

func (r *OrderRepository) List(
	ctx context.Context,
	limit int,
	offset int,
) ([]model.Order, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			customer_id,
			items,
			status,
			idempotency_key,
			request_hash,
			attempts,
			max_attempts,
			next_attempt_at,
			created_at,
			updated_at
		FROM orders
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]model.Order, 0)

	for rows.Next() {
		var order model.Order
		var itemsJSON []byte

		err := rows.Scan(
			&order.ID,
			&order.CustomerID,
			&itemsJSON,
			&order.Status,
			&order.IdempotencyKey,
			&order.RequestHash,
			&order.Attempts,
			&order.MaxAttempts,
			&order.NextAttemptAt,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}

		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("decode order items: %w", err)
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orders: %w", err)
	}

	return orders, nil
}

func (r *OrderRepository) ConfirmPayment(
	ctx context.Context,
	id uuid.UUID,
) (*model.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin payment confirmation: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var current model.OrderStatus

	err = tx.QueryRow(ctx, `
		SELECT status
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&current)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load order status: %w", err)
	}

	// การยืนยันซ้ำหลังชำระเงินแล้วไม่ทำให้สถานะเปลี่ยนซ้ำ
	if current == model.StatusQueued {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit repeated confirmation: %w", err)
		}
		return r.GetByID(ctx, id)
	}

	if !model.CanTransition(current, model.StatusQueued) {
		return nil, model.ErrInvalidTransition
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`, model.StatusQueued, id)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO order_timeline (
			order_id, from_status, to_status, event, details
		)
		VALUES ($1, $2, $3, $4, $5)
	`,
		id,
		current,
		model.StatusQueued,
		"PAYMENT_CONFIRMED",
		"Payment confirmed",
	)
	if err != nil {
		return nil, fmt.Errorf("insert payment timeline: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit payment confirmation: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *OrderRepository) GetTimeline(
	ctx context.Context,
	id uuid.UUID,
) ([]model.OrderTimeline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, order_id, from_status, to_status,
		       event, details, created_at
		FROM order_timeline
		WHERE order_id = $1
		ORDER BY created_at ASC, id ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query timeline: %w", err)
	}
	defer rows.Close()

	timeline := make([]model.OrderTimeline, 0)

	for rows.Next() {
		var entry model.OrderTimeline
		var fromStatus *string

		if err := rows.Scan(
			&entry.ID,
			&entry.OrderID,
			&fromStatus,
			&entry.ToStatus,
			&entry.Event,
			&entry.Details,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan timeline: %w", err)
		}

		if fromStatus != nil {
			status := model.OrderStatus(*fromStatus)
			entry.FromStatus = &status
		}

		timeline = append(timeline, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate timeline: %w", err)
	}

	return timeline, nil
}

func (r *OrderRepository) CreateIssue(
	ctx context.Context,
	id uuid.UUID,
	description string,
) (*model.OrderIssue, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}

	issue := &model.OrderIssue{}

	err := r.db.QueryRow(ctx, `
		INSERT INTO order_issues (order_id, description)
		SELECT id, $2
		FROM orders
		WHERE id = $1
		RETURNING id, order_id, description, created_at
	`, id, description).Scan(
		&issue.ID,
		&issue.OrderID,
		&issue.Description,
		&issue.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("create order issue: %w", err)
	}

	return issue, nil
}

// ClaimNext reserves one eligible order for a worker.
// SKIP LOCKED prevents two workers claiming the same row simultaneously.
func (r *OrderRepository) ClaimNext(
	ctx context.Context,
) (*model.Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim order: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	order := &model.Order{}
	var itemsJSON []byte

	err = tx.QueryRow(ctx, `
		SELECT
			id,
			customer_id,
			items,
			status,
			idempotency_key,
			request_hash,
			attempts,
			max_attempts,
			next_attempt_at,
			created_at,
			updated_at
		FROM orders
		WHERE
			status = $1
			AND attempts < max_attempts
			AND (
				next_attempt_at IS NULL
				OR next_attempt_at <= NOW()
			)
		ORDER BY created_at ASC, id ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, model.StatusQueued).Scan(
		&order.ID,
		&order.CustomerID,
		&itemsJSON,
		&order.Status,
		&order.IdempotencyKey,
		&order.RequestHash,
		&order.Attempts,
		&order.MaxAttempts,
		&order.NextAttemptAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select next order: %w", err)
	}

	if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
		return nil, fmt.Errorf("decode claimed order items: %w", err)
	}

	previousStatus := order.Status

	err = tx.QueryRow(ctx, `
		UPDATE orders
		SET status = $1,
		    attempts = attempts + 1,
		    next_attempt_at = NULL,
		    updated_at = NOW()
		WHERE id = $2
		RETURNING attempts, updated_at
	`, model.StatusInProgress, order.ID).Scan(
		&order.Attempts,
		&order.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("mark order in progress: %w", err)
	}

	order.Status = model.StatusInProgress

	_, err = tx.Exec(ctx, `
		INSERT INTO order_timeline (
			order_id, from_status, to_status, event, details
		)
		VALUES ($1, $2, $3, $4, $5)
	`,
		order.ID,
		previousStatus,
		order.Status,
		"WORKER_STARTED",
		fmt.Sprintf("Processing attempt %d", order.Attempts),
	)
	if err != nil {
		return nil, fmt.Errorf("insert worker timeline: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim order: %w", err)
	}

	return order, nil
}

func (r *OrderRepository) FinishProcessing(
	ctx context.Context,
	id uuid.UUID,
	maxAttempts int,
	processErr error,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin finish processing: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var current model.OrderStatus
	var attempts int
	var storedMaxAttempts int

	err = tx.QueryRow(ctx, `
		SELECT status, attempts, max_attempts
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&current, &attempts, &storedMaxAttempts)

	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("load processing order: %w", err)
	}

	if current != model.StatusInProgress {
		return model.ErrInvalidTransition
	}

	if processErr == nil {
		_, err = tx.Exec(ctx, `
			UPDATE orders
			SET status = $1, updated_at = NOW()
			WHERE id = $2
		`, model.StatusCompleted, id)
		if err != nil {
			return fmt.Errorf("complete order: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO order_timeline (
				order_id, from_status, to_status, event, details
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
			id,
			current,
			model.StatusCompleted,
			"ORDER_COMPLETED",
			"Order processing completed",
		)
		if err != nil {
			return fmt.Errorf("insert completion timeline: %w", err)
		}
	} else {
		effectiveMaxAttempts := storedMaxAttempts
		if maxAttempts > 0 && maxAttempts < effectiveMaxAttempts {
			effectiveMaxAttempts = maxAttempts
		}

		nextStatus := model.StatusQueued
		var nextAttemptAt time.Time

		if attempts >= effectiveMaxAttempts {
			nextStatus = model.StatusFailed
		} else {
			// Exponential backoff: 2s, 4s, 8s... capped at 5 minutes.
			delay := 2 * time.Second * time.Duration(1<<min(attempts-1, 7))
			if delay > 5*time.Minute {
				delay = 5 * time.Minute
			}
			nextAttemptAt = time.Now().Add(delay)
		}

		if nextStatus == model.StatusQueued {
			_, err = tx.Exec(ctx, `
				UPDATE orders
				SET status = $1,
				    next_attempt_at = $2,
				    updated_at = NOW()
				WHERE id = $3
			`, nextStatus, nextAttemptAt, id)
		} else {
			_, err = tx.Exec(ctx, `
				UPDATE orders
				SET status = $1,
				    next_attempt_at = NULL,
				    updated_at = NOW()
				WHERE id = $2
			`, nextStatus, id)
		}
		if err != nil {
			return fmt.Errorf("update failed order: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO order_timeline (
				order_id, from_status, to_status, event, details
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
			id,
			current,
			nextStatus,
			"PROCESSING_FAILED",
			processErr.Error(),
		)
		if err != nil {
			return fmt.Errorf("insert failure timeline: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit finish processing: %w", err)
	}

	return nil
}
