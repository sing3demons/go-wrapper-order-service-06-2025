package main

import (
	"os"

	config "github.com/sing3demons/go-order-service/configs"
	"github.com/sing3demons/go-order-service/mongo"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/logger"
	"github.com/sing3demons/go-order-service/pkg/router"
	"github.com/sing3demons/go-order-service/product"
)

func main() {
	conf := config.NewConfig()

	// conf.LoadConfigJson("configs/config.json")
	conf.LoadEnv("configs")

	log := logger.NewLogger(conf)
	defer log.Sync()

	mongoClient := mongo.New(mongo.Config{
		URI:      os.Getenv("MONGO_URI"),
		Host:     conf.Get("MONGO_HOST"),
		Database: "order_service",
	})
	mongoClient.UseLogger(log)

	mongoClient.Connect()

	col := mongoClient.Collection("products")
	product.NewProductService(col)

	app := router.NewApplication(conf, log)
	app.CreateTopic("product_created")

	productService := product.NewProductService(col)
	handler := product.NewHandler(productService)

	app.Post("/products", handler.CreateProduct)

	// app.Consumer("test-topic", func(c *router.Context) error {
	// 	var msg string
	// 	if err := c.Bind(&msg); err != nil {
	// 		fmt.Println("Error binding message:", err)
	// 		return err
	// 	}
	// 	fmt.Println("Received message:", msg)
	// 	return nil
	// })

	app.Consumer("product_created", func(c *router.Context) error {
		var req router.KafkaPayload
		if err := c.Bind(&req); err != nil {
			c.Log.Error(logAction.CONSUMING("product_created", ""), "Failed to bind request body", err)
			c.Log.AddSummary(commonlog.EventTag{
				Node:        "consuming",
				Command:     "product_created",
				Code:        "400",
				Description: err.Error(),
			})
			return err
		}
		c.Log.Info(logAction.CONSUMING("product_created", ""), req)
		c.Log.AddSummary(commonlog.EventTag{
			Node:        "consuming",
			Command:     "product_created",
			Code:        "",
			Description: "success",
		})
		c.Log.Flush()
		return nil
	})

	app.Start()
}
