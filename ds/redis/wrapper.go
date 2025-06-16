package redis

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/logistics-id/engine/common"
	"go.uber.org/zap"
)

var cache *Redis

// NewConnection initializes Redis connection pool and global defaultCache instance.
// Also assigns the global Logger for package-wide logging.
func NewConnection(cfg *Config, l *zap.Logger) error {
	pool := &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", cfg.Server, redis.DialPassword(cfg.Password))
		},
	}

	l = l.With(
		zap.String("component", "ds.redis"),
		zap.String("dsn", fmt.Sprintf("%s@%s", cfg.Server, cfg.Prefix)),
		zap.String("database", cfg.Prefix),
	)

	cache = &Redis{
		Prefix: cfg.Prefix,
		Pool:   pool,
		Logger: l,
	}

	if err := cache.Ping(); err != nil {
		l.Error("RED/CONN FAILED", zap.Error(err))
		return err
	}

	l.Info("RED/CONN CONNECTED")

	return nil
}

func GetConn() redis.Conn {
	return cache.Pool.Get()
}

func GetPool() *redis.Pool {
	return cache.Pool
}

// Save stores value under the given key in global defaultCache instance, logs the operation.
func Save(ctx context.Context, key string, value any) error {
	if cache == nil {
		return ErrNotInitialized()
	}

	started := time.Now()
	err := cache.Save(key, value)

	cache.Logger.Info("RED/QUERY",
		zap.String("action", "save"),
		zap.String("key", key),
		zap.Duration("duration", time.Since(started)),
		zap.String("request_id", common.GetContextRequestID(ctx)),
		zap.Error(err),
	)

	return err
}

// Read retrieves value stored under the given key into out from global defaultCache, logs the operation.
func Read(ctx context.Context, key string, out any) error {
	if cache == nil {
		return ErrNotInitialized()
	}

	started := time.Now()
	err := cache.Read(key, out)

	cache.Logger.Info("RED/QUERY",
		zap.String("action", "read"),
		zap.String("key", key),
		zap.Duration("duration", time.Since(started)),
		zap.String("request_id", common.GetContextRequestID(ctx)),
		zap.Error(err),
	)

	return err
}

func GetCmd(cmd string, key string) ([]string, error) {
	return cache.GetStrings(cmd, key)
}

// Delete removes the given key from global defaultCache instance, logs the operation.
func Delete(ctx context.Context, key string) error {
	if cache == nil {
		return ErrNotInitialized()
	}

	started := time.Now()
	err := cache.Delete(key)

	cache.Logger.Info("RED/QUERY",
		zap.String("action", "delete"),
		zap.String("key", key),
		zap.Duration("duration", time.Since(started)),
		zap.String("request_id", common.GetContextRequestID(ctx)),
		zap.Error(err),
	)

	return err
}

func ConfigDefault(prefix string) *Config {
	return &Config{
		Prefix:   prefix,
		Server:   os.Getenv("REDIS_SERVER"),
		Password: os.Getenv("REDIS_AUTH_PASSWORD"),
	}
}

// ErrNotInitialized returns an error for uninitialized defaultCache.
func ErrNotInitialized() error {
	return fmt.Errorf("redis defaultCache is not initialized; call NewConnection first")
}
