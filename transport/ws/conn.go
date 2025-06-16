package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConn wraps a Gorilla WebSocket connection with mutex for safe concurrent writes.
type WebSocketConn struct {
	*sync.Mutex
	raw *websocket.Conn
}

// UpgradeConn upgrades an HTTP connection to a WebSocket connection.
// It configures read limits, deadlines, and keep-alive via pong handler.
// Returns a wrapped WebSocketConn or an error if upgrade fails.
func UpgradeConn(w http.ResponseWriter, r *http.Request) (*WebSocketConn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins — update for production if needed.
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	conn.SetReadLimit(65536) // Limit message size to 64KB
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Refresh deadline on pong
		return nil
	})

	return &WebSocketConn{
		Mutex: &sync.Mutex{},
		raw:   conn,
	}, nil
}

// ReadMessage reads the next message from the WebSocket connection.
func (w *WebSocketConn) ReadMessage() (int, []byte, error) {
	return w.raw.ReadMessage()
}

// WriteMessage sends a message to the WebSocket client in a thread-safe manner.
func (w *WebSocketConn) WriteMessage(msg []byte) error {
	w.Lock()
	defer w.Unlock()
	return w.raw.WriteMessage(websocket.TextMessage, msg)
}

// SendReply sends a structured JSON message with a given type and payload.
func (w *WebSocketConn) SendReply(messageType string, payload any) error {
	resp := &Message{
		Type:    messageType,
		Payload: payload,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return w.WriteMessage(b)
}

// Close sends a close control frame and then closes the WebSocket connection.
func (w *WebSocketConn) Close() error {
	_ = w.raw.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "closing"))
	return w.raw.Close()
}
