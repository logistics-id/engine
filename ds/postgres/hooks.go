package postgres

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type ZapQueryHook struct {
	Logger *zap.Logger
}

func (h *ZapQueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	event.StartTime = time.Now()
	return ctx
}

func (h *ZapQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	requestID := "-"
	if rid := ctx.Value("X-Request-ID"); rid != nil {
		if str, ok := rid.(string); ok {
			requestID = str
		}
	}

	h.Logger.Info("PG/QUERY",
		zap.String("query", event.Query),
		zap.Duration("duration", time.Since(event.StartTime)),
		zap.String("request_id", requestID),
		zap.Error(event.Err),
	)
}
