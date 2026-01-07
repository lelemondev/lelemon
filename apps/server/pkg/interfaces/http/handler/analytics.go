package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/lelemon/server/pkg/application/analytics"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// AnalyticsHandler handles analytics requests
type AnalyticsHandler struct {
	service *analytics.Service
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(service *analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

// Summary handles GET /api/v1/analytics/summary
func (h *AnalyticsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
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

	result, err := h.service.GetSummary(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Usage handles GET /api/v1/analytics/usage
func (h *AnalyticsHandler) Usage(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
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

	result, err := h.service.GetUsage(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data": result,
	})
}
