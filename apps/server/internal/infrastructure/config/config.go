package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the server
type Config struct {
	// Server
	Port        int
	FrontendURL string

	// Logging
	LogLevel  string // debug, info, warn, error
	LogFormat string // json, text

	// Database
	DatabaseURL string

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// Load loads configuration from environment variables
func Load() *Config {
	baseURL := getEnv("BASE_URL", "http://localhost:8080")

	return &Config{
		Port:               getEnvInt("PORT", 8080),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		LogFormat:          getEnv("LOG_FORMAT", "json"),
		DatabaseURL:        getEnv("DATABASE_URL", "sqlite://./data/lelemon.db"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production-please"),
		JWTExpiration:      getEnvDuration("JWT_EXPIRATION", 24*7*time.Hour), // 7 days
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", baseURL+"/api/v1/auth/google/callback"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
