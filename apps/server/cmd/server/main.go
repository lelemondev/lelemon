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

	// Initialize primary store (users, projects)
	primaryStore, err := store.New(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to initialize primary store", "error", err)
		os.Exit(1)
	}

	// Initialize analytics store (traces, spans) - defaults to primary
	analyticsStore := primaryStore
	if cfg.AnalyticsDatabaseURL != "" {
		analyticsStore, err = store.New(cfg.AnalyticsDatabaseURL)
		if err != nil {
			log.Error("failed to initialize analytics store", "error", err)
			os.Exit(1)
		}
		log.Info("using separate analytics store")
	}

	// Run migrations
	ctx := context.Background()
	if err := primaryStore.Migrate(ctx); err != nil {
		log.Error("failed to run primary migrations", "error", err)
		os.Exit(1)
	}
	if analyticsStore != primaryStore {
		if err := analyticsStore.Migrate(ctx); err != nil {
			log.Error("failed to run analytics migrations", "error", err)
			os.Exit(1)
		}
	}
	log.Info("database migrations completed")

	// Initialize auth services
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiration)
	oauthService := auth.NewOAuthService(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	// Initialize application services
	// - Services that handle traces/spans/analytics use analyticsStore
	// - Services that handle users/projects use primaryStore
	pricing := service.NewPricingCalculator()
	ingestSvc := ingest.NewAsyncService(analyticsStore, pricing, 1000, 4)
	traceSvc := trace.NewService(analyticsStore, pricing)
	analyticsSvc := analytics.NewService(analyticsStore)
	projectSvc := project.NewService(primaryStore)
	authSvc := appauth.NewService(primaryStore, jwtService, oauthService)

	// Create router
	router := apphttp.NewRouter(apphttp.RouterConfig{
		PrimaryStore:   primaryStore,
		AnalyticsStore: analyticsStore,
		IngestSvc:      ingestSvc,
		TraceSvc:       traceSvc,
		AnalyticsSvc:   analyticsSvc,
		ProjectSvc:     projectSvc,
		AuthSvc:        authSvc,
		JWTService:     jwtService,
		FrontendURL:    cfg.FrontendURL,
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

	// Close database connections
	if err := primaryStore.Close(); err != nil {
		log.Error("primary store close error", "error", err)
	}
	if analyticsStore != primaryStore {
		if err := analyticsStore.Close(); err != nil {
			log.Error("analytics store close error", "error", err)
		}
	}

	log.Info("server stopped gracefully")
}
