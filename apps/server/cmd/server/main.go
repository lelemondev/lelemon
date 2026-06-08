package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lelemon/server/pkg/application/analytics"
	appauth "github.com/lelemon/server/pkg/application/auth"
	"github.com/lelemon/server/pkg/application/ingest"
	"github.com/lelemon/server/pkg/application/project"
	"github.com/lelemon/server/pkg/application/trace"
	"github.com/lelemon/server/pkg/domain/service"
	"github.com/lelemon/server/pkg/infrastructure/auth"
	"github.com/lelemon/server/pkg/infrastructure/config"
	"github.com/lelemon/server/pkg/infrastructure/logger"
	"github.com/lelemon/server/pkg/infrastructure/store"
	apphttp "github.com/lelemon/server/pkg/interfaces/http"
)

// envOr returns the value of environment variable key, or fallback if unset/empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

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
		"allowed_origins", cfg.AllowedOrigins,
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

	// Auto-sync model pricing from external sources in the background.
	// Non-blocking and offline-safe: the built-in table stays in effect
	// until/unless a refresh succeeds. Precedence: LiteLLM > OpenRouter > local.
	//   PRICING_AUTOSYNC=false       disable all external sources
	//   PRICING_SOURCE_URL=...       override the LiteLLM (primary) URL
	//   PRICING_OPENROUTER=false     disable the OpenRouter (secondary) source
	//   PRICING_OPENROUTER_URL=...   override the OpenRouter URL
	if os.Getenv("PRICING_AUTOSYNC") != "false" {
		sources := service.PricingSources{
			LiteLLMURL: envOr("PRICING_SOURCE_URL", service.DefaultLiteLLMPricingURL),
			Interval:   service.DefaultPricingRefreshInterval,
		}
		if os.Getenv("PRICING_OPENROUTER") != "false" {
			sources.OpenRouterURL = envOr("PRICING_OPENROUTER_URL", service.DefaultOpenRouterPricingURL)
		}
		service.StartPricingRefresh(ctx, sources)
		log.Info("pricing auto-sync enabled", "litellm", sources.LiteLLMURL, "openrouter", sources.OpenRouterURL)
	}

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
		AllowedOrigins: cfg.AllowedOrigins,
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
