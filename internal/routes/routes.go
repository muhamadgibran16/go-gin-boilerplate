package routes

import (
	"net/http"

	authHandler "github.com/gibran/go-gin-boilerplate/internal/handler/auth"
	healthHandler "github.com/gibran/go-gin-boilerplate/internal/handler/health"
	userHandler "github.com/gibran/go-gin-boilerplate/internal/handler/user"
	"github.com/gibran/go-gin-boilerplate/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Handlers holds all the route handlers
type Handlers struct {
	Health *healthHandler.Handler
	Auth   *authHandler.Handler
	User   *userHandler.Handler
}

// Setup registers all routes to the Gin engine
func Setup(r *gin.Engine, handlers *Handlers, jwtSecret string) {
	// Default route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Hello World",
		})
	})

	// Health check route (no version prefix)
	r.GET("/health", handlers.Health.Check)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Health
		v1.GET("/health", handlers.Health.Check)

		// Auth (Public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handlers.Auth.Register)
			auth.POST("/login", handlers.Auth.Login)
			auth.POST("/refresh", handlers.Auth.Refresh)
		}

		// Users (Protected)
		users := v1.Group("/users")
		users.Use(middleware.Auth(jwtSecret))
		{
			users.GET("", middleware.ValidateQueryParams([]string{"page", "perPage"}), handlers.User.GetMany)
			users.GET("/:id", handlers.User.GetOne)
			users.PUT("/:id", handlers.User.Update)
			users.DELETE("/:id", handlers.User.Delete)
			users.POST("/logout", handlers.Auth.Logout)
		}
	}
}
