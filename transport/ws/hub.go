package ws

import (
	"sync"

	"go.uber.org/zap"
)

// Hub tracks user connections.
type Hub struct {
	mu      sync.RWMutex
	sockets map[string]map[*Conn]struct{}
	logger  *zap.Logger
}

func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		sockets: map[string]map[*Conn]struct{}{},
		logger:  logger,
	}
}

func (h *Hub) Add(userID string, conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.sockets[userID]; !ok {
		h.sockets[userID] = map[*Conn]struct{}{}
	}
	h.sockets[userID][conn] = struct{}{}
	h.logger.Info("connection added", zap.String("userID", userID))
}

func (h *Hub) Remove(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.sockets[conn.UserID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			h.logger.Info("last connection removed", zap.String("userID", conn.UserID))
			delete(h.sockets, conn.UserID)
		} else {
			h.logger.Info("connection removed", zap.String("userID", conn.UserID))
		}
	}
}

func (h *Hub) SendLocal(userID string, msg []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.sockets[userID] {
		select {
		case conn.Send <- msg:
		default:
			h.logger.Warn("dropped message due to full channel", zap.String("userID", userID))
		}
	}
	return nil
}

// ListUserIDs returns all currently connected user IDs.
func (h *Hub) ListUserIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]string, 0, len(h.sockets))
	for userID := range h.sockets {
		ids = append(ids, userID)
	}
	return ids
}
