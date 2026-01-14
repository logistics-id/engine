# NATS Broker Library

A lightweight, high-performance NATS client wrapper using [nats.go](https://github.com/nats-io/nats.go). Perfect for internal microservices communication requiring low latency.

## Features

- **Simple Pub/Sub**: Easy-to-use API for publishing and subscribing.
- **JSON Support**: Automatic JSON marshaling and unmarshaling.
- **Request/Reply**: Native support for synchronous request-response patterns.
- **Prefix Namespacing**: Automatic subject prefixing for service isolation.

## Dependencies

- [github.com/nats-io/nats.go](https://github.com/nats-io/nats.go)
- [go.uber.org/zap](https://github.com/uber-go/zap)

## Installation

```bash
go get github.com/logistics-id/engine/broker/nats
```

## Quick Start

### 1. Configuration & Initialization

**Environment Variables:**
```bash
NATS_SERVER=localhost:4222
NATS_AUTH_USERNAME=user
NATS_AUTH_PASSWORD=pass
```

**Initialization:**
```go
package main

import (
    "context"
    "github.com/logistics-id/engine/broker/nats"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Load default config
    cfg := nats.ConfigDefault("myservice.v1")

    // Initialize
    if err := nats.NewConnection(cfg, logger); err != nil {
        logger.Fatal("Failed to connect to NATS", zap.Error(err))
    }
    defer nats.CloseConnection()
}
```

## API Reference

### Publishing

Sends a fire-and-forget message.

```go
type Event struct {
    ID string
}

// Publishes to "myservice.v1.user.created"
err := nats.Publish("user.created", Event{ID: "u-123"})
```

### Subscribing

Listens for messages on a subject. The handler receives the context and the unmarshaled payload (as `any`, you may need to type assert or modify handler in future versions).

*Note: The current `wrapper.go` exposes `func(ctx context.Context, msg any)`, but `client.go` implementation might vary.*

```go
err := nats.Subscribe("user.created", func(ctx context.Context, msg any) {
    // msg is map[string]interface{} or the struct if unmarshaled
    logger.Info("Received", zap.Any("msg", msg))
})
```

### Request / Reply

Sends a request and waits for a response.

```go
req := Request{ID: "123"}
var resp Response

// Sends request and unmarshals response into resp
err := defaultClient.Request("user.get", req, &resp)
```
*(Note: `Request` is available on the `Client` struct but not yet exposed via static wrapper functions in some versions. Check `wrapper.go`.)*

## Client Wrapper

For direct access to `Request` method or advanced features:

```go
// Access the internal singleton if needed, or create a new client
client, err := nats.NewClient(cfg, logger)
client.Request("subject", req, &resp)
```
