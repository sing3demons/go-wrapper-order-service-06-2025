package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	config "github.com/sing3demons/go-order-service/configs"
)

var errNotPointer = errors.New("input should be a pointer to a variable")

type Message struct {
	ctx context.Context

	Topic    string
	Value    []byte
	MetaData any
	config   config.KafkaConfig

	Committer
	sessionId     string
	transactionId string
	requestId     string
	Header        Header
	Payload       []byte
}

type contextKey string

const (
	SessionIdKey     contextKey = "sessionId"
	TransactionIdKey contextKey = "transactionId"
	RequestIdKey     contextKey = "requestId"
)

func NewMessage(ctx context.Context, config config.KafkaConfig, value []byte) *Message {
	if ctx == nil {
		ctx = context.Background()
	}

	rId := uuid.New().String()

	data := DataConsumer{}
	if err := json.Unmarshal(value, &data); err != nil {
		data.Body = string(value)
		data.Header.Broker = config.Broker
	}

	if data.Header.Session == "" {
		data.Header.Session = uuid.New().String()
	}

	if data.Header.Transaction == "" {
		data.Header.Transaction = uuid.New().String()
	}
	ctx = context.WithValue(ctx, SessionIdKey, data.Header.Session)
	ctx = context.WithValue(ctx, TransactionIdKey, data.Header.Transaction)
	ctx = context.WithValue(ctx, RequestIdKey, rId)

	jsonValue, err := json.Marshal(data)
	if err != nil {
		jsonValue = value
	}

	return &Message{
		ctx:           ctx,
		config:        config,
		sessionId:     data.Header.Session,
		transactionId: data.Header.Transaction,
		requestId:     rId,
		Header:        data.Header,
		Payload:       jsonValue,
	}
}

func (m *Message) Context() context.Context {
	return m.ctx
}

func (m *Message) Headers() map[string]any {
	headers := make(map[string]any)

	h := m.Header
	t := reflect.TypeOf(h)
	v := reflect.ValueOf(h)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		if value.IsValid() && !value.IsZero() {
			// Convert field name to lower case and use it as key
			key := field.Name
			if len(key) > 0 {
				key = string(key[0]|32) + key[1:] // Convert first letter to lower case
			}
			headers[key] = value.Interface()
		}
	}

	// Manually extract fields from Header struct
	if m.Header.Broker != "" {
		headers["broker"] = m.Header.Broker
	}
	if m.Header.Session != "" {
		headers[sessionId] = m.Header.Session
	}
	if m.Header.Transaction != "" {
		headers[transactionId] = m.Header.Transaction
	}

	headers[requestId] = m.requestId

	return headers
}

func (m *Message) Param(p string) string {
	if p == "topic" {
		return m.Topic
	}

	return ""
}

func (k *Message) SessionId() string {
	return k.sessionId
}

// TransactionId() string
func (m *Message) TransactionId() string {
	return m.transactionId
}

func (m *Message) RequestId() string {
	return m.requestId
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
	return m.config.Broker
}

func (m *Message)  Method() string {
	return "kafka"
}
func (m *Message) URL() string {
	if m.ctx == nil {
		return ""
	}

	url := fmt.Sprintf("kafka://%s/%s", m.config.Broker, m.Topic)
	if m.ctx.Value(SessionIdKey) != nil {
		url += "?sessionId=" + m.ctx.Value(SessionIdKey).(string)
	}
	if m.ctx.Value(TransactionIdKey) != nil {
		url += "&transactionId=" + m.ctx.Value(TransactionIdKey).(string)
	}
	if m.ctx.Value(RequestIdKey) != nil {
		url += "&requestId=" + m.ctx.Value(RequestIdKey).(string)
	}

	return url
}

func (*Message) Params(string) []string {
	return nil
}

type kafkaMessage struct {
	msg    *kafka.Message
	reader Reader
	// logger commonlog.LoggerService
}

func newKafkaMessage(msg *kafka.Message, reader Reader) *kafkaMessage {
	return &kafkaMessage{
		msg:    msg,
		reader: reader,
		// logger: logger,
	}
}

func (kmsg *kafkaMessage) Commit() {
	if kmsg.reader != nil {
		err := kmsg.reader.CommitMessages(context.Background(), *kmsg.msg)
		if err != nil {
			// kmsg.logger.Errorf("unable to commit message on kafka")
			log.Println("unable to commit message on kafka:", err.Error())
		}
	}
}
