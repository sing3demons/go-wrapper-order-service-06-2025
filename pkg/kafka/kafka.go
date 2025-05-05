package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrConsumerGroupNotProvided = errors.New("consumer group id not provided")
	errBrokerNotProvided        = errors.New("kafka broker address not provided")
	errPublisherNotConfigured   = errors.New("can't publish message. Publisher not configured or topic is empty")
	errBatchSize                = errors.New("KAFKA_BATCH_SIZE must be greater than 0")
	errBatchBytes               = errors.New("KAFKA_BATCH_BYTES must be greater than 0")
	errBatchTimeout             = errors.New("KAFKA_BATCH_TIMEOUT must be greater than 0")
	errClientNotConnected       = errors.New("kafka client not connected")
	// errUnsupportedSASLMechanism    = errors.New("unsupported SASL mechanism")
	// errSASLCredentialsMissing      = errors.New("SASL credentials missing")
	// errUnsupportedSecurityProtocol = errors.New("unsupported security protocol")
)

const (
	DefaultBatchSize    = 100
	DefaultBatchBytes   = 1048576
	DefaultBatchTimeout = 1000
	defaultRetryTimeout = 10 * time.Second
	protocolPlainText   = "PLAINTEXT"
	protocolSASL        = "SASL_PLAINTEXT"
	protocolSSL         = "SSL"
	protocolSASLSSL     = "SASL_SSL"
)

type Config struct {
	Broker           string
	Partition        int
	ConsumerGroupID  string
	OffSet           int
	BatchSize        int
	BatchBytes       int
	BatchTimeout     int
	RetryTimeout     time.Duration
	SASLMechanism    string
	SASLUser         string
	SASLPassword     string
	SecurityProtocol string
	TLS              TLSConfig
	AutoCreateTopic  bool
}

type TLSConfig struct {
	CertFile           string
	KeyFile            string
	CACertFile         string
	InsecureSkipVerify bool
}

type kafkaClient struct {
	dialer *kafka.Dialer
	conn   Connection

	writer Writer
	reader map[string]Reader

	mu *sync.RWMutex

	logger Logger
	config Config
}

type KafkaClient interface {
	Publish(ctx context.Context, topic string, message []byte) error
	Subscribe(ctx context.Context, topic string) (*Message, error)
	Close() (err error)

	DeleteTopic(name string) error
	Controller() (broker kafka.Broker, err error)
	CreateTopic(name string) error
}

func New(conf *Config, logger Logger) KafkaClient {
	err := validateConfigs(conf)
	if err != nil {
		logger.Errorf("could not initialize kafka, error: %v", err)
		return nil
	}

	logger.Debugf("connecting to Kafka broker '%s'", conf.Broker)

	dialer, conn, writer, reader, err := initializeKafkaClient(conf, logger)
	if err != nil {
		logger.Errorf("failed to connect to kafka at %v, error: %v", conf.Broker, err)

		client := &kafkaClient{
			logger: logger,
			config: *conf,
			// metrics: metrics,
			mu: &sync.RWMutex{},
		}

		go retryConnect(client, conf, logger)

		return client
	}

	return &kafkaClient{
		config: *conf,
		dialer: dialer,
		reader: reader,
		conn:   conn,
		logger: logger,
		writer: writer,
		mu:     &sync.RWMutex{},
	}
}

func validateConfigs(conf *Config) error {
	if err := validateRequiredFields(conf); err != nil {
		return err
	}

	return nil
}

func validateRequiredFields(conf *Config) error {
	if conf.Broker == "" {
		return errBrokerNotProvided
	}

	if conf.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0: %w", errBatchSize)
	}

	if conf.BatchBytes <= 0 {
		return fmt.Errorf("batch bytes must be greater than 0: %w", errBatchBytes)
	}

	if conf.BatchTimeout <= 0 {
		return fmt.Errorf("batch timeout must be greater than 0: %w", errBatchTimeout)
	}

	return nil
}

func (k *kafkaClient) Publish(ctx context.Context, topic string, message []byte) error {
	traceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
	fmt.Printf("Trace ID: %s from kafka\n", traceID)
	ctx, span := otel.GetTracerProvider().Tracer("gokp").Start(ctx, "kafka-publish")
	defer span.End()

	if k.writer == nil || topic == "" {
		return errPublisherNotConfigured
	}

	start := time.Now()
	err := k.writer.WriteMessages(ctx,
		kafka.Message{
			Topic: topic,
			Value: message,
			Time:  time.Now(),
		},
	)
	end := time.Since(start)

	if err != nil {
		k.logger.Errorf("failed to publish message to kafka broker, error: %v", err)
		return err
	}

	k.logger.Debug(&Log{
		Mode:          "PUB",
		CorrelationID: span.SpanContext().TraceID().String(),
		MessageValue:  string(message),
		Topic:         topic,
		Host:          k.config.Broker,
		PubSubBackend: "KAFKA",
		Time:          end.Microseconds(),
	})

	// k.metrics.IncrementCounter(ctx, "app_pubsub_publish_success_count", "topic", topic)

	return nil
}

