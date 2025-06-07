package product

import (
	"net/http"

	"github.com/google/uuid"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/router"
)

type Handler interface {
	CreateProduct(ctx *router.Context) error
}
type handler struct {
}

func NewHandler() Handler {
	return &handler{}
}

func (h *handler) CreateProduct(ctx *router.Context) error {
	cmd := "create_product"

	var body ProductInfo
	if err := ctx.Bind(&body); err != nil {
		ctx.Log.SetSummary(commonlog.LogEventTag{
			Node:        "client",
			Command:     cmd,
			Code:        "400",
			Description: err.Error(),
		}).Error(logAction.INBOUND(cmd, ""), err.Error())

		return ctx.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	ctx.Log.SetSummary(commonlog.LogEventTag{
		Node:        "client",
		Command:     cmd,
		Code:        "",
		Description: "success",
	}).Info(logAction.INBOUND(cmd, ""), map[string]any{
		"headers": ctx.Headers(),
		"body":    body,
	})

	// call api
	// ctx.Log.Info(logAction.HTTP_REQUEST("todo", ""), map[string]any{
	// 	"method": "GET",
	// 	"url":    "http://localhost:3000/todo",
	// })
	// req, err := http.NewRequest(http.MethodGet, "http://localhost:3000/todo", nil)
	// if err != nil {
	// 	return ctx.JSON(500, map[string]string{"error": err.Error()})
	// }

	// req.Header.Set("Content-Type", "application/json")

	// httpClient := http.Client{
	// 	Timeout: 5 * time.Second,
	// }
	// resp, err := httpClient.Do(req)
	// if err != nil {
	// 	// Example: Detect timeout error
	// 	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
	// 		return ctx.JSON(http.StatusGatewayTimeout, map[string]string{"error": "user api request timed out"})

	// 	}
	// 	// Detect connection refused
	// 	if strings.Contains(err.Error(), "connection refused") {
	// 		return ctx.JSON(http.StatusBadGateway, map[string]string{"error": "failed to connect to user api"})

	// 	}
	// 	// Fallback generic error response
	// 	return ctx.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})

	// }
	// defer resp.Body.Close()
	// if resp.StatusCode < 200 || resp.StatusCode >= 300 {
	// 	errorBody, err := io.ReadAll(resp.Body) // Optional: log response body for debugging
	// 	if err != nil {
	// 		fmt.Println("Error reading error response body:", err)
	// 		return ctx.JSON(resp.StatusCode, map[string]string{"error": "Failed to read error response body"})
	// 	}
	// 	fmt.Printf("HTTP error: %d %s\nResponse body: %s\n", resp.StatusCode, http.StatusText(resp.StatusCode), errorBody)
	// 	return ctx.JSON(resp.StatusCode, map[string]string{"error": "HTTP error: " + http.StatusText(resp.StatusCode)})
	// }

	// bodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	fmt.Println("Error reading response body:", err)
	// 	return ctx.JSON(500, map[string]string{"error": "Failed to read response body"})
	// }
	// ctx.Log.Info(logAction.HTTP_RESPONSE("todo", ""), map[string]any{
	// 	"status":  resp.Status,
	// 	"headers": resp.Header,
	// 	"body":    string(bodyBytes),
	// })

	productId, _ := uuid.NewV7()
	body.ProductId = productId.String()
	if err := ctx.Publish(ctx, "product_created", body); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to publish product created event"})
	}

	return ctx.JSON(http.StatusCreated, map[string]string{
		"message":    "success",
		"status":     "processing",
		"product_id": body.ProductId,
	})
}

func (h *handler) GetProduct(ctx *router.Context) error {
	cmd := "get_product"

	var body ProductInfo
	if err := ctx.Bind(&body); err != nil {
		ctx.Log.SetSummary(commonlog.LogEventTag{
			Node:        "client",
			Command:     cmd,
			Code:        "400",
			Description: err.Error(),
		}).Error(logAction.INBOUND(cmd, ""), err.Error())

		return ctx.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	ctx.Log.SetSummary(commonlog.LogEventTag{
		Node:        "client",
		Command:     cmd,
		Code:        "",
		Description: "success",
	}).Info(logAction.INBOUND(cmd, ""), map[string]any{
		"headers": ctx.Headers(),
		"body":    body,
	})

	// if err := h.ProductService.GetProduct(ctx, &body); err != nil {
	// 	return ctx.JSON(500, map[string]string{"error": err.Error()})
	// }

	return ctx.JSON(http.StatusOK, map[string]string{"message": "success"})
}
