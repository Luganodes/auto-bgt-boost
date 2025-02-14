package api

import (
	"bgt_boost/internal/config"
	"bgt_boost/internal/repository"

	"github.com/gin-gonic/gin"
)

func DatabaseMiddleware(dbRepository *repository.DbRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("dbRepository", dbRepository)
		c.Next()
	}
}

func AdminMiddleware(config *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != config.AdminAPIKey {
			UnauthorizedResponse(c, "Invalid API key")
			c.Abort()
			return
		}
		c.Next()
	}
}
