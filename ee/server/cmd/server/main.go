package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Core imports
	"github.com/lelemon/server/pkg/application/analytics"
	appauth "github.com/lelemon/server/pkg/application/auth"
	"github.com/lelemon/server/pkg/application/ingest"
	"github.com/lelemon/server/pkg/application/project"
	"github.com/lelemon/server/pkg/application/trace"
	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/domain/service"
	"github.com/lelemon/server/pkg/infrastructure/auth"
	"github.com/lelemon/server/pkg/infrastructure/config"
	"github.com/lelemon/server/pkg/infrastructure/logger"
	"github.com/lelemon/server/pkg/infrastructure/store"
	coreHttp "github.com/lelemon/server/pkg/interfaces/http"

	// Enterprise imports
	"github.com/lelemon/ee/server/application/billing"
	"github.com/lelemon/ee/server/application/organization"
	"github.com/lelemon/ee/server/application/rbac"
	"github.com/lelemon/ee/server/infrastructure/lemonsqueezy"
	entStore "github.com/lelemon/ee/server/infrastructure/store"
	entHttp "github.com/lelemon/ee/server/interfaces/http"
)

// userStoreAdapter adapts repository.Store to organization.UserStore interface
type userStoreAdapter struct {
	store repository.Store
}

func (a *userStoreAdapter) GetUserByID(ctx context.Context, id string) (organization.UserInfo, error) {
	return a.store.GetUserByID(ctx, id)
}

func (a *userStoreAdapter) GetUserByEmail(ctx context.Context, email string) (organization.UserInfo, error) {
	return a.store.GetUserByEmail(ctx, email)
}

func main() {
	// Load configuration
	cfg := config.Load()
	entCfg := loadEnterpriseConfig()

	// Setup structured logging
	logCfg := logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
		Output: os.Stdout,
	}
	logger.Setup(logCfg)

	log := slog.Default()
	log.Info("starting lelemon enterprise server",
		"version", "1.0.0-enterprise",
		"port", cfg.Port,
		"log_level", cfg.LogLevel,
		"allowed_origins", cfg.AllowedOrigins,
	)

	// ============================================
	// CORE: Initialize stores and services
	// ============================================

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

	// Run core migrations
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
	log.Info("core database migrations completed")

	// Initialize core auth services
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiration)
	oauthService := auth.NewOAuthService(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	// Initialize core application services
	pricing := service.NewPricingCalculator()
	ingestSvc := ingest.NewAsyncService(analyticsStore, pricing, 1000, 4)
	traceSvc := trace.NewService(analyticsStore, pricing)
	analyticsSvc := analytics.NewService(analyticsStore)
	projectSvc := project.NewService(primaryStore)
	authSvc := appauth.NewService(primaryStore, jwtService, oauthService)

	// ============================================
	// ENTERPRISE: Initialize stores and services
	// ============================================

	// Open raw SQL connection for enterprise store
	db, err := openDB(cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to open enterprise DB connection", "error", err)
		os.Exit(1)
	}

	// Create enterprise store
	enterpriseStore := entStore.New(primaryStore, db)

	// Run enterprise migrations
	if err := enterpriseStore.MigrateEnterprise(ctx); err != nil {
		log.Error("failed to run enterprise migrations", "error", err)
		os.Exit(1)
	}
	log.Info("enterprise database migrations completed")

	// Initialize Lemon Squeezy client
	lsClient := lemonsqueezy.NewClient(
		entCfg.LemonSqueezyAPIKey,
		entCfg.LemonSqueezyWebhookSecret,
		entCfg.LemonSqueezyStoreID,
	)

	// Initialize enterprise services
	userStore := &userStoreAdapter{store: primaryStore}
	orgSvc := organization.NewService(enterpriseStore, userStore)
	rbacSvc := rbac.NewService(enterpriseStore)
	billingConfig := &billing.Config{
		ProVariantID:        entCfg.ProVariantID,
		EnterpriseVariantID: entCfg.EnterpriseVariantID,
	}
	billingSvc := billing.NewService(enterpriseStore, lsClient, billingConfig)

	// ============================================
	// ROUTER: Create core router with enterprise extension
	// ============================================

	// Create enterprise extension
	enterpriseExtension := entHttp.NewEnterpriseExtension(
		orgSvc,
		rbacSvc,
		billingSvc,
		lsClient,
	)

	// Create router with enterprise features enabled
	router := coreHttp.NewRouter(coreHttp.RouterConfig{
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
		// Enterprise features
		Extensions:     []coreHttp.RouterExtension{enterpriseExtension},
		FeaturesConfig: coreHttp.EnterpriseFeaturesConfig(),
	})

	// Create server
	server := coreHttp.NewServer(router, cfg.Port)

	// Start server in goroutine
	go func() {
		log.Info("enterprise server listening", "port", cfg.Port)
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
	if err := db.Close(); err != nil {
		log.Error("enterprise db close error", "error", err)
	}

	log.Info("enterprise server stopped gracefully")
}

// EnterpriseConfig holds enterprise-specific configuration
type EnterpriseConfig struct {
	LemonSqueezyAPIKey        string
	LemonSqueezyWebhookSecret string
	LemonSqueezyStoreID       string
	ProVariantID              string
	EnterpriseVariantID       string
}

// loadEnterpriseConfig loads enterprise configuration from environment
func loadEnterpriseConfig() *EnterpriseConfig {
	return &EnterpriseConfig{
		LemonSqueezyAPIKey:        getEnv("LEMONSQUEEZY_API_KEY", ""),
		LemonSqueezyWebhookSecret: getEnv("LEMONSQUEEZY_WEBHOOK_SECRET", ""),
		LemonSqueezyStoreID:       getEnv("LEMONSQUEEZY_STORE_ID", ""),
		ProVariantID:              getEnv("LEMONSQUEEZY_PRO_VARIANT_ID", ""),
		EnterpriseVariantID:       getEnv("LEMONSQUEEZY_ENTERPRISE_VARIANT_ID", ""),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// openDB opens a database connection based on the URL
func openDB(databaseURL string) (*sql.DB, error) {
	// For SQLite
	if len(databaseURL) > 9 && databaseURL[:9] == "sqlite://" {
		path := databaseURL[9:]
		return sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	}

	// For PostgreSQL
	if len(databaseURL) > 11 && (databaseURL[:11] == "postgres://" || databaseURL[:13] == "postgresql://") {
		return sql.Open("pgx", databaseURL)
	}

	// Default to SQLite
	return sql.Open("sqlite", databaseURL+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
}
