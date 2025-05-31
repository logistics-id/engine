package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	target string
	log    *zap.Logger
}

func NewClient(_ context.Context, config *Config, logger *zap.Logger) *Client {
	log := logger.Named("grpc.client")
	reg := NewRedisRegistry(config.Namespace, 0)

	// Service discovery
	target, err := reg.PickOne(context.Background(), config.ServiceName)
	if err != nil {
		log.Fatal("DISCOVERY FAILED", zap.Error(err))
	}

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Fatal("DIAL FAILED", zap.String("target", target), zap.Error(err))
	}

	log.Info("CONNECTED", zap.String("target", target))
	return &Client{
		conn:   conn,
		target: target,
		log:    log,
	}
}

func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Close() error {
	c.log.Info("gRPC/CLIENT closing", zap.String("target", c.target))
	return c.conn.Close()
}
