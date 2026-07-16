package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yurifa/expense-tracker-api/internal/http-server/httpctx"
	"github.com/yurifa/expense-tracker-api/internal/http-server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	t.Run("should generate a new request ID if not provided", func(t *testing.T) {
		t.Parallel()
		router := gin.New()
		router.Use(middleware.RequestID())
		router.GET("/", func(c *gin.Context) {
			rid := httpctx.RequestID(c)
			assert.NotEmpty(t, rid)
			c.String(http.StatusOK, rid)
		})

		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Body.String())
	})

	t.Run("should use the provided request ID", func(t *testing.T) {
		t.Parallel()
		router := gin.New()
		router.Use(middleware.RequestID())
		router.GET("/", func(c *gin.Context) {
			rid := httpctx.RequestID(c)
			assert.NotEmpty(t, rid)
			c.String(http.StatusOK, rid)
		})

		provided := uuid.NewString()
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", provided)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, provided, w.Body.String())
	})
}
