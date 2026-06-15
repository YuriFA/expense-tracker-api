package handlers

import "github.com/gin-gonic/gin"

const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeAccountNotFound  = "ACCOUNT_NOT_FOUND"
	ErrCodeInternal         = "INTERNAL_ERROR"
)

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"code":    code,
		"message": message,
	})
}
