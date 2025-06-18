package ws

import (
	"os"

	"github.com/gomodule/redigo/redis"
	"github.com/logistics-id/engine/broker/rabbitmq"
	"go.uber.org/zap"
)

// Default is an optional global WebSocket instance.
var Default *WebSocket

func NewDefault(redisPool *redis.Pool, broker *rabbitmq.Client, logger *zap.Logger, Origins ...string) *WebSocket {
	hostname, _ := os.Hostname()

	registry := NewRedisRegistry(redisPool)
	hub := NewHub(logger.With(zap.String("component", "hub")))
	router := NewRouter(logger.With(zap.String("component", "router")))

	limiter := NewRedisRateLimiter(redisPool, logger)
	ackstore := NewAckStore(redisPool, logger)

	sender := NewRMQSender(hostname, broker, hub, registry, logger.With(zap.String("component", "sender")))

	ws := &WebSocket{
		Hub:         hub,
		Router:      router,
		Sender:      sender,
		Registry:    registry,
		RateLimiter: limiter,
		AckStore:    ackstore,
		PodID:       hostname,
		Logger:      logger,
		Origins:     Origins,
	}

	ws.Router.Register("ack", ackstore.AckHandler)
	ws.Router.Register("restore", ws.restoreHandler)

	return ws
}
