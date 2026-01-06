package handler

import (
	"encoding/json"
	"net/http"

	"github.com/lelemon/server/internal/application/ingest"
	"github.com/lelemon/server/internal/interfaces/http/middleware"
)

// IngestHandler handles event ingestion requests
type IngestHandler struct {
	service *ingest.Service
}

// NewIngestHandler creates a new ingest handler
func NewIngestHandler(service *ingest.Service) *IngestHandler {
	return &IngestHandler{service: service}
}

// Handle processes POST /api/v1/ingest requests
func (h *IngestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Get authenticated project from context
	project := middleware.GetProject(r.Context())
	if project == nil {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req ingest.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Process events
	resp, err := h.service.Ingest(r.Context(), project.ID, &req)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if !resp.Success {
		w.WriteHeader(http.StatusMultiStatus)
	}
	json.NewEncoder(w).Encode(resp)
}
