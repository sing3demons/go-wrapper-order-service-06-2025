package order

import (
	"net/http"

	"github.com/google/uuid"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/router"
)

type Handler struct {
	store StoreOrder
}

func NewHandler(store StoreOrder) *Handler {
	return &Handler{store: store}
}

func (h *Handler) CreateOrder(ctx *router.Context) error {
	// Log the incoming request
	summary := commonlog.LogEventTag{
		Node:        "client",
		Command:     "create_order",
		Code:        "",
		Description: "Creating a new order",
	}
	var order Order
	if err := ctx.Bind(&order); err != nil {
		summary.Code = "400"
		summary.Description = err.Error()

		ctx.Log.SetSummary(summary).Error(logAction.INBOUND("create_order"), map[string]any{
			"headers": ctx.Headers(),
		})
		return ctx.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	ctx.Log.SetSummary(summary).Info(logAction.INBOUND("create_order"), map[string]any{
		"headers": ctx.Headers(),
		"body":    order,
	})
	_, err := h.store.Create(ctx, &order)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": "Failed to create order"})
	}
	return ctx.JSON(http.StatusCreated, map[string]string{"message": "Order created successfully"})
}

func (h *Handler) GetOrders(ctx *router.Context) error {
	summary := commonlog.LogEventTag{
		Node:        "client",
		Command:     "get_orders",
		Code:        "",
		Description: "Retrieving all orders",
	}
	ctx.Log.SetSummary(summary).Info(logAction.INBOUND("get_orders"), map[string]any{
		"headers": ctx.Headers(),
	})
	orders, err := h.store.GetAll(ctx)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": "Failed to retrieve orders"})
	}
	return ctx.JSON(http.StatusOK, orders)
}

func (h *Handler) GetOrderById(ctx *router.Context) error {
	summary := commonlog.LogEventTag{
		Node:        "client",
		Command:     "get_order_by_id",
		Code:        "",
		Description: "success",
	}
	id := ctx.PathParam("id")

	orderId, err := uuid.Parse(id)
	if err != nil {
		summary.Code = "400"
		summary.Description = "Invalid order ID format"
		ctx.Log.SetSummary(summary).Error(logAction.INBOUND("get_order_by_id"), map[string]any{
			"headers": ctx.Headers(),
			"url":     ctx.URL(),
			"method":  ctx.Method(),
			"param":   map[string]string{"id": id},
		})
		return ctx.JSON(400, map[string]string{"error": "Invalid order ID format"})
	}

	ctx.Log.SetSummary(summary).Info(logAction.INBOUND("get_order_by_id"), map[string]any{
		"headers": ctx.Headers(),
		"url":     ctx.URL(),
		"method":  ctx.Method(),
		"param":   map[string]string{"id": id},
	})
	order, err := h.store.GetByID(ctx, orderId)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": "Failed to retrieve order"})
	}
	return ctx.JSON(http.StatusOK, order)
}
