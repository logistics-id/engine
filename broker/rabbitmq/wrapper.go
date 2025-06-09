package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
)

// This file provides a high-level wrapper for RabbitMQ client operations,
// simplifying the process of connecting, publishing, and subscribing to topics
// using a default client instance. It abstracts away the underlying client
// management and configuration, making it easier to integrate RabbitMQ-based
// messaging into your application.
//
// Usage Overview:
//   1. Use ConfigDefault to generate a configuration from environment variables.
//   2. Call NewConnection to initialize the default RabbitMQ client.
//   3. Use Publish and Subscribe to send and receive messages.
//   4. Call CloseConnection to gracefully close the connection.
//
// Environment Variables Required:
//   - RABBIT_SERVER: RabbitMQ server address
//   - RABBIT_AUTH_USERNAME: RabbitMQ username
//   - RABBIT_AUTH_PASSWORD: RabbitMQ password
//
// Example:
//   cfg := rabbitmq.ConfigDefault("myPrefix")
//   logger := zap.NewExample()
//   err := rabbitmq.NewConnection(cfg, logger)
//   if err != nil { ... }
//   rabbitmq.Publish(ctx, "topic", data)
//   rabbitmq.Subscribe("topic", handler)
//   rabbitmq.CloseConnection()

var defaultClient *Client

// ConfigDefault creates a Config struct using environment variables.
// Make sure to load the environment variables before calling this function.
// The prefix parameter is used as a namespace for topics.
func ConfigDefault(prefix string) *Config {
	c := &Config{
		Server:       os.Getenv("RABBIT_SERVER"),
		Username:     os.Getenv("RABBIT_AUTH_USERNAME"),
		Password:     os.Getenv("RABBIT_AUTH_PASSWORD"),
		Prefix:       prefix,
		Exchange:     "engine.service",
		ExchangeType: "topic",
		Durable:      true,
		QueueTTL:     30 * time.Second, DeadLetter: "engine.service.dlx",
	}
	c.Datasource = fmt.Sprintf("amqp://%s:%s@%s/", c.Username, c.Password, c.Server)
	return c
}

// NewConnection initializes the default RabbitMQ client using the provided config and logger.
// It must be called before using Publish or Subscribe.
func NewConnection(cfg *Config, logger *zap.Logger) error {
	c, err := NewClient(cfg, logger.With(zap.String("component", "broker.rabbitmq")))

	if err == nil {
		defaultClient = c
	}

	return err
}

// Subscribe registers a handler for the specified topic using the default client.
// The handler will be called for each message received on the topic.
func Subscribe(topic string, handler any) error {
	return defaultClient.Subscribe(concatPrefix(topic), topic, handler)
}

// Publish sends data to the specified topic using the default client.
func Publish(ctx context.Context, topic string, data any) error {
	return defaultClient.Publish(ctx, concatPrefix(topic), data)
}

// CloseConnection gracefully closes the default RabbitMQ client connection.
func CloseConnection() error {
	return defaultClient.Close()
}

// concatPrefix adds the configured prefix to the topic name for namespacing.
func concatPrefix(topic string) string {
	return fmt.Sprintf("%s.%s", defaultClient.config.Prefix, topic)
}
