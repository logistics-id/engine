package grpc

import (
	"context"
	"math/rand"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client struct {
	conn   *grpc.ClientConn
	target string
	log    *zap.Logger
}

func NewClient(ctx context.Context, config *Config, logger *zap.Logger) *Client {
	log := logger.Named("grpc.client")
	reg := NewRedisRegistry(config.Namespace, 0)

	instances, err := reg.Discover(ctx, config.ServiceName)
	if err != nil {
		log.Fatal("DISCOVERY FAILED", zap.Error(err))
	}
	if len(instances) == 0 {
		log.Fatal("NO INSTANCES FOUND", zap.String("service", config.ServiceName))
	}

	rand.Seed(time.Now().UnixNano())
	target := instances[rand.Intn(len(instances))]

	dialCtx, cancel := context.WithTimeout(ctx, config.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, target,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatal("DIAL FAILED", zap.String("target", target), zap.Error(err))
	}

	log.Info("CONNECTED", zap.String("target", target))
	return &Client{conn: conn, target: target, log: log}
}

func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Close() error {
	c.log.Info("gRPC/CLIENT closing", zap.String("target", c.target))
	return c.conn.Close()
}
