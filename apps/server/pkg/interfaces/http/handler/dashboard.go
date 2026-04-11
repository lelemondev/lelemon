package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/analytics"
	"github.com/lelemon/server/pkg/application/project"
	"github.com/lelemon/server/pkg/application/trace"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// DashboardHandler handles dashboard requests (session auth)
type DashboardHandler struct {
	projectSvc   *project.Service
	traceSvc     *trace.Service
	analyticsSvc *analytics.Service
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(
	projectSvc *project.Service,
	traceSvc *trace.Service,
	analyticsSvc *analytics.Service,
) *DashboardHandler {
	return &DashboardHandler{
		projectSvc:   projectSvc,
		traceSvc:     traceSvc,
		analyticsSvc: analyticsSvc,
	}
}

// ListProjects handles GET /api/v1/dashboard/projects
func (h *DashboardHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Map to response (hide API key, only show first 8 chars)
	type projectResp struct {
		ID        string                  `json:"id"`
		Name      string                  `json:"name"`
		APIKey    string                  `json:"apiKey"`
		Settings  entity.ProjectSettings `json:"settings"`
		CreatedAt time.Time               `json:"createdAt"`
		UpdatedAt time.Time               `json:"updatedAt"`
	}

	resp := make([]projectResp, len(projects))
	for i, p := range projects {
		apiKeyPreview := p.APIKey
		if len(apiKeyPreview) > 12 {
			apiKeyPreview = apiKeyPreview[:12] + "..."
		}
		resp[i] = projectResp{
			ID:        p.ID,
			Name:      p.Name,
			APIKey:    apiKeyPreview,
			Settings:  p.Settings,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CreateProject handles POST /api/v1/dashboard/projects
func (h *DashboardHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req project.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"Name is required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.projectSvc.Create(r.Context(), user.Email, &req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// UpdateProject handles PATCH /api/v1/dashboard/projects/{id}
func (h *DashboardHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		http.Error(w, `{"error":"Project ID required"}`, http.StatusBadRequest)
		return
	}

	var req project.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.projectSvc.Update(r.Context(), projectID, user.Email, &req); err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeleteProject handles DELETE /api/v1/dashboard/projects/{id}
func (h *DashboardHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		http.Error(w, `{"error":"Project ID required"}`, http.StatusBadRequest)
		return
	}

	if err := h.projectSvc.Delete(r.Context(), projectID, user.Email); err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RotateProjectAPIKey handles POST /api/v1/dashboard/projects/{id}/api-key
func (h *DashboardHandler) RotateProjectAPIKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		http.Error(w, `{"error":"Project ID required"}`, http.StatusBadRequest)
		return
	}

	// Verify ownership first
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	result, err := h.projectSvc.RotateAPIKey(r.Context(), projectID)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetTraces handles GET /api/v1/dashboard/projects/{id}/traces
func (h *DashboardHandler) GetTraces(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		http.Error(w, `{"error":"Project ID required"}`, http.StatusBadRequest)
		return
	}

	// Verify ownership
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	// Parse filters
	filter := entity.TraceFilter{Limit: 50, Offset: 0}
	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		status := entity.TraceStatus(v)
		filter.Status = &status
	}
	if v := r.URL.Query().Get("name"); v != "" {
		filter.Name = &v
	}
	if v := r.URL.Query().Get("sessionId"); v != "" {
		filter.SessionID = &v
	}
	if v := r.URL.Query().Get("userId"); v != "" {
		filter.UserID = &v
	}
	if tags := r.URL.Query()["tags"]; len(tags) > 0 {
		filter.Tags = tags
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.To = &t
		}
	}

	result, err := h.traceSvc.List(r.Context(), projectID, filter)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetTrace handles GET /api/v1/dashboard/projects/{id}/traces/{traceId}
func (h *DashboardHandler) GetTrace(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	traceID := chi.URLParam(r, "traceId")

	// Verify ownership
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	result, err := h.traceSvc.GetDetail(r.Context(), projectID, traceID)
	if err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"Trace not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetSessions handles GET /api/v1/dashboard/projects/{id}/sessions
func (h *DashboardHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		http.Error(w, `{"error":"Project ID required"}`, http.StatusBadRequest)
		return
	}

	// Verify ownership
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	// Parse filters
	filter := entity.SessionFilter{Limit: 50, Offset: 0}
	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	result, err := h.traceSvc.ListSessions(r.Context(), projectID, filter)
	slog.Info("ListSessions called", "projectID", projectID, "err", err)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}


// GetStats handles GET /api/v1/dashboard/projects/{id}/stats
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")

	// Verify ownership
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	req := &analytics.SummaryRequest{}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.To = &t
		}
	}

	result, err := h.analyticsSvc.GetSummary(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetUsage handles GET /api/v1/dashboard/projects/{id}/usage
func (h *DashboardHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")

	// Verify ownership
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	req := &analytics.UsageRequest{}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.To = &t
		}
	}
	if v := r.URL.Query().Get("granularity"); v != "" {
		req.Granularity = v
	}

	result, err := h.analyticsSvc.GetUsage(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": result,
	})
}

