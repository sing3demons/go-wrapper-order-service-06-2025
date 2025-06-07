package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	config "github.com/sing3demons/go-order-service/configs"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	kafkaService "github.com/sing3demons/go-order-service/pkg/kafka"
)

type Context struct {
	context.Context
	Request
	http.ResponseWriter
	kafkaService.KafkaClient
	Logger commonlog.LoggerService
	Log    commonlog.CustomLoggerService
	conf   *config.Config
}

type Request interface {
	Context() context.Context
	Param(string) string
	PathParam(string) string
	Bind(any) error
	HostName() string
	Params(string) []string
	SessionId() string
	TransactionId() string
	RequestId() string
	Headers() map[string]any
	Method() string
	URL() string
}

func (c *Context) Bind(i any) error {
	return c.Request.Bind(i)
}

type LogService struct {
	appLog     commonlog.LoggerService
	detailLog  commonlog.LoggerService
	summaryLog commonlog.LoggerService
}

func newContext(w http.ResponseWriter, r Request, k kafkaService.KafkaClient, logger LogService, conf *config.Config) *Context {
	t := commonlog.NewTimer()
	kpLog := commonlog.NewLogger(logger.detailLog, logger.summaryLog, t)
	ctx := &Context{
		Context:        r.Context(),
		Request:        r,
		ResponseWriter: w,
		KafkaClient:    k,
		conf:           conf,
		Logger:         logger.appLog,
	}

	broker := "none"
	if w == nil {
		broker = r.HostName()
	}

	kpLog.Init(commonlog.LogDto{
		Channel:          "none",
		UseCase:          "none",
		UseCaseStep:      "none",
		Broker:           broker,
		TransactionId:    ctx.Request.TransactionId(),
		SessionId:        ctx.Request.SessionId(),
		RequestId:        ctx.Request.RequestId(),
		AppName:          conf.App.Name,
		ComponentVersion: conf.App.Version,
		ComponentName:    conf.App.ComponentName,
		Instance:         ctx.Request.HostName(),
		// DateTime:         time.Now().Format(time.RFC3339),
		OriginateServiceName: func() string {
			if w != nil {
				return "HTTP Service"
			}
			return "Event Source"
		}(),
		RecordType: "detail",
	})

	ctx.Log = kpLog
	return ctx
}

type Header struct {
	Broker      string
	Channel     string
	UseCase     string
	UseCaseStep string
	Identity    struct {
		Device interface{}
		Public string
		User   string
	}
	Session     string
	Transaction string
}
type KafkaPayload struct {
	Header Header      `json:"header"`
	Body   interface{} `json:"body"`
}

func (c *Context) Publish(ctx *Context, topic string, message any) error {
	var msg []byte
	start := time.Now()

	body := KafkaPayload{
		Body: message,
	}
	body.Header.Broker = c.conf.Kafka.Broker
	body.Header.UseCase = topic
	body.Header.Session = ctx.Request.SessionId()
	body.Header.Transaction = ctx.Request.TransactionId()
	body.Header.Channel = topic
	body.Header.UseCaseStep = "publish"
	body.Header.Identity.Device = ctx.Request.HostName()
	body.Header.Identity.User = ctx.Request.HostName()

	ctx.Log.Info(logAction.PRODUCING(topic, ""), map[string]any{
		"body": map[string]any{
			"topic": topic,
			"value": body,
		}})

	var err error
	msg, err = json.Marshal(body)
	if err != nil {
		c.Logger.Errorf("failed to marshal message: %v", err)
		return err
	}
	description := "success"

	if err := c.KafkaClient.Publish(c.Context, topic, msg); err != nil {
		description = err.Error()
		c.Logger.Errorf("failed to publish message to topic %s: %v", topic, err)
	}
	end := time.Since(start)
	c.Log.SetSummary(commonlog.LogEventTag{
		Node:        "kafka",
		Command:     topic,
		Code:        "200",
		Description: description,
		ResTime:     end.Microseconds(),
	}).Info(logAction.PRODUCED(topic, ""), description)

	return nil
}

func (c *Context) JSON(code int, v any) error {
	if c.ResponseWriter != nil {
		c.ResponseWriter.Header().Set("Content-Type", "application/json; charset=UTF8")
		c.ResponseWriter.WriteHeader(code)

		if err := json.NewEncoder(c.ResponseWriter).Encode(v); err != nil {
			c.Log.Error(logAction.OUTBOUND("client", ""), err.Error())
			c.Log.End(code, err.Error())
			return err
		}
		c.Log.Info(logAction.OUTBOUND("client", ""), v)
	}

	c.Log.End(code, "")
	return nil
}

type SubscribeFunc func(c *Context) error

type SubscriptionManager struct {
	kafkaService.KafkaClient
	subscriptions map[string]SubscribeFunc
	Logger        LogService
	conf          *config.Config
}

func newSubscriptionManager(kafkaSvc kafkaService.KafkaClient, logger LogService, conf *config.Config) SubscriptionManager {
	return SubscriptionManager{
		KafkaClient:   kafkaSvc,
		subscriptions: make(map[string]SubscribeFunc),
		Logger:        logger,
		conf:          conf,
	}
}

// startSubscriber continuously subscribes to a topic and handles messages using the provided handler.
func (s *SubscriptionManager) startSubscriber(ctx context.Context, topic string, handler SubscribeFunc) error {
	for {
		select {
		case <-ctx.Done():
			s.Logger.appLog.Logf("shutting down subscriber for topic %s", topic)
			return nil
		default:
			err := s.handleSubscription(ctx, topic, handler)
			if err != nil {
				s.Logger.appLog.Errorf("error in subscription for topic %s: %v", topic, err)
			}
		}
	}
}

func (s *SubscriptionManager) handleSubscription(ctx context.Context, topic string, handler SubscribeFunc) error {
	msg, err := s.KafkaClient.Subscribe(ctx, topic)

	if err != nil {
		s.Logger.appLog.Errorf("error while reading from topic %v, err: %v", topic, err.Error())
		return err
	}

	if msg == nil {
		return nil
	}

	// newContext creates a new context from the msg.Context()
	logService := LogService{
		appLog:     s.Logger.appLog,
		detailLog:  s.Logger.detailLog,
		summaryLog: s.Logger.summaryLog,
	}
	msgCtx := newContext(nil, msg, s.KafkaClient, logService, s.conf)
	err = func(ctx *Context) error {
		defer func() {
			panicRecovery(recover(), ctx.Logger)
		}()

		return handler(ctx)
	}(msgCtx)

	if err != nil {
		// fmt.Printf("error in handler for topic %s: %v", topic, err)
		s.Logger.appLog.Errorf("error in handler for topic %s: %v", topic, err)

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

func panicRecovery(re any, log commonlog.LoggerService) {
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

	// log.Error(panicLog{
	// 	Error:      e,
	// 	StackTrace: string(debug.Stack()),
	// })

	fmt.Println(panicLog{
		Error:      e,
		StackTrace: string(debug.Stack()),
	})
}
