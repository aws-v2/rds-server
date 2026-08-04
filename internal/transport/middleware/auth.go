package middleware

import (
	"fmt"
	"log"
	"net/http"
	"rds/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		userID := c.GetHeader("X-User-Id")
		role := c.GetHeader("X-User-Role")
		authMethod := c.GetHeader("X-Auth-Method")
		requestID := c.GetHeader("X-Request-Id")

		fmt.Printf("userID=%v", c.Request.URL)
		// TODO: this solution works but its inelegant, change this to somehtign better
		headers := [4]string{userID, role, authMethod, requestID}

		c.Set("userId", userID)
		c.Set("role", role)
		c.Set("authMethod", authMethod)
		c.Set("requestID", requestID)
		// the idea behind authMethod !="None" is about public endpoints,
		// ie the public manifest & docs,
		//  the authMethod is set to "None" in the api gateway for all public endpoints,
		// the assumption here is
		if strings.Contains(requestID, "public-req") {
			c.Next()

		} else {
			for iter, header := range headers {

				if header == "" {
					log.Printf("[Middleware] header %v not available, droping request", iter)
					utils.RespondError(c, http.StatusUnauthorized, fmt.Errorf("Missing headers you dumb f*ck"))
					c.Abort()
					return
				}

			}
		}

	}
}
