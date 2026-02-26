package main

import (
	"log"

	"github.com/gibran/go-gin-boilerplate/config"
	"github.com/gibran/go-gin-boilerplate/database"
	"github.com/gibran/go-gin-boilerplate/internal/server"
	"go.uber.org/zap"
)

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
