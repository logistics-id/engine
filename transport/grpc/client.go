package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type Client struct {
	conn   *grpc.ClientConn
	target string
	log    *zap.Logger
}

var ErrServiceNotInitialized = errors.New("grpc.Service is not initialized")

type GRPCClientFactory[T any] func(grpc.ClientConnInterface) T

// NewClient creates a new gRPC client for the given serviceName.
// It uses the registry, logger, and config from the global Service instance.
func NewClient(ctx context.Context, serviceName string) (*Client, error) {
	if Service == nil {
		return nil, ErrServiceNotInitialized
	}

	log := Service.logger.With(
		zap.String("action", "client"),
		zap.String("service_name", serviceName),
		zap.String("request_id", ctx.Value("request_id").(string)),
	)

	reg := Service.registry
	dialTimeout := Service.config.DialTimeout
	if dialTimeout == 0 {
		dialTimeout = 5 * time.Second // default
	}

	target, err := reg.PickOne(ctx, serviceName)
	if err != nil {
		log.Error("DISCOVERY FAILED", zap.Error(err))
		return nil, err
	}

	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(NewZapClientLogger(log)),
	)

	if err != nil {
		log.Error("DIAL FAILED", zap.String("service_host", target), zap.Error(err))
		return nil, err
	}

	return &Client{
		conn:   conn,
		target: target,
		log:    log,
	}, nil
}

func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// GetClient returns a typed gRPC client and a closer.
//   - ctx: your context
//   - serviceName: the gRPC service name (as registered in your system)
//   - factory: generated constructor, e.g. pb.NewAuthServiceClient
//
// Usage:
//
//	client, closeFn, err := grpc.GetClient(ctx, "auth-service", pb.NewAuthServiceClient)
//	defer closeFn()
//
// Now use `client` as your typed client.
func GetClient[T any](
	ctx context.Context,
	serviceName string,
	factory GRPCClientFactory[T],
) (client T, closer func(), err error) {
	cli, err := NewClient(ctx, serviceName)
	if err != nil {
		var zero T
		return zero, nil, err
	}

	return factory(cli.Conn()), func() { cli.Close() }, nil
}

func NewZapClientLogger(log *zap.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {

		reqID := ctx.Value("request_id").(string)

		var reqPayload []byte
		if pb, ok := req.(proto.Message); ok {
			reqPayload, _ = json.Marshal(pb)
		}

		md := metadata.Pairs("x-request-id", reqID)
		ctx = metadata.NewOutgoingContext(ctx, md)

		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)

		var respPayload []byte
		if pb, ok := reply.(proto.Message); ok {
			respPayload, err = json.Marshal(pb)
		}

		log.Info("GRPC/CLIENT",
			zap.String("method", method),
			zap.String("service_host", cc.Target()),
			zap.String("request_id", reqID),
			zap.Any("payload", json.RawMessage(reqPayload)),
			zap.Any("response", json.RawMessage(respPayload)),
			zap.Duration("duration", time.Duration(time.Since(start))),
			zap.Error(err),
		)
		return err
	}
}
