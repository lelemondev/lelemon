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

// SessionAuth creates middleware that authenticates requests via JWT.
// Checks httpOnly cookie first, then falls back to Authorization header.
func SessionAuth(jwtService *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractSessionToken(r)
			if token == "" {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				if err == auth.ErrExpiredToken {
					http.Error(w, `{"error":"Token expired"}`, http.StatusUnauthorized)
				} else {
					http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				}
				return
			}

			userCtx := &UserContext{
				UserID: claims.UserID,
				Email:  claims.Email,
			}
			ctx := context.WithValue(r.Context(), UserContextKey, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractSessionToken gets the JWT from cookie first, then Authorization header
func extractSessionToken(r *http.Request) string {
	// 1. Try httpOnly cookie (preferred, secure)
	if cookie, err := r.Cookie("lelemon_session"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Fall back to Authorization header (backward compat, SDK usage)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}

// GetUser retrieves the authenticated user from context
func GetUser(ctx context.Context) *UserContext {
	user, ok := ctx.Value(UserContextKey).(*UserContext)
	if !ok {
		return nil
	}
	return user
}
