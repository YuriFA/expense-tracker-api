package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"expense-tracker-api/internal/config"
	httpserver "expense-tracker-api/internal/http-server"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage/sqlite"
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

	if err := db.RunMigrations(); err != nil {
		log.Error("failed to run migrations", logger.Error(err))
		os.Exit(1)
	}

	log.Info("Storage initialized")

	handlers := handlers.NewHandler(log, db, &cfg.HTTPServer)
	router := httpserver.NewRouter(log, db, handlers, &cfg.HTTPServer)

	log.Info("Starting server", slog.String("address", cfg.Address))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", logger.Error(err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info("shutting down server", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.WriteTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", logger.Error(err))
		os.Exit(1)
	}
	if err := db.Close(); err != nil {
		log.Error("failed to close database", logger.Error(err))
	}
	log.Info("Server exiting")
}
