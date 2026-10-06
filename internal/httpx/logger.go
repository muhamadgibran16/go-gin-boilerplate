package httpx

import (
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/ctxlog"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger stores a request-scoped logger (with request_id) in the request context,
// available through ctxlog.From(ctx), and writes an access log line for each request.
// Must run after RequestID.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		reqLogger := logger.With(zap.String("request_id", c.GetString(ContextKeyRequestID)))
		c.Request = c.Request.WithContext(ctxlog.With(c.Request.Context(), reqLogger))

		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if query != "" {
			path = path + "?" + query
		}

		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user-agent", c.Request.UserAgent()),
		}
		// The request logger already carries request_id, and user_id once authenticated
		accessLog := ctxlog.From(c.Request.Context())

		switch {
		case statusCode >= 500:
			accessLog.Error("request", fields...)
		case statusCode >= 400:
			accessLog.Warn("request", fields...)
		default:
			accessLog.Info("request", fields...)
		}
	}
}
