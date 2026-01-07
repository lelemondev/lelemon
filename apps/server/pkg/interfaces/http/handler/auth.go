package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/lelemon/server/pkg/application/auth"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	service     *auth.Service
	frontendURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(service *auth.Service, frontendURL string) *AuthHandler {
	return &AuthHandler{
		service:     service,
		frontendURL: frontendURL,
	}
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req auth.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		http.Error(w, `{"error":"Email, password and name are required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.Register(r.Context(), &req)
	if err != nil {
		switch err {
		case auth.ErrEmailExists:
			http.Error(w, `{"error":"Email already registered"}`, http.StatusConflict)
		case auth.ErrWeakPassword:
			http.Error(w, `{"error":"Password must be at least 8 characters"}`, http.StatusBadRequest)
		default:
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"Email and password are required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.Login(r.Context(), &req)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			http.Error(w, `{"error":"Invalid email or password"}`, http.StatusUnauthorized)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GoogleAuth handles GET /api/v1/auth/google
func (h *AuthHandler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	if !h.service.IsOAuthConfigured() {
		http.Error(w, `{"error":"OAuth not configured"}`, http.StatusNotImplemented)
		return
	}

	// Generate state for CSRF protection
	state := generateState()

	// Set state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to Google
	url := h.service.GetGoogleAuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles GET /api/v1/auth/google/callback
func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		http.Redirect(w, r, h.frontendURL+"/login?error=invalid_state", http.StatusTemporaryRedirect)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Check for error
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Redirect(w, r, h.frontendURL+"/login?error="+errParam, http.StatusTemporaryRedirect)
		return
	}

	// Exchange code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, h.frontendURL+"/login?error=no_code", http.StatusTemporaryRedirect)
		return
	}

	result, err := h.service.HandleGoogleCallback(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/login?error=auth_failed", http.StatusTemporaryRedirect)
		return
	}

	// Redirect to frontend with token
	http.Redirect(w, r, h.frontendURL+"/auth/callback?token="+result.Token, http.StatusTemporaryRedirect)
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.service.GetCurrentUser(r.Context(), user.UserID)
	if err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"User not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Refresh handles POST /api/v1/auth/refresh (optional - for token refresh)
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Get fresh user data and generate new token
	result, err := h.service.GetCurrentUser(r.Context(), user.UserID)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"user": result,
	})
}
