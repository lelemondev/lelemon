package config

import (
	"log"
	"os"
	"strconv"
	"strings"
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
	DatabaseURL          string
	AnalyticsDatabaseURL string // Optional: separate store for traces/spans/analytics

	// JWT
	JWTSecret     string
	JWTExpiration time.Duration

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// Security
	AllowedOrigins []string // CORS allowed origins (empty = allow FrontendURL only)
	Environment    string   // development, staging, production
}

// Load loads configuration from environment variables
func Load() *Config {
	baseURL := getEnv("BASE_URL", "http://localhost:8080")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")
	env := getEnv("ENVIRONMENT", "development")

	// Parse allowed origins - defaults to frontend URL if not specified
	allowedOrigins := getEnvList("ALLOWED_ORIGINS", ",")
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{frontendURL}
	}

	// Validate JWT_SECRET in production
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production-please")
	if env == "production" {
		if jwtSecret == "change-me-in-production-please" || len(jwtSecret) < 32 {
			log.Fatal("FATAL: JWT_SECRET must be set to a secure value (min 32 chars) in production")
		}
	}

	return &Config{
		Port:                 getEnvInt("PORT", 8080),
		FrontendURL:          frontendURL,
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		LogFormat:            getEnv("LOG_FORMAT", "json"),
		DatabaseURL:          getEnv("DATABASE_URL", "sqlite://./data/lelemon.db"),
		AnalyticsDatabaseURL: getEnv("ANALYTICS_DATABASE_URL", ""),
		JWTSecret:            jwtSecret,
		JWTExpiration:        getEnvDuration("JWT_EXPIRATION", 24*7*time.Hour), // 7 days
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getEnv("GOOGLE_REDIRECT_URL", baseURL+"/api/v1/auth/google/callback"),
		AllowedOrigins:       allowedOrigins,
		Environment:          env,
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

func getEnvList(key, sep string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
