package product

import (
	"encoding/json"
	"net/http"

	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/router"
)

type consumer struct {
	ProductService ProductService
}

func NewProductConsumer(productService ProductService) Handler {
	return &consumer{ProductService: productService}
}

func (h *consumer) CreateProduct(c *router.Context) error {
	var req router.KafkaPayload
	if err := c.Bind(&req); err != nil {
		c.Log.SetSummary(commonlog.LogEventTag{
			Node:        "consuming",
			Command:     c.QueryParam("topic"),
			Code:        "400",
			Description: err.Error(),
		}).Error(logAction.CONSUMING("product_created"), err.Error())

		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	c.Log.SetSummary(commonlog.LogEventTag{
		Node:        "consuming",
		Command:     c.QueryParam("topic"),
		Code:        "",
		Description: "success",
	}).Info(logAction.CONSUMING(c.QueryParam("topic")), req)

	var body ProductInfo
	byteData, err := json.Marshal(req.Body)
	if err != nil {
		c.Log.Error(logAction.APP_LOGIC("convert to data to byte"), err.Error())
	}
	if err := json.Unmarshal(byteData, &body); err != nil {
		c.Log.Error(logAction.APP_LOGIC("convert byte to data"), err.Error())
	}

	if err := h.ProductService.CreateProduct(c, &body); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, "")
}
