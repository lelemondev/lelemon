package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/prompt"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// PromptHandler serves the API-key (SDK) surface for prompts.
//
// API-key callers have no human identity, so CreateVersion always receives
// nil for `createdBy`. The dashboard handler (dashboard_prompt.go) threads
// the JWT user's email through instead.
type PromptHandler struct {
	svc *prompt.Service
}

// NewPromptHandler wires a new API-key prompt handler.
func NewPromptHandler(svc *prompt.Service) *PromptHandler {
	return &PromptHandler{svc: svc}
}

// writePromptError maps domain errors → HTTP status. Shared between the two
// auth surfaces. ErrConflict is meaningful here (UNIQUE on version label),
// so it gets its own 409.
func writePromptError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not found")
	case errors.Is(err, entity.ErrBadRequest):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrConflict):
		writeJSONError(w, http.StatusConflict, "a version with that label already exists for this prompt")
	case errors.Is(err, entity.ErrUnsupported):
		writeJSONError(w, http.StatusNotImplemented, "prompts require a SQLite or Postgres primary store")
	default:
		slog.Error("prompt handler error", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
	}
}

func promptListFilter(r *http.Request) entity.PromptFilter {
	f := entity.PromptFilter{}
	if v := r.URL.Query().Get("name"); v != "" {
		f.Name = &v
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}
	return f
}

func promptVersionListFilter(r *http.Request) entity.PromptVersionFilter {
	f := entity.PromptVersionFilter{}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			f.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}
	return f
}

// ============================================
// API-KEY — project from middleware, createdBy nil
// ============================================

// CreatePrompt handles POST /api/v1/prompts
func (h *PromptHandler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req prompt.CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.Create(r.Context(), project.ID, req)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// ListPrompts handles GET /api/v1/prompts
func (h *PromptHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := h.svc.List(r.Context(), project.ID, promptListFilter(r))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// GetPrompt handles GET /api/v1/prompts/{promptId}
func (h *PromptHandler) GetPrompt(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.Get(r.Context(), project.ID, chi.URLParam(r, "promptId"))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// UpdatePrompt handles PATCH /api/v1/prompts/{promptId}
func (h *PromptHandler) UpdatePrompt(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req prompt.UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.Update(r.Context(), project.ID, chi.URLParam(r, "promptId"), req); err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DeletePrompt handles DELETE /api/v1/prompts/{promptId}
func (h *PromptHandler) DeletePrompt(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.Delete(r.Context(), project.ID, chi.URLParam(r, "promptId")); err != nil {
		writePromptError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreatePromptVersion handles POST /api/v1/prompts/{promptId}/versions
func (h *PromptHandler) CreatePromptVersion(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req prompt.CreatePromptVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// API-key callers have no human identity → createdBy is nil.
	view, err := h.svc.CreateVersion(r.Context(), project.ID, chi.URLParam(r, "promptId"), req, nil)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// GetPromptVersion handles GET /api/v1/prompts/{promptId}/versions/{versionId}
func (h *PromptHandler) GetPromptVersion(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.GetVersion(r.Context(), project.ID,
		chi.URLParam(r, "promptId"), chi.URLParam(r, "versionId"))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// ListPromptVersions handles GET /api/v1/prompts/{promptId}/versions
func (h *PromptHandler) ListPromptVersions(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := h.svc.ListVersions(r.Context(), project.ID,
		chi.URLParam(r, "promptId"), promptVersionListFilter(r))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}
