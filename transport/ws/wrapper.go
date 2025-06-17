package ws

import (
	"os"

	"github.com/gomodule/redigo/redis"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Default is an optional global WebSocket instance.
var Default *WebSocket

func NewDefault(redisPool *redis.Pool, rabbitChan *amqp.Channel, logger *zap.Logger, Origins ...string) *WebSocket {
	hostname, _ := os.Hostname()

	registry := NewRedisRegistry(redisPool)
	hub := NewHub(logger.With(zap.String("component", "hub")))
	router := NewRouter(logger.With(zap.String("component", "router")))

	limiter := NewRedisRateLimiter(redisPool, logger)
	ackstore := NewAckStore(redisPool, logger)

	sender, e := NewRabbitSender(rabbitChan, hostname, hub, registry, logger)

	if e != nil {
		panic(e)
	}

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
