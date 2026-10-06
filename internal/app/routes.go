package app

import (
	"net/http"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/config"
	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gibran/go-gin-boilerplate/internal/modules/auth"
	"github.com/gibran/go-gin-boilerplate/internal/modules/user"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/gibran/go-gin-boilerplate/docs"
)

// registerRoutes registers all routes to the Gin engine
func registerRoutes(r *gin.Engine, cfg *config.Config, h handlers) {
	requireAuth := auth.Middleware(cfg.JWTSecret)
	// One shared limiter, so a user's quota covers all authenticated endpoints together
	perUserLimit := httpx.RateLimiterPerUser(apiLimitPerUser, time.Minute)

	// Default route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Hello World",
		})
	})

	// Health check routes (no version prefix): liveness and readiness
	r.GET("/health", h.health.Check)
	r.GET("/health/ready", h.health.Ready)

	// Swagger route (disabled by default in production)
	if cfg.SwaggerEnabled {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Health
		v1.GET("/health", h.health.Check)

		// Auth. Register and login get a stricter per-IP limit (login is also limited per email
		// in auth.Service). Refresh and logout are called often by clients, so they are not.
		authGroup := v1.Group("/auth")
		{
			authLimit := httpx.RateLimiter(authLimitPerIP, time.Minute)
			authGroup.POST("/register", authLimit, h.auth.Register)
			authGroup.POST("/login", authLimit, h.auth.Login)
			authGroup.POST("/refresh", h.auth.Refresh)
			authGroup.POST("/logout", requireAuth, perUserLimit, h.auth.Logout)
		}

		// Users (admin only)
		users := v1.Group("/users")
		users.Use(requireAuth, perUserLimit, auth.RequireRoles(user.RoleAdmin))
		{
			users.GET("", httpx.ValidateQueryParams([]string{"page", "perPage", "sort", "search"}), h.user.GetMany)
			users.GET("/:id", h.user.GetOne)
			users.PUT("/:id", h.user.Update)
			users.DELETE("/:id", h.user.Delete)
		}
	}
}
