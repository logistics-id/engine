package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Client wraps the NATS connection and logger.
type Client struct {
	conn   *nats.Conn
	logger *zap.Logger
	config *Config
}

// NewClient initializes a NATS client with the given config and logger.
func NewClient(cfg *Config, logger *zap.Logger) (*Client, error) {
	dsn := cfg.Datasource
	if dsn == "" {
		dsn = fmt.Sprintf("nats://%s:%s@%s", cfg.Username, cfg.Password, cfg.Server)
	}

	nc, err := nats.Connect(dsn)
	if err != nil {
		logger.Fatal("NATS/CONN FAILED", zap.String("dsn", dsn), zap.Any("config", cfg), zap.Error(err))

		return nil, err
	}

	logger.Info("NATS/CONN CONNECTED", zap.String("dsn", dsn))

	return &Client{
		conn:   nc,
		logger: logger,
		config: cfg,
	}, nil
}

// Publish sends a message to a subject with structured logging.
func (c *Client) Publish(subject string, payload any) error {
	fullSubject := fmt.Sprintf("%s.%s", c.config.Prefix, subject)

	data, err := json.Marshal(payload)
	if err != nil {
		c.logger.Error("Failed to marshal message payload",
			zap.String("subject", fullSubject),
			zap.Any("payload", payload),
			zap.Error(err),
		)
		return err
	}

	err = c.conn.Publish(fullSubject, data)
	if err != nil {
		c.logger.Error("Failed to publish message",
			zap.String("subject", fullSubject),
			zap.Any("payload", payload),
			zap.Error(err),
		)
	} else {
		c.logger.Debug("Published message",
			zap.String("subject", fullSubject),
			zap.Any("payload", payload),
		)
	}

	return err
}

// Subscribe sets up a queue subscriber with context logging support.
func (c *Client) Subscribe(subject string, handler func(context.Context, any)) error {
	queue := c.config.Prefix
	fullSubject := fmt.Sprintf("%s.%s", c.config.Prefix, subject)

	_, err := c.conn.QueueSubscribe(fullSubject, queue, func(msg *nats.Msg) {
		var data any
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			c.logger.Error("Failed to unmarshal message",
				zap.String("subject", msg.Subject),
				zap.ByteString("raw", msg.Data),
				zap.Error(err),
			)
			return
		}

		ctx := context.Background() // Could enrich context here with metadata if needed
		c.logger.Info("Received message",
			zap.String("subject", msg.Subject),
			zap.Any("data", data),
		)

		handler(ctx, data)

		// c.conn.
	})

	if err != nil {
		c.logger.Error("Failed to subscribe to subject",
			zap.String("subject", fullSubject),
			zap.Error(err),
		)
	}

	return err
}

// Request sends a request and waits for a reply.
func (c *Client) Request(subject string, req any, resp any) error {
	fullSubject := fmt.Sprintf("%s.%s", c.config.Prefix, subject)

	reqData, err := json.Marshal(req)
	if err != nil {
		c.logger.Error("Failed to marshal request",
			zap.String("subject", fullSubject),
			zap.Any("request", req),
			zap.Error(err),
		)
		return err
	}

	msg, err := c.conn.Request(fullSubject, reqData, 30*time.Second)
	if err != nil {
		c.logger.Error("NATS request failed",
			zap.String("subject", fullSubject),
			zap.Any("request", req),
			zap.Error(err),
		)
		return err
	}

	err = json.Unmarshal(msg.Data, resp)
	if err != nil {
		c.logger.Error("Failed to unmarshal response",
			zap.String("subject", fullSubject),
			zap.ByteString("response", msg.Data),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// Close shuts down the connection gracefully.
func (c *Client) Close() error {
	c.logger.Info("NATS/CONN CLOSED")

	return c.conn.Drain()
}
