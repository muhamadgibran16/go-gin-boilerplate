package httpx

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout gives every request a deadline. Database queries and other calls that use the
// request context are cancelled when it expires, and ErrorHandler answers 503 instead of
// leaving the client hanging (e.g. when the database stops responding).
// Keep it below the server's WriteTimeout, so the JSON error can still be written.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
