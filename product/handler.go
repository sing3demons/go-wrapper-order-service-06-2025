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
	logger := ctx.CommonLog()

	var body ProductInfo
	if err := ctx.Bind(&body); err != nil {
		logger.Error(logAction.INBOUND(cmd, ""), "Failed to bind request body", err)
		logger.AddSummary(commonlog.EventTag{
			Node:        "client",
			Command:     cmd,
			Code:        "400",
			Description: err.Error(),
		})
		ctx.JSON(400, map[string]string{"error": "Invalid request body"})
		return err
	}

	logger.Info(logAction.INBOUND(cmd, ""), map[string]any{"body": body})
	logger.AddSummary(commonlog.EventTag{
		Node:        "client",
		Command:     cmd,
		Code:        "",
		Description: "success",
	})

	if err := h.ProductService.CreateProduct(ctx, &body); err != nil {
		ctx.JSON(500, map[string]string{"error": err.Error()})
		return err
	}

	ctx.Publish(ctx, "product_created", body)

	ctx.JSON(http.StatusCreated, map[string]string{"message": "success"})
	logger.Info(logAction.OUTBOUND(cmd, ""), map[string]any{"body": body})
	logger.Flush()
	return nil

}
