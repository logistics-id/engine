package log

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
)

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

func customNameEncoder(name string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(ColorWhite(Bold(name)))
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
