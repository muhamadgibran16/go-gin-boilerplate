package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit caps the request body at maxBytes, so a client cannot exhaust memory by
// sending a huge payload. Reading past the limit fails, and BindJSON answers 413.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			Abort(c, errPayloadTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
