package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"expense-tracker-api/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func bindAndValidateJSON[T any](c *gin.Context, log *slog.Logger, req *T) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
			log.Info("validation failed", logger.Error(err))
			writeValidationError(c, verrs)
			return false
		}
		log.Info("invalid request body", logger.Error(err))
		writeError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid request body")
		return false
	}

	return true
}
