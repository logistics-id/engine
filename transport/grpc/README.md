# gRPC Transport Library

A gRPC server wrapper that integrates service discovery (via Redis) and structured logging.

## Features

- **Service Registry**: Automatically registers the service in Redis for discovery by other services.
- **Logging Interceptor**: Logs all unary gRPC calls with execution time and metadata using `zap`.
- **Health Checks**: Background heartbeat mechanism to keep the service registration alive.
- **Context Propagation**: Automatically extracts and injects request IDs using `common.ContextRequestIDKey`.

## Installation

```bash
go get github.com/logistics-id/engine/transport/grpc
```

## Quick Start

### 1. Configuration & Initialization

```go
package main

import (
    "context"
    "time"
    "github.com/logistics-id/engine/transport/grpc"
    "go.uber.org/zap"
    pb "github.com/your/repo/proto"
)

func main() {
    logger, _ := zap.NewProduction()

    cfg := &grpc.Config{
        ServiceName:       "user-service",
        Address:           ":9090",
        AdvertisedAddress: "10.0.0.5:9090", // Address accessible by others
        Namespace:         "microservices",
        TTL:               10 * time.Second,
    }

    // Initialize Server
    server := grpc.NewServer(cfg, logger, nil, func(s *google_grpc.Server) {
        // Register your gRPC service implementation
        pb.RegisterUserServiceServer(s, &MyUserService{})
    })

    // Start Server
    server.Start(context.Background())
}
```

### Service Discovery

The server uses `RedisRegistry` (by default) to store service locations.

- **Register**: On start, adds key `namespace:service_name:address`.
- **Heartbeat**: Periodically refreshes the TTL of the key.
- **Unregister**: Removes the key on graceful shutdown.

Clients can use the registry to find available instances of a service.
