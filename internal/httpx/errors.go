package httpx

import (
	"context"
	"errors"

	"github.com/gibran/go-gin-boilerplate/internal/pkg/apperror"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/ctxlog"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Abort records err and stops the handler chain. ErrorHandler turns it into the response:
// *apperror.Error uses its kind and message; any other error becomes a logged 500.
//
//	user, err := h.service.Get(ctx, id)
//	if err != nil {
//		httpx.Abort(c, err)
//		return
//	}
func Abort(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

var errRequestTimeout = apperror.Unavailable("Service temporarily unavailable, please try again")

// ErrorHandler writes the JSON error response for errors recorded with Abort.
// Internal errors are logged with the request logger and hidden from the client.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		appErr := apperror.From(err)
		if errors.Is(err, context.DeadlineExceeded) {
			// The request ran out of time (see Timeout), typically a slow or unreachable database
			ctxlog.From(c.Request.Context()).Error("request timed out", zap.Error(err))
			appErr = errRequestTimeout
		}

		message := appErr.Message
		if appErr.Kind == apperror.KindInternal {
			ctxlog.From(c.Request.Context()).Error("request failed", zap.Error(err))
			message = "Internal server error"
		}

		res := ErrorResponse{Status: "error", Message: message}
		if len(appErr.Fields) > 0 {
			res.Errors = appErr.Fields
		}
		c.JSON(appErr.Kind.HTTPStatus(), res)
	}
}
