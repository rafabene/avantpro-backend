package logging

import (
	"log/slog"
	"os"

	"github.com/rafabene/avantpro-backend/internal/domain"
)

// SlogLogger implementa domain.Logger usando slog do stdlib
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger cria um novo logger usando slog
func NewSlogLogger(level string) domain.Logger {
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &SlogLogger{logger: logger}
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) With(args ...any) domain.Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}
