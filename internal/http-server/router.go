package httpserver

import (
	"log/slog"
	"reflect"
	"strings"
	"time"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/http-server/middleware"
	"expense-tracker-api/internal/storage/sqlite"

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
	authApi := api.Group("/auth")
	authApi.POST("/register", handlers.Register)
	authApi.POST("/login", handlers.Login)
	authApi.POST("/logout", handlers.Logout)

	privateApi := api.Group("/", middleware.AuthRequired(db, log, cfg))
	privateApi.GET("/auth/me", handlers.Me)
	privateApi.GET("/accounts", handlers.ListAccounts)
	privateApi.POST("/accounts", handlers.CreateAccount)
	privateApi.GET("/accounts/:id", handlers.GetAccount)
	privateApi.PATCH("/accounts/:id", handlers.UpdateAccount)
	privateApi.DELETE("/accounts/:id", handlers.DeleteAccount)
	privateApi.GET("/accounts/balances", handlers.GetAccountBalances)

	privateApi.GET("/categories", handlers.ListCategories)
	privateApi.POST("/categories", handlers.CreateCategory)
	privateApi.GET("/categories/:id", handlers.GetCategory)
	privateApi.PATCH("/categories/:id", handlers.UpdateCategory)
	privateApi.DELETE("/categories/:id", handlers.DeleteCategory)

	privateApi.GET("/transactions", handlers.ListTransactions)
	privateApi.POST("/transactions", handlers.CreateTransaction)
	privateApi.GET("/transactions/:id", handlers.GetTransaction)
	privateApi.PATCH("/transactions/:id", handlers.UpdateTransaction)
	privateApi.DELETE("/transactions/:id", handlers.DeleteTransaction)

	return router
}
