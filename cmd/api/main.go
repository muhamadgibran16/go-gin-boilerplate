package main

import (
	"log"

	"github.com/gibran/go-gin-boilerplate/config"
	"github.com/gibran/go-gin-boilerplate/database"
	"github.com/gibran/go-gin-boilerplate/internal/server"
	"go.uber.org/zap"
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
	var logger *zap.Logger
	var err error

	if cfg.AppEnv == "production" {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	// Connect to database
	db := database.Connect(cfg)

	// Create and run server
	srv := server.New(cfg, logger, db)
	srv.Run()
}
