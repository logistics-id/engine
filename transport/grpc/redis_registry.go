package grpc

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/logistics-id/engine/ds/redis"
)

type RedisServiceRegistry struct {
	Namespace string
	TTL       time.Duration
}

func NewRedisRegistry(namespace string, ttl time.Duration) *RedisServiceRegistry {
	return &RedisServiceRegistry{
		Namespace: namespace,
		TTL:       ttl,
	}
}

func (r *RedisServiceRegistry) key(service string) string {
	if r.Namespace == "" {
		return fmt.Sprintf("services:%s", service)
	}
	return fmt.Sprintf("%s:services:%s", r.Namespace, service)
}

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

func (r *RedisServiceRegistry) Unregister(ctx context.Context, serviceName, address string) error {
	conn := redis.GetConn()
	defer conn.Close()

	key := r.key(serviceName)
	_, err := conn.Do("SREM", key, address)
	return err
}

func (r *RedisServiceRegistry) Discover(ctx context.Context, serviceName string) ([]string, error) {
	return redis.GetCmd("SMEMBERS", r.key(serviceName))
}

func (r *RedisServiceRegistry) Heartbeat(ctx context.Context, serviceName, address string, ttl time.Duration) {
	go func() {
		ticker := time.NewTicker(ttl / 2)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = r.Register(ctx, serviceName, address, ttl)
			}
		}
	}()
}

func (r *RedisServiceRegistry) PickOne(ctx context.Context, serviceName string) (string, error) {
	addresses, err := r.Discover(ctx, serviceName)
	if err != nil {
		return "", err
	}
	if len(addresses) == 0 {
		return "", fmt.Errorf("no healthy instances for service: %s", serviceName)
	}

	rand.Seed(time.Now().UnixNano())
	return addresses[rand.Intn(len(addresses))], nil
}
