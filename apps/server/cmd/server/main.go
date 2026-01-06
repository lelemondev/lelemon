package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lelemon/server/internal/application/analytics"
	appauth "github.com/lelemon/server/internal/application/auth"
	"github.com/lelemon/server/internal/application/ingest"
	"github.com/lelemon/server/internal/application/project"
	"github.com/lelemon/server/internal/application/trace"
	"github.com/lelemon/server/internal/domain/service"
	"github.com/lelemon/server/internal/infrastructure/auth"
	"github.com/lelemon/server/internal/infrastructure/config"
	"github.com/lelemon/server/internal/infrastructure/logger"
	"github.com/lelemon/server/internal/infrastructure/store"
	apphttp "github.com/lelemon/server/internal/interfaces/http"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup structured logging
	logCfg := logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
		Output: os.Stdout,
	}
	logger.Setup(logCfg)

	log := slog.Default()
	log.Info("starting lelemon server",
		"version", "1.0.0",
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
	)

	// Initialize store
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to initialize store", "error", err)
		os.Exit(1)
	}

	// Run migrations
	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	log.Info("database migrations completed")

	// Initialize auth services
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiration)
	oauthService := auth.NewOAuthService(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	// Initialize application services
	pricing := service.NewPricingCalculator()
	ingestSvc := ingest.NewAsyncService(db, pricing, 1000, 4)
	traceSvc := trace.NewService(db, pricing)
	analyticsSvc := analytics.NewService(db)
	projectSvc := project.NewService(db)
	authSvc := appauth.NewService(db, jwtService, oauthService)

	// Create router
	router := apphttp.NewRouter(apphttp.RouterConfig{
		Store:        db,
		IngestSvc:    ingestSvc,
		TraceSvc:     traceSvc,
		AnalyticsSvc: analyticsSvc,
		ProjectSvc:   projectSvc,
		AuthSvc:      authSvc,
		JWTService:   jwtService,
		FrontendURL:  cfg.FrontendURL,
	})

	// Create server
	server := apphttp.NewServer(router, cfg.Port)

	// Start server in goroutine
	go func() {
		log.Info("server listening", "port", cfg.Port)
		if err := server.Start(); err != nil {
			log.Error("server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("shutdown signal received", "signal", sig.String())

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", "error", err)
	}

	// Stop ingest worker (drain pending jobs)
	ingestSvc.Stop(10 * time.Second)

	// Close database connection
	if err := db.Close(); err != nil {
		log.Error("database close error", "error", err)
	}

	log.Info("server stopped gracefully")
}
