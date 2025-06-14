package grpc

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/logistics-id/engine/common"
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
			vals := md.Get(string(common.ContextRequestIDKey))
			if len(vals) > 0 {
				reqID = vals[0]
			}
		}

		var reqPayload []byte
		if pb, ok := req.(proto.Message); ok {
			reqPayload, _ = json.Marshal(pb)
		}

		start := time.Now()

		// ctx = context.WithC
		ctx = context.WithValue(ctx, common.ContextRequestIDKey, reqID)

		resp, err = handler(ctx, req)

		var respPayload []byte
		if err == nil && resp != nil {
			if pb, ok := resp.(proto.Message); ok {
				respPayload, err = json.Marshal(pb)
			}
		}

		l := log.With(
			zap.String("action", "server.response"),
			zap.String("method", info.FullMethod),
			zap.String("peer", peerAddr),
			zap.String("request_id", reqID),
			zap.Any("payload", json.RawMessage(reqPayload)),
			zap.Any("response", json.RawMessage(respPayload)),
			zap.Duration("duration", time.Since(start)),
		)

		if err != nil {
			l.Error("GRPC/SERVER", zap.Error(err))
		} else {
			l.Info("GRPC/SERVER")
		}

		return resp, err
	}
}
