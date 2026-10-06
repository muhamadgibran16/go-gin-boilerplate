// Package app is the composition root: it wires config, database, modules and
// HTTP middleware together and runs the server. Feature code lives in internal/modules.
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gibran/go-gin-boilerplate/internal/config"
	"github.com/gibran/go-gin-boilerplate/internal/httpx"
	"github.com/gibran/go-gin-boilerplate/internal/modules/auth"
	"github.com/gibran/go-gin-boilerplate/internal/modules/health"
	"github.com/gibran/go-gin-boilerplate/internal/modules/user"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// maxBodyBytes caps request bodies; larger requests get 413
	maxBodyBytes = 1 << 20 // 1 MiB

	// requestTimeout cancels slow requests (e.g. an unresponsive database) with a 503.
	// It must stay below the server WriteTimeout (10s) so the error can still be written.
	requestTimeout = 8 * time.Second

	// Rate limits, per minute. Layered so that no single key can be abused:
	// IP for flood protection, user ID for authenticated traffic, email for login brute force.
	globalLimitPerIP   = 300 // every request; high enough for many users behind one IP
	authLimitPerIP     = 20  // register and login
	apiLimitPerUser    = 100 // authenticated endpoints, keyed by the user ID from the JWT
	maxFailedLogins    = 5   // failed logins per email before it is locked...
	loginLockoutWindow = 15 * time.Minute
)

// Server holds the HTTP server and its dependencies
type Server struct {
	config *config.Config
	logger *zap.Logger
	db     *gorm.DB
	engine *gin.Engine
}

// handlers holds the HTTP handlers of every module
type handlers struct {
	health *health.Handler
	auth   *auth.Handler
	user   *user.Handler
}

// New creates a new Server instance
func New(cfg *config.Config, logger *zap.Logger, db *gorm.DB) *Server {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	// Answer a wrong method on a known route with 405 instead of 404
	engine.HandleMethodNotAllowed = true

	// Only trust X-Forwarded-For from configured proxies, otherwise clients could
	// spoof their IP and bypass the rate limiter.
	if err := engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		logger.Fatal("Invalid TRUSTED_PROXIES", zap.Error(err))
	}

	// Global middleware. Logger and CORS run before the rate limiter so that
	// rejected (429) requests are still logged and readable by browsers.
	engine.Use(httpx.Recovery(logger))
	engine.Use(httpx.RequestID())
	engine.Use(httpx.Logger(logger))
	engine.Use(httpx.ErrorHandler())
	engine.Use(httpx.Timeout(requestTimeout))
	engine.Use(httpx.Security())
	if origins := corsOrigins(cfg); len(origins) > 0 {
		engine.Use(httpx.CORS(origins))
	}
	engine.Use(httpx.RateLimiter(globalLimitPerIP, time.Minute))
	engine.Use(httpx.BodyLimit(maxBodyBytes))

	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get database handle", zap.Error(err))
	}

	// Wire modules
	userRepo := user.NewRepository(db)
	loginFailures := ratelimit.NewFailureCounter(maxFailedLogins, loginLockoutWindow)
	authSvc := auth.NewService(userRepo, loginFailures, auth.Config{
		JWTSecret:       cfg.JWTSecret,
		AccessTokenTTL:  time.Duration(cfg.JWTAccessExpireHours) * time.Hour,
		RefreshTokenTTL: time.Duration(cfg.JWTRefreshExpireDays) * 24 * time.Hour,
	})
	userSvc := user.NewService(userRepo)

	registerRoutes(engine, cfg, handlers{
		health: health.NewHandler(cfg.AppName, sqlDB),
		auth:   auth.NewHandler(authSvc),
		user:   user.NewHandler(userSvc),
	})

	return &Server{
		config: cfg,
		logger: logger,
		db:     db,
		engine: engine,
	}
}

// corsOrigins returns the allowed CORS origins. An empty result disables CORS,
// which means only same-origin browser requests are allowed.
func corsOrigins(cfg *config.Config) []string {
	if len(cfg.CORSAllowedOrigins) > 0 {
		return cfg.CORSAllowedOrigins
	}
	if cfg.IsProduction() {
		return nil
	}
	return []string{"*"}
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
		s.logger.Error("Server forced to shutdown", zap.Error(err))
	}

	if sqlDB, err := s.db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			s.logger.Error("Failed to close database connection", zap.Error(err))
		}
	}

	s.logger.Info("Server exited")
}
