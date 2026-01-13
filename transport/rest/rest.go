package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Config struct {
	Server string
	IsDev  bool
}

type RestServer struct {
	Router *mux.Router
	Config *Config
	Log    *zap.Logger
	srv    *http.Server
}

type HandlerFunc func(*Context) error

// NewServer creates and configures the REST server
func NewServer(cfg *Config, logger *zap.Logger, register func(*RestServer)) *RestServer {
	logger = logger.With(
		zap.String("component", "transport.rest"),
		zap.String("action", "server"),
	)

	r := mux.NewRouter()

	// Built-in middleware
	r.Use(CORSMiddleware())
	r.Use(RequestIDMiddleware())
	r.Use(RecoveryMiddleware(logger))
	r.Use(LoggingMiddleware(logger))

	// Collect middleware chain for special handlers
	builtInMiddleware := []func(http.Handler) http.Handler{
		CORSMiddleware(),
		RequestIDMiddleware(),
		RecoveryMiddleware(logger),
		LoggingMiddleware(logger),
	}

	// Standard 404 and 405 handling
	notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{
			Context:  r.Context(),
			Request:  r,
			Response: w,
			logger:   logger,
		}

		_ = ctx.Error(http.StatusNotFound, MsgNotFound, nil)
	})
	r.NotFoundHandler = chainMiddleware(notFoundHandler, builtInMiddleware)

	methodNotAllowedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization, X-Requested-With, X-Terminal-ID, X-Gate-Lane-ID",
			)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		ctx := &Context{
			Context:  r.Context(),
			Request:  r,
			Response: w,
			logger:   logger,
		}
		_ = ctx.Error(http.StatusMethodNotAllowed, Message("method not allowed"), nil)
	})
	r.MethodNotAllowedHandler = chainMiddleware(methodNotAllowedHandler, builtInMiddleware)

	// Add /healthz route
	registerDefaultRoutes(r)

	srv := &RestServer{
		Router: r,
		Config: cfg,
		Log:    logger,
	}

	// Register application routes
	register(srv)

	if cfg.IsDev {
		debugRoutes(r)
	}

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

	// Start the server asynchronously
	s.Log.Info("REST/SERVER STARTED", zap.String("addr", s.Config.Server))
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Log.Error("REST/SERVER", zap.Error(err))
		}
	}()
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
func (s *RestServer) handle(method, path string, handler HandlerFunc, mws []func(http.Handler) http.Handler, name ...string) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{
			Context:  r.Context(),
			Request:  r,
			Response: w,
			logger:   s.Log,
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

	route := s.Router.Handle(path, wrapped).Methods(method)
	if len(name) > 0 {
		route.Name(name[0])
	}
}

// Shorthand route registration
func (s *RestServer) GET(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodGet, path, handler, mws, handlerName(handler))
}

func (s *RestServer) POST(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodPost, path, handler, mws, handlerName(handler))
}

func (s *RestServer) PUT(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodPut, path, handler, mws, handlerName(handler))
}

func (s *RestServer) DELETE(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodDelete, path, handler, mws, handlerName(handler))
}

func (s *RestServer) PATCH(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodPatch, path, handler, mws, handlerName(handler))
}

func (s *RestServer) OPTIONS(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodOptions, path, handler, mws, handlerName(handler))
}

func (s *RestServer) HEAD(path string, handler HandlerFunc, mws []func(http.Handler) http.Handler) {
	s.handle(http.MethodHead, path, handler, mws, handlerName(handler))
}

// WithAuth applies JWT and optional role middleware
func (s *RestServer) WithAuth(requireAuth bool, roles ...string) []func(http.Handler) http.Handler {
	mws := []func(http.Handler) http.Handler{}
	if requireAuth {
		mws = append(mws, JWTAuthMiddleware())
		if len(roles) > 0 {
			mws = append(mws, RequireRole(roles[0]))
		}
	}
	return mws
}

func (s *RestServer) Restricted(permission ...string) []func(http.Handler) http.Handler {
	mws := []func(http.Handler) http.Handler{}
	mws = append(mws, JWTAuthMiddleware())
	if len(permission) > 0 {
		mws = append(mws, RequirePermission(permission[0]))
	}
	return mws
}

// Registers built-in system routes
func registerDefaultRoutes(r *mux.Router) {
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		resp := map[string]string{
			"host":    os.Getenv("HOSTNAME"),
			"service": os.Getenv("SERVICE_NAME"),
			"time":    time.Now().String(),
			"version": os.Getenv("SERVICE_VERSION"),
		}

		_ = json.NewEncoder(w).Encode(resp)
	}).Methods(http.MethodGet)
}

// cleanHandlerName returns a simplified function name with package and method.
func handlerName(fn any) string {
	ptr := reflect.ValueOf(fn).Pointer()
	full := runtime.FuncForPC(ptr).Name()
	parts := strings.Split(full, "/")
	return parts[len(parts)-1]
}
