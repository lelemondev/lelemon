package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/lelemon/server/pkg/application/analytics"
	"github.com/lelemon/server/pkg/domain/entity"
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

// parseTime parses an RFC3339 date or returns an error
func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

// parsePeriodParams extracts and validates from/to/prefix/limit from query params.
// Returns 400 on invalid date format.
func parsePeriodParams(w http.ResponseWriter, r *http.Request) (*analytics.PeriodRequest, bool) {
	req := &analytics.PeriodRequest{}

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := parseTime(v)
		if err != nil {
			http.Error(w, `{"error":"Invalid 'from' date format. Use RFC3339 (e.g. 2026-04-01T00:00:00Z)"}`, http.StatusBadRequest)
			return nil, false
		}
		req.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := parseTime(v)
		if err != nil {
			http.Error(w, `{"error":"Invalid 'to' date format. Use RFC3339 (e.g. 2026-04-11T00:00:00Z)"}`, http.StatusBadRequest)
			return nil, false
		}
		req.To = &t
	}

	req.Prefix = r.URL.Query().Get("prefix")
	req.Tag = r.URL.Query().Get("tag")
	req.SessionID = r.URL.Query().Get("sessionId")
	req.UserID = r.URL.Query().Get("userId")
	req.Name = r.URL.Query().Get("name")

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			http.Error(w, `{"error":"Invalid 'limit'. Must be between 1 and 1000"}`, http.StatusBadRequest)
			return nil, false
		}
		req.Limit = n
	}

	return req, true
}

// parseGranularityParams extracts and validates from/to/granularity from query params.
func parseGranularityParams(w http.ResponseWriter, r *http.Request) (*analytics.UsageRequest, bool) {
	req := &analytics.UsageRequest{}

	if v := r.URL.Query().Get("from"); v != "" {
		t, err := parseTime(v)
		if err != nil {
			http.Error(w, `{"error":"Invalid 'from' date format. Use RFC3339"}`, http.StatusBadRequest)
			return nil, false
		}
		req.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := parseTime(v)
		if err != nil {
			http.Error(w, `{"error":"Invalid 'to' date format. Use RFC3339"}`, http.StatusBadRequest)
			return nil, false
		}
		req.To = &t
	}
	if v := r.URL.Query().Get("granularity"); v != "" {
		if !entity.ValidGranularity(v) {
			http.Error(w, `{"error":"Invalid 'granularity'. Must be 'hour', 'day', or 'week'"}`, http.StatusBadRequest)
			return nil, false
		}
		req.Granularity = v
	}

	return req, true
}

func respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": data})
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
		if t, err := parseTime(v); err == nil {
			req.From = &t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := parseTime(v); err == nil {
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

	req, ok := parseGranularityParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetUsage(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

// Models handles GET /api/v1/analytics/models
func (h *AnalyticsHandler) Models(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetModelStats(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

// Tags handles GET /api/v1/analytics/tags
func (h *AnalyticsHandler) Tags(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetTagStats(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

// TopUsers handles GET /api/v1/analytics/top-users
func (h *AnalyticsHandler) TopUsers(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetTopUsers(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

// Heatmap handles GET /api/v1/analytics/heatmap
func (h *AnalyticsHandler) Heatmap(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetHourlyHeatmap(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

// LatencyDistribution handles GET /api/v1/analytics/latency/distribution
func (h *AnalyticsHandler) LatencyDistribution(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	req, ok := parsePeriodParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetLatencyDistribution(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}

// LatencyTimeSeries handles GET /api/v1/analytics/latency/timeseries
func (h *AnalyticsHandler) LatencyTimeSeries(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	req, ok := parseGranularityParams(w, r)
	if !ok {
		return
	}

	result, err := h.service.GetLatencyTimeSeries(r.Context(), project.ID, req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	respondJSON(w, result)
}
