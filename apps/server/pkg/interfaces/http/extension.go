package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/infrastructure/auth"
	"github.com/lelemon/server/pkg/interfaces/http/handler"
)

// RouterExtension allows external packages to extend the core router
// without modifying its source code. This follows the Open/Closed principle.
type RouterExtension interface {
	// MountRoutes adds routes to the provided chi router.
	// Called after core routes are registered.
	MountRoutes(r chi.Router, deps *RouterDeps)
}

// RouterDeps provides access to core dependencies for extensions.
// Extensions can use these to integrate with core services.
type RouterDeps struct {
	// Stores
	PrimaryStore   repository.Store
	AnalyticsStore repository.Store

	// Auth
	JWTService *auth.JWTService

	// Helper to get user ID from request context (set by SessionAuth middleware)
	GetUserID func(r *http.Request) string

	// Helper to get user email from request context (set by SessionAuth middleware)
	GetUserEmail func(r *http.Request) string
}

// EnterpriseFeaturesConfig returns features config for enterprise edition.
// Use this when creating an enterprise server.
func EnterpriseFeaturesConfig() *handler.FeaturesConfig {
	return &handler.FeaturesConfig{
		Edition: "enterprise",
		Features: map[string]bool{
			"organizations": true,
			"rbac":          true,
			"billing":       true,
			"sso":           true,
		},
	}
}
