package rest

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ctxKey string

const (
	userKey ctxKey = "user"
)

type CustomClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := uuid.New().String()
			ctx := context.WithValue(r.Context(), "request_id", reqID)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", reqID)
			next.ServeHTTP(w, r)
		})
	}
}

func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := r.Context().Value("request_id")

			next.ServeHTTP(w, r)

			logger.Info("REST/SERVER",
				zap.String("action", "request"),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote", r.RemoteAddr),
				zap.String("request_id", fmt.Sprint(reqID)),
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
					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.ByteString("stack", debug.Stack()),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func JWTAuthMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims := &CustomClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(userKey).(*CustomClaims)
			if !ok || claims.Role != role {
				http.Error(w, "Forbidden: role mismatch", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func FromContext(r *http.Request) *CustomClaims {
	claims, _ := r.Context().Value(userKey).(*CustomClaims)
	return claims
}

func chainMiddleware(h http.Handler, mws []func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
