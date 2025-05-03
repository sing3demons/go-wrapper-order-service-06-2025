package main

import (
	"context"
	"fmt"

	kafkaService "github.com/sing3demons/go-order-service/kafka"
	"github.com/sing3demons/go-order-service/router"
	"go.uber.org/zap"
)

type kafkaLogger struct {
	*zap.Logger
}

func (k *kafkaLogger) Debugf(format string, args ...any) {
	k.Logger.Sugar().Debugf(format, args...)
}

func (k *kafkaLogger) Debug(args ...any) {
	k.Logger.Sugar().Debug(args...)
}

func (k *kafkaLogger) Logf(format string, args ...any) {
	k.Logger.Sugar().Infof(format, args...)
}
func (k *kafkaLogger) Log(args ...any) {
	k.Logger.Sugar().Info(args...)
}
func (k *kafkaLogger) Errorf(format string, args ...any) {
	// fmt.Printf(format, args...)
	k.Logger.Sugar().Errorf(format, args...)
}
func (k *kafkaLogger) Error(args ...any) {
	k.Logger.Sugar().Error(args...)
}

func NewLogger(logger *zap.Logger) kafkaService.Logger {
	return &kafkaLogger{Logger: logger}
}

func main() {
	ZLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer ZLogger.Sync()

	logger := NewLogger(ZLogger)
	logger.Debugf("test debug %s", "test")

	conf := &router.Config{
		AppName:    "go-app",
		AppPort:    "8080",
		AppVersion: "1.0.0",
		KafkaConfig: kafkaService.Config{
			Broker:          "localhost:29092",
			BatchSize:       100,
			BatchBytes:      1048576,
			BatchTimeout:    1000,
			ConsumerGroupID: "test-group",
		},
		TracerHost: "localhost:4318",
	}

	app := router.NewApplication(conf, logger)

	var index uint64 = 0

	app.Get("/test", func(c *router.Context) error {
		c.Publish(context.Background(), "test-topic", "test_message_"+fmt.Sprint(index))
		index++
		return c.JSON(200, map[string]string{"message": "test"})
	})

	app.Subscribe("test-topic", func(c *router.Context) error {
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
