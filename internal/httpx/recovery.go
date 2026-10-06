package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns a Gin middleware that recovers from panics and logs the error
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("request_id", c.GetString(ContextKeyRequestID)),
					zap.String("path", c.Request.URL.Path),
					zap.Stack("stack"),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"status":  "error",
					"message": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
