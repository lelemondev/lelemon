package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/repository"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// ProjectContextKey is the context key for the authenticated project
	ProjectContextKey contextKey = "project"
)

// APIKeyAuth creates middleware that authenticates requests via API key
func APIKeyAuth(store repository.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, `{"error":"Invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			apiKey := parts[1]
			if apiKey == "" || !strings.HasPrefix(apiKey, "le_") {
				http.Error(w, `{"error":"Invalid API key format"}`, http.StatusUnauthorized)
				return
			}

			// Hash the API key
			hash := sha256.Sum256([]byte(apiKey))
			hashStr := hex.EncodeToString(hash[:])

			// Look up project by API key hash
			project, err := store.GetProjectByAPIKeyHash(r.Context(), hashStr)
			if err != nil {
				if err == entity.ErrNotFound {
					http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
				} else {
					http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
				}
				return
			}

			// Add project to context
			ctx := context.WithValue(r.Context(), ProjectContextKey, project)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetProject retrieves the authenticated project from context
func GetProject(ctx context.Context) *entity.Project {
	project, ok := ctx.Value(ProjectContextKey).(*entity.Project)
	if !ok {
		return nil
	}
	return project
}
