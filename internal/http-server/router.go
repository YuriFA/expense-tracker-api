package httpserver

import (
	"log/slog"
	"reflect"
	"strings"
	"time"

	"expense-tracker-api/internal/config"
	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/http-server/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func NewRouter(log *slog.Logger, handlers *handlers.Handler, config config.HTTPServer) *gin.Engine {
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
		AllowOrigins:     config.CorsConfig.AllowedOrigins,
		AllowMethods:     config.CorsConfig.AllowedMethods,
		AllowHeaders:     config.CorsConfig.AllowedHeaders,
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(gin.Recovery())
	router.Use(middleware.SlogLogger(log))

	api := router.Group("/api")
	api.POST("/auth/register", handlers.Register)

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

	return router
}
