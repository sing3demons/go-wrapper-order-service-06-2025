package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/trace"
)

var errNotPointer = errors.New("input should be a pointer to a variable")

type Message struct {
	ctx context.Context

	Topic    string
	Value    []byte
	MetaData any

	Committer
}

func NewMessage(ctx context.Context) *Message {
	if ctx == nil {
		return &Message{ctx: context.Background()}
	}

	return &Message{ctx: ctx}
}

func (m *Message) Context() context.Context {
	return m.ctx
}

func (m *Message) Param(p string) string {
	if p == "topic" {
		return m.Topic
	}

	return ""
}

func (k *Message) SessionId() string {
	session := k.ctx.Value("x-session-id")
	if session == nil {
		// set context with session if not already set
		session = uuid.New().String()
		k.ctx = context.WithValue(k.ctx, "x-session-id", session)
	}
	if sessionStr, ok := session.(string); ok {
		return sessionStr
	}
	return trace.SpanFromContext(k.ctx).SpanContext().TraceID().String()
}

// TransactionId() string
func (m *Message) TransactionId() string {
	transaction := m.ctx.Value("x-transaction-id")
	if transaction == nil {
		// set context with transaction if not already set
		transaction = uuid.New().String()
		m.ctx = context.WithValue(m.ctx, "x-transaction-id", transaction)
	}
	if transactionStr, ok := transaction.(string); ok {
		return transactionStr
	}
	return trace.SpanFromContext(m.ctx).SpanContext().TraceID().String()
}

func (m *Message) PathParam(p string) string {
	return m.Param(p)
}

// Bind binds the message value to the input variable. The input should be a pointer to a variable.
func (m *Message) Bind(i any) error {
	fmt.Println("Binding message value to variable", string(m.Value))
	if reflect.ValueOf(i).Kind() != reflect.Ptr {
		return errNotPointer
	}

	switch v := i.(type) {
	case *string:
		return m.bindString(v)
	case *float64:
		return m.bindFloat64(v)
	case *int:
		return m.bindInt(v)
	case *bool:
		return m.bindBool(v)
	default:
		return m.bindStruct(i)
	}
}

func (m *Message) bindString(v *string) error {
	*v = string(m.Value)
	return nil
}

func (m *Message) bindFloat64(v *float64) error {
	f, err := strconv.ParseFloat(string(m.Value), 64)
	if err != nil {
		return err
	}

	*v = f

	return nil
}

func (m *Message) bindInt(v *int) error {
	in, err := strconv.Atoi(string(m.Value))
	if err != nil {
		return err
	}

	*v = in

	return nil
}

func (m *Message) bindBool(v *bool) error {
	b, err := strconv.ParseBool(string(m.Value))
	if err != nil {
		return err
	}

	*v = b

	return nil
}

func (m *Message) bindStruct(i any) error {
	return json.Unmarshal(m.Value, i)
}

func (m *Message) HostName() string {
	return ""
}

func (*Message) Params(string) []string {
	return nil
}

type kafkaMessage struct {
	msg    *kafka.Message
	reader Reader
	logger Logger
}

func newKafkaMessage(msg *kafka.Message, reader Reader, logger Logger) *kafkaMessage {
	return &kafkaMessage{
		msg:    msg,
		reader: reader,
		logger: logger,
	}
}

func (kmsg *kafkaMessage) Commit() {
	if kmsg.reader != nil {
		err := kmsg.reader.CommitMessages(context.Background(), *kmsg.msg)
		if err != nil {
			kmsg.logger.Errorf("unable to commit message on kafka")
		}
	}
}
