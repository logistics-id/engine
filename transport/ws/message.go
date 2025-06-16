package ws

import "context"

// Message represents a generic WebSocket message with a type and payload.
type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// MessageHandler defines the function signature for handling incoming WebSocket messages.
type MessageHandler func(ctx context.Context, conn *WebSocketConn, payload any)

// MessageRoute holds a handler and an optional binder for a specific message type.
type MessageRoute struct {
	Handler MessageHandler
	BindTo  func() any
}
