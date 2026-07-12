package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/cookie"
	"expense-tracker-api/internal/http-server/keys"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/util"

	"github.com/gin-gonic/gin"
)

func AuthRequired(db *sqlite.Storage, log *slog.Logger, cfg *config.HTTPServer) gin.HandlerFunc {
	op := "httpserver.middleware.AuthRequired"
	log = log.With(slog.String("op", op))

	return func(c *gin.Context) {
		sessionID, err := c.Request.Cookie(cfg.SessionConfig.CookieName)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		session, err := db.GetSessionByID(sessionID.Value)
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

		expiresAt, err := util.ParseDatetime(session.ExpiresAt)
		if err != nil {
			log.Error("failed to parse session expiration date", slog.String("error", err.Error()))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if cfg.SessionConfig.SlidingExpiration && time.Until(expiresAt) < cfg.SessionConfig.TTL/4 {
			newExpiresAt := time.Now().UTC().Add(cfg.SessionConfig.TTL)
			if err := db.ExtendSession(session.ID, newExpiresAt); err != nil {
				log.Error("failed to extend session", slog.String("error", err.Error()))
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			c.SetCookieData(
				cookie.BuildSession(
					cfg.SessionConfig,
					session.ID,
					int(cfg.SessionConfig.TTL.Seconds()),
				),
			)
		}

		c.Set(keys.CurrentUserKey, user)
		c.Next()
	}
}
