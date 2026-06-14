package logger

import (
	"context"
	"log/slog"
)

type DiscardHandler struct {
	slog.Handler
}

func NewDiscardHandler() *DiscardHandler {
	return &DiscardHandler{}
}

func (h *DiscardHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h *DiscardHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *DiscardHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (h *DiscardHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func NewDiscardLogger() *slog.Logger {
	return slog.New(NewDiscardHandler())
}
