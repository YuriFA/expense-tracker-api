package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/keys"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
)

func AuthRequired(db *sqlite.Storage, log *slog.Logger, cfg *config.HTTPServer) gin.HandlerFunc {
	op := "httpserver.middleware.AuthRequired"
	log = log.With(slog.String("op", op))

	return func(c *gin.Context) {
		sessionId, err := c.Request.Cookie(cfg.SessionConfig.CookieName)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		session, err := db.GetSessionByID(sessionId.Value)
		if err != nil {
			if errors.Is(err, storage.ErrSessionNotFound) {
				log.Info("invalid or expired session")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			log.Error("failed to get session by ID", slog.String("error", err.Error()))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		user, err := db.GetUserByID(session.UserID)
		if err != nil {
			log.Error("failed to get user by ID", slog.String("error", err.Error()))
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set(keys.CurrentUserKey, user)
		c.Next()
	}
}
