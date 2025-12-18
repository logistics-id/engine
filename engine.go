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
	"go.uber.org/zap"
)

type (
	config struct {
		Name    string
		Version string
		IsDev   bool
	}

	Lifecycle struct {
		mu      sync.Mutex
		onStart []StartHook
		onStop  []StopHook
	}

	StartHook func(ctx context.Context) error
	StopHook  func(ctx context.Context)
)

var (
	Config    *config
	Logger    *zap.Logger
	lifecycle = &Lifecycle{}
)

func OnStart(hook StartHook) {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	lifecycle.onStart = append(lifecycle.onStart, hook)
}

// Register a shutdown hook (LIFO)
func OnStop(hook StopHook) {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	lifecycle.onStop = append([]StopHook{hook}, lifecycle.onStop...)
}

// Orchestrate startup/shutdown and run main app logic
func Run(appMain func(ctx context.Context)) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for _, hook := range lifecycle.onStart {
		if err := hook(ctx); err != nil {
			os.Stderr.WriteString("Startup hook failed: " + err.Error() + "\n")
			os.Exit(1)
		}
	}

	go appMain(ctx)

	<-ctx.Done()

	for _, hook := range lifecycle.onStop {
		hook(ctx)
	}

	time.Sleep(6 * time.Second)
}

func Init(name string, version string, isDev bool) {
	Config = &config{
		Name:    name,
		Version: version,
		IsDev:   isDev,
	}

	Logger = log.NewLogger(Config.Name, Config.IsDev).With(
		zap.String("version", Config.Version),
	)

	Logger.Info(fmt.Sprintf("Starting Service: %s", Config.Name))
}
