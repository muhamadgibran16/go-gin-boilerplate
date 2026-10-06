package health

import (
	"context"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gin-gonic/gin"
)

// Pinger checks connectivity to a dependency, e.g. *sql.DB
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Handler handles health check requests
type Handler struct {
	serviceName string
	db          Pinger
}

// NewHandler creates a new health handler
func NewHandler(serviceName string, db Pinger) *Handler {
	return &Handler{serviceName: serviceName, db: db}
}

// Check handles GET /health (liveness)
// @Summary Liveness check
// @Description Returns 200 while the process is running. Does not check dependencies.
// @Tags health
// @Produce json
// @Success 200 {object} httpx.Response
// @Router /health [get]
func (h *Handler) Check(c *gin.Context) {
	httpx.Success(c, "Server is running", gin.H{
		"service": h.serviceName,
		"version": "1.0.0",
	})
}

// Ready handles GET /health/ready (readiness)
// @Summary Readiness check
// @Description Returns 200 when the server can reach its dependencies (database), 503 otherwise.
// @Tags health
// @Produce json
// @Success 200 {object} httpx.Response
// @Failure 503 {object} httpx.ErrorResponse
// @Router /health/ready [get]
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		httpx.ServiceUnavailable(c, "Database is unreachable")
		return
	}

	httpx.Success(c, "Server is ready", gin.H{
		"service":  h.serviceName,
		"database": "up",
	})
}
