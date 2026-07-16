package middleware

import (
	"expense-tracker-api/internal/http-server/keys"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(keys.RequestIDHeader)

		if rid == "" {
			rid = uuid.NewString()
			c.Request.Header.Set(keys.RequestIDHeader, rid)
		} else {
			err := uuid.Validate(rid)
			if err != nil {
				rid = uuid.NewString()
				c.Request.Header.Set(keys.RequestIDHeader, rid)
			}
		}
		c.Header(keys.RequestIDHeader, rid)
		c.Next()
	}
}
