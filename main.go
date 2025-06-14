package main

import (
	"os"

	config "github.com/sing3demons/go-order-service/configs"
	"github.com/sing3demons/go-order-service/mongo"
	"github.com/sing3demons/go-order-service/order"
	"github.com/sing3demons/go-order-service/pkg/logger"
	"github.com/sing3demons/go-order-service/pkg/router"
	"github.com/sing3demons/go-order-service/postgres"
	"github.com/sing3demons/go-order-service/product"
)

func main() {
	conf := config.NewConfig()

	// conf.LoadConfigJson("configs/config.json")
	conf.LoadEnv("configs")

	log := logger.NewLogger(conf.Log.App)
	defer log.Sync()

	mongoClient := mongo.New(mongo.Config{
		URI:      os.Getenv("MONGO_URI"),
		Host:     conf.Get("MONGO_HOST"),
		Database: "order_service",
	})
	mongoClient.UseLogger(log)
	mongoClient.Connect()
	col := mongoClient.Collection("products")

	postgresClient, err := postgres.New()
	if err != nil {
		log.Errorf("Failed to connect to PostgreSQL", err)
		os.Exit(1)
	}

	app := router.NewApplication(conf, log)
	app.LogDetail(logger.NewLogger(conf.Log.Detail))
	app.LogSummary(logger.NewLogger(conf.Log.Summary))
	app.StartKafka()

	app.CreateTopic("product_created")

	productService := product.NewProductService(col)
	handler := product.NewHandler()
	consumer := product.NewProductConsumer(productService)

	orderService := order.New(postgresClient)
	orderHandler := order.NewHandler(orderService)

	app.Post("/products", handler.CreateProduct)
	app.Post("/orders", orderHandler.CreateOrder)
	app.Get("/orders/{id}", orderHandler.GetOrderById)
	app.Get("/orders", orderHandler.GetOrders)

	// app.Consumer("test-topic", func(c *router.Context) error {
	// 	var msg string
	// 	if err := c.Bind(&msg); err != nil {
	// 		fmt.Println("Error binding message:", err)
	// 		return err
	// 	}

	// 	fmt.Println("Received message:", msg)
	// 	return nil
	// })

	app.Consumer("product_created", consumer.CreateProduct)

	app.Start()
}
