package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Config struct {
	ServiceName       string
	Address           string
	AdvertisedAddress string
	Namespace         string
	TTL               time.Duration
	DialTimeout       time.Duration
}

type service struct {
	Server   *Server
	config   *Config
	logger   *zap.Logger
	registry ServiceRegistry
}

var Service *service

func NewService(config *Config, logger *zap.Logger, register func(*grpc.Server)) *service {
	config.TTL = 30 * time.Second
	config.DialTimeout = 5 * time.Second

	logger = logger.With(zap.String("component", "transport.grpc"))

	reg := NewRedisRegistry(config.Namespace, config.TTL)
	Service = &service{
		Server:   NewServer(config, logger, reg, register),
		config:   config,
		logger:   logger,
		registry: reg,
	}

	return Service
}

func (s *service) Start(ctx context.Context) {
	go s.Server.Start(ctx)
}

func (s *service) Shutdown(ctx context.Context) {
	s.Server.Shutdown(ctx)
}

func (s *service) Registry() ServiceRegistry {
	return s.registry
}
