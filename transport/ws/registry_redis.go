package ws

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
)

// RedisRegistry implements Registry using redigo.
type RedisRegistry struct {
	Pool   *redis.Pool
	TTL    time.Duration
	Prefix string
}

func (r *RedisRegistry) key(userID string) string {
	return r.Prefix + ":user:" + userID
}

func (r *RedisRegistry) MarkOnline(ctx context.Context, userID, podID string) error {
	conn := r.Pool.Get()
	defer conn.Close()
	key := r.key(userID)
	_, err := conn.Do("SADD", key, podID)
	if err != nil {
		return err
	}
	if r.TTL > 0 {
		_, _ = conn.Do("EXPIRE", key, int(r.TTL.Seconds()))
	}
	return nil
}

func (r *RedisRegistry) MarkOffline(ctx context.Context, userID, podID string) error {
	conn := r.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("SREM", r.key(userID), podID)
	return err
}

func (r *RedisRegistry) GetUserPods(ctx context.Context, userID string) ([]string, error) {
	conn := r.Pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", r.key(userID)))
}

func NewRedisRegistry(redisPool *redis.Pool) *RedisRegistry {
	return &RedisRegistry{
		Pool:   redisPool,
		TTL:    24 * 60 * 60 * time.Second,
		Prefix: "ws",
	}
}
