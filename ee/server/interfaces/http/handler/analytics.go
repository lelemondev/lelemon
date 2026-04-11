package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lelemon/ee/server/domain/entity"
	"github.com/lelemon/ee/server/domain/repository"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	store repository.AnalyticsStore
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(store repository.AnalyticsStore) *AnalyticsHandler {
	return &AnalyticsHandler{
		store: store,
	}
}

// GetCostBreakdown handles GET /projects/{projectId}/analytics/cost-breakdown
// Query params:
//   - tagPrefix: filter tags by prefix (e.g., "org:", "campaign:")
//   - from: start date (RFC3339 format)
//   - to: end date (RFC3339 format)
//   - limit: max number of results (default: 10)
func (h *AnalyticsHandler) GetCostBreakdown(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	// Parse filter parameters
	filter := entity.NewCostBreakdownFilter()

	if prefix := r.URL.Query().Get("tagPrefix"); prefix != "" {
		filter.TagPrefix = prefix
	}

	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			WriteError(w, entity.ErrInvalidInput)
			return
		}
		filter.From = &t
	}

	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			WriteError(w, entity.ErrInvalidInput)
			return
		}
		filter.To = &t
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		l, err := strconv.Atoi(limit)
		if err != nil || l <= 0 {
			WriteError(w, entity.ErrInvalidInput)
			return
		}
		filter.Limit = l
	}

	result, err := h.store.GetCostBreakdownByTags(r.Context(), projectID, filter)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetErrorMetrics handles GET /projects/{projectId}/analytics/errors
// Query params:
//   - tagPrefix: filter tags by prefix (e.g., "org:", "campaign:")
//   - from: start date (RFC3339 format)
//   - to: end date (RFC3339 format)
//   - topLimit: max number of top errors (default: 10)
func (h *AnalyticsHandler) GetErrorMetrics(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	// Parse filter parameters
	filter := entity.NewErrorFilter()

	if prefix := r.URL.Query().Get("tagPrefix"); prefix != "" {
		filter.TagPrefix = prefix
	}

	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			WriteError(w, entity.ErrInvalidInput)
			return
		}
		filter.From = &t
	}

	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			WriteError(w, entity.ErrInvalidInput)
			return
		}
		filter.To = &t
	}

	if topLimit := r.URL.Query().Get("topLimit"); topLimit != "" {
		l, err := strconv.Atoi(topLimit)
		if err != nil || l <= 0 {
			WriteError(w, entity.ErrInvalidInput)
			return
		}
		filter.TopLimit = l
	}

	result, err := h.store.GetErrorMetrics(r.Context(), projectID, filter)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}
