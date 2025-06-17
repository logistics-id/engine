package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/logistics-id/engine/common"
	"go.uber.org/zap"
)

// WebSocket is the main engine instance.
type WebSocket struct {
	Hub         *Hub
	Router      *Router
	Sender      Sender
	Registry    Registry
	RateLimiter RateLimiter
	AckStore    *AckStore
	PodID       string
	Logger      *zap.Logger
	Origins     []string
	IPFilter    func(ip string) bool
}

func NewWebSocket(cfg Config) *WebSocket {
	ws := &WebSocket{
		Hub:         NewHub(cfg.Logger.With(zap.String("component", "hub"), zap.String("pod", cfg.PodID))),
		Router:      NewRouter(cfg.Logger.With(zap.String("component", "router"), zap.String("pod", cfg.PodID))),
		Sender:      cfg.Sender,
		Registry:    cfg.Registry,
		RateLimiter: cfg.RateLimiter,
		AckStore:    cfg.AckStore,
		PodID:       cfg.PodID,
		Logger:      cfg.Logger,
		Origins:     cfg.Origins,
		IPFilter:    cfg.IPFilter,
	}
	if cfg.AckStore != nil {
		ws.Router.Register("ack", cfg.AckStore.AckHandler)
		ws.Router.Register("restore", ws.restoreHandler)

	}
	return ws
}

func (ws *WebSocket) On(msgType string, handler HandlerFunc) {
	ws.Router.Register(msgType, handler)
}

func (ws *WebSocket) SendToUser(ctx context.Context, userID string, payload Envelope) error {
	msg, err := json.Marshal(payload)
	if err != nil {
		if ws.Logger != nil {
			ws.Logger.Error("failed to marshal message", zap.Error(err))
		}
		return err
	}
	if payload.RequiresAck && ws.AckStore != nil && payload.ID != "" {
		ws.AckStore.Save(userID, payload.ID, msg)
	}
	return ws.Sender.SendToUser(ctx, userID, msg)
}

func (ws *WebSocket) RegisterConn(w http.ResponseWriter, r *http.Request, ctx context.Context) error {
	originCheck := func(r *http.Request) bool {
		if len(ws.Origins) == 0 {
			return true
		}
		origin := r.Header.Get("Origin")
		for _, allowed := range ws.Origins {
			if strings.EqualFold(origin, allowed) {
				return true
			}
		}

		ws.Logger.Warn("connection rejected: origin not allowed", zap.String("origin", origin))
		return false
	}

	ip := r.RemoteAddr
	if ws.IPFilter != nil && !ws.IPFilter(ip) {
		ws.Logger.Warn("connection rejected: IP not allowed", zap.String("ip", ip))
		http.Error(w, "Forbidden", http.StatusForbidden)
		return nil
	}

	upgrader := websocket.Upgrader{
		CheckOrigin:       originCheck,
		EnableCompression: true,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.Logger.Warn("websocket upgrade failed", zap.Error(err))
		return err
	}

	uc := common.GetContextSession(ctx)
	userID := uc.UserID

	c := &Conn{
		UserID:   uc.UserID,
		WS:       conn,
		Send:     make(chan []byte, 64),
		Close:    make(chan struct{}),
		LastSeen: time.Now(),
	}
	ws.Hub.Add(userID, c)
	err = ws.Registry.MarkOnline(ctx, userID, ws.PodID)
	if err == nil {
		ws.Logger.Info("user connected", zap.String("userID", userID))
	}

	if ws.AckStore != nil {
		go ws.retryUnacked(userID)
	}

	go ws.readLoop(ctx, c)
	go ws.writeLoop(c)

	return nil
}

