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

type Service struct {
	Server *Server
	Client *Client
	config *Config
	logger *zap.Logger
}

func NewService(config *Config, logger *zap.Logger, register func(*grpc.Server)) *Service {
	return &Service{
		Server: NewServer(config, logger.Named("grpc"), register),
		config: config,
		logger: logger.Named("grpc"),
	}
}

func (s *Service) Start(ctx context.Context) {
	go s.Server.Start(ctx)
	s.Client = NewClient(ctx, s.config, s.logger)
}

func (s *Service) Shutdown(ctx context.Context) {
	s.Server.Shutdown(ctx)
	if s.Client != nil {
		if err := s.Client.Close(); err != nil {
			s.logger.Error("gRPC/CLIENT CLOSE ERROR", zap.Error(err))
		}
	}
}
