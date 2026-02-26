package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// App
	AppName string
	AppEnv  string
	AppPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT
	JWTSecret            string
	JWTAccessExpireHours int
	JWTRefreshExpireDays int
}

// Load reads configuration from environment variables
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		AppName: getEnv("APP_NAME", "go-gin-boilerplate"),
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "go_gin_boilerplate"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret:            getEnv("JWT_SECRET", "your-super-secret-key"),
		JWTAccessExpireHours: getEnvInt("JWT_ACCESS_EXPIRE_HOURS", 1),
		JWTRefreshExpireDays: getEnvInt("JWT_REFRESH_EXPIRE_DAYS", 7),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var res int
		fmt.Sscanf(value, "%d", &res)
		return res
	}
	return defaultValue
}
