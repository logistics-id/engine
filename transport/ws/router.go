package ws

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
)

// Router dispatches messages to registered handlers.
type Router struct {
	handlers map[string]HandlerFunc
	logger   *zap.Logger
}

func NewRouter(logger *zap.Logger) *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
		logger:   logger,
	}
}

func (r *Router) Register(msgType string, handler HandlerFunc) {
	r.handlers[msgType] = handler
	r.logger.Debug("handler registered", zap.String("type", msgType))
}

func (r *Router) Dispatch(ctx context.Context, msgType string, payload json.RawMessage, conn *Conn) error {
	if handler, ok := r.handlers[msgType]; ok {
		return handler(ctx, conn, payload)
	}
	if r.logger != nil {
		r.logger.Warn("no handler for message type", zap.String("type", msgType), zap.String("userID", conn.UserID))
	}
	return nil // optionally return error
}

// Bind decodes raw payload into typed struct.
func Bind[T any](payload json.RawMessage) (T, error) {
	var v T
	err := json.Unmarshal(payload, &v)
	return v, err
}
