package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
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

// ProjectAuth authenticates a project-scoped request via EITHER a project API key (the normal
// SDK/agent path) OR a trusted service call. The service path lets the out-of-process MCP
// authorization server act for a project it resolved through OAuth — without a project API key:
// it presents the shared `MCP_STORE_SECRET` as the Bearer token plus an `X-Project-Id` header,
// and we load that project into context exactly like the API-key path. All downstream handlers
// (traces, analytics, /projects/me) are unchanged — they read the project from context.
func ProjectAuth(store repository.Store, serviceSecret string) func(http.Handler) http.Handler {
	apiKeyAuth := APIKeyAuth(store)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			token := ""
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}

			// Service path: a trusted caller acting for a project by id. Only taken when the
			// token matches the configured service secret (constant-time); otherwise fall through
			// to the API-key path so a malformed/absent secret never weakens that flow.
			if serviceSecret != "" && token != "" &&
				subtle.ConstantTimeCompare([]byte(token), []byte(serviceSecret)) == 1 {
				projectID := r.Header.Get("X-Project-Id")
				if projectID == "" {
					http.Error(w, `{"error":"X-Project-Id is required"}`, http.StatusBadRequest)
					return
				}
				project, err := store.GetProjectByID(r.Context(), projectID)
				if err != nil {
					if err == entity.ErrNotFound {
						http.Error(w, `{"error":"unknown project"}`, http.StatusNotFound)
					} else {
						http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
					}
					return
				}
				ctx := context.WithValue(r.Context(), ProjectContextKey, project)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Normal path: project API key.
			apiKeyAuth(next).ServeHTTP(w, r)
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
