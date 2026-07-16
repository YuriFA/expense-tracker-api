package handlers

import (
	"strings"

	"github.com/yurifa/expense-tracker-api/internal/http-server/httperr"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// writeValidationError converts validator.ValidationErrors into the standard
// validation error response shape and writes it. Validator-specific glue lives
// here; everything else (codes, types, plain error writes) goes through the
// shared httperr package directly.
func writeValidationError(c *gin.Context, verrs validator.ValidationErrors) {
	fields := make([]httperr.FieldError, len(verrs))
	for i, fe := range verrs {
		fields[i] = httperr.FieldError{
			Field:   fe.Field(),
			Message: formatValidationMessage(fe),
		}
	}
	httperr.WriteValidation(c, httperr.ErrCodeValidationFailed, "validation failed", fields)
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
	case "email":
		return fe.Field() + " must be a valid email address"
	default:
		return fe.Field() + " failed '" + fe.Tag() + "' validation"
	}
}
