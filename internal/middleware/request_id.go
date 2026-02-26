package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID returns a middleware that adds a unique X-Request-ID to each request and response
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Inject to context
		c.Set("requestID", requestID)

		// Set response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