func (ws *WebSocket) readLoop(ctx context.Context, c *Conn) {
	defer func() {
		_ = c.WS.Close()
		close(c.Close)
		ws.Hub.Remove(c)
		_ = ws.Registry.MarkOffline(ctx, c.UserID, ws.PodID)
		if ws.Logger != nil {
			ws.Logger.Info("user disconnected", zap.String("userID", c.UserID))
		}
	}()
	c.WS.SetReadLimit(65536)
	c.WS.SetReadDeadline(time.Now().Add(30 * time.Second))
	c.WS.SetPongHandler(func(string) error {
		c.WS.SetReadDeadline(time.Now().Add(30 * time.Second))
		return nil
	})
	for {
		_, msg, err := c.WS.ReadMessage()
		if err != nil {
			if ws.Logger != nil {
				ws.Logger.Warn("read message error", zap.Error(err))
			}
			return
		}
		if ws.RateLimiter != nil && !ws.RateLimiter.Allow(ctx, c.UserID) {
			if ws.Logger != nil {
				ws.Logger.Warn("rate limit exceeded", zap.String("userID", c.UserID))
			}
			continue
		}
		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			if ws.Logger != nil {
				ws.Logger.Warn("invalid JSON payload", zap.Error(err))
			}
			continue
		}
		_ = ws.Router.Dispatch(ctx, env.Type, env.Payload, c)
	}
}

func (ws *WebSocket) writeLoop(c *Conn) {
	ping := time.NewTicker(10 * time.Second)
	defer ping.Stop()
	for {
		select {
		case msg := <-c.Send:
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.TextMessage, msg); err != nil {
				if ws.Logger != nil {
					ws.Logger.Warn("write message error", zap.Error(err))
				}
				return
			}
		case <-ping.C:
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.Close:
			return
		}
	}
}

func (ws *WebSocket) retryUnacked(userID string) {
	conn := ws.AckStore.Pool.Get()
	defer conn.Close()
	pattern := fmt.Sprintf("%s:%s:*", ws.AckStore.Prefix, userID)
	replies, err := redis.Values(conn.Do("KEYS", pattern))
	if err != nil {
		ws.Logger.Warn("failed to scan for unacked messages", zap.String("userID", userID), zap.Error(err))
		return
	}
	for _, key := range replies {
		keyStr, _ := redis.String(key, nil)
		data, err := redis.Bytes(conn.Do("GET", keyStr))
		if err != nil {
			continue
		}
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		if env.ExpiresAt > 0 && time.Now().UnixMilli() > env.ExpiresAt {
			ws.Logger.Info("skipped expired message", zap.String("userID", userID), zap.String("msgID", env.ID))
			_, _ = conn.Do("DEL", keyStr) // clean up expired
			continue
		}
		_ = ws.Hub.SendLocal(userID, data)
		ws.Logger.Info("resent unacked message", zap.String("userID", userID), zap.String("key", keyStr))
	}
}

func (ws *WebSocket) restoreHandler(ctx context.Context, c *Conn, raw json.RawMessage) error {
	if ws.AckStore == nil {
		ws.Logger.Debug("no ack storages")
		return nil
	}

	var req restorePayload
	if err := json.Unmarshal(raw, &req); err != nil {
		ws.Logger.Debug("unmarshall failed", zap.Error(err))

		return err
	}

	conn := ws.AckStore.Pool.Get()
	defer conn.Close()

	pattern := fmt.Sprintf("%s:%s:*", ws.AckStore.Prefix, c.UserID)
	replies, err := redis.Values(conn.Do("KEYS", pattern))
	if err != nil {
		ws.Logger.Warn("restore: redis scan failed", zap.Error(err))
		return nil
	}

	if len(replies) == 0 {
		msg := Envelope{
			Type:    "restore",
			Payload: json.RawMessage(`"no message"`),
		}

		data, _ := json.Marshal(msg)
		c.Send <- data
		return nil
	}

	ws.Logger.Debug("restoring messages", zap.Any("s", replies))

	now := time.Now().UnixMilli()
	for _, key := range replies {
		keyStr, _ := redis.String(key, nil)
		data, err := redis.Bytes(conn.Do("GET", keyStr))
		if err != nil {
			ws.Logger.Debug("theres no pending messages")
			continue
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		if env.ExpiresAt > 0 && env.ExpiresAt < now {
			continue
		}
		if req.Since > 0 && env.ExpiresAt > 0 && env.ExpiresAt < req.Since {
			continue
		}
		_ = ws.Hub.SendLocal(c.UserID, data)
		ws.Logger.Info("restored message", zap.String("userID", c.UserID), zap.String("msgID", env.ID))
	}

	return nil
}
