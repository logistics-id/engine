package grpc

import (
	"context"
	"time"
)

// ServiceRegistry defines the interface for service registration and discovery.
type ServiceRegistry interface {
	Register(ctx context.Context, serviceName, address string, ttl time.Duration) error
	Unregister(ctx context.Context, serviceName, address string) error
	Discover(ctx context.Context, serviceName string) ([]string, error)
	Heartbeat(ctx context.Context, serviceName, address string, ttl time.Duration)
	PickOne(ctx context.Context, serviceName string) (string, error)
}
