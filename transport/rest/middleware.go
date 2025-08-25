package rest

import (
	"bytes"
	"context"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/logistics-id/engine/common"
	"go.uber.org/zap"
)

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := uuid.New().String()
			ctx := context.WithValue(r.Context(), common.ContextRequestIDKey, reqID)
			r = r.WithContext(ctx)
			w.Header().Set(string(common.ContextRequestIDKey), reqID)
			next.ServeHTTP(w, r)
		})
	}
}

func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := common.GetContextRequestID(r.Context())

			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(rec, r)

			logger.Info("REST/SERVER",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.Int("status", rec.statusCode),
				zap.String("user_agent", r.UserAgent()),
				zap.String("remote", getRealIP(r)),
				zap.String("request_id", reqID),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					ctx := &Context{
						Context:  r.Context(),
						Request:  r,
						Response: w,
					}

					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.ByteString("stack", debug.Stack()),
					)

					ctx.Error(http.StatusInternalServerError, MsgInternalError, err)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func JWTAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := &Context{
				Context:  r.Context(),
				Request:  r,
				Response: w,
			}

			// 1) Try Authorization: Bearer <jwt>
			tokenStr := ""
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}

			// 2) Fallback: query param ?token=... or ?access_token=...
			if tokenStr == "" {
				q := r.URL.Query()
				if v := strings.TrimSpace(q.Get("token")); v != "" {
					tokenStr = v
				} else if v := strings.TrimSpace(q.Get("access_token")); v != "" {
					tokenStr = v
				}
			}

			// 3) Optional fallback: Sec-WebSocket-Protocol carrying the token
			//    e.g. client: new WebSocket(url, ['jwt', '<token>'])
			if tokenStr == "" {
				if protos := r.Header.Values("Sec-WebSocket-Protocol"); len(protos) > 0 {
					// Flatten and pick the last non-empty that isn't a known label
					for _, p := range protos {
						for _, part := range strings.Split(p, ",") {
							part = strings.TrimSpace(part)
							if part == "" || strings.EqualFold(part, "jwt") || strings.EqualFold(part, "bearer") {
								continue
							}
							tokenStr = part
						}
					}
				}
			}

			if tokenStr == "" {
				ctx.Error(http.StatusUnauthorized, MsgUnauthorized, nil)
				return
			}

			claims, err := common.TokenDecode(tokenStr)
			if err != nil || claims == nil {
				ctx.Error(http.StatusUnauthorized, MsgUnauthorized, nil)
				return
			}

			ctxUsr := context.WithValue(r.Context(), common.ContextUserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctxUsr))
		})
	}
}

func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !common.ValidTokenPermission(r.Context(), perm) {
				ctx := &Context{
					Context:  r.Context(),
					Request:  r,
					Response: w,
				}

				ctx.Error(http.StatusForbidden, MsgForbidden, nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := &Context{
				Context:  r.Context(),
				Request:  r,
				Response: w,
			}

			// todo, fix role based on permissions
			_, ok := r.Context().Value(common.ContextUserKey).(*common.SessionClaims)
			// if !ok || claims.Role != role {
			if !ok {
				ctx.Error(http.StatusForbidden, MsgForbidden, nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func chainMiddleware(h http.Handler, mws []func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
