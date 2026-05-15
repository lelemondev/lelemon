package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/lelemon/server/pkg/application/analytics"
	appauth "github.com/lelemon/server/pkg/application/auth"
	"github.com/lelemon/server/pkg/application/dataset"
	"github.com/lelemon/server/pkg/application/eval"
	"github.com/lelemon/server/pkg/application/ingest"
	"github.com/lelemon/server/pkg/application/project"
	"github.com/lelemon/server/pkg/application/prompt"
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
	DatasetSvc     *dataset.Service // Phase 1: evals & prompt management
	EvalSvc        *eval.Service    // Phase 2A: evals & prompt management
	PromptSvc      *prompt.Service  // Phase 3A: evals & prompt management
	JWTService     *auth.JWTService
	FrontendURL    string

	// Security
	AllowedOrigins []string // CORS allowed origins

	// Extensions allow adding routes without modifying core code.
	// Used by enterprise edition to add organization, billing, etc.
	Extensions []RouterExtension

	// FeaturesConfig defines what features are available.
	// If nil, defaults to community edition features.
	FeaturesConfig *handler.FeaturesConfig
}

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.Logging)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxBodySize(5 << 20)) // 5MB max request body
	r.Use(corsMiddleware(cfg.AllowedOrigins))

	// Health checks (no auth required)
	healthHandler := handler.NewHealthHandler(cfg.PrimaryStore, cfg.AnalyticsStore)
	r.Get("/health", healthHandler.Handle)
	r.Get("/health/live", handler.LivenessHandler)
	r.Get("/health/ready", healthHandler.ReadinessHandler)

	// Rate limiters
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)         // 100 req/min per project
	authRateLimiter := middleware.NewRateLimiter(10, time.Minute)      // 10 req/min per IP for auth

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Features endpoint (no auth - frontend needs this to detect edition)
		featuresHandler := handler.NewFeaturesHandler(cfg.FeaturesConfig)
		r.Get("/features", featuresHandler.Handle)

		// Auth routes (rate limited by IP to prevent brute force)
		authHandler := handler.NewAuthHandler(cfg.AuthSvc, cfg.FrontendURL)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RateLimitByIP(authRateLimiter))
			r.Post("/auth/register", authHandler.Register)
			r.Post("/auth/login", authHandler.Login)
		})
		// OAuth routes (no rate limit - redirect-based)
		r.Get("/auth/google", authHandler.GoogleAuth)
		r.Get("/auth/google/callback", authHandler.GoogleCallback)
		r.Post("/auth/oauth/exchange", authHandler.ExchangeOAuthToken)
		r.Post("/auth/logout", authHandler.Logout)

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
			r.Get("/analytics/models", analyticsHandler.Models)
			r.Get("/analytics/tags", analyticsHandler.Tags)
			r.Get("/analytics/top-users", analyticsHandler.TopUsers)
			r.Get("/analytics/heatmap", analyticsHandler.Heatmap)
			r.Get("/analytics/latency/distribution", analyticsHandler.LatencyDistribution)
			r.Get("/analytics/latency/timeseries", analyticsHandler.LatencyTimeSeries)

			// Project (current - via API key)
			projectHandler := handler.NewProjectHandler(cfg.ProjectSvc)
			r.Get("/projects/me", projectHandler.GetCurrent)
			r.Patch("/projects/me", projectHandler.UpdateCurrent)
			r.Post("/projects/api-key", projectHandler.RotateAPIKey)

			// Datasets (Phase 1 of evals feature). Same surface as the dashboard
			// routes below, but project resolves from the API key — not the URL.
			if cfg.DatasetSvc != nil {
				datasetHandler := handler.NewDatasetHandler(cfg.DatasetSvc)
				r.Post("/datasets", datasetHandler.CreateDataset)
				r.Get("/datasets", datasetHandler.ListDatasets)
				r.Get("/datasets/{datasetId}", datasetHandler.GetDataset)
				r.Patch("/datasets/{datasetId}", datasetHandler.UpdateDataset)
				r.Delete("/datasets/{datasetId}", datasetHandler.DeleteDataset)
				r.Get("/datasets/{datasetId}/items", datasetHandler.ListDatasetItems)
				r.Post("/datasets/{datasetId}/items", datasetHandler.CreateDatasetItem)
				r.Post("/datasets/{datasetId}/items/from-trace", datasetHandler.AddDatasetItemFromTrace)
				r.Post("/datasets/{datasetId}/items/import", datasetHandler.ImportDatasetItems)
				r.Get("/datasets/{datasetId}/items/{itemId}", datasetHandler.GetDatasetItem)
				r.Delete("/datasets/{datasetId}/items/{itemId}", datasetHandler.DeleteDatasetItem)
			}

			// Evals (Phase 2A of evals feature). The eval-harness loop:
			// SDK calls StartRun → PostResult per item → Finalize.
			if cfg.EvalSvc != nil {
				evalHandler := handler.NewEvalHandler(cfg.EvalSvc)
				r.Post("/evals", evalHandler.CreateEval)
				r.Get("/evals", evalHandler.ListEvals)
				r.Get("/evals/{evalId}", evalHandler.GetEval)
				r.Delete("/evals/{evalId}", evalHandler.DeleteEval)

				r.Post("/eval-runs", evalHandler.StartRun)
				r.Get("/eval-runs", evalHandler.ListRuns)
				r.Get("/eval-runs/{runId}", evalHandler.GetRun)
				r.Post("/eval-runs/{runId}/results", evalHandler.PostResult)
				r.Get("/eval-runs/{runId}/results", evalHandler.ListResults)
				r.Post("/eval-runs/{runId}/finalize", evalHandler.FinalizeRun)
			}

			// Prompts (Phase 3A). Versions are append-only; API-key callers
			// have no human identity, so createdBy is nil for them.
			if cfg.PromptSvc != nil {
				promptHandler := handler.NewPromptHandler(cfg.PromptSvc)
				r.Post("/prompts", promptHandler.CreatePrompt)
				r.Get("/prompts", promptHandler.ListPrompts)
				r.Get("/prompts/{promptId}", promptHandler.GetPrompt)
				r.Patch("/prompts/{promptId}", promptHandler.UpdatePrompt)
				r.Delete("/prompts/{promptId}", promptHandler.DeletePrompt)
				r.Get("/prompts/{promptId}/versions", promptHandler.ListPromptVersions)
				r.Post("/prompts/{promptId}/versions", promptHandler.CreatePromptVersion)
				r.Get("/prompts/{promptId}/versions/{versionId}", promptHandler.GetPromptVersion)
			}
		})

		// Dashboard routes (session auth)
		r.Group(func(r chi.Router) {
			r.Use(middleware.SessionAuth(cfg.JWTService))

			dashboardHandler := handler.NewDashboardHandler(cfg.ProjectSvc, cfg.TraceSvc, cfg.AnalyticsSvc, cfg.DatasetSvc, cfg.EvalSvc, cfg.PromptSvc)

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
			r.Get("/dashboard/projects/{id}/usage", dashboardHandler.GetUsage)

			// Analytics V2 (dashboard auth, project-scoped)
			r.Get("/dashboard/projects/{id}/analytics/models", dashboardHandler.GetModelStats)
			r.Get("/dashboard/projects/{id}/analytics/tags", dashboardHandler.GetTagStats)
			r.Get("/dashboard/projects/{id}/analytics/top-users", dashboardHandler.GetTopUsers)
			r.Get("/dashboard/projects/{id}/analytics/heatmap", dashboardHandler.GetHeatmap)
			r.Get("/dashboard/projects/{id}/analytics/latency/distribution", dashboardHandler.GetLatencyDistribution)
			r.Get("/dashboard/projects/{id}/analytics/latency/timeseries", dashboardHandler.GetLatencyTimeSeries)

			// Datasets (Phase 1 of evals feature). Gated on the service being
			// wired so an operator deploying without it gets a clean 404, not
			// a nil-pointer panic.
			if cfg.DatasetSvc != nil {
				r.Get("/dashboard/projects/{id}/datasets", dashboardHandler.ListProjectDatasets)
				r.Post("/dashboard/projects/{id}/datasets", dashboardHandler.CreateProjectDataset)
				r.Get("/dashboard/projects/{id}/datasets/{datasetId}", dashboardHandler.GetProjectDataset)
				r.Patch("/dashboard/projects/{id}/datasets/{datasetId}", dashboardHandler.UpdateProjectDataset)
				r.Delete("/dashboard/projects/{id}/datasets/{datasetId}", dashboardHandler.DeleteProjectDataset)

				r.Get("/dashboard/projects/{id}/datasets/{datasetId}/items", dashboardHandler.ListProjectDatasetItems)
				r.Post("/dashboard/projects/{id}/datasets/{datasetId}/items", dashboardHandler.CreateProjectDatasetItem)
				r.Post("/dashboard/projects/{id}/datasets/{datasetId}/items/from-trace", dashboardHandler.AddProjectDatasetItemFromTrace)
				r.Post("/dashboard/projects/{id}/datasets/{datasetId}/items/import", dashboardHandler.ImportProjectDatasetItems)
				r.Get("/dashboard/projects/{id}/datasets/{datasetId}/items/{itemId}", dashboardHandler.GetProjectDatasetItem)
				r.Delete("/dashboard/projects/{id}/datasets/{datasetId}/items/{itemId}", dashboardHandler.DeleteProjectDatasetItem)
			}

			// Evals (Phase 2A). Dashboard surface for inspecting eval definitions
			// and SDK-started runs. Run lifecycle endpoints (start/post/finalize)
			// live on the API-key surface above — they're driven by CI/scripts.
			if cfg.EvalSvc != nil {
				r.Get("/dashboard/projects/{id}/evals", dashboardHandler.ListProjectEvals)
				r.Post("/dashboard/projects/{id}/evals", dashboardHandler.CreateProjectEval)
				r.Get("/dashboard/projects/{id}/evals/{evalId}", dashboardHandler.GetProjectEval)
				r.Delete("/dashboard/projects/{id}/evals/{evalId}", dashboardHandler.DeleteProjectEval)

				r.Get("/dashboard/projects/{id}/eval-runs", dashboardHandler.ListProjectEvalRuns)
				r.Get("/dashboard/projects/{id}/eval-runs/{runId}", dashboardHandler.GetProjectEvalRun)
				r.Get("/dashboard/projects/{id}/eval-runs/{runId}/results", dashboardHandler.ListProjectEvalRunResults)
			}

			// Prompts (Phase 3A). Versions threaded with createdBy = JWT user.
			if cfg.PromptSvc != nil {
				r.Get("/dashboard/projects/{id}/prompts", dashboardHandler.ListProjectPrompts)
				r.Post("/dashboard/projects/{id}/prompts", dashboardHandler.CreateProjectPrompt)
				r.Get("/dashboard/projects/{id}/prompts/{promptId}", dashboardHandler.GetProjectPrompt)
				r.Patch("/dashboard/projects/{id}/prompts/{promptId}", dashboardHandler.UpdateProjectPrompt)
				r.Delete("/dashboard/projects/{id}/prompts/{promptId}", dashboardHandler.DeleteProjectPrompt)

				r.Get("/dashboard/projects/{id}/prompts/{promptId}/versions", dashboardHandler.ListProjectPromptVersions)
				r.Post("/dashboard/projects/{id}/prompts/{promptId}/versions", dashboardHandler.CreateProjectPromptVersion)
				r.Get("/dashboard/projects/{id}/prompts/{promptId}/versions/{versionId}", dashboardHandler.GetProjectPromptVersion)
			}
		})
	})

	// Mount extensions (e.g., enterprise routes)
	if len(cfg.Extensions) > 0 {
		deps := &RouterDeps{
			PrimaryStore:   cfg.PrimaryStore,
			AnalyticsStore: cfg.AnalyticsStore,
			JWTService:     cfg.JWTService,
			GetUserID: func(req *http.Request) string {
				user := middleware.GetUser(req.Context())
				if user == nil {
					return ""
				}
				return user.UserID
			},
			GetUserEmail: func(req *http.Request) string {
				user := middleware.GetUser(req.Context())
				if user == nil {
					return ""
				}
				return user.Email
			},
		}

		for _, ext := range cfg.Extensions {
			ext.MountRoutes(r, deps)
		}
	}

	return r
}

// corsMiddleware handles CORS headers with origin allowlist
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	// Build a map for O(1) lookups
	originMap := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		originMap[origin] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && originMap[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Vary", "Origin")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
