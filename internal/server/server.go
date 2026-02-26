package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gibran/go-gin-boilerplate/config"
	authHandler "github.com/gibran/go-gin-boilerplate/internal/handler/auth"
	healthHandler "github.com/gibran/go-gin-boilerplate/internal/handler/health"
	userHandler "github.com/gibran/go-gin-boilerplate/internal/handler/user"
	"github.com/gibran/go-gin-boilerplate/internal/middleware"
	repository "github.com/gibran/go-gin-boilerplate/internal/repository/user"
	"github.com/gibran/go-gin-boilerplate/internal/routes"
	authService "github.com/gibran/go-gin-boilerplate/internal/service/auth"
	userService "github.com/gibran/go-gin-boilerplate/internal/service/user"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Server holds the HTTP server and its dependencies
type Server struct {
	config *config.Config
	logger *zap.Logger
	db     *gorm.DB
	engine *gin.Engine
}

// New creates a new Server instance
func New(cfg *config.Config, logger *zap.Logger, db *gorm.DB) *Server {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Global middleware
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.CORS())
	engine.Use(middleware.Security())

	// Initialize Layers
	userRepo := repository.NewUserRepository(db)
	
	authSvc := authService.NewAuthService(userRepo, cfg)
	userSvc := userService.NewUserService(userRepo)

	handlers := &routes.Handlers{
		Health: healthHandler.NewHandler(),
		Auth:   authHandler.NewHandler(authSvc),
		User:   userHandler.NewHandler(userSvc),
	}

	// Setup Routes
	routes.Setup(engine, handlers, cfg.JWTSecret)

	return &Server{
		config: cfg,
		logger: logger,
		db:     db,
		engine: engine,
	}
}

// Run starts the HTTP server with graceful shutdown
func (s *Server) Run() {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.AppPort),
		Handler:      s.engine,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		s.logger.Info("Starting server",
			zap.String("name", s.config.AppName),
			zap.String("env", s.config.AppEnv),
			zap.String("port", s.config.AppPort),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	s.logger.Info("Server exited")
}
