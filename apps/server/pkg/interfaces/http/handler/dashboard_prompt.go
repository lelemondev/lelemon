package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/prompt"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// Dashboard prompt endpoints (session auth, JWT). Same surface as the
// API-key handler in prompt.go, but `createdBy` on new versions is the
// authenticated user's email — the dashboard knows there's a human in
// the loop, the API key does not.

// ListProjectPrompts handles GET /api/v1/dashboard/projects/{id}/prompts
func (h *DashboardHandler) ListProjectPrompts(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	page, err := h.promptSvc.List(r.Context(), projectID, promptListFilter(r))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// CreateProjectPrompt handles POST /api/v1/dashboard/projects/{id}/prompts
func (h *DashboardHandler) CreateProjectPrompt(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req prompt.CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.promptSvc.Create(r.Context(), projectID, req)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// GetProjectPrompt handles GET /api/v1/dashboard/projects/{id}/prompts/{promptId}
func (h *DashboardHandler) GetProjectPrompt(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	view, err := h.promptSvc.Get(r.Context(), projectID, chi.URLParam(r, "promptId"))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// UpdateProjectPrompt handles PATCH /api/v1/dashboard/projects/{id}/prompts/{promptId}
func (h *DashboardHandler) UpdateProjectPrompt(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req prompt.UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.promptSvc.Update(r.Context(), projectID, chi.URLParam(r, "promptId"), req); err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DeleteProjectPrompt handles DELETE /api/v1/dashboard/projects/{id}/prompts/{promptId}
func (h *DashboardHandler) DeleteProjectPrompt(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	if err := h.promptSvc.Delete(r.Context(), projectID, chi.URLParam(r, "promptId")); err != nil {
		writePromptError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CreateProjectPromptVersion handles POST /api/v1/dashboard/projects/{id}/prompts/{promptId}/versions
func (h *DashboardHandler) CreateProjectPromptVersion(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req prompt.CreatePromptVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Thread the dashboard user's email through as createdBy. The middleware
	// has already authenticated; verifyProjectOwnership above also checks
	// ownership, so the user is guaranteed non-nil at this point.
	user := middleware.GetUser(r.Context())
	var createdBy *string
	if user != nil && user.Email != "" {
		email := user.Email
		createdBy = &email
	}

	view, err := h.promptSvc.CreateVersion(r.Context(), projectID,
		chi.URLParam(r, "promptId"), req, createdBy)
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// GetProjectPromptVersion handles GET /api/v1/dashboard/projects/{id}/prompts/{promptId}/versions/{versionId}
func (h *DashboardHandler) GetProjectPromptVersion(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	view, err := h.promptSvc.GetVersion(r.Context(), projectID,
		chi.URLParam(r, "promptId"), chi.URLParam(r, "versionId"))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// ListProjectPromptVersions handles GET /api/v1/dashboard/projects/{id}/prompts/{promptId}/versions
func (h *DashboardHandler) ListProjectPromptVersions(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	page, err := h.promptSvc.ListVersions(r.Context(), projectID,
		chi.URLParam(r, "promptId"), promptVersionListFilter(r))
	if err != nil {
		writePromptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}
