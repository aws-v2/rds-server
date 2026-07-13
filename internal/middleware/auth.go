package middleware

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
)

func AuthContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		userID := c.GetHeader("X-User-Id")
		role := c.GetHeader("X-User-Role")
		authMethod := c.GetHeader("X-Auth-Method")

		// Store into Gin context
		c.Set("userId", userID)
		c.Set("role", role)
		c.Set("authMethod", authMethod)

		// Propagate to Request context so services can access them
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, "userId", userID)
		ctx = context.WithValue(ctx, "role", role)
		ctx = context.WithValue(ctx, "authMethod", authMethod)
		c.Request = c.Request.WithContext(ctx)

		log.Printf("the user id is: %s", userID)

		c.Next()
	}
}
