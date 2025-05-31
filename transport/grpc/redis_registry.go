package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/logistics-id/engine/ds/redis" // aliased as-is
)

type RedisServiceRegistry struct {
	Namespace string
	TTL       time.Duration
}

// NewRedisRegistry creates a Redis-backed service registry.
// `namespace` is optional and isolates keys like: grpc:services:user-service
func NewRedisRegistry(namespace string, ttl time.Duration) *RedisServiceRegistry {
	return &RedisServiceRegistry{
		Namespace: namespace,
		TTL:       ttl,
	}
}

// key formats the Redis key with optional namespace.
func (r *RedisServiceRegistry) key(service string) string {
	if r.Namespace == "" {
		return fmt.Sprintf("services:%s", service)
	}
	return fmt.Sprintf("%s:services:%s", r.Namespace, service)
}

// Register adds this instance to Redis with TTL.
func (r *RedisServiceRegistry) Register(ctx context.Context, serviceName, address string, ttl time.Duration) error {
	conn := redis.GetConn()
	defer conn.Close()

	key := r.key(serviceName)

	if _, err := conn.Do("SADD", key, address); err != nil {
		return err
	}
	if _, err := conn.Do("EXPIRE", key, int(ttl.Seconds())); err != nil {
		return err
	}

	return nil
}

// Unregister removes this instance from Redis.
func (r *RedisServiceRegistry) Unregister(ctx context.Context, serviceName, address string) error {
	conn := redis.GetConn()
	defer conn.Close()

	key := r.key(serviceName)
	_, err := conn.Do("SREM", key, address)
	return err
}

// Discover lists all currently registered addresses.
func (r *RedisServiceRegistry) Discover(ctx context.Context, serviceName string) ([]string, error) {
	return redis.GetCmd("SMEMBERS", r.key(serviceName))
}
