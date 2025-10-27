package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Conn wraps an active WebSocket connection.
type Conn struct {
	UserID   string
	WS       *websocket.Conn
	Send     chan []byte
	Close    chan struct{}
	LastSeen time.Time
}

func (c *Conn) Reply(payload any) error {
	msg, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	select {
	case c.Send <- msg:
		return nil
	default:
		return fmt.Errorf("send buffer full for user %s", c.UserID)
	}
}

// Envelope defines the wire format for message delivery.
type Envelope struct {
	UserID      string          `json:"user_id,omitempty"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	ID          string          `json:"id,omitempty"`
	RequiresAck bool            `json:"requiresAck,omitempty"`
	ExpiresAt   int64           `json:"expiresAt,omitempty"` // epoch millis
}

type Config struct {
	Hub         *Hub
	Sender      Sender
	Registry    Registry
	RateLimiter RateLimiter
	AckStore    *AckStore
	PodID       string
	Logger      *zap.Logger
	Origins     []string             // optional allowed origin list
	IPFilter    func(ip string) bool // optional IP filter
}

type restorePayload struct {
	Since int64 `json:"since"`
}

// HandlerFunc defines a typed WebSocket message handler.
type HandlerFunc func(ctx context.Context, conn *Conn, payload json.RawMessage) error

type TypedHandlerFunc func(ctx context.Context, conn *Conn, payload any) error

// Registry interface tracks user presence.
type Registry interface {
	MarkOnline(ctx context.Context, userID, podID string) error
	MarkOffline(ctx context.Context, userID, podID string) error
	GetUserPods(ctx context.Context, userID string) ([]string, error)
	GetUsers(ctx context.Context) ([]string, error)
}

// Sender interface abstracts local + cross-pod message delivery.
type Sender interface {
	SendToUser(ctx context.Context, userID string, msg []byte) error
}
