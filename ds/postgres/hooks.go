package postgres

import (
	"context"
	"strings"
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
	if rid := ctx.Value("request_id"); rid != nil {
		if str, ok := rid.(string); ok {
			requestID = str
		}
	}

	log := h.Logger.With(
		zap.String("event", event.Operation()),
		zap.String("query", strings.ReplaceAll(event.Query, "\"", "")),
		zap.String("request_id", requestID),
		zap.Duration("duration", time.Since(event.StartTime)),
	)

	if event.Err != nil {
		log.Error("PG/QUERY ", zap.Error(event.Err))
	} else {
		log.Info("PG/QUERY")
	}
}
