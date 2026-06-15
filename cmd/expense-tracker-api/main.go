package main

import (
	"log/slog"
	"net/http"
	"os"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.MustLoad()

	log := logger.New(logger.Options{Environment: cfg.Env, AppName: "expense-tracker-api"})

	log.Info("Logger initialized", slog.String("env", cfg.Env))
	log.Debug("Debug message: Logger is set to debug level")

	db, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to initialize storage", logger.Error(err))
		os.Exit(1)
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	handlers := handlers.NewHandler(log, db)
	router.GET("/accounts", handlers.ListAccounts)
	router.POST("/accounts", handlers.CreateAccount)
	router.GET("/accounts/:id", handlers.GetAccount)
	router.PATCH("/accounts/:id", handlers.UpdateAccount)
	router.DELETE("/accounts/:id", handlers.DeleteAccount)

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
