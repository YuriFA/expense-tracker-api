package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	ErrCodeInvalidRequest       = "INVALID_REQUEST"
	ErrCodeValidationFailed     = "VALIDATION_FAILED"
	ErrCodeAccountNotFound      = "ACCOUNT_NOT_FOUND"
	ErrCodeCategoryNotFound     = "CATEGORY_NOT_FOUND"
	ErrCodeCategoryTypeMismatch = "CATEGORY_TYPE_MISMATCH"
	ErrCodeTransactionNotFound  = "TRANSACTION_NOT_FOUND"
	ErrCodeInternal             = "INTERNAL_ERROR"
	ErrCodeForbidden            = "FORBIDDEN"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"code":    code,
		"message": message,
	})
}

func writeValidationError(c *gin.Context, verrs validator.ValidationErrors) {
	fields := make([]FieldError, len(verrs))
	for i, fe := range verrs {
		fields[i] = FieldError{
			Field:   fe.Field(),
			Message: formatValidationMessage(fe),
		}
	}
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    ErrCodeValidationFailed,
		"message": "validation failed",
		"errors":  fields,
	})
}

func formatValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "gte":
		return fe.Field() + " must be greater than or equal to " + fe.Param()
	case "min":
		return fe.Field() + " must be at least " + fe.Param() + " characters"
	case "max":
		return fe.Field() + " must be at most " + fe.Param() + " characters"
	default:
		return fe.Field() + " failed '" + fe.Tag() + "' validation"
	}
}
