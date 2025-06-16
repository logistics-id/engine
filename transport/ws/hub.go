package ws

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/logistics-id/engine/common"
	"go.uber.org/zap"
)

type Hub struct {
	mu       sync.RWMutex
	active   map[string]map[string]*WebSocketConn
	handlers map[string]MessageRoute
	registry *PresenceRegistry
	log      *zap.Logger
}

// On registers a handler for a specific message type, with an optional binder for payload binding.
func (h *Hub) On(msgType string, handler MessageHandler, binders ...func() any) {
	var binder func() any
	if len(binders) > 0 {
		binder = binders[0]
	}
	h.handlers[msgType] = MessageRoute{
		Handler: handler,
		BindTo:  binder,
	}
}

// Register adds a new WebSocket connection to the hub for the current user session.
// It also registers the user presence in Redis and starts listening for messages.
func (h *Hub) Register(ctx context.Context, conn *WebSocketConn) {
	sessionClaim := common.GetContextSession(ctx)

	podID := getPodID()

	h.mu.Lock()
	if _, ok := h.active[sessionClaim.UserID]; !ok {
		h.active[sessionClaim.UserID] = make(map[string]*WebSocketConn)
	}

	connID := uuid.New().String()
	h.active[sessionClaim.UserID][connID] = conn
	h.mu.Unlock()

	// registering the user to the redis
	h.registry.Add(sessionClaim.UserID, podID)

	go h.listen(ctx, sessionClaim.UserID, connID, conn)
}

// Unregister removes a WebSocket connection from the hub for the current user session.
// It also removes the user presence from Redis if no more connections exist.
func (h *Hub) Unregister(ctx context.Context, connID string) {
	sessionClaim := common.GetContextSession(ctx)

	podID := getPodID()

	h.mu.Lock()
	delete(h.active[sessionClaim.UserID], connID)
	if len(h.active[sessionClaim.UserID]) == 0 {
		delete(h.active, sessionClaim.UserID)
	}

	// removing from redis
	h.registry.Remove(sessionClaim.UserID, podID)

	h.mu.Unlock()
}

// listen handles incoming messages and ping/pong keep-alive for a WebSocket connection.
// It dispatches messages to registered handlers and manages connection lifecycle.
func (h *Hub) listen(ctx context.Context, userID, connID string, conn *WebSocketConn) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				conn.Lock()
				err := conn.raw.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
				conn.Unlock()
				if err != nil {
					h.log.Warn("ping failed", zap.Error(err))
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	conn.raw.SetCloseHandler(func(code int, text string) error {
		h.log.Info("client initiated close", zap.Int("code", code), zap.String("text", text))
		return nil
	})

	for {
		conn.raw.SetReadDeadline(time.Now().Add(60 * time.Second))

		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			h.log.Warn("read message failed", zap.Error(err))
			_ = conn.raw.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "closing"))
			break
		}

		h.log.Debug("received ws message", zap.String("userID", userID), zap.ByteString("msg", rawMsg))

		var incoming *Message
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			h.log.Warn("invalid ws json", zap.ByteString("raw", rawMsg), zap.Error(err))
			conn.SendReply("error", "error parsing command")
			continue
		}

		if route, ok := h.handlers[incoming.Type]; ok {
			var payload any = incoming.Payload

			if route.BindTo != nil {
				raw, _ := json.Marshal(incoming.Payload)
				bound := route.BindTo()
				if err := json.Unmarshal(raw, &bound); err != nil {
					conn.SendReply("error", "invalid payload")
					continue
				}
				payload = bound
			}

			route.Handler(ctx, conn, payload)
		} else {
			h.log.Warn("unregistered ws message type", zap.String("type", incoming.Type))
			conn.SendReply("error", "unknown command")
		}
	}

	conn.Close()
	h.Unregister(ctx, connID)
}

// Send sends a message to all active WebSocket connections for a given user.
func (h *Hub) Send(userID string, msg []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, ok := h.active[userID]
	if !ok {
		return nil
	}
	for _, c := range conns {
		_ = c.WriteMessage(msg)
	}
	return nil
}

var hubInstance *Hub

// InitHub initializes the singleton Hub instance with a Redis pool and logger.
func InitHub(redisPool *redis.Pool, logger *zap.Logger) *Hub {
	registry := NewPresenceRegistry(redisPool)

	hubInstance = &Hub{
		log:      logger,
		registry: registry,
		handlers: make(map[string]MessageRoute),
		active:   make(map[string]map[string]*WebSocketConn),
	}

	return hubInstance
}

// GetHub returns the singleton Hub instance.
func GetHub() *Hub {
	return hubInstance
}

// getPodID returns the hostname of the current pod or machine.
func getPodID() string {
	host, _ := os.Hostname()

	return host
}
