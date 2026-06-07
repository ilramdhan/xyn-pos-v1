package logger

import (
	"log/slog"
	"os"

	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration.
type Config struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json" or "text"
}

// New creates a slog.Logger backed by a zap production logger.
// It sets the logger as the global slog default.
func New(cfg Config) *slog.Logger {
	level := parseLevel(cfg.Level)
	zapLevel := toZapLevel(level)

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = zap.NewAtomicLevelAt(zapLevel)
	if cfg.Format == "text" {
		zapCfg.Encoding = "console"
	}

	zapLogger, err := zapCfg.Build()
	if err != nil {
		// Fallback to default slog if zap fails
		slog.Error("failed to build zap logger", "err", err)
		return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	}

	handler := slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler()
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func toZapLevel(l slog.Level) zapcore.Level {
	switch l {
	case slog.LevelDebug:
		return zapcore.DebugLevel
	case slog.LevelWarn:
		return zapcore.WarnLevel
	case slog.LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
