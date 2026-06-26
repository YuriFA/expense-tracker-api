package handlers_test

import (
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	time.Local = time.UTC // Set the default timezone to UTC for tests
	os.Exit(m.Run())
}
