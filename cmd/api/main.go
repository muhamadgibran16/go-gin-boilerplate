package main

import (
	"context"
	"log"
	"os"

	"github.com/gibran/go-gin-boilerplate/internal/app"
	"github.com/gibran/go-gin-boilerplate/internal/config"
	"github.com/gibran/go-gin-boilerplate/internal/database"
	"github.com/gibran/go-gin-boilerplate/internal/pkg/termcolor"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

// @title           Go Gin Boilerplate API
// @version         1.0
// @description     Boilerplate API with Clean Architecture, Auth, and Security.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and your JWT token.

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger, err := newLogger(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	// Connect to database
	db := database.Connect(cfg)

	// Migrations are not applied automatically, only reported
	warnPendingMigrations(db, logger)

	// Create and run server
	srv := app.New(cfg, logger, db)
	srv.Run()
}

// warnPendingMigrations logs a warning when the database schema is behind the embedded migrations
func warnPendingMigrations(db *gorm.DB, logger *zap.Logger) {
	migrator, err := database.NewMigrator(db)
	if err != nil {
		logger.Warn("Could not check database migrations", zap.Error(err))
		return
	}

	current, target, err := migrator.GetVersions(context.Background())
	if err != nil {
		logger.Warn("Could not check database migrations", zap.Error(err))
		return
	}

	if current < target {
		logger.Warn("Database has pending migrations, run `make db:migrate`",
			zap.Int64("current_version", current),
			zap.Int64("latest_version", target),
		)
	}
}

// newLogger returns a JSON logger for log collectors, or a readable console logger
// (colored levels when writing to a terminal) when LOG_FORMAT=pretty
func newLogger(cfg *config.Config) (*zap.Logger, error) {
	if !cfg.PrettyLogs() {
		return zap.NewProduction()
	}

	zc := zap.NewDevelopmentConfig()
	zc.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	if termcolor.Enabled(os.Stderr) {
		zc.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	// Stack traces only for errors; the development default (warnings too) is noisy
	zc.DisableStacktrace = true
	return zc.Build(zap.AddStacktrace(zapcore.ErrorLevel))
}
