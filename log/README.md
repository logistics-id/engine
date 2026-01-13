# Logger Package

The `engine/log` package provides a standardized logging interface built on top of [uber-go/zap](https://github.com/uber-go/zap). It automatically switches between human-readable console output and structured JSON based on the environment.

## Features

- **Environment Awareness**:
  - **Dev Mode**: Pretty-printed console output with colors and simplified format.
  - **Production Mode**: Structured JSON output optimized for log aggregators (Elasticsearch, Loki, etc.).
- **Consistent Metadata**: Automatically includes `host`, `service`, `version` (via context), and `pid`.
- **Custom Encoders**: Tailored timestamp and caller formatters for better readability in dev mode.

## Installation

```bash
go get github.com/logistics-id/engine/log
```

## Quick Start

### Initialization

Use `NewLogger` to create a standard logger instance. This is typically handled by `engine.Init` but can be used standalone.

```go
package main

import (
    "github.com/logistics-id/engine/log"
    "go.uber.org/zap"
)

func main() {
    // Create a logger named "my-service"
    // true = Development Mode (Console)
    // false = Production Mode (JSON)
    logger := log.NewLogger("my-service", true)

    logger.Info("Service started",
        zap.String("port", ":8080"),
        zap.Int("workers", 5),
    )

    // Output (Dev):
    // 13/01 10:00:00 INFO @main.go:15 Service started {"port": ":8080", "workers": 5}
}
```

## Configuration

The logger configuration is internal but adapts based on the `isDev` flag passed to `NewLogger`.

| Feature | Dev Mode (`true`) | Production Mode (`false`) |
|---|---|---|
| **Encoding** | Console (Text) | JSON |
| **Level** | DEBUG | INFO |
| **Output** | Stderr | Stderr |
| **Colors** | Yes | No |
| **Time Format** | `DD/MM HH:mm:ss` | ISO8601 |
