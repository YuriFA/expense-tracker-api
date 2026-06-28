package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func SlogLogger(log *slog.Logger) gin.HandlerFunc {
	op := "httpserver.middleware.SlogLogger"
	log = log.With(slog.String("op", op))

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		entry := log.With(
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", c.Request.URL.RawQuery),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
		)

		switch {
		case status >= 500:
			entry.Error("request")
		case status >= 400:
			entry.Warn("request")
		default:
			entry.Info("request")
		}
	}
}
