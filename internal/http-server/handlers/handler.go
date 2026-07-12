package handlers

import (
	"log/slog"

	"expense-tracker-api/internal/auth"
	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/storage/sqlite"
)

type Handler struct {
	Logger      *slog.Logger
	DB          *sqlite.Storage
	Config      *config.HTTPServer
	RateLimiter *auth.LoginRateLimiter
}

func NewHandler(
	log *slog.Logger,
	db *sqlite.Storage,
	cfg *config.HTTPServer,
	rateLimiter *auth.LoginRateLimiter,
) *Handler {
	return &Handler{
		Logger:      log,
		DB:          db,
		Config:      cfg,
		RateLimiter: rateLimiter,
	}
}
