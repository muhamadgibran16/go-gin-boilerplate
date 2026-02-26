package middleware

import (
	"strings"

	"github.com/gibran/go-gin-boilerplate/pkg/response"
	"github.com/gibran/go-gin-boilerplate/pkg/security"
	"github.com/gin-gonic/gin"
)

// Auth returns a Gin middleware that validates JWT tokens
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "Authorization header must be Bearer token")
			c.Abort()
			return
		}

		claims, err := security.ValidateToken(parts[1], secret)
		if err != nil {
			response.Unauthorized(c, "Invalid or expired token")
			c.Abort()
			return
		}

		// Inject user info into context
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
