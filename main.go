package main

import (
	"context"
	"fmt"
	"log"
	"time"

	kafkaService "github.com/sing3demons/go-order-service/kafka"
	"github.com/sing3demons/go-order-service/router"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
	fmt.Printf(format, args...)
}
func (k *kafkaLogger) Error(args ...any) {
	k.Logger.Sugar().Error(args...)
}

func NewLogger(logger *zap.Logger) kafkaService.Logger {
	return &kafkaLogger{Logger: logger}
}

func main() {
	traceProvider, err := startTracing()
	if err != nil {
		log.Fatalf("traceprovider: %v", err)
	}
	defer func() {
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("traceprovider: %v", err)
		}
	}()

	_ = traceProvider.Tracer("my-app")
	ZLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer ZLogger.Sync()

	logger := NewLogger(ZLogger)
	logger.Debugf("test debug %s", "test")

	app := router.NewApplication(&kafkaService.Config{
		Broker:          "localhost:29092",
		BatchSize:       100,
		BatchBytes:      1048576,
		BatchTimeout:    1000,
		ConsumerGroupID: "test-group",
	}, logger)

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

func startTracing() (*trace.TracerProvider, error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint("localhost:4318"),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	tracerprovider := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("product-app"),
			),
		),
	)

	otel.SetTracerProvider(tracerprovider)

	return tracerprovider, nil
}
