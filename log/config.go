package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a named logger using the global config.
func NewLogger(name string, isDev bool) *zap.Logger {
	var logger *zap.Logger

	if isDev {
		logger = consoleLogger()
	} else {
		logger = jsonLogger()
	}

	return logger.Named(os.Getenv("PLATFORM")).Named(name)
}

func initialFields() map[string]any {
	hostname, _ := os.Hostname()

	return map[string]any{
		"host": hostname,
	}
}

func jsonLogger() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "lvl",
			NameKey:        "service",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "trace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		InitialFields: initialFields(),
	}

	logger, _ := cfg.Build()

	return logger
}

func consoleLogger() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "console",
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			NameKey:    "service",
			EncodeName: customNameEncoder,

			MessageKey: "message",

			TimeKey:    "time",
			EncodeTime: customTimeEncoder,

			LevelKey:    "level",
			EncodeLevel: customLevelEncoder,

			CallerKey:      "caller",
			EncodeCaller:   customCallerEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		},
	}

	logger, _ := cfg.Build()

	return logger
}
