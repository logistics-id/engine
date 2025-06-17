package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

// AckStore manages message tracking and acknowledgment.
type AckStore struct {
	Pool   *redis.Pool
	TTL    time.Duration
	Prefix string // e.g., "ws:ack"
	Logger *zap.Logger
}

// Save stores a message that needs to be acknowledged.
func (a *AckStore) Save(userID, msgID string, msg []byte) {
	conn := a.Pool.Get()
	defer conn.Close()
	key := a.Prefix + ":" + userID + ":" + msgID
	_, err := conn.Do("SETEX", key, int(a.TTL.Seconds()), msg)
	if err != nil && a.Logger != nil {
		a.Logger.Error("failed to save ack message", zap.String("userID", userID), zap.String("msgID", msgID), zap.Error(err))
	}
}

// AckHandler handles incoming ack messages.
func (a *AckStore) AckHandler(ctx context.Context, conn *Conn, payload json.RawMessage) error {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return err
	}
	key := a.Prefix + ":" + conn.UserID + ":" + body.ID
	c := a.Pool.Get()
	defer c.Close()
	_, err := c.Do("DEL", key)
	if err != nil && a.Logger != nil {
		a.Logger.Warn("failed to delete ack entry", zap.String("userID", conn.UserID), zap.String("msgID", body.ID), zap.Error(err))
	}
	return nil
}

func NewAckStore(pool *redis.Pool, logger *zap.Logger) *AckStore {
	return &AckStore{
		Pool:   pool,
		Logger: logger,
		Prefix: "ws:ack",
		TTL:    10 * time.Minute,
	}
}
