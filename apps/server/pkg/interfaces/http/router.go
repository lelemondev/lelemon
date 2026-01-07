package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/lelemon/server/pkg/application/analytics"
	appauth "github.com/lelemon/server/pkg/application/auth"
	"github.com/lelemon/server/pkg/application/ingest"
	"github.com/lelemon/server/pkg/application/project"
	"github.com/lelemon/server/pkg/application/trace"
	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/infrastructure/auth"
	"github.com/lelemon/server/pkg/interfaces/http/handler"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// RouterConfig holds the dependencies for the router
type RouterConfig struct {
	PrimaryStore   repository.Store // users, projects (API key auth)
	AnalyticsStore repository.Store // traces, spans (health checks)
	IngestSvc      *ingest.Service
	TraceSvc       *trace.Service
	AnalyticsSvc   *analytics.Service
	ProjectSvc     *project.Service
	AuthSvc        *appauth.Service
	JWTService     *auth.JWTService
	FrontendURL    string
}

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logging)
	r.Use(chimiddleware.Recoverer)
	r.Use(corsMiddleware)

	// Health checks (no auth required)
	healthHandler := handler.NewHealthHandler(cfg.PrimaryStore, cfg.AnalyticsStore)
	r.Get("/health", healthHandler.Handle)
	r.Get("/health/live", handler.LivenessHandler)
	r.Get("/health/ready", healthHandler.ReadinessHandler)

	// Rate limiter: 100 requests per minute per project
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Auth routes (no auth required)
		authHandler := handler.NewAuthHandler(cfg.AuthSvc, cfg.FrontendURL)
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Get("/auth/google", authHandler.GoogleAuth)
		r.Get("/auth/google/callback", authHandler.GoogleCallback)

		// Auth routes (session auth required)
		r.Group(func(r chi.Router) {
			r.Use(middleware.SessionAuth(cfg.JWTService))
			r.Get("/auth/me", authHandler.Me)
			r.Post("/auth/refresh", authHandler.Refresh)
		})

		// Ingest endpoint (no rate limit - SDK already batches)
		r.Group(func(r chi.Router) {
			r.Use(middleware.APIKeyAuth(cfg.PrimaryStore))

			ingestHandler := handler.NewIngestHandler(cfg.IngestSvc)
			r.Post("/ingest", ingestHandler.Handle)
		})

		// API Key authenticated routes (rate limited)
		r.Group(func(r chi.Router) {
			r.Use(middleware.APIKeyAuth(cfg.PrimaryStore))
			r.Use(middleware.RateLimit(rateLimiter))

			// Traces
			traceHandler := handler.NewTraceHandler(cfg.TraceSvc)
			r.Post("/traces", traceHandler.Create)
			r.Get("/traces", traceHandler.List)
			r.Get("/traces/{id}", traceHandler.Get)
			r.Patch("/traces/{id}", traceHandler.Update)
			r.Post("/traces/{id}/spans", traceHandler.AddSpan)

			// Sessions
			r.Get("/sessions", traceHandler.ListSessions)

			// Analytics
			analyticsHandler := handler.NewAnalyticsHandler(cfg.AnalyticsSvc)
			r.Get("/analytics/summary", analyticsHandler.Summary)
			r.Get("/analytics/usage", analyticsHandler.Usage)

			// Project (current - via API key)
			projectHandler := handler.NewProjectHandler(cfg.ProjectSvc)
			r.Get("/projects/me", projectHandler.GetCurrent)
			r.Patch("/projects/me", projectHandler.UpdateCurrent)
			r.Post("/projects/api-key", projectHandler.RotateAPIKey)
		})

		// Dashboard routes (session auth)
		r.Group(func(r chi.Router) {
			r.Use(middleware.SessionAuth(cfg.JWTService))

			dashboardHandler := handler.NewDashboardHandler(cfg.ProjectSvc, cfg.TraceSvc, cfg.AnalyticsSvc)

			// Projects
			r.Get("/dashboard/projects", dashboardHandler.ListProjects)
			r.Post("/dashboard/projects", dashboardHandler.CreateProject)
			r.Patch("/dashboard/projects/{id}", dashboardHandler.UpdateProject)
			r.Delete("/dashboard/projects/{id}", dashboardHandler.DeleteProject)
			r.Post("/dashboard/projects/{id}/api-key", dashboardHandler.RotateProjectAPIKey)

			// Project data
			r.Get("/dashboard/projects/{id}/traces", dashboardHandler.GetTraces)
			r.Delete("/dashboard/projects/{id}/traces", dashboardHandler.DeleteAllTraces)
			r.Get("/dashboard/projects/{id}/traces/{traceId}", dashboardHandler.GetTrace)
			r.Get("/dashboard/projects/{id}/sessions", dashboardHandler.GetSessions)
			r.Get("/dashboard/projects/{id}/stats", dashboardHandler.GetStats)
		})
	})

	return r
}

// corsMiddleware handles CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
