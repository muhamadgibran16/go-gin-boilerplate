package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultJWTSecret   = "your-super-secret-key"
	minJWTSecretLength = 32
)

// Config holds the application configuration
type Config struct {
	// App
	AppName string
	AppEnv  string
	AppPort string

	// TrustedProxies is the list of proxy IPs/CIDRs allowed to set X-Forwarded-For.
	// Empty means no proxy is trusted and the client IP is taken from the TCP connection.
	TrustedProxies []string

	// CORSAllowedOrigins lists origins allowed to call the API from a browser.
	// Empty means "*" in development and no cross-origin access in production.
	CORSAllowedOrigins []string

	// SwaggerEnabled exposes /swagger. Defaults to true outside production.
	SwaggerEnabled bool

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

// IsProduction reports whether the app runs in production mode
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load reads configuration from environment variables
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	appEnv := getEnv("APP_ENV", "development")

	cfg := &Config{
		AppName: getEnv("APP_NAME", "go-gin-boilerplate"),
		AppEnv:  appEnv,
		AppPort: getEnv("APP_PORT", "8080"),

		TrustedProxies:     getEnvList("TRUSTED_PROXIES"),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS"),
		SwaggerEnabled:     getEnvBool("SWAGGER_ENABLED", appEnv != "production"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "go_gin_boilerplate"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		JWTSecret:            getEnv("JWT_SECRET", defaultJWTSecret),
		JWTAccessExpireHours: getEnvInt("JWT_ACCESS_EXPIRE_HOURS", 1),
		JWTRefreshExpireDays: getEnvInt("JWT_REFRESH_EXPIRE_DAYS", 7),
	}

	cfg.validate()
	return cfg
}

// validate stops the app on insecure configuration in production and warns otherwise
func (c *Config) validate() {
	weakSecret := c.JWTSecret == "" ||
		strings.HasPrefix(c.JWTSecret, defaultJWTSecret) ||
		len(c.JWTSecret) < minJWTSecretLength

	if weakSecret {
		if c.IsProduction() {
			log.Fatalf("JWT_SECRET must be set to a random value of at least %d characters in production", minJWTSecretLength)
		}
		log.Println("WARNING: JWT_SECRET is weak or default, do not use this configuration in production")
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	res, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || res <= 0 {
		log.Fatalf("%s must be a positive integer, got %q", key, value)
	}
	return res
}

func getEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	res, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		log.Fatalf("%s must be a boolean, got %q", key, value)
	}
	return res
}

func getEnvList(key string) []string {
	var res []string
	for _, item := range strings.Split(os.Getenv(key), ",") {
		if item = strings.TrimSpace(item); item != "" {
			res = append(res, item)
		}
	}
	return res
}
