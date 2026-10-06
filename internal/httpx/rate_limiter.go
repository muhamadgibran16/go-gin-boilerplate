package httpx

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// RateLimiter allows at most limit requests per period for each client IP.
// The client IP comes from c.ClientIP(), so the engine's trusted proxies must be configured correctly.
func RateLimiter(limit int64, period time.Duration) gin.HandlerFunc {
	return newRateLimiter(limit, period, func(c *gin.Context) string {
		return "ip:" + c.ClientIP()
	})
}

// RateLimiterPerUser allows at most limit requests per period for each authenticated user,
// so users sharing an IP address (office, campus, mobile carrier) do not share a quota.
// It must run after the auth middleware; unauthenticated requests fall back to the client IP.
//
// The user ID comes from the signed JWT, so clients cannot change it to get a new quota
// (unlike a device ID or User-Agent header, which clients control).
func RateLimiterPerUser(limit int64, period time.Duration) gin.HandlerFunc {
	return newRateLimiter(limit, period, func(c *gin.Context) string {
		if userID, ok := CurrentUserID(c); ok {
			return "user:" + userID.String()
		}
		return "ip:" + c.ClientIP()
	})
}

func newRateLimiter(limit int64, period time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
	// Each limiter gets its own in-memory store so counters are not shared between limiters.
	// Use a shared store (e.g. Redis) when running multiple instances.
	instance := limiter.New(memory.NewStore(), limiter.Rate{Period: period, Limit: limit})

	return mgin.NewMiddleware(instance,
		mgin.WithKeyGetter(key),
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			TooManyRequests(c, "Too many requests, please try again later")
		}),
	)
}
