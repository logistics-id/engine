# RabbitMQ Broker Library

A robust RabbitMQ client wrapper using [amqp091-go](https://github.com/rabbitmq/amqp091-go), designed for reliability with automatic reconnection, dead-letter exchanges, and type-safe message handling.

## Features

- **Auto-Reconnection**: Automatically attempts to reconnect and resubscribe if the connection is lost.
- **Dead Letter Support**: Configurable DLX for failed message handling.
- **Reflection-Based Handling**: Simplifies subscribers by automatically unmarshaling JSON payloads into struct arguments.
- **Context Propagation**: Automatically propagates `X-Request-ID` (mapped to `common.ContextRequestIDKey`) and other context metadata.
- **Integrated Logging**: Detailed logs for publish/subscribe events using `zap`.

## Dependencies

- [github.com/rabbitmq/amqp091-go](https://github.com/rabbitmq/amqp091-go)
- [go.uber.org/zap](https://github.com/uber-go/zap)

## Installation

```bash
go get github.com/logistics-id/engine/broker/rabbitmq
```

## Quick Start

### 1. Configuration & Initialization

Load configuration from environment variables and initialize the connection.

**Environment Variables:**
```bash
RABBIT_SERVER=localhost:5672
RABBIT_AUTH_USERNAME=guest
RABBIT_AUTH_PASSWORD=guest
```

**Initialization:**
```go
package main

import (
    "context"
    "github.com/logistics-id/engine/broker/rabbitmq"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Create config with a prefix (namespace)
    cfg := rabbitmq.ConfigDefault("myservice")

    // Initialize global connection
    if err := rabbitmq.NewConnection(cfg, logger); err != nil {
        logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
    }
    defer rabbitmq.CloseConnection()

    // Keep app running...
}
```

## API Reference

### Publishing Messages

Use `Publish` to send a message to a topic (routing key). The data will be automatically marshaled to JSON.

```go
type OrderCreated struct {
    ID     string
    Amount float64
}

// Publish to "myservice.orders.created" (prefix + topic)
err := rabbitmq.Publish(ctx, "orders.created", OrderCreated{ID: "123", Amount: 50.0})
```

### Subscribing to Messages

Use `Subscribe` to listen for messages. The library uses reflection to match the handler argument type.

#### Handler Signature
The handler function must accept two arguments:
1. **Payload**: A struct (pass by value) matching the JSON message.
2. **Delivery**: `amqp.Delivery` for accessing raw message details (headers, etc).

It usually returns an `error`. If it returns an error, the message is `Nack`-ed (requeued or sent to DLX). If `nil`, it is `Ack`-ed.

```go
// Define payload struct
type OrderCreated struct {
    ID     string
    Amount float64
}

// Define handler
func handleOrderCreated(payload OrderCreated, d amqp.Delivery) error {
    fmt.Printf("Received order: %s amount: %f\n", payload.ID, payload.Amount)
    return nil // Success
}

// Register subscription
// Queue name will be "myservice.orders.created"
err := rabbitmq.Subscribe("orders.created", handleOrderCreated)
```

### Advanced Configuration

You can customize the `Config` struct before initialization.

```go
cfg := rabbitmq.ConfigDefault("myservice")
cfg.ExchangeType = "direct" // Default is "topic"
cfg.Durable = false         // Default is true
cfg.QueueTTL = 60 * time.Second
```

### Manual Client

If you need multiple connections or don't want to use the global singleton:

```go
client, err := rabbitmq.NewClient(cfg, logger)
client.Publish(ctx, "topic", data)
```
