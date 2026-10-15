package handler

import (
	"errors"
	"strconv"

	"order-tracking-api/internal/model"
	"order-tracking-api/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(s *service.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req model.CreateOrderRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid JSON request body",
		})
	}

	order, created, err := h.service.CreateOrder(
		c.UserContext(),
		req,
		c.Get("Idempotency-Key"),
	)
	if err != nil {
		if errors.Is(err, model.ErrIdempotencyConflict) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return writeServiceError(c, err)
	}

	statusCode := fiber.StatusOK
	if created {
		statusCode = fiber.StatusCreated
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"data": order,
		"meta": fiber.Map{
			"created": created,
		},
	})
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	id, err := parseOrderID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order ID",
		})
	}

	order, err := h.service.GetOrder(c.UserContext(), id)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(fiber.Map{"data": order})
}

func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	limit, err := parseQueryInt(c, "limit", 20)
	if err != nil || limit < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "limit must be a positive integer",
		})
	}

	offset, err := parseQueryInt(c, "offset", 0)
	if err != nil || offset < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "offset must be a non-negative integer",
		})
	}

	orders, err := h.service.ListOrders(
		c.UserContext(),
		limit,
		offset,
	)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(fiber.Map{
		"data": orders,
		"meta": fiber.Map{
			"limit":  min(limit, 100),
			"offset": offset,
		},
	})
}

func (h *OrderHandler) ConfirmPayment(c *fiber.Ctx) error {
	id, err := parseOrderID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order ID",
		})
	}

	order, err := h.service.ConfirmPayment(c.UserContext(), id)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(fiber.Map{"data": order})
}

func (h *OrderHandler) GetTimeline(c *fiber.Ctx) error {
	id, err := parseOrderID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order ID",
		})
	}

	timeline, err := h.service.GetTimeline(c.UserContext(), id)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.JSON(fiber.Map{"data": timeline})
}

func (h *OrderHandler) ReportIssue(c *fiber.Ctx) error {
	id, err := parseOrderID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid order ID",
		})
	}

	var req struct {
		Description string `json:"description"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid JSON request body",
		})
	}

	issue, err := h.service.ReportIssue(
		c.UserContext(),
		id,
		req.Description,
	)
	if err != nil {
		return writeServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": issue,
	})
}

func parseOrderID(raw string) (uuid.UUID, error) {
	return uuid.Parse(raw)
}

func parseQueryInt(
	c *fiber.Ctx,
	key string,
	defaultValue int,
) (int, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}

	return strconv.Atoi(raw)
}

func writeServiceError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, model.ErrOrderNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "order not found",
		})

	case errors.Is(err, model.ErrInvalidOrder):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})

	case errors.Is(err, model.ErrInvalidTransition):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})

	default:
		// เก็บรายละเอียดจริงไว้ใน server log แทนการส่งให้ client
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
}
