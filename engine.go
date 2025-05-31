package engine

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/logistics-id/engine/log"
	"github.com/oklog/run"
	"go.uber.org/zap"
)

type Config struct {
	Name    string
	Version string
	Host    string
	IsDev   bool
}

var (
	Service               *Config
	Logger                *zap.Logger
	Routine               run.Group
	dependenciesReadyOnce sync.Once
	dependenciesReady     = make(chan struct{})
)

// Start initializes the service configuration and logger.
func Start(cfg *Config) *Config {
	Service = cfg
	Logger = NewLogger(cfg.Name)
	Logger.Info(fmt.Sprintf("Starting Service: %s", Service.Name))

	return Service
}

func Shutdown(cancel context.CancelFunc) {
	// Wait for the service to stop
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	Logger.Info(fmt.Sprintf("Shutdown Service: %s", Service.Name))

	cancel()

	// Give some time for graceful shutdown
	time.Sleep(6 * time.Second)
}

// NewLogger creates a named logger using the global config.
func NewLogger(name string) *zap.Logger {
	return log.BuildLogger(Service.IsDev).Named(name).With(zap.String("host", Service.Host))
}

// SignalDependenciesReady should be called once after dependencies are ready
func DependenciesReady() {
	dependenciesReadyOnce.Do(func() {
		close(dependenciesReady)
	})
}

// WaitForDependencies blocks until dependencies are ready
func WaitForDependencies() {
	<-dependenciesReady
}

func WaitForShutdownSignal() error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	Logger.Info("Received shutdown signal")
	return nil
}
