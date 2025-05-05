package main

// import (
// 	"encoding/json"
// 	"fmt"

// 	"gofr.dev/pkg/gofr"
// )

// func main() {
// 	app := gofr.New()

// 	var index uint64 = 1

// 	app.GET("/test", func(ctx *gofr.Context) (any, error) {
// 			type orderStatus struct {
// 			OrderId string `json:"orderId"`
// 			Status  string `json:"status"`
// 		}

// 		data := orderStatus{
// 			OrderId: fmt.Sprintf("order-%d", index),
// 			Status:  "shipped",
// 		}

// 		msg, _ := json.Marshal(data)

// 		err := ctx.GetPublisher().Publish(ctx, "order-logs", msg)
// 		if err != nil {
// 			return nil, err
// 		}
// 		index++
// 		return map[string]string{"message": "test"}, nil
// 	})

// 	app.Run() // listens and serves on localhost:8000
// }

// package main

import (
	"context"
	"fmt"

	"github.com/sing3demons/go-order-service/pkg/kafka"
	"github.com/sing3demons/go-order-service/pkg/logger"
	"github.com/sing3demons/go-order-service/router"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	log := logger.NewLogger()
	defer log.Sync()

	conf := &router.Config{
		AppName:    "gokp-app",
		AppPort:    "3000",
		AppVersion: "1.0.0",
		KafkaConfig: kafka.Config{
			Broker:          "localhost:29092",
			BatchSize:       100,
			BatchBytes:      1048576,
			BatchTimeout:    1000,
			ConsumerGroupID: "test-group",
			AutoCreateTopic: true,
		},
		TracerHost: "localhost:4318",
	}

	app := router.NewApplication(conf, log)

	app.Get("/ping", func(c *router.Context) error {
		traceID := trace.SpanFromContext(c).SpanContext().TraceID().String()

		c.Publish("test-topic", "test_message_"+traceID)
		return c.JSON(200, map[string]string{"message": "test"})
	})

	app.Consumer("test-topic", func(c *router.Context) error {
		var msg string
		if err := c.Bind(&msg); err != nil {
			fmt.Println("Error binding message:", err)
			return err
		}
		fmt.Println("Received message:", msg)
		return nil
	})

	app.Start(context.Background())
}
