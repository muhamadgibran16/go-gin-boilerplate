package httpx

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/ctxlog"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerOptions configures the access log
type LoggerOptions struct {
	// Pretty writes one human-readable line per request to Out instead of a JSON log entry.
	// Use it in development; keep JSON in production so log collectors can parse it.
	Pretty bool
	// Color adds ANSI colors to pretty lines. Enable it only when Out is a terminal.
	Color bool
	// Out receives pretty lines
	Out io.Writer
}

// Logger stores a request-scoped logger (with request_id) in the request context,
// available through ctxlog.From(ctx), and writes an access log line for each request.
// Must run after RequestID.
func Logger(logger *zap.Logger, opts LoggerOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		requestID := c.GetString(ContextKeyRequestID)
		reqLogger := logger.With(zap.String("request_id", requestID))
		c.Request = c.Request.WithContext(ctxlog.With(c.Request.Context(), reqLogger))

		path := c.Request.URL.Path
		if query := c.Request.URL.RawQuery; query != "" {
			path += "?" + query
		}

		c.Next()

		entry := accessEntry{
			Time:      start,
			Status:    c.Writer.Status(),
			Latency:   time.Since(start),
			ClientIP:  c.ClientIP(),
			Method:    c.Request.Method,
			Path:      path,
			RequestID: requestID,
		}
		if len(c.Errors) > 0 {
			entry.Error = c.Errors.Last().Error()
		}

		if opts.Pretty {
			_, _ = io.WriteString(opts.Out, entry.pretty(opts.Color))
			return
		}
		entry.log(ctxlog.From(c.Request.Context()), c.Request.UserAgent())
	}
}

type accessEntry struct {
	Time      time.Time
	Status    int
	Latency   time.Duration
	ClientIP  string
	Method    string
	Path      string
	RequestID string
	Error     string
}

// log writes a structured entry. The request logger already carries request_id,
// and user_id once authenticated.
func (e accessEntry) log(l *zap.Logger, userAgent string) {
	fields := []zap.Field{
		zap.Int("status", e.Status),
		zap.String("method", e.Method),
		zap.String("path", e.Path),
		zap.String("ip", e.ClientIP),
		zap.Duration("latency", e.Latency),
		zap.String("user-agent", userAgent),
	}

	switch {
	case e.Status >= 500:
		l.Error("request", fields...)
	case e.Status >= 400:
		l.Warn("request", fields...)
	default:
		l.Info("request", fields...)
	}
}

// ANSI escape codes
const (
	reset = "\033[0m"
	dim   = "\033[2m"
	red   = "\033[31m"
)

// pretty formats the entry as one line, e.g.
//
//	2026-10-06 15:40:01 | 200 |    1.234ms |       127.0.0.1 | GET     /api/v1/users?page=1  req=4f1c…
func (e accessEntry) pretty(color bool) string {
	statusColor, methodColor, resetColor, dimColor, errColor := "", "", "", "", ""
	if color {
		statusColor, methodColor = statusBadgeColor(e.Status), methodBadgeColor(e.Method)
		resetColor, dimColor, errColor = reset, dim, red
	}

	line := fmt.Sprintf("%s |%s %3d %s| %12s | %15s |%s %-7s %s %s  %sreq=%s%s",
		e.Time.Format("2006-01-02 15:04:05"),
		statusColor, e.Status, resetColor,
		e.Latency.Round(time.Microsecond),
		e.ClientIP,
		methodColor, e.Method, resetColor,
		e.Path,
		dimColor, e.RequestID, resetColor,
	)
	if e.Error != "" {
		line += fmt.Sprintf("  %serror=%s%s", errColor, e.Error, resetColor)
	}
	return line + "\n"
}

func statusBadgeColor(status int) string {
	switch {
	case status >= http.StatusInternalServerError:
		return "\033[97;41m" // white on red
	case status >= http.StatusBadRequest:
		return "\033[90;43m" // black on yellow
	case status >= http.StatusMultipleChoices:
		return "\033[90;47m" // black on white
	default:
		return "\033[97;42m" // white on green
	}
}

func methodBadgeColor(method string) string {
	switch method {
	case http.MethodGet:
		return "\033[97;44m" // white on blue
	case http.MethodPost:
		return "\033[97;46m" // white on cyan
	case http.MethodPut:
		return "\033[90;43m" // black on yellow
	case http.MethodPatch:
		return "\033[97;42m" // white on green
	case http.MethodDelete:
		return "\033[97;41m" // white on red
	case http.MethodHead:
		return "\033[97;45m" // white on magenta
	default:
		return "\033[90;47m" // black on white (OPTIONS, ...)
	}
}
