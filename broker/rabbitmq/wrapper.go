package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

var (
	defaultClient *Client
)

// ConfigDefault creating an config readed from .env file
// make sure you load the env file in your init app
func ConfigDefault(prefix string) *Config {
	c := &Config{
		Server:       os.Getenv("RABBIT_SERVER"),
		Username:     os.Getenv("RABBIT_AUTH_USERNAME"),
		Password:     os.Getenv("RABBIT_AUTH_PASSWORD"),
		Prefix:       prefix,
		Exchange:     "engine.service",
		ExchangeType: "topic",
		Durable:      true,
		QueueTTL:     30 * time.Second,
		DeadLetter:   "engine.service.dlx",
	}

	c.Datasource = fmt.Sprintf("amqp://%s:%s@%s/", c.Username, c.Password, c.Server)

	return c
}

func NewConnection(cfg *Config, logger *zap.Logger) error {
	c, err := NewClient(cfg, logger)
	if err == nil {
		defaultClient = c
	}

	return err
}

func Subscribe(topic string, handler any) error {
	return defaultClient.Subscribe(concatPrefix(topic), topic, handler)
}

func Publish(ctx context.Context, topic string, data any) error {
	return defaultClient.Publish(ctx, concatPrefix(topic), data)
}

func CloseConnection() error {
	return defaultClient.Close()
}

func concatPrefix(topic string) string {
	return fmt.Sprintf("%s.%s", defaultClient.config.Prefix, topic)
}
