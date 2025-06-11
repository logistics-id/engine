package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/logistics-id/engine/common"
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
	if rid := ctx.Value(common.ContextRequestIDKey); rid != nil {
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
		if errors.Is(event.Err, sql.ErrNoRows) {
			log.Warn("PG/QUERY", zap.Error(event.Err))
		} else {
			log.Error("PG/QUERY ", zap.Error(event.Err))
		}
	} else {
		log.Info("PG/QUERY")
	}
}
