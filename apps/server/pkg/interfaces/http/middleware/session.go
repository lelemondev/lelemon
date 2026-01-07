package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/lelemon/server/pkg/infrastructure/auth"
)

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey contextKey = "user"
)

// UserContext holds user information from JWT
type UserContext struct {
	UserID string
	Email  string
}

// SessionAuth creates middleware that authenticates requests via JWT
func SessionAuth(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
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

			token := parts[1]
			if token == "" {
				http.Error(w, `{"error":"Token required"}`, http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				if err == auth.ErrExpiredToken {
					http.Error(w, `{"error":"Token expired"}`, http.StatusUnauthorized)
				} else {
					http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				}
				return
			}

			// Add user to context
			userCtx := &UserContext{
				UserID: claims.UserID,
				Email:  claims.Email,
			}
			ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser retrieves the authenticated user from context
func GetUser(ctx context.Context) *UserContext {
	user, ok := ctx.Value(UserContextKey).(*UserContext)
	if !ok {
		return nil
	}
	return user
}
