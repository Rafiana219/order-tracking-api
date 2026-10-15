package worker

import (
	"context"
	"log"
	"time"

	"order-tracking-api/internal/model"
	"order-tracking-api/internal/repository"

	"github.com/google/uuid"
)

type OrderWorker struct {
	repo        *repository.OrderRepository
	interval    time.Duration
	maxAttempts int
}

func NewOrderWorker(
	repo *repository.OrderRepository,
	interval time.Duration,
	maxAttempts int,
) *OrderWorker {
	if interval <= 0 {
		interval = 3 * time.Second
	}

	if maxAttempts < 1 {
		maxAttempts = 3
	}

	return &OrderWorker{
		repo:        repo,
		interval:    interval,
		maxAttempts: maxAttempts,
	}
}

func (w *OrderWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("order worker started")

	// เริ่มทำงานทันที ไม่ต้องรอ ticker รอบแรก
	w.processOne(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("order worker stopped")
			return

		case <-ticker.C:
			w.processOne(ctx)
		}
	}
}

func (w *OrderWorker) processOne(ctx context.Context) {
	order, err := w.repo.ClaimNext(ctx)
	if err != nil {
		log.Printf("worker: claim order failed: %v", err)
		return
	}

	if order == nil {
		return
	}

	log.Printf(
		"worker: processing order_id=%s attempt=%d",
		order.ID,
		order.Attempts,
	)

	processErr := w.processOrder(ctx, order)

	finishCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := w.repo.FinishProcessing(
		finishCtx,
		order.ID,
		w.maxAttempts,
		processErr,
	); err != nil {
		log.Printf(
			"worker: finish order_id=%s failed: %v",
			order.ID,
			err,
		)
	}
}

// processOrder เป็นเพียงตัวอย่างจำลองการทำงาน
// ยังไม่ได้เชื่อมต่อระบบสินค้า การชำระเงิน หรือบริการภายนอกจริง
func (w *OrderWorker) processOrder(
	ctx context.Context,
	order *model.Order,
) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-time.After(500 * time.Millisecond):
	}

	if order.ID == uuid.Nil {
		return model.ErrInvalidOrder
	}

	if len(order.Items) == 0 {
		return model.ErrInvalidOrder
	}

	// TODO: เพิ่ม business logic จริง เช่น
	// 1. ตรวจสอบสินค้า
	// 2. เรียกบริการภายนอก
	// 3. บันทึกผลลัพธ์
	//
	// ตอนนี้ถือว่าประมวลผลสำเร็จหลังจำลอง 500ms
	return nil
}
