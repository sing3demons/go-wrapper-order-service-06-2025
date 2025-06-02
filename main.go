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
	"time"

	config "github.com/sing3demons/go-order-service/configs"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/common-log/masking"
	"github.com/sing3demons/go-order-service/pkg/logger"
	"github.com/sing3demons/go-order-service/router"
)

func main() {
	log := logger.NewLogger()
	defer log.Sync()

	conf := config.NewConfig(config.AppConfig{
		// AppName:    "gokp-app",
		// AppPort:    "3000",
		// AppVersion: "1.0.0",
		// KafkaConfig: config.KafkaConfig{
		// 	Broker:          "localhost:29092",
		// 	BatchSize:       100,
		// 	BatchBytes:      1048576,
		// 	BatchTimeout:    1000,
		// 	ConsumerGroupID: "test-group",
		// 	AutoCreateTopic: true,
		// },
		// TracerHost: "localhost:4318",
	})

	conf.LoadEnv("configs")

	app := router.NewApplication(conf, log)

	app.Get("/ping", func(c *router.Context) error {
		// traceID := trace.SpanFromContext(c).SpanContext().TraceID().String()
		maskingData := []masking.MaskingOptionDto{
			{
				MaskingField: "body.key1",
				MaskingType:  masking.Full,
				IsArray:      true,
			},
		}
		kpLog := c.CommonLog()
		kpLog.Info(logAction.INBOUND("ping", ""), map[string]any{"body": map[string]any{"key1": "value1", "key2": "value2"}}, maskingData...)
		kpLog.AddSummary(commonlog.EventTag{
			Node:        "client",
			Command:     "ping",
			Code:        "",
			Description: "success",
		})

		startTime := time.Now()
		kpLog.SetDependencyMetadata(commonlog.LogDependencyMetadata{
			Dependency: "mongo",
		}).Info(logAction.DB_REQUEST("find user", ""), "")

		kpLog.SetDependencyMetadata(commonlog.LogDependencyMetadata{
			Dependency:   "mongo",
			ResponseTime: time.Since(startTime).Milliseconds(),
			ResultCode:   "200",
		}).Info(logAction.DB_RESPONSE("find user", ""), map[string]any{"user": "testUser"})
		kpLog.AddSummary(commonlog.EventTag{
			Node:        "mongo",
			Command:     "find user",
			Code:        "200",
			Description: "success",
			ResTime:     time.Since(startTime).Milliseconds(),
		})

		response := map[string]string{"message": "pong"}
		kpLog.Info(logAction.OUTBOUND("ping", ""), response)
		kpLog.Flush()
		return c.JSON(200, response)
	})

	// app.Consumer("test-topic", func(c *router.Context) error {
	// 	var msg string
	// 	if err := c.Bind(&msg); err != nil {
	// 		fmt.Println("Error binding message:", err)
	// 		return err
	// 	}
	// 	fmt.Println("Received message:", msg)
	// 	return nil
	// })

	app.Start(context.Background())
}
