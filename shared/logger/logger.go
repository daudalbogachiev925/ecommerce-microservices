package logger

import (
	"log/slog"
	"os"
)

// New делает логгер, который сразу пишет service и json — удобно парсить
func New(service string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With("service", service)
}
