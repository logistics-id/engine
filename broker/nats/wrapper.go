package nats

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"
)

// Config contains the NATS connection parameters.
type Config struct {
	Server     string
	Username   string
	Password   string
	Datasource string
	Prefix     string
}

// internal shared client singleton
var (
	defaultClient           *Client
	defaultLogger           *zap.Logger
	ErrClientNotInitialized = errors.New("nats client not initialized; call InitializeDefaultClient first")
)

// InitializeDefaultClient creates the default singleton NATS client for package-level functions.
// Must be called before using Publish/Subscribe functions.
// example
// nats.NewConnection(cfg, logger)
//
//	defer func() {
//		    if err := nats.CloseConnection(); err != nil {
//		        logger.Error("failed to close nats client", zap.Error(err))
//		    }
//		}()
func NewConnection(cfg *Config, logger *zap.Logger) error {
	c, err := NewClient(cfg, logger)
	if err == nil {
		defaultClient = c
	}

	return err
}

// ConfigDefault creating an config readed from .env file
// make sure you load the env file in your init app
func ConfigDefault(prefix string) *Config {
	return &Config{
		Server:   os.Getenv("NATS_SERVER"),
		Username: os.Getenv("NATS_AUTH_USERNAME"),
		Password: os.Getenv("NATS_AUTH_PASSWORD"),
		Prefix:   prefix,
	}
}

// Publish sends a message using the default client.
func Publish(subject string, payload any) error {
	if defaultClient == nil {
		return ErrClientNotInitialized
	}

	return defaultClient.Publish(subject, payload)
}

// Subscribe subscribes using the default client.
func Subscribe(subject string, handler func(ctx context.Context, msg any)) error {
	if defaultClient == nil {
		return ErrClientNotInitialized
	}

	return defaultClient.Subscribe(subject, handler)
}

// CloseConnection closes the default client connection.
func CloseConnection() error {
	if defaultClient == nil {
		return ErrClientNotInitialized
	}

	return defaultClient.Close()
}
