package httpserver

import (
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/config"
	"github.com/yurifa/expense-tracker-api/internal/http-server/handlers"
	"github.com/yurifa/expense-tracker-api/internal/http-server/middleware"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func NewRouter(
	log *slog.Logger,
	db *sqlite.Storage,
	handlers *handlers.Handler,
	cfg *config.HTTPServer,
) *gin.Engine {
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

	router := gin.New()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CorsConfig.AllowedOrigins,
		AllowMethods:     cfg.CorsConfig.AllowedMethods,
		AllowHeaders:     cfg.CorsConfig.AllowedHeaders,
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(gin.Recovery())
	router.Use(middleware.SlogLogger(log))

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/logout", handlers.Logout)
		}

		private := api.Group("/", middleware.AuthRequired(db, log, cfg))
		{
			private.GET("/auth/me", handlers.Me)
			private.GET("/accounts", handlers.ListAccounts)
			private.POST("/accounts", handlers.CreateAccount)
			private.GET("/accounts/:id", handlers.GetAccount)
			private.PATCH("/accounts/:id", handlers.UpdateAccount)
			private.DELETE("/accounts/:id", handlers.DeleteAccount)
			private.GET("/accounts/balances", handlers.GetAccountBalances)

			private.GET("/categories", handlers.ListCategories)
			private.POST("/categories", handlers.CreateCategory)
			private.GET("/categories/:id", handlers.GetCategory)
			private.PATCH("/categories/:id", handlers.UpdateCategory)
			private.DELETE("/categories/:id", handlers.DeleteCategory)

			private.GET("/transactions", handlers.ListTransactions)
			private.POST("/transactions", middleware.Idempotency(db, log), handlers.CreateTransaction)
			private.GET("/transactions/:id", handlers.GetTransaction)
			private.PATCH("/transactions/:id", handlers.UpdateTransaction)
			private.DELETE("/transactions/:id", handlers.DeleteTransaction)
		}
	}

	// API docs (OpenAPI spec + Redoc UI)
	router.StaticFile("/docs", "docs/api/redoc.html")
	router.StaticFile("/docs/openapi.yaml", "docs/api/openapi.yaml")

	return router
}
