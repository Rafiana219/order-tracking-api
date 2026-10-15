package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"order-tracking-api/internal/config"
	"order-tracking-api/internal/handler"
	"order-tracking-api/internal/middleware"
	"order-tracking-api/internal/repository"
	"order-tracking-api/internal/router"
	"order-tracking-api/internal/service"
	"order-tracking-api/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := cfg.NewDBPool(ctx)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	orderRepo := repository.NewOrderRepository(db)
	orderService := service.NewOrderService(orderRepo)
	orderHandler := handler.NewOrderHandler(orderService)

	app := fiber.New(fiber.Config{
		AppName:      "Order Tracking API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	})

	app.Use(middleware.Trace())

	router.Register(app, orderHandler, db)

	orderWorker := worker.NewOrderWorker(
		orderRepo,
		cfg.WorkerInterval,
		cfg.WorkerMaxAttempts,
	)

	workerCtx, stopWorker := context.WithCancel(
		context.Background(),
	)
	defer stopWorker()

	go orderWorker.Start(workerCtx)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.Listen(":" + cfg.AppPort)
	}()

	log.Printf("API starting on port %s", cfg.AppPort)

	signalCtx, stopSignal := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignal()

	select {
	case err := <-serverErrors:
		if err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	case <-signalCtx.Done():
		log.Println("Shutdown signal received")
	}

	stopWorker()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
