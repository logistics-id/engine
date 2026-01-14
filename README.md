# logistics-id/engine

**Engine** is a powerful, production-ready framework for rapidly developing Go microservices at [logistics-id](https://github.com/logistics-id).

Engine accelerates microservice delivery by providing robust, integrated libraries for essential infrastructure, including databases (MongoDB, PostgreSQL, Redis), message brokers (RabbitMQ), and transport protocols (REST, gRPC). Its built-in lifecycle management, structured logging, and unified architecture eliminate boilerplate and complexity.

With Engine, development can confidently build, deploy, and scale microservices faster—focusing purely on delivering impactful business logic, not managing infrastructure.

## ✨ Features

- **Lifecycle hooks:** Register `OnStart` and `OnStop` functions for any dependency.
- **Signal handling:** Handles `SIGINT`/`SIGTERM` for graceful shutdown.
- **Centralized config and structured logging:** Built-in logger with [zap](https://github.com/uber-go/zap).
- **Modular libraries:** Built-in for MongoDB, PostgreSQL, Redis, RabbitMQ, REST, gRPC, and more.

---

## 📡 Built-in Communication Libraries

This engine package provides robust, production-ready libraries for working with external systems and inter-service communication, including:

- **Data Stores:**
  - **[MongoDB](ds/mongo/README.md)**: `engine/ds/mongo` — Full-featured library for MongoDB operations and lifecycle management.
  - **[PostgreSQL](ds/postgres/README.md)**: `engine/ds/postgres` — Abstraction for PostgreSQL with connection pooling and helpers.
  - **[Redis](ds/redis/README.md)**: `engine/ds/redis` — Fast access and management of Redis with ready-to-use utilities.
- **Message Brokers:**
  - **[RabbitMQ](broker/rabbitmq/README.md)**: `engine/broker/rabbitmq` — Complete AMQP (RabbitMQ) client with publisher/subscriber abstractions and reliability features.
  - **[NATS](broker/nats/README.md)**: `engine/broker/nats` — NATS client integration for lightweight messaging.
- **Transport Protocols:**
  - **[gRPC](transport/grpc/README.md)**: `engine/transport/grpc` — Idiomatic server/client layer, service discovery, and registry integration.
  - **[REST](transport/rest/README.md)**: `engine/transport/rest` — Flexible HTTP/REST server with built-in middleware and error handling.
  - **[WebSockets](transport/ws/README.md)**: `engine/transport/ws` — WebSocket server implementation for real-time communication.

## 🛠 Core & Utilities

- **[Common](common/README.md)**: `engine/common` — Base repositories, use cases, and shared utilities.
- **[Validation](validate/README.md)**: `engine/validate` — Input validation and assertions.
- **[Logging](log/README.md)**: `engine/log` — Environment-aware structure logging.

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
	engine.Init("service-name", "v1.0.0", false)
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
