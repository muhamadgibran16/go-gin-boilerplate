package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimiter returns a middleware that limits the request rate per IP address
func RateLimiter() gin.HandlerFunc {
	// Define the rate: 100 requests per minute
	rate := limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}

	// Use an in-memory store for the limiter
	store := memory.NewStore()

	// Create a new limiter
	instance := limiter.New(store, rate)

	// Return the Gin middleware
	return mgin.NewMiddleware(instance, mgin.WithLimitReachedHandler(func(c *gin.Context) {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"status":  "error",
			"message": "Too many requests, please try again later",
		})
	}))
}
