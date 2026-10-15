package router

import (
	"context"
	"time"

	"order-tracking-api/internal/handler"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(
	app *fiber.App,
	orderHandler *handler.OrderHandler,
	db *pgxpool.Pool,
) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	app.Get("/ready", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			2*time.Second,
		)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(
				fiber.Map{
					"status": "not_ready",
					"error":  "database unavailable",
				},
			)
		}

		return c.JSON(fiber.Map{
			"status": "ready",
		})
	})

	api := app.Group("/api")
	orders := api.Group("/orders")

	orders.Post("/", orderHandler.CreateOrder)
	orders.Get("/", orderHandler.ListOrders)
	orders.Get("/:id", orderHandler.GetOrder)
	orders.Post("/:id/confirm-payment", orderHandler.ConfirmPayment)
	orders.Get("/:id/timeline", orderHandler.GetTimeline)
	orders.Post("/:id/issues", orderHandler.ReportIssue)
}
