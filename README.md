# logistics-id/engine

A lightweight, modular lifecycle and dependency manager for Go microservices.

This package is designed for [logistics-id](https://github.com/logistics-id) projects.
It speed up the development of microservices with Built-in support for databases, caches, brokers, and transport layers—so you can focus on building features, not boilerplate.

## ✨ Features

- **Lifecycle hooks:** Register `OnStart` and `OnStop` functions for any dependency.
- **Signal handling:** Handles `SIGINT`/`SIGTERM` for graceful shutdown.
- **Centralized config and structured logging:** Built-in logger with [zap](https://github.com/uber-go/zap).
- **Modular connectors:** Built-in for MongoDB, PostgreSQL, Redis, RabbitMQ, REST, gRPC, and more.

---

## 📡 Built-in Communication Libraries

This engine package provides robust, production-ready libraries for working with external systems and inter-service communication, including:

- **Data Stores:**
  - **MongoDB**: `engine/ds/mongo` — Full-featured library for MongoDB operations and lifecycle management.
  - **PostgreSQL**: `engine/ds/postgres` — Abstraction for PostgreSQL with connection pooling and helpers.
  - **Redis**: `engine/ds/redis` — Fast access and management of Redis with ready-to-use utilities.
- **Message Brokers:**
  - **RabbitMQ**: `engine/broker/rabbitmq` — Complete AMQP (RabbitMQ) client with publisher/subscriber abstractions and reliability features.
- **Transport Protocols:**
  - **gRPC**: `engine/protocol/grpc` — Idiomatic server/client layer, service discovery, and registry integration.
  - **REST**: `engine/protocol/rest` — Flexible HTTP/REST server with built-in middleware and error handling.

Each library exposes standardized initialization and shutdown APIs, and includes patterns for logging, context propagation, and best practices.

---

## 📝 Example: `main.go`

```go
package main

import (
    "context"
    "os"

    "github.com/joho/godotenv"

    "github.com/logistics-id/engine"
    "github.com/logistics-id/engine/ds/mongo"
    "github.com/logistics-id/engine/broker/rabbitmq"
    "github.com/logistics-id/engine/transport/rest"

    "github.com/yourrepo/service/proto"
)

func init() {
    godotenv.Load()
	engine.Init(proto.ServiceName)
}

func main() {
    engine.OnStart(func(ctx context.Context) error {
		mongo.NewConnection(mongo.ConfigDefault(os.Getenv("MONGODB_DATABASE")), engine.Logger)

		return rabbitmq.NewConnection(rabbitmq.ConfigDefault(engine.Config.Name), engine.Logger)
    })

    engine.OnStop(func(ctx context.Context) {
        rabbitmq.CloseConnection()
         mongo.CloseConnection()
    })

    engine.Run(func(ctx context.Context) {
        restServer := rest.NewServer(&rest.Config{
            Server:    os.Getenv("REST_SERVER"),
            IsDev:     engine.Config.IsDev,
            JwtSecret: os.Getenv("JWT_SECRET"),
        }, engine.Logger, registerRoutes)
        go restServer.Start(ctx)
        defer restServer.Shutdown(ctx)

        <-ctx.Done()
    })
}
```
