package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lelemon/server/internal/application/project"
	"github.com/lelemon/server/internal/domain/entity"
	"github.com/lelemon/server/internal/interfaces/http/middleware"
)

// ProjectHandler handles project requests
type ProjectHandler struct {
	service *project.Service
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(service *project.Service) *ProjectHandler {
	return &ProjectHandler{service: service}
}

// GetCurrent handles GET /api/v1/projects/me
func (h *ProjectHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	proj := middleware.GetProject(r.Context())
	if proj == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	result := h.service.GetCurrent(r.Context(), proj)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// UpdateCurrent handles PATCH /api/v1/projects/me
func (h *ProjectHandler) UpdateCurrent(w http.ResponseWriter, r *http.Request) {
	proj := middleware.GetProject(r.Context())
	if proj == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req project.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateCurrent(r.Context(), proj.ID, &req); err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RotateAPIKey handles POST /api/v1/projects/api-key
func (h *ProjectHandler) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	proj := middleware.GetProject(r.Context())
	if proj == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.service.RotateAPIKey(r.Context(), proj.ID)
	if err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
