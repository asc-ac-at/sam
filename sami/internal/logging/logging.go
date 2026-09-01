package logging

import (
	"io"
	"log/slog"
	"os"
)

type Config struct {
	Level slog.Level
}

func NewLogger(w io.Writer, cfg Config) *slog.Logger {
	handler := &slog.HandlerOptions{
		Level: cfg.Level,
	}
	return slog.New(slog.NewTextHandler(w, handler))
}

func Setup(cfg Config) *slog.Logger {
	return NewLogger(os.Stdout, cfg)
}
