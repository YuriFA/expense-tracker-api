package handlers

import (
	"log/slog"

	"github.com/yurifa/expense-tracker-api/internal/auth"
	"github.com/yurifa/expense-tracker-api/internal/config"
	"github.com/yurifa/expense-tracker-api/internal/http-server/httpctx"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
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

// loggerFor returns a request-scoped logger with `op` and `request_id` attributes.
// request_id comes from the X-Request-ID header populated by middleware.RequestID.
func (h *Handler) loggerFor(c *gin.Context, op string) *slog.Logger {
	return h.Logger.With(
		slog.String("op", op),
		slog.String("request_id", httpctx.RequestID(c)),
	)
}
