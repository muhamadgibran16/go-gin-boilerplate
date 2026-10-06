package httpx

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextKeyRequestID is the context key holding the current request ID
const ContextKeyRequestID = "requestID"

// validRequestID limits client-provided IDs so they cannot inject arbitrary content into logs
var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// RequestID returns a middleware that adds a unique X-Request-ID to each request and response
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !validRequestID.MatchString(requestID) {
			requestID = uuid.New().String()
		}

		// Inject to context
		c.Set(ContextKeyRequestID, requestID)

		// Set response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}
