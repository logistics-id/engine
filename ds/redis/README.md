# Redis Data Source Library

A simplified Redis client wrapper based on [Redigo](https://github.com/gomodule/redigo), providing connection pooling, JSON serialization helpers, and a global access pattern.

## Features

- **Connection Pooling**: Efficiently manages a pool of connections.
- **Global Singleton**: Easy access via global functions after initialization.
- **JSON Serialization**: Automatically marshals/unmarshals structs to/from JSON when saving/reading.
- **Prefix Namespacing**: built-in support for key prefixes to avoid collisions.
- **Integrated Logging**: Operations are logged with execution time using `zap`.

## Dependencies

- [github.com/gomodule/redigo](https://github.com/gomodule/redigo)
- [go.uber.org/zap](https://github.com/uber-go/zap)

## Installation

```bash
go get github.com/logistics-id/engine/ds/redis
```

## Quick Start

### 1. Configuration & Initialization

Initialize the connection once at application startup.

**Using Environment Variables:**

```bash
# .env
REDIS_SERVER=localhost:6379
REDIS_AUTH_PASSWORD=secret
```

```go
package main

import (
    "context"
    "github.com/logistics-id/engine/ds/redis"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Load config from env, set prefix to "myapp"
    cfg := redis.ConfigDefault("myapp")

    // Initialize (creates the global pool)
    if err := redis.NewConnection(cfg, logger); err != nil {
        logger.Fatal("Failed to connect to Redis", zap.Error(err))
    }

    // Application logic...
}
```

## API Reference

### Global Functions

Once initialized, you can use these package-level functions.

#### Save (JSON)

Stores a value as a JSON string.

```go
type Session struct {
    UserID string
    Role   string
}

session := Session{UserID: "123", Role: "admin"}
// Key becomes "myapp:session:123" if prefix is "myapp"
err := redis.Save(ctx, "session:123", session)
```

#### Read (JSON)

Retrieves and unmarshals a JSON value.

```go
var result Session
err := redis.Read(ctx, "session:123", &result)
```

#### Delete

Removes a key.

```go
err := redis.Delete(ctx, "session:123")
```

#### Raw Connection

If you need to execute raw commands supported by Redigo:

```go
conn := redis.GetConn()
defer conn.Close()

s, err := redis.String(conn.Do("GET", "some_key"))
```

### Client wrapper

The library also exposes the underlying `Redis` struct if you prefer not to use the global singleton, although `NewConnection` sets up the global instance by default.

```go
// Accessing the underlying pool
pool := redis.GetPool()
```

## Configuration

The `Config` struct controls the connection:

```go
type Config struct {
    Prefix   string // Key prefix (e.g., "service-name")
    Server   string // "host:port"
    Password string // Auth password
}
```
