package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, message []byte) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (*Message, error)
}

type ConfigClient interface {
	Publisher
	Subscriber

	CreateTopic(context context.Context, name string) error
	DeleteTopic(context context.Context, name string) error

	Close() error
}

type Committer interface {
	Commit()
}


type Reader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Stats() kafka.ReaderStats
	Close() error
}

type Writer interface {
	WriteMessages(ctx context.Context, msg ...kafka.Message) error
	Close() error
	Stats() kafka.WriterStats
}

type Connection interface {
	Controller() (broker kafka.Broker, err error)
	CreateTopics(topics ...kafka.TopicConfig) error
	DeleteTopics(topics ...string) error
	Close() error
}
