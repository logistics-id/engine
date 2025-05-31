package redis

import (
	"encoding/json"

	"github.com/gomodule/redigo/redis"
	"go.uber.org/zap"
)

// Config contains Redis connection settings.
type Config struct {
	Prefix   string // Key prefix
	Server   string // Redis server address
	Password string // Redis password
}

// Redis wraps a Redis connection pool and key prefix.
type Redis struct {
	Prefix string      // Prefix for all Redis keys
	Pool   *redis.Pool // Connection pool
	Logger *zap.Logger
}

// Save marshals 'value' to JSON and stores it in Redis under the key with prefix.
func (r *Redis) Save(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	conn := r.Pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", r.key(key), data)
	return err
}

// Read retrieves the JSON value from Redis under the key and unmarshals into 'out'.
func (r *Redis) Read(key string, out any) error {
	conn := r.Pool.Get()
	defer conn.Close()

	data, err := redis.Bytes(conn.Do("GET", r.key(key)))
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

// Delete removes the key from Redis.
func (r *Redis) Delete(key string) error {
	conn := r.Pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", r.key(key))
	return err
}

// ping checks Redis connectivity.
func (r *Redis) Ping() error {
	conn := r.Pool.Get()
	defer conn.Close()

	_, err := redis.String(conn.Do("PING"))
	return err
}

func (r *Redis) GetStrings(command string, key string) ([]string, error) {
	conn := r.Pool.Get()
	defer conn.Close()

	return redis.Strings(conn.Do(command, key))
}

// key adds prefix to the key.
func (r *Redis) key(k string) string {
	if r.Prefix == "" {
		return k
	}
	return r.Prefix + ":" + k
}

// Close closes the Redis connection pool.
func (r *Redis) Close() error {
	return r.Pool.Close()
}
