package health

import (
	"github.com/gibran/go-gin-boilerplate/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler handles health check requests
type Handler struct{}

// NewHandler creates a new health handler
func NewHandler() *Handler {
	return &Handler{}
}

// Check handles GET /health
// @Summary Health check
// @Description Returns the health status of the server
// @Tags health
// @Produce json
// @Success 200 {object} response.Response
// @Router /health [get]
func (h *Handler) Check(c *gin.Context) {
	response.Success(c, "Server is running", gin.H{
		"service": "go-gin-boilerplate",
		"version": "1.0.0",
	})
}
