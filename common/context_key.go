package common

type ContextKey string

const (
	ContextRequestIDKey        ContextKey = "request_id"
	ContextUserKey             ContextKey = "user"
	ContextSessionKey          ContextKey = "session"
	ContextCorrelationIDKey    ContextKey = "correlation_id"
	ContextLocaleKey           ContextKey = "locale"
	ContextAuthTokenKey        ContextKey = "auth_token"
	ContextRolesKey            ContextKey = "roles"
	ContextLoggerKey           ContextKey = "logger"
	ContextClientIPKey         ContextKey = "client_ip"
	ContextRequestStartTimeKey ContextKey = "request_start_time"
	ContextTraceIDKey          ContextKey = "trace_id"
	ContextSpanIDKey           ContextKey = "span_id"
)
