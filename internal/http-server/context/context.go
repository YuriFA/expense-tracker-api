package context

import (
	"expense-tracker-api/internal/http-server/keys"
	"expense-tracker-api/internal/storage"

	"github.com/gin-gonic/gin"
)

func CurrentUser(c *gin.Context) *storage.User {
	val, exists := c.Get(keys.CurrentUserKey)
	if !exists {
		return nil
	}
	user, ok := val.(*storage.User)
	if !ok {
		return nil
	}
	return user
}
