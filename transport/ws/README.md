# WebSocket Transport Library

A scalable, real-time WebSocket server implementation designed for distributed systems. It uses Redis for user presence and RabbitMQ for message broadcasting across multiple pods.

## Features

- **Distributed Architecture**: Supports horizontal scaling using Redis (presence) and RabbitMQ (pub/sub).
- **User-Centric API**: Send messages to users (`SendToUser`) regardless of which pod they are connected to.
- **Rate Limiting**: Built-in Redis-based rate limiting per user.
- **ACK Mechanism**: Reliable message delivery with acknowledgments and retry logic.
- **Hub & Router**: Organized message handling based on message types.

## Installation

```bash
go get github.com/logistics-id/engine/transport/ws
```

## Usage

### 1. Initialization

Use `NewDefault` to set up a fully configured WebSocket server with all dependencies.

```go
// Dependencies
redisPool := redis.Pool{...}
rmqClient := rabbitmq.NewClient(...)

// Initialize WS Server
wsServer := ws.NewDefault(redisPool, rmqClient, logger, "http://localhost:3000") // Allowed origins
```

### 2. Registering Handlers

Register handlers to process incoming messages from clients.

```go
wsServer.On("chat_message", func(ctx context.Context, c *ws.Conn, payload json.RawMessage) error {
    var msg ChatMessage
    json.Unmarshal(payload, &msg)

    logger.Info("Received chat", zap.String("text", msg.Text))
    return nil
})
```

### 3. Connection Handler

Integrate the WebSocket upgrader into your HTTP server. Only authenticated requests can upgrade to a WebSocket connection.

> [!IMPORTANT]
> **Authentication Required**: `RegisterConn` relies on `common.GetContextSession(ctx)` to identify the user. Ensure your HTTP handler is protected by authentication middleware (like `rest.JWTAuthMiddleware`) that populates `common.ContextUserKey`.

```go
// Assuming 'server' is your rest.RestServer
server.GET("/ws", func(ctx *rest.Context) error {
    // ctx.Context() already contains SessionClaims if protected by WithAuth
    if err := wsServer.RegisterConn(ctx.Response, ctx.Request, ctx.Context()); err != nil {
        ctx.logger.Error("Upgrade failed", zap.Error(err))
    }
    return nil
}, server.WithAuth(true))
```

### 4. Sending Messages

Send messages to connected users. The engine attempts to send locally first; if the user is on another pod, it publishes to RabbitMQ.

```go
payload := ws.Envelope{
    Type: "notification",
    Payload: json.RawMessage(`{"text": "You have a new alert!"}`),
    ID: "msg-123", // Optional, for ACK
    RequiresAck: true,
}

err := wsServer.SendToUser(ctx, "user-123", payload)
```

## Architecture

1.  **Hub**: Manages local connections (in-memory).
2.  **Registry**: Stores active user sessions in Redis (`user_id -> pod_id`).
3.  **Sender**: Handles routing.
    - Checks Registry.
    - If user is local: Sends directly via Hub.
    - If user is remote: Publishes to RabbitMQ topic corresponding to the target pod.
    - Target pod consumes message and sends via its Hub.
