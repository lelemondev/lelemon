package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/trace"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// TraceHandler handles trace requests
type TraceHandler struct {
	service *trace.Service
}

// NewTraceHandler creates a new trace handler
func NewTraceHandler(service *trace.Service) *TraceHandler {
	return &TraceHandler{service: service}
}

// Create handles POST /api/v1/traces
func (h *TraceHandler) Create(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req trace.CreateTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.Create(r.Context(), project.ID, &req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// Get handles GET /api/v1/traces/{id}
func (h *TraceHandler) Get(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	traceID := chi.URLParam(r, "id")
	if traceID == "" {
		http.Error(w, `{"error":"Trace ID required"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.Get(r.Context(), project.ID, traceID)
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

// List handles GET /api/v1/traces
func (h *TraceHandler) List(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	filter := entity.TraceFilter{
		Limit:  50,
		Offset: 0,
	}

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
	if v := r.URL.Query().Get("sessionId"); v != "" {
		filter.SessionID = &v
	}
	if v := r.URL.Query().Get("userId"); v != "" {
		filter.UserID = &v
	}
	if v := r.URL.Query().Get("status"); v != "" {
		status := entity.TraceStatus(v)
		filter.Status = &status
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

	result, err := h.service.List(r.Context(), project.ID, filter)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Update handles PATCH /api/v1/traces/{id}
func (h *TraceHandler) Update(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	traceID := chi.URLParam(r, "id")
	if traceID == "" {
		http.Error(w, `{"error":"Trace ID required"}`, http.StatusBadRequest)
		return
	}

	var req trace.UpdateTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Update(r.Context(), project.ID, traceID, &req); err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"Trace not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// AddSpan handles POST /api/v1/traces/{id}/spans
func (h *TraceHandler) AddSpan(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	traceID := chi.URLParam(r, "id")
	if traceID == "" {
		http.Error(w, `{"error":"Trace ID required"}`, http.StatusBadRequest)
		return
	}

	var req trace.CreateSpanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	result, err := h.service.AddSpan(r.Context(), project.ID, traceID, &req)
	if err != nil {
		if err == entity.ErrNotFound {
			http.Error(w, `{"error":"Trace not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

// ListSessions handles GET /api/v1/sessions
func (h *TraceHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	filter := entity.SessionFilter{
		Limit:  50,
		Offset: 0,
	}

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
	if v := r.URL.Query().Get("userId"); v != "" {
		filter.UserID = &v
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

	result, err := h.service.ListSessions(r.Context(), project.ID, filter)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
