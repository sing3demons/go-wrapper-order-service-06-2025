package product

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/router"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProductService interface {
	CreateProduct(ctx *router.Context, body *ProductInfo) error
	GetProduct(id string) (string, float64, error)
}

type productService struct {
	col *mongo.Collection
}

func NewProductService(col *mongo.Collection) ProductService {
	return &productService{col: col}
}
func (s *productService) CreateProduct(ctx *router.Context, body *ProductInfo) error {
	cmd := "create_product"
	dependency := s.col.Database().Name()

	start := time.Now()
	body.CreatedAt = start
	body.UpdatedAt = start
	body.DeletedAt = nil

	if body.ProductId == "" {
		productId, _ := uuid.NewV7()
		body.ProductId = productId.String()
	}

	processReqLog := ProcessMongoReq{
		Collection: "products",
		Method:     "InsertOne",
		Query:      nil,
		Document:   body,
		Options:    nil,
	}

	ctx.Log.SetDependencyMetadata(commonlog.LogDependencyMetadata{
		Dependency: dependency,
	}).Info(logAction.DB_REQUEST(logAction.DB_CREATE, cmd),
		map[string]any{
			"Body": processReqLog,
		})

	fmt.Println("mongo request: ===============> ", processReqLog.RawString())

	pCtx, cancel := context.WithTimeout(ctx.Context, 10*time.Second)
	defer cancel()

	result, err := s.col.InsertOne(pCtx, body)
	end := time.Since(start)

	if err != nil {
		ctx.Log.SetSummary(commonlog.LogEventTag{
			Node:        "mongo",
			Command:     cmd,
			Code:        "500",
			Description: err.Error(),
			ResTime:     end.Microseconds(),
		}).Error(logAction.DB_RESPONSE(logAction.DB_CREATE, cmd), err.Error())

		return fmt.Errorf("failed to create product: %w", err)
	}

	ctx.Log.SetSummary(commonlog.LogEventTag{
		Node:        "mongo",
		Command:     cmd,
		Code:        "200",
		Description: "success",
		ResTime:     end.Microseconds(),
	}).SetDependencyMetadata(commonlog.LogDependencyMetadata{
		Dependency:   dependency,
		ResponseTime: end.Microseconds(),
		ResultCode:   "200",
	}).Info(logAction.DB_RESPONSE(logAction.DB_CREATE, cmd), map[string]any{
		"Body": result,
	})

	// Implementation for creating a product
	// For now, just return a dummy ID and no error
	return nil
}
func (s *productService) GetProduct(id string) (string, float64, error) {
	// Implementation for retrieving a product
	// For now, just return a dummy name and price
	return "Dummy Product", 99.99, nil
}
