package log

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func BuildLogger(isDev bool) *zap.Logger {
	logger, _ := zapConfig(isDev).Build()
	return logger
}

func zapConfig(isDev bool) zap.Config {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "lvl",
			NameKey:        "eng",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "trace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	if isDev {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		cfg.Development = true
		cfg.Encoding = "console"
		cfg.EncoderConfig = zapcore.EncoderConfig{
			NameKey:      "log",
			MessageKey:   "message",
			TimeKey:      "time",
			LevelKey:     "level",
			CallerKey:    "file",
			EncodeTime:   customTimeEncoder,
			EncodeLevel:  customLevelEncoder,
			EncodeCaller: customCallerEncoder,
		}
	}

	return cfg
}

// Custom encoders
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(ColorGray(t.Format("02/01 15:04:05")))
}

func customLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(colorized(level))
}

func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(ColorGray(fmt.Sprintf("@%s", caller.TrimmedPath())))
}

func colorized(level zapcore.Level) string {
	switch level {
	case zapcore.DebugLevel:
		return ColorGreen(level.CapitalString())
	case zapcore.InfoLevel:
		return ColorWhite(Bold(level.CapitalString()))
	case zapcore.WarnLevel:
		return ColorGray(Bold(level.CapitalString()))
	default: // Error, DPanic, Panic, Fatal
		return ColorRed(Bold(level.CapitalString()))
	}
}
