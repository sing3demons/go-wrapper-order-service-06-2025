package config

import (
	"os"
	"time"
)

type AppConfig struct {
	AppName     string      `json:"app_name"`
	AppPort     string      `json:"app_port"`
	AppVersion  string      `json:"app_version"`
	KafkaConfig KafkaConfig `json:"kafka"`
	TracerHost  string      `json:"tracer_host"`
}

type KafkaConfig struct {
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
	TLS              TLSKafkaConfig
	AutoCreateTopic  bool
}

type TLSKafkaConfig struct {
	CertFile           string
	KeyFile            string
	CACertFile         string
	InsecureSkipVerify bool
}

type IConfig interface {
	Get(string) string
	GetOrDefault(string, string) string
}

func NewConfig(cfg AppConfig) *AppConfig {
	return &cfg
}

func (c *AppConfig) LoadEnv(configFolder string) {
	// Assuming NewEnvFile loads environment variables into the process,
	// so just call it for its side effects.
	NewEnvFile(configFolder)
}

func (*AppConfig) Get(key string) string {
	return os.Getenv(key)
}

func (*AppConfig) GetOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultValue
}
