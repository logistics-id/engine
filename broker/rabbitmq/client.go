package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Config defines RabbitMQ connection and exchange settings
type Config struct {
	Server       string
	Username     string
	Password     string
	Datasource   string
	Prefix       string
	Exchange     string
	ExchangeType string
	Durable      bool
	QueueTTL     time.Duration
	DeadLetter   string
}

// Client wraps RabbitMQ connection, channel, and subscriber management
type Client struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	config      *Config
	logger      *zap.Logger
	exchange    string
	subscribers []subscriberMeta

	closed chan struct{}
	mu     sync.Mutex
	wg     sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

type subscriberMeta struct {
	Queue      string
	RoutingKey string
	Handler    any
}

// NewClient initializes the RabbitMQ client and connects
func NewClient(cfg *Config, logger *zap.Logger) (*Client, error) {
	c := &Client{
		config:      cfg,
		logger:      logger,
		exchange:    cfg.Exchange,
		subscribers: []subscriberMeta{},
		closed:      make(chan struct{}),
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	if err := c.connect(); err != nil {
		return nil, err
	}

	// Monitor connection for close events to reconnect
	go c.monitorConnection()

	return c, nil
}

// connect establishes a new connection and channel to RabbitMQ
func (c *Client) connect() error {
	conn, err := amqp.Dial(c.config.Datasource)
	if err != nil {
		c.logger.Error("RMQ/CONNECT FAILED", zap.String("dsn", c.config.Datasource), zap.Error(err))
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		c.logger.Error("RMQ/CHANNEL FAILED", zap.Error(err))
		return err
	}

	// Declare the exchange, create if not exists
	err = ch.ExchangeDeclare(
		c.exchange,
		c.config.ExchangeType,
		c.config.Durable,
		false, false, false, nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		c.logger.Error("RMQ/EXCHANGE DECLARE FAILED", zap.Error(err))
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.channel = ch
	c.mu.Unlock()

	c.logger.Info("RMQ/CONNECTED", zap.String("exchange", c.exchange))
	return nil
}

// monitorConnection listens for connection close events and reconnects
func (c *Client) monitorConnection() {
	connClose := c.conn.NotifyClose(make(chan *amqp.Error))
	for {
		select {
		case err := <-connClose:
			if err != nil {
				c.logger.Warn("RMQ/CONNECTION CLOSED", zap.Error(err))
				// Try reconnect but exit if ctx cancelled
				c.reconnect()
				return
			}
		case <-c.closed:
			return
		case <-c.ctx.Done():
			return
		}
	}
}

// reconnect tries to reconnect and resubscribe on failure
func (c *Client) reconnect() {
	for {
		select {
		case <-c.closed:
			return
		case <-c.ctx.Done():
			return
		default:
		}

		time.Sleep(3 * time.Second)
		c.logger.Info("RMQ/RECONNECTING...")
		if err := c.connect(); err == nil {
			c.logger.Info("RMQ/RECONNECTED")

			// Resubscribe all previous subscribers
			for _, sub := range c.subscribers {
				if err := c.Subscribe(sub.Queue, sub.RoutingKey, sub.Handler); err != nil {
					c.logger.Error("RMQ/RESUBSCRIBE FAILED", zap.String("queue", sub.Queue), zap.Error(err))
				}
			}

			// Restart monitoring connection after successful reconnect
			go c.monitorConnection()
			return
		}
	}
}

// Publish sends a JSON-encoded message to a topic (routing key)
func (c *Client) Publish(ctx context.Context, topic string, data any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Auto reconnect if connection/channel closed
	if c.conn == nil || c.conn.IsClosed() || c.channel == nil || c.channel.IsClosed() {
		c.logger.Warn("RMQ/PUB: connection or channel closed, reconnecting")
		c.reconnect()
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("RMQ/PUB: marshal error %w", err)
	}

	requestID, _ := ctx.Value("X-Request-ID").(string)
	headers := amqp.Table{}
	if requestID != "" {
		headers["X-Request-ID"] = requestID
	}

	err = c.channel.PublishWithContext(ctx,
		c.exchange,
		topic,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers:     headers,
		},
	)

	fields := []zap.Field{
		zap.String("topic", topic),
		zap.Any("request_id", requestID),
		zap.Any("payload", json.RawMessage(body)),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
		c.logger.Error("RMQ/PUB FAILED", fields...)
		return err
	}

	c.logger.Info("RMQ/PUB", fields...)
	return nil
}

// Subscribe declares queue/bindings and starts a consumer with a fixed handler signature
func (c *Client) Subscribe(queue string, routingKey string, handler any) error {
	c.mu.Lock()
	c.subscribers = append(c.subscribers, subscriberMeta{Queue: queue, RoutingKey: routingKey, Handler: handler})
	c.mu.Unlock()

	c.wg.Add(1)
	go c.runSubscriber(queue, routingKey, handler)

	return nil
}

func (c *Client) runSubscriber(queue string, routingKey string, handler any) {
	defer c.wg.Done()

	backoff := time.Second

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("RMQ/SUB: shutting down subscriber", zap.String("queue", queue), zap.String("routingKey", routingKey))
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil || conn.IsClosed() {
			c.logger.Warn("RMQ/SUB: waiting for connection...")
			time.Sleep(backoff)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			c.logger.Error("RMQ/SUB: failed to create channel", zap.Error(err))
			time.Sleep(backoff)
			continue
		}

		args := amqp.Table{}
		if c.config.QueueTTL > 0 {
			args["x-message-ttl"] = int32(c.config.QueueTTL.Milliseconds())
		}
		if c.config.DeadLetter != "" {
			args["x-dead-letter-exchange"] = c.config.DeadLetter
		}

		q, err := ch.QueueDeclare(queue, c.config.Durable, false, false, false, args)
		if err != nil {
			c.logger.Error("RMQ/SUB: queue declare failed", zap.Error(err))
			ch.Close()
			time.Sleep(backoff)
			continue
		}

		err = ch.QueueBind(q.Name, routingKey, c.exchange, false, nil)
		if err != nil {
			c.logger.Error("RMQ/SUB: queue bind failed", zap.Error(err))
			ch.Close()
			time.Sleep(backoff)
			continue
		}

		msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
		if err != nil {
			c.logger.Error("RMQ/SUB: consume failed", zap.Error(err))
			ch.Close()
			time.Sleep(backoff)
			continue
		}

		closeChan := make(chan *amqp.Error)
		ch.NotifyClose(closeChan)

		argName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

		var fields = []zap.Field{
			zap.String("queue", queue),
			zap.String("topic", routingKey),
			zap.String("handler", argName),
		}

		c.logger.Info("RMQ/SUBS STARTED", fields...)

		// Message processing loop
		processDone := make(chan error, 1)
		go func() {
			for d := range msgs {
				requestID := d.Headers["X-Request-ID"]

				raw := json.RawMessage(string(d.Body))

				var fields = []zap.Field{
					zap.String("queue", queue),
					zap.String("topic", routingKey),
					zap.String("message_id", d.MessageId),
					zap.Any("request_id", requestID),
					zap.Any("payload", &raw),
					zap.String("handler", argName),
				}

				// Deserialize message payload into expected type
				target := reflect.New(reflect.TypeOf(handler).In(0)).Interface()
				if err := json.Unmarshal(d.Body, target); err != nil {
					fields = append(fields, zap.Error(err))
					c.logger.Error("RMQ/SUB: json unmarshal failed", fields...)
					d.Nack(false, false) // reject without requeue
					continue
				}

				c.logger.Info("RMQ/SUB", fields...)

				// Call user handler func(msg any, delivery amqp.Delivery)
				// using reflection to invoke
				results := reflect.ValueOf(handler).Call([]reflect.Value{
					reflect.ValueOf(target).Elem(), // pass the struct, not pointer
					reflect.ValueOf(d),
				})

				// If handler returns error (last return), check it
				if len(results) == 1 {
					if err, ok := results[0].Interface().(error); ok && err != nil {
						c.logger.Error("RMQ/SUB: handler returned error", zap.Error(err))
						d.Nack(false, true) // requeue on handler error
						continue
					}
				}
			}
			processDone <- nil
		}()

		// Wait until channel closed or processing ends (usually channel close)
		select {
		case err := <-closeChan:
			if err != nil {
				c.logger.Warn("RMQ/SUB: channel closed with error", fields...)
			} else {
				c.logger.Debug("RMQ/SUB: channel closed normally", fields...)
			}
		case <-processDone:
			c.logger.Debug("RMQ/SUB: message processing ended", fields...)
		case <-c.ctx.Done():
			c.logger.Info("RMQ/SUB: shutting down during message processing", fields...)
		}

		ch.Close()
		time.Sleep(backoff)
	}
}

// Close gracefully closes channel and connection and waits for goroutines
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Signal shutdown to all goroutines
	c.cancel()
	close(c.closed)

	c.logger.Info("RMQ/CLOSING: waiting for subscribers to finish")
	c.wg.Wait()

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			c.logger.Warn("RMQ/CHANNEL CLOSE FAILED", zap.Error(err))
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.logger.Warn("RMQ/CONNECTION CLOSE FAILED", zap.Error(err))
		}
	}

	c.logger.Info("RMQ/CLOSED")
	return nil
}