// DeleteAllTraces handles DELETE /api/v1/dashboard/projects/{id}/traces
func (h *DashboardHandler) DeleteAllTraces(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	projectID := chi.URLParam(r, "id")
	if projectID == "" {
		http.Error(w, `{"error":"Project ID required"}`, http.StatusBadRequest)
		return
	}

	// Verify ownership
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	found := false
	for _, p := range projects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
		return
	}

	deleted, err := h.traceSvc.DeleteAll(r.Context(), projectID)
	if err != nil {
		slog.Error("Failed to delete traces", "projectID", projectID, "error", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("Deleted all traces", "projectID", projectID, "deleted", deleted)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int64{"Deleted": deleted})
}

// verifyProjectOwnership checks the user owns the project. Returns projectID or writes error.
func (h *DashboardHandler) verifyProjectOwnership(w http.ResponseWriter, r *http.Request) (string, bool) {
	user := middleware.GetUser(r.Context())
	if user == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return "", false
	}

	projectID := chi.URLParam(r, "id")
	projects, err := h.projectSvc.List(r.Context(), user.Email)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return "", false
	}

	for _, p := range projects {
		if p.ID == projectID {
			return projectID, true
		}
	}

	http.Error(w, `{"error":"Project not found"}`, http.StatusNotFound)
	return "", false
}

func dashboardRespondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": data})
}

// GetModelStats handles GET /api/v1/dashboard/projects/{id}/analytics/models
func (h *DashboardHandler) GetModelStats(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}
	result, err := h.analyticsSvc.GetModelStats(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	dashboardRespondJSON(w, result)
}

// GetTagStats handles GET /api/v1/dashboard/projects/{id}/analytics/tags
func (h *DashboardHandler) GetTagStats(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}
	result, err := h.analyticsSvc.GetTagStats(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	dashboardRespondJSON(w, result)
}

// GetTopUsers handles GET /api/v1/dashboard/projects/{id}/analytics/top-users
func (h *DashboardHandler) GetTopUsers(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}
	result, err := h.analyticsSvc.GetTopUsers(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	dashboardRespondJSON(w, result)
}

// GetHeatmap handles GET /api/v1/dashboard/projects/{id}/analytics/heatmap
func (h *DashboardHandler) GetHeatmap(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}
	result, err := h.analyticsSvc.GetHourlyHeatmap(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	dashboardRespondJSON(w, result)
}

// GetLatencyDistribution handles GET /api/v1/dashboard/projects/{id}/analytics/latency/distribution
func (h *DashboardHandler) GetLatencyDistribution(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}
	result, err := h.analyticsSvc.GetLatencyDistribution(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	dashboardRespondJSON(w, result)
}

// GetLatencyTimeSeries handles GET /api/v1/dashboard/projects/{id}/analytics/latency/timeseries
func (h *DashboardHandler) GetLatencyTimeSeries(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}

	req := &analytics.UsageRequest{}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			req.To = &t
		}
	}
	if v := r.URL.Query().Get("granularity"); v != "" {
		if !entity.ValidGranularity(v) {
			http.Error(w, `{"error":"Invalid granularity"}`, http.StatusBadRequest)
			return
		}
		req.Granularity = v
	}

	result, err := h.analyticsSvc.GetLatencyTimeSeries(r.Context(), projectID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	dashboardRespondJSON(w, result)
}
