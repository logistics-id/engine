package common

import "context"

type ContextKey string

const (
	ContextRequestIDKey        ContextKey = "request_id"
	ContextUserKey             ContextKey = "user"
	ContextSessionKey          ContextKey = "session"
	ContextCorrelationIDKey    ContextKey = "correlation_id"
	ContextLocaleKey           ContextKey = "locale"
	ContextClientIPKey         ContextKey = "client_ip"
	ContextRequestStartTimeKey ContextKey = "request_start_time"
	ContextTraceIDKey          ContextKey = "trace_id"
	ContextSpanIDKey           ContextKey = "span_id"
)

func GetContextRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ContextRequestIDKey).(string); ok {
		return v
	}
	return ""
}

func GetContextSession(ctx context.Context) *SessionClaims {
	if v, ok := ctx.Value(ContextUserKey).(*SessionClaims); ok {
		return v
	}

	return nil
}
