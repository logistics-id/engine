package grpc

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	config   *Config
	log      *zap.Logger
	server   *grpc.Server
	listener net.Listener
	reg      ServiceRegistry
}

func NewServer(config *Config, logger *zap.Logger, reg ServiceRegistry, register func(*grpc.Server)) *Server {
	logger = logger.With(
		zap.String("action", "server"),
		zap.String("service_name", config.ServiceName),
	)

	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		logger.Fatal("gRPC/PORT BIND FAILED", zap.String("addr", config.Address), zap.Error(err))
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(NewZapServerLogger(logger)),
	)
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
	if err := s.reg.Register(ctx, s.config.ServiceName, s.config.AdvertisedAddress, s.config.TTL); err != nil {
		s.log.Fatal("GRPC/SERVER REGISTRY FAILED", zap.Error(err))
	}

	go s.reg.Heartbeat(ctx, s.config.ServiceName, s.config.AdvertisedAddress, s.config.TTL)

	s.log.Info("GRPC/SERVER STARTED", zap.String("addr", s.config.Address))

	go func() {
		if err := s.server.Serve(s.listener); err != nil {
			s.log.Fatal("GRPC/SERVE FAILED", zap.Error(err))
		}
	}()

	<-ctx.Done()
	s.Shutdown(ctx)
	return nil
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.reg.Unregister(ctx, s.config.ServiceName, s.config.AdvertisedAddress); err != nil {
		s.log.Error("GRPC/SERVER DEREGISTER FAILED", zap.Error(err))
	}

	s.server.GracefulStop()
	s.log.Debug("GRPC/SERVER shutdown complete")
}

func NewZapServerLogger(log *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		var peerAddr string
		if p, ok := peer.FromContext(ctx); ok {
			peerAddr = p.Addr.String()
		}

		var reqID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			vals := md.Get("request_id")
			if len(vals) > 0 {
				reqID = vals[0]
			}
		}

		var reqPayload string
		if pb, ok := req.(proto.Message); ok {
			if b, err := json.Marshal(pb); err == nil {
				reqPayload = string(b)
			}
		}

		start := time.Now()

		resp, err = handler(ctx, req)

		var respPayload string
		if pb, ok := resp.(proto.Message); ok {
			if b, err := json.Marshal(pb); err == nil {
				respPayload = string(b)
			}
		}

		log.Info("GRPC/SERVER",
			zap.String("action", "server.response"),
			zap.String("method", info.FullMethod),
			zap.String("peer", peerAddr),
			zap.String("request_id", reqID),
			zap.String("payload", reqPayload),
			zap.String("response", respPayload),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return resp, err
	}
}
