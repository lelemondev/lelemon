package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/dataset"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// DatasetHandler serves the API-key (SDK) surface for datasets. The project
// always comes from the authenticated API key — never from the request body or
// URL — per .claude/rules/multi-tenant.md.
type DatasetHandler struct {
	svc *dataset.Service
}

// NewDatasetHandler wires a new API-key dataset handler.
func NewDatasetHandler(svc *dataset.Service) *DatasetHandler {
	return &DatasetHandler{svc: svc}
}

// ---------- helpers (dataset-specific; generic ones live in responses.go) -

// writeDatasetError maps domain errors to HTTP status codes. Used by both
// auth surfaces so the wire contract stays consistent.
func writeDatasetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not found")
	case errors.Is(err, entity.ErrBadRequest):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrUnsupported):
		// Surface this clearly — operators running ClickHouse-as-primary need to
		// understand they have to switch to SQLite/Postgres for evals features.
		writeJSONError(w, http.StatusNotImplemented, "datasets require a SQLite or Postgres primary store")
	default:
		slog.Error("dataset handler error", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
	}
}

// datasetItemListFilter parses common query params for item listing.
func datasetItemListFilter(r *http.Request) entity.DatasetItemFilter {
	f := entity.DatasetItemFilter{}
	if v := r.URL.Query().Get("sourceTraceId"); v != "" {
		f.SourceTraceID = &v
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

func datasetListFilter(r *http.Request) entity.DatasetFilter {
	f := entity.DatasetFilter{}
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

// ============================================
// API-KEY ENDPOINTS — project from middleware
// ============================================

// CreateDataset handles POST /api/v1/datasets
func (h *DatasetHandler) CreateDataset(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dataset.CreateDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.Create(r.Context(), project.ID, req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// ListDatasets handles GET /api/v1/datasets
func (h *DatasetHandler) ListDatasets(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := h.svc.List(r.Context(), project.ID, datasetListFilter(r))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// GetDataset handles GET /api/v1/datasets/{datasetId}
func (h *DatasetHandler) GetDataset(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.Get(r.Context(), project.ID, chi.URLParam(r, "datasetId"))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// UpdateDataset handles PATCH /api/v1/datasets/{datasetId}
func (h *DatasetHandler) UpdateDataset(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dataset.UpdateDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.Update(r.Context(), project.ID, chi.URLParam(r, "datasetId"), req); err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DeleteDataset handles DELETE /api/v1/datasets/{datasetId}
func (h *DatasetHandler) DeleteDataset(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.Delete(r.Context(), project.ID, chi.URLParam(r, "datasetId")); err != nil {
		writeDatasetError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListDatasetItems handles GET /api/v1/datasets/{datasetId}/items
func (h *DatasetHandler) ListDatasetItems(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := h.svc.ListItems(r.Context(), project.ID, chi.URLParam(r, "datasetId"), datasetItemListFilter(r))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// CreateDatasetItem handles POST /api/v1/datasets/{datasetId}/items
func (h *DatasetHandler) CreateDatasetItem(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dataset.CreateDatasetItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.CreateItem(r.Context(), project.ID, chi.URLParam(r, "datasetId"), req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// AddDatasetItemFromTrace handles POST /api/v1/datasets/{datasetId}/items/from-trace
func (h *DatasetHandler) AddDatasetItemFromTrace(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dataset.AddDatasetItemFromTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.AddItemFromTrace(r.Context(), project.ID, chi.URLParam(r, "datasetId"), req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// ImportDatasetItems handles POST /api/v1/datasets/{datasetId}/items/import
func (h *DatasetHandler) ImportDatasetItems(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req dataset.ImportDatasetItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.svc.Import(r.Context(), project.ID, chi.URLParam(r, "datasetId"), req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetDatasetItem handles GET /api/v1/datasets/{datasetId}/items/{itemId}
func (h *DatasetHandler) GetDatasetItem(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.GetItem(r.Context(), project.ID,
		chi.URLParam(r, "datasetId"), chi.URLParam(r, "itemId"))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// DeleteDatasetItem handles DELETE /api/v1/datasets/{datasetId}/items/{itemId}
func (h *DatasetHandler) DeleteDatasetItem(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.DeleteItem(r.Context(), project.ID,
		chi.URLParam(r, "datasetId"), chi.URLParam(r, "itemId")); err != nil {
		writeDatasetError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