func (k *kafkaClient) Subscribe(ctx context.Context, topic string) (*Message, error) {
	if !k.isConnected() {
		time.Sleep(defaultRetryTimeout)

		return nil, errClientNotConnected
	}

	if k.config.ConsumerGroupID == "" {
		k.logger.Error("cannot subscribe as consumer_id is not provided in configs")

		return &Message{}, ErrConsumerGroupNotProvided
	}

	ctx, span := otel.GetTracerProvider().Tracer("gokp").Start(ctx, "kafka-subscribe")
	defer span.End()

	// k.metrics.IncrementCounter(ctx, "app_pubsub_subscribe_total_count", "topic", topic, "consumer_group", k.config.ConsumerGroupID)

	var reader Reader
	// Lock the reader map to ensure only one subscriber access the reader at a time
	k.mu.Lock()

	if k.reader == nil {
		k.reader = make(map[string]Reader)
	}

	if k.reader[topic] == nil {
		k.reader[topic] = k.getNewReader(topic)
	}

	// Release the lock on the reader map after update
	k.mu.Unlock()

	start := time.Now()

	// Read a single message from the topic
	reader = k.reader[topic]
	msg, err := reader.FetchMessage(ctx)

	if err != nil {
		k.logger.Errorf("failed to read message from kafka topic %s: %v", topic, err)

		return nil, err
	}

	m := NewMessage(ctx)
	m.Value = msg.Value
	m.Topic = topic
	m.Committer = newKafkaMessage(&msg, k.reader[topic], k.logger)

	end := time.Since(start)

	k.logger.Debug(&Log{
		Mode:          "SUB",
		CorrelationID: span.SpanContext().TraceID().String(),
		MessageValue:  string(msg.Value),
		Topic:         topic,
		Host:          k.config.Broker,
		PubSubBackend: "KAFKA",
		Time:          end.Microseconds(),
	})

	// k.metrics.IncrementCounter(ctx, "app_pubsub_subscribe_success_count", "topic", topic, "consumer_group", k.config.ConsumerGroupID)

	return m, err
}

func (k *kafkaClient) Close() (err error) {
	for _, r := range k.reader {
		err = errors.Join(err, r.Close())
	}

	if k.writer != nil {
		err = k.writer.Close()
	}

	if k.conn != nil {
		err = errors.Join(k.conn.Close())
	}

	return err
}

func initializeKafkaClient(conf *Config, logger Logger) (*kafka.Dialer, Connection, Writer, map[string]Reader, error) {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	// 🛠 FIX: set default batch values before Writer
	if conf.BatchSize <= 0 {
		conf.BatchSize = 100 // Default batch size
	}
	if conf.BatchBytes <= 0 {
		conf.BatchBytes = 1 * 1024 * 1024 // Default 1MB
	}
	if conf.BatchTimeout <= 0 {
		conf.BatchTimeout = int((10 * time.Millisecond).Milliseconds()) // Default 10ms
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", conf.Broker)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{conf.Broker},
		Dialer:       dialer,
		BatchSize:    conf.BatchSize,
		BatchBytes:   conf.BatchBytes,
		BatchTimeout: time.Duration(conf.BatchTimeout),
	})

	reader := make(map[string]Reader)

	logger.Logf("connected to Kafka broker '%s'", conf.Broker)

	return dialer, conn, writer, reader, nil

}

func (k *kafkaClient) getNewReader(topic string) Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		GroupID:     k.config.ConsumerGroupID,
		Brokers:     []string{k.config.Broker},
		Topic:       topic,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		Dialer:      k.dialer,
		StartOffset: int64(k.config.OffSet),
	})

	return reader
}

func (k *kafkaClient) DeleteTopic(name string) error {
	return k.conn.DeleteTopics(name)
}

func (k *kafkaClient) Controller() (broker kafka.Broker, err error) {
	return k.conn.Controller()
}

func (k *kafkaClient) CreateTopic(name string) error {
	topics := kafka.TopicConfig{Topic: name, NumPartitions: 1, ReplicationFactor: 1}

	err := k.conn.CreateTopics(topics)
	if err != nil {
		return err
	}

	return nil
}

// retryConnect handles the retry mechanism for connecting to the Kafka broker.
func retryConnect(client *kafkaClient, conf *Config, logger Logger) {
	for {
		time.Sleep(defaultRetryTimeout)

		dialer, conn, writer, reader, err := initializeKafkaClient(conf, logger)
		if err != nil {
			logger.Errorf("could not connect to Kafka at '%v', error: %v", conf.Broker, err)
			continue
		}

		client.mu.Lock()
		client.conn = conn
		client.dialer = dialer
		client.writer = writer
		client.reader = reader
		client.mu.Unlock()

		return
	}
}

func (k *kafkaClient) isConnected() bool {
	if k.conn == nil {
		return false
	}

	_, err := k.conn.Controller()

	return err == nil
}
