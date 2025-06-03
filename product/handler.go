package product

import (
	"net/http"

	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/router"
)

type Handler interface {
	CreateProduct(ctx *router.Context) error
}
type handler struct {
	ProductService ProductService
}

func NewHandler(productService ProductService) Handler {
	return &handler{
		ProductService: productService,
	}
}

func (h *handler) CreateProduct(ctx *router.Context) error {
	cmd := "create_product"

	var body ProductInfo
	if err := ctx.Bind(&body); err != nil {
		ctx.Log.Error(logAction.INBOUND(cmd, ""), "Failed to bind request body", err)
		ctx.Log.AddSummary(commonlog.EventTag{
			Node:        "client",
			Command:     cmd,
			Code:        "400",
			Description: err.Error(),
		})
		return ctx.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	ctx.Log.Info(logAction.INBOUND(cmd, ""), map[string]any{
		"headers": ctx.Headers(),
		"body": body,
	})
	ctx.Log.AddSummary(commonlog.EventTag{
		Node:        "client",
		Command:     cmd,
		Code:        "",
		Description: "success",
	})

	if err := h.ProductService.CreateProduct(ctx, &body); err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	ctx.Publish(ctx, "product_created", body)

	return ctx.JSON(http.StatusCreated, map[string]string{"message": "success"})
}
