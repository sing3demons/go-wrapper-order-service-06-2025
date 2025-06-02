package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	config "github.com/sing3demons/go-order-service/configs"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	httpService "github.com/sing3demons/go-order-service/pkg/http"
	kafkaService "github.com/sing3demons/go-order-service/pkg/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"golang.org/x/sync/errgroup"
)

type App struct {
	SubscriptionManager
	httpServer    *httpService.Router
	traceProvider *trace.TracerProvider
	Logger        commonlog.LoggerService
	conf          *config.Config
}

type IApplication interface {
	Get(pattern string, handler Handler)
	Put(pattern string, handler Handler)
	Post(pattern string, handler Handler)
	Delete(pattern string, handler Handler)
	Patch(pattern string, handler Handler)
	Consumer(topic string, handler SubscribeFunc)
	Start(ctx context.Context)
	CreateTopic(topic string)
}

func NewApplication(conf *config.Config, logger commonlog.LoggerService) IApplication {
	var traceProvider *trace.TracerProvider
	if conf.TracerHost != "" {
		tp, err := startTracing(conf.App.Name, conf.TracerHost)
		if err != nil {
			logger.Errorf("failed to start tracing: %v", err)
		} else {
			traceProvider = tp
		}
	}

	app := &App{
		Logger: logger,
		conf:   conf,
	}

	kafkaClient := kafkaService.New(&conf.Kafka, logger)
	httpServiceClient := httpService.NewRouter()
	app.httpServer = httpServiceClient
	if traceProvider != nil {
		app.traceProvider = traceProvider
		app.httpServer.UseMiddleware(func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
				tr := otel.GetTracerProvider().Tracer("gokp-dev")
				ctx, span := tr.Start(ctx, fmt.Sprintf("%s %s", strings.ToUpper(r.Method), r.URL.Path))
				defer span.End()
				handler.ServeHTTP(w, r.WithContext(ctx))
			})
		})
	}
	if kafkaClient != nil {
		app.SubscriptionManager = newSubscriptionManager(kafkaClient, logger, conf)
	}
	return app
}

func (a *App) add(method, pattern string, h Handler) {
	a.httpServer.Add(method, pattern, handler{
		function:       h,
		requestTimeout: time.Duration(10) * time.Second,
		KafkaClient:    a.SubscriptionManager.KafkaClient,
		Logger:         a.Logger,
		conf:           a.conf,
	})
}

func startTracing(appName, endpoint string) (*trace.TracerProvider, error) {
	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultScheduleDelay*time.Millisecond),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(appName),
			),
		),
	)

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider, nil
}

func (a *App) Get(pattern string, handler Handler) {
	a.add(http.MethodGet, pattern, handler)
}

func (a *App) Put(pattern string, handler Handler) {
	a.add(http.MethodPut, pattern, handler)
}
func (a *App) Post(pattern string, handler Handler) {
	a.add(http.MethodPost, pattern, handler)
}
func (a *App) Delete(pattern string, handler Handler) {
	a.add(http.MethodDelete, pattern, handler)
}
func (a *App) Patch(pattern string, handler Handler) {
	a.add(http.MethodPatch, pattern, handler)
}

func (a *App) CreateTopic(topic string) {
	a.KafkaClient.CreateTopic(topic)
}

func (a *App) Consume(ctx context.Context, topic string) (*kafkaService.Message, error) {
	msg, err := a.SubscriptionManager.Subscribe(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}
	return msg, nil
}

func (a *App) GetSubscriber() kafkaService.Subscriber {
	if a.KafkaClient == nil {
		return nil
	}

	return a.KafkaClient
}
func (a *App) Consumer(topic string, handler SubscribeFunc) {
	if topic == "" || handler == nil {
		a.Logger.Error("invalid subscription: topic and handler must not be empty or nil")
		return
	}

	if a.GetSubscriber() == nil {
		a.Logger.Error("subscriber not initialized in the container")
		return
	}

	if a.conf.Kafka.AutoCreateTopic {
		err := a.KafkaClient.CreateTopic(topic)
		if err != nil {
			a.Logger.Error("failed to create topic %s: %v", topic, err)
			return
		}
	}

	a.SubscriptionManager.subscriptions[topic] = handler
}

func (a *App) startSubscriptions(ctx context.Context) error {
	if len(a.SubscriptionManager.subscriptions) == 0 {
		return nil
	}

	group := errgroup.Group{}
	// Start subscribers concurrently using go-routines
	for topic, handler := range a.SubscriptionManager.subscriptions {
		subscriberTopic, subscriberHandler := topic, handler

		group.Go(func() error {
			return a.SubscriptionManager.startSubscriber(ctx, subscriberTopic, subscriberHandler)
		})
	}

	return group.Wait()
}

func (a *App) Start(ctx context.Context) {
	if a.conf.Server.AppPort == "" {
		a.Logger.Error("server port is not configured")
		return
	}
	a.Logger.Log("starting application on port: " + a.conf.Server.AppPort)
	wg := sync.WaitGroup{}

	// Start HTTP Server
	wg.Add(1)
	s := &http.Server{
		Addr:           ":" + a.conf.Server.AppPort,
		Handler:        a.httpServer,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		defer wg.Done()
		fmt.Println("server started at", s.Addr)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen:", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		err := a.startSubscriptions(ctx)
		if err != nil {
			a.Logger.Error("Subscription Error : %v", err)
		}
	}()

	wg.Wait()

	<-ctx.Done()
	a.Logger.Log("shutting down gracefully, press Ctrl+C again to force")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(timeoutCtx); err != nil {
		fmt.Println(err)
	}

	defer func() {
		if err := a.traceProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("traceprovider: %v", err)
		}
	}()
}
