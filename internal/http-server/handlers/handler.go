package handlers

import (
	"log/slog"

	"expense-tracker-api/internal/storage/sqlite"
)

type Handler struct {
	Logger *slog.Logger
	DB     *sqlite.Storage
}

func NewHandler(log *slog.Logger, db *sqlite.Storage) *Handler {
	return &Handler{
		Logger: log,
		DB:     db,
	}
}
