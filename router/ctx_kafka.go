package router

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime/debug"

	kafkaService "github.com/sing3demons/go-order-service/pkg/kafka"
)

type Context struct {
	context.Context
	Request
	http.ResponseWriter
	kafkaService.KafkaClient
	Logger
}

type Request interface {
	Context() context.Context
	Param(string) string
	PathParam(string) string
	Bind(any) error
	HostName() string
	Params(string) []string
}

func (c *Context) Bind(i any) error {
	return c.Request.Bind(i)
}

func newContext(w http.ResponseWriter, r Request, k kafkaService.KafkaClient, logger Logger) *Context {
	return &Context{
		Context:        r.Context(),
		Request:        r,
		ResponseWriter: w,
		KafkaClient:    k,
		Logger:         logger,
	}
}

func (c *Context) Publish(topic string, message any) error {
	var msg []byte

	if _, ok := message.([]byte); ok {
		msg = message.([]byte)
	} else {
		var err error
		msg, err = json.Marshal(message)
		if err != nil {
			c.Logger.Errorf("failed to marshal message: %v", err)
			return err
		}
	}

	return c.KafkaClient.Publish(c.Context, topic, msg)
}

func (c *Context) JSON(code int, v any) error {
	c.ResponseWriter.Header().Set("Content-Type", "application/json; charset=UTF8")
	c.ResponseWriter.WriteHeader(code)

	if err := json.NewEncoder(c.ResponseWriter).Encode(v); err != nil {
		c.Logger.Errorf("failed to write response: %v", err)
		return err
	}
	return nil
}

type SubscribeFunc func(c *Context) error

type SubscriptionManager struct {
	kafkaService.KafkaClient
	subscriptions map[string]SubscribeFunc
	Logger
}

func newSubscriptionManager(kafkaSvc kafkaService.KafkaClient, logger Logger) SubscriptionManager {
	return SubscriptionManager{
		KafkaClient:   kafkaSvc,
		subscriptions: make(map[string]SubscribeFunc),
		Logger:        logger,
	}
}

// startSubscriber continuously subscribes to a topic and handles messages using the provided handler.
func (s *SubscriptionManager) startSubscriber(ctx context.Context, topic string, handler SubscribeFunc) error {
	for {
		select {
		case <-ctx.Done():
			s.Logger.Logf("shutting down subscriber for topic %s", topic)
			return nil
		default:
			err := s.handleSubscription(ctx, topic, handler)
			if err != nil {
				s.Logger.Errorf("error in subscription for topic %s: %v", topic, err)
			}
		}
	}
}

func (s *SubscriptionManager) handleSubscription(ctx context.Context, topic string, handler SubscribeFunc) error {
	msg, err := s.KafkaClient.Subscribe(ctx, topic)

	if err != nil {
		s.Logger.Errorf("error while reading from topic %v, err: %v", topic, err.Error())
		return err
	}

	if msg == nil {
		return nil
	}

	// newContext creates a new context from the msg.Context()
	msgCtx := newContext(nil, msg, s.KafkaClient, s.Logger)
	err = func(ctx *Context) error {
		defer func() {
			panicRecovery(recover(), ctx.Logger)
		}()

		return handler(ctx)
	}(msgCtx)

	if err != nil {
		// fmt.Printf("error in handler for topic %s: %v", topic, err)
		s.Logger.Errorf("error in handler for topic %s: %v", topic, err)

		return nil
	}

	if msg.Committer != nil {
		// commit the message if the subscription function does not return error
		msg.Commit()
	}

	return nil
}

type panicLog struct {
	Error      string `json:"error,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

func panicRecovery(re any, log Logger) {
	if re == nil {
		return
	}

	var e string
	switch t := re.(type) {
	case string:
		e = t
	case error:
		e = t.Error()
	default:
		e = "Unknown panic type"
	}

	log.Error(panicLog{
		Error:      e,
		StackTrace: string(debug.Stack()),
	})
}
