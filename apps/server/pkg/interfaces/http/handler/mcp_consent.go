package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// MCPConsentHandler mints the short-lived consent token that bridges the dashboard's session to
// the MCP authorization server. The MCP (a different origin) cannot read the dashboard cookie, so
// after the logged-in user approves and picks a project here, we sign a 3-minute HS256 JWT bound to
// {userId, projectId} that the user's browser carries back to the MCP `/authorize` (which verifies
// it with the same MCP_CONSENT_SECRET). The token grants identity, not access — the OAuth code +
// PKCE are still required to obtain real tokens.
type MCPConsentHandler struct {
	store  repository.Store
	secret []byte
}

func NewMCPConsentHandler(store repository.Store, secret string) *MCPConsentHandler {
	return &MCPConsentHandler{store: store, secret: []byte(secret)}
}

// ConsentTTL is intentionally short — the token rides back through a URL query parameter.
const ConsentTTL = 3 * time.Minute

// ConsentAudience binds the token to its single purpose (RFC 7519 `aud`).
const ConsentAudience = "mcp-consent"

func (h *MCPConsentHandler) Mint(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil || user.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProjectID == "" {
		http.Error(w, `{"error":"projectId is required"}`, http.StatusBadRequest)
		return
	}

	// Multi-tenant: only the project's owner may grant MCP access to it.
	owns, err := h.store.IsProjectOwner(r.Context(), req.ProjectID, user.Email)
	if err != nil {
		slog.Error("consent ownership check failed", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if !owns {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       user.UserID,
		"projectId": req.ProjectID,
		"aud":       ConsentAudience,
		"iat":       now.Unix(),
		"exp":       now.Add(ConsentTTL).Unix(),
	}).SignedString(h.secret)
	if err != nil {
		slog.Error("consent token signing failed", "error", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"consentToken": token}); err != nil {
		slog.Error("consent response encode failed", "error", err)
	}
}
