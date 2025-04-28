package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	kafkaService "github.com/sing3demons/go-order-service/kafka"
	httpService "github.com/sing3demons/go-order-service/pkg/http"
	"golang.org/x/sync/errgroup"
)

type App struct {
	SubscriptionManager
	httpServer *httpService.Router
	Logger
}

func NewApplication(conf *kafkaService.Config, logger Logger) *App {
	kafkaClient := kafkaService.New(conf, logger)
	httpServiceClient := httpService.NewRouter()

	subscriptionManager := newSubscriptionManager(kafkaClient, logger)

	return &App{
		SubscriptionManager: subscriptionManager,
		httpServer:          httpServiceClient,
		Logger:              logger,
	}
}
func (a *App) add(method, pattern string, h Handler) {

	a.httpServer.Add(method, pattern, handler{
		function:       h,
		requestTimeout: time.Duration(10) * time.Second,
		KafkaClient:    a.SubscriptionManager.KafkaClient,
		Logger:         a.Logger,
	})
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
func (a *App) Subscribe(topic string, handler SubscribeFunc) {
	if topic == "" || handler == nil {
		err := fmt.Errorf("invalid subscription: topic and handler must not be empty or nil")
		fmt.Println(err)

		return
	}

	if a.GetSubscriber() == nil {
		err := fmt.Errorf("subscriber not initialized in the container")
		fmt.Println(err)
		return
	}

	a.SubscriptionManager.subscriptions[topic] = handler
}

type token struct{}

type Group struct {
	cancel func(error)

	wg sync.WaitGroup

	sem chan token

	errOnce sync.Once
	err     error
}

func (g *Group) done() {
	if g.sem != nil {
		<-g.sem
	}
	g.wg.Done()
}
func WithContext(ctx context.Context) (*Group, context.Context) {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{cancel: cancel}, ctx
}

// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel(g.err)
	}
	return g.err
}
func (g *Group) Go(f func() error) {
	if g.sem != nil {
		g.sem <- token{}
	}

	g.wg.Add(1)
	go func() {
		defer g.done()

		if err := f(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel(g.err)
				}
			})
		}
	}()
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
	wg := sync.WaitGroup{}

	// Start HTTP Server
	wg.Add(1)
	s := &http.Server{
		Addr:           ":3000",
		Handler:        a.httpServer,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
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
			fmt.Printf("Subscription Error : %v", err)
		}
	}()

	wg.Wait()

	<-ctx.Done()
	fmt.Println("shutting down gracefully, press Ctrl+C again to force")

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Shutdown(timeoutCtx); err != nil {
		fmt.Println(err)
	}
}
