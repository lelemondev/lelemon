package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// ServiceAuth authenticates trusted service-to-service callers (the MCP authorization server
// calling the internal OAuth store API) with a shared secret presented as a Bearer token.
// Constant-time comparison; fails closed when the secret is unset.
func ServiceAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				http.Error(w, `{"error":"service auth not configured"}`, http.StatusServiceUnavailable)
				return
			}
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" ||
				subtle.ConstantTimeCompare([]byte(parts[1]), []byte(secret)) != 1 {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
