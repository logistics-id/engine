package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Config struct {
	Server    string
	IsDev     bool
	JwtSecret string
}

type RestServer struct {
	Router *mux.Router
	Config *Config
	Log    *zap.Logger
	srv    *http.Server
}

// NewServer creates and configures the REST server
func NewServer(cfg *Config, logger *zap.Logger, register func(*RestServer)) *RestServer {
	r := mux.NewRouter()

	// Built-in middleware
	r.Use(RequestIDMiddleware())
	r.Use(RecoveryMiddleware(logger))
	r.Use(LoggingMiddleware(logger))
	r.Use(CORSMiddleware())

	// Standard 404 and 405 handling
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{Context: r.Context(), Request: r, Response: w}
		_ = ctx.Error(http.StatusNotFound, MsgNotFound, nil)
	})

	r.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{Context: r.Context(), Request: r, Response: w}
		_ = ctx.Error(http.StatusMethodNotAllowed, Message("method not allowed"), nil)
	})

	// Add /healthz route
	registerDefaultRoutes(r)

	srv := &RestServer{
		Router: r,
		Config: cfg,
		Log: logger.With(
			zap.String("component", "transport.rest"),
			zap.String("action", "server"),
		),
	}

	// Register application routes
	register(srv)

	return srv
}

// Start launches the HTTP server and listens for shutdown via context
func (s *RestServer) Start(ctx context.Context) {
	s.srv = &http.Server{
		Addr:         s.Config.Server,
		Handler:      s.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server
	s.Log.Info("REST/SERVER STARTED", zap.String("addr", s.Config.Server))
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.Log.Error("REST/SERVER", zap.Error(err))
	}

	// Shutdown listener
	<-ctx.Done()
	s.Shutdown(ctx)
}

// Shutdown explicitly shuts down the server
func (s *RestServer) Shutdown(ctx context.Context) {
	s.Log.Debug("REST/SERVER Shutting Down")
	if shutdownErr := s.srv.Shutdown(ctx); shutdownErr != nil {
		s.Log.Error("REST/SERVER shutdown error", zap.Error(shutdownErr))
	} else {
		s.Log.Debug("REST/SERVER server shut down cleanly")
	}
}

// Generic route handler with middleware support
func (s *RestServer) handle(method, path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{
			Context:  r.Context(),
			Request:  r,
			Response: w,
		}

		if err := handler(ctx); err != nil {
			var httpErr HTTPError
			if errors.As(err, &httpErr) {
				_ = ctx.Error(httpErr.Code, Message(httpErr.Message), nil)
			} else {
				_ = ctx.Error(http.StatusInternalServerError, MsgInternalError, err.Error())
			}
		}
	})

	var wrapped http.Handler = h
	wrapped = chainMiddleware(wrapped, mws)

	s.Router.Handle(path, wrapped).Methods(method)
}

// Shorthand route registration
func (s *RestServer) GET(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodGet, path, handler, mws)
}
func (s *RestServer) POST(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodPost, path, handler, mws)
}
func (s *RestServer) PUT(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodPut, path, handler, mws)
}
func (s *RestServer) DELETE(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodDelete, path, handler, mws)
}
func (s *RestServer) PATCH(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodPatch, path, handler, mws)
}
func (s *RestServer) OPTIONS(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodOptions, path, handler, mws)
}
func (s *RestServer) HEAD(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodHead, path, handler, mws)
}

// WithAuth applies JWT and optional role middleware
func (s *RestServer) WithAuth(requireAuth bool, roles ...string) []func(http.Handler) http.Handler {
	mws := []func(http.Handler) http.Handler{}
	if requireAuth {
		mws = append(mws, JWTAuthMiddleware(s.Config.JwtSecret))
		if len(roles) > 0 {
			mws = append(mws, RequireRole(roles[0]))
		}
	}
	return mws
}

// Registers built-in system routes
func registerDefaultRoutes(r *mux.Router) {
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods(http.MethodGet)
}
