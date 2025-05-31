package grpc

import (
	"context"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	config   *Config
	log      *zap.Logger
	server   *grpc.Server
	listener net.Listener
	reg      ServiceRegistry
}

func NewServer(config *Config, logger *zap.Logger, register func(*grpc.Server)) *Server {
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		logger.Fatal("gRPC/PORT BIND FAILED", zap.String("addr", config.Address), zap.Error(err))
	}

	s := grpc.NewServer()
	register(s)

	return &Server{
		config:   config,
		log:      logger,
		server:   s,
		listener: listener,
		reg:      NewRedisRegistry(config.Namespace, config.TTL),
	}
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.reg.Register(ctx, s.config.ServiceName, s.config.Address, s.config.TTL); err != nil {
		s.log.Fatal("gRPC/REGISTRY FAILED", zap.Error(err))
	}

	s.log.Info("gRPC/SRV starting", zap.String("addr", s.config.Address))

	if err := s.server.Serve(s.listener); err != nil {
		s.log.Fatal("gRPC/SERVE FAILED", zap.Error(err))
	}

	<-ctx.Done()
	s.Shutdown(ctx)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) {
	s.log.Info("gRPC/SRV shutting down")

	if err := s.reg.Unregister(ctx, s.config.ServiceName, s.config.Address); err != nil {
		s.log.Error("gRPC/DEREGISTER FAILED", zap.Error(err))
	}

	s.server.GracefulStop()
	s.log.Info("gRPC/SRV shutdown complete")
}
