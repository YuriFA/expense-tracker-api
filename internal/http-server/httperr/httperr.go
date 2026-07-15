// Package httperr provides the shared HTTP error response shape and a small
// helper to write it. Handlers and middleware use the same types and codes so
// clients can parse errors uniformly regardless of which layer produced them.
package httperr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	ErrCodeUserAlreadyExists      = "USER_ALREADY_EXISTS"
	ErrCodeInvalidCredentials     = "INVALID_CREDENTIALS"
	ErrCodeInvalidRequest         = "INVALID_REQUEST"
	ErrCodeValidationFailed       = "VALIDATION_FAILED"
	ErrCodeAccountNotFound        = "ACCOUNT_NOT_FOUND"
	ErrCodeCategoryNotFound       = "CATEGORY_NOT_FOUND"
	ErrCodeCategoryAlreadyExists  = "CATEGORY_ALREADY_EXISTS"
	ErrCodeCategoryTypeMismatch   = "CATEGORY_TYPE_MISMATCH"
	ErrCodeTransactionNotFound    = "TRANSACTION_NOT_FOUND"
	ErrCodeInternal               = "INTERNAL_ERROR"
	ErrCodeForbidden              = "FORBIDDEN"
	ErrCodeAccountInUse           = "ACCOUNT_IN_USE"
	ErrCodeCategoryInUse          = "CATEGORY_IN_USE"
	ErrCodeInvalidRefs            = "INVALID_REFS"
	ErrCodeSameAccountTransfer    = "SAME_ACCOUNT_TRANSFER"
	ErrCodeUnauthorized           = "UNAUTHORIZED"
	ErrCodeTooManyRequests        = "TOO_MANY_REQUESTS"
	ErrCodeIdempotencyKeyMissing  = "IDEMPOTENCY_KEY_MISSING"
	ErrCodeIdempotencyKeyInUse    = "IDEMPOTENCY_KEY_IN_USE"
	ErrCodeIdempotencyKeyMismatch = "IDEMPOTENCY_KEY_MISMATCH"
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

// Write sends a JSON error response with the given status, code, and message.
// It aborts the gin context so subsequent handlers do not run.
func Write(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// WriteValidation sends a 400 with the standard validation-error shape.
func WriteValidation(c *gin.Context, code, message string, errors []FieldError) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ValidationErrorResponse{
		ErrorResponse: ErrorResponse{
			Code:    code,
			Message: message,
		},
		Errors: errors,
	})
}
