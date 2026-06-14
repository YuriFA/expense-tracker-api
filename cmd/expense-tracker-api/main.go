package main

import (
	"log/slog"
	"net/http"
	"os"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/middleware"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	// "rest-api-youtube/internal/config"
	// "rest-api-youtube/internal/logger"
	// "rest-api-youtube/internal/storage/sqlite"
	//
	// "rest-api-youtube/internal/http-server/handlers/redirect"
	// hDelete "rest-api-youtube/internal/http-server/handlers/url/delete"
	// "rest-api-youtube/internal/http-server/handlers/url/save"
	// mwLogger "rest-api-youtube/internal/http-server/middleware/logger"
	// "github.com/go-chi/chi/v5"
	// "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.MustLoad()

	log := logger.New(logger.Options{Environment: cfg.Env, AppName: "expense-tracker-api"})

	log.Info("Logger initialized", slog.String("env", cfg.Env))
	log.Debug("Debug message: Logger is set to debug level")

	_, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to initialize storage", logger.Error(err))
		os.Exit(1)
	}

	router := gin.Default()

	router.Use(requestid.New())
	router.Use(middleware.Logger(log))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	log.Info("Starting server", slog.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", logger.Error(err))
	}

	log.Error("Server stopped unexpectedly")
}
