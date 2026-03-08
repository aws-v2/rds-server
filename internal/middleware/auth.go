package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthMiddleware creates a middleware that validates API keys
func APIKeyAuthMiddleware(validator APIKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("x-api-key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing x-api-key header"})
			c.Abort()
			return
		}

		// Split key "ID:SECRET"
		parts := strings.SplitN(apiKey, ":", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key format. Expected accessKeyId:secretAccessKey"})
			c.Abort()
			return
		}

		accessKeyID := parts[0]
		secretAccessKey := parts[1]

		// Validate key using the validator
		userID, err := validator.ValidateKey(accessKeyID, secretAccessKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			c.Abort()
			return
		}

		// Store userId in context
		c.Set("userId", userID)

		// Continue to next handler
		c.Next()
	}
}
