package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NotFoundHandler answers unknown routes with the standard JSON error instead of
// Gin's plain-text "404 page not found". Register it with engine.NoRoute.
func NotFoundHandler(c *gin.Context) {
	NotFound(c, "Route not found")
}

// MethodNotAllowedHandler answers known routes called with a wrong method (405).
// Register it with engine.NoMethod and set engine.HandleMethodNotAllowed = true.
func MethodNotAllowedHandler(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, ErrorResponse{
		Status:  "error",
		Message: "Method not allowed",
	})
}
