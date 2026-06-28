package handlers

import (
	"net/http"
	"strings"

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
	ErrCodeAccountInUse         = "ACCOUNT_IN_USE"
	ErrCodeCategoryInUse        = "CATEGORY_IN_USE"
	ErrCodeInvalidRefs          = "INVALID_REFS"
	ErrCodeSameAccountTransfer  = "SAME_ACCOUNT_TRANSFER"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationErrorResponse struct {
	ErrorResponse
	Errors []FieldError `json:"errors"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Code:    code,
		Message: message,
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
	c.JSON(http.StatusBadRequest, ValidationErrorResponse{
		ErrorResponse: ErrorResponse{
			Code:    ErrCodeValidationFailed,
			Message: "validation failed",
		},
		Errors: fields,
	})
}

func formatValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "gt":
		return fe.Field() + " must be greater than " + fe.Param()
	case "gte":
		return fe.Field() + " must be greater than or equal to " + fe.Param()
	case "min":
		return fe.Field() + " must be at least " + fe.Param() + " characters"
	case "max":
		return fe.Field() + " must be at most " + fe.Param() + " characters"
	case "oneof":
		params := strings.Split(fe.Param(), " ")
		for i, param := range params {
			params[i] = "'" + param + "'"
		}
		return fe.Field() + " must be either " + strings.Join(params, " or ")
	case "uuid":
		return fe.Field() + " must be a valid UUID"
	default:
		return fe.Field() + " failed '" + fe.Tag() + "' validation"
	}
}
