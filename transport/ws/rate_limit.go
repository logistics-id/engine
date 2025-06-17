package ws

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

// RateLimiter defines an interface to throttle users.
type RateLimiter interface {
	Allow(ctx context.Context, userID string) bool
}

// RedisRateLimiter implements a fixed-window rate limiter per user using Redis.
type RedisRateLimiter struct {
	Pool   *redis.Pool
	Limit  int           // max messages per window
	Window time.Duration // window size (e.g., 1 * time.Minute)
	Prefix string        // e.g., "ws:rl"
	Logger *zap.Logger
}

func (r *RedisRateLimiter) Allow(ctx context.Context, userID string) bool {
	conn := r.Pool.Get()
	defer conn.Close()

	key := r.Prefix + ":" + userID
	count, err := redis.Int(conn.Do("INCR", key))
	if err != nil {
		if r.Logger != nil {
			r.Logger.Error("redis rate limit INCR failed", zap.String("userID", userID), zap.Error(err))
		}
		return true // fail-open
	}

	if count == 1 {
		_, _ = conn.Do("EXPIRE", key, int(r.Window.Seconds()))
	}

	if count > r.Limit {
		if r.Logger != nil {
			r.Logger.Warn("user rate limited", zap.String("userID", userID), zap.Int("count", count))
		}
		return false
	}
	return true
}

func NewRedisRateLimiter(pool *redis.Pool, logger *zap.Logger) *RedisRateLimiter {
	return &RedisRateLimiter{
		Pool:   pool,
		Limit:  20,
		Window: 10 * time.Second,
		Prefix: "ws:rl",
		Logger: logger,
	}
}
