package main

import (
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"strings"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/logger"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
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

	err = db.SeedCategories()
	if err != nil {
		log.Error("failed to seed categories", logger.Error(err))
		os.Exit(1)
	}
	log.Info("Storage initialized and categories seeded")

	// Format validation error messages to use JSON field names instead of struct field names
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
			if name == "-" {
				return fld.Name
			}
			return name
		})
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	handlers := handlers.NewHandler(log, db)
	api := router.Group("/api")
	api.GET("/accounts", handlers.ListAccounts)
	api.POST("/accounts", handlers.CreateAccount)
	api.GET("/accounts/:id", handlers.GetAccount)
	api.PATCH("/accounts/:id", handlers.UpdateAccount)
	api.DELETE("/accounts/:id", handlers.DeleteAccount)
	api.GET("/accounts/balances", handlers.GetAccountBalances)

	api.GET("/categories", handlers.ListCategories)
	api.POST("/categories", handlers.CreateCategory)
	api.GET("/categories/:id", handlers.GetCategory)
	api.PATCH("/categories/:id", handlers.UpdateCategory)
	api.DELETE("/categories/:id", handlers.DeleteCategory)

	api.GET("/transactions", handlers.ListTransactions)
	api.POST("/transactions", handlers.CreateTransaction)
	api.GET("/transactions/:id", handlers.GetTransaction)
	api.PATCH("/transactions/:id", handlers.UpdateTransaction)
	api.DELETE("/transactions/:id", handlers.DeleteTransaction)

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
