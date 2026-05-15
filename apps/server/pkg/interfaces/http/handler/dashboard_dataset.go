package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/dataset"
)

// Dashboard dataset endpoints (session auth, JWT).
//
// Every method routes through verifyProjectOwnership FIRST — the URL's
// `{id}` is the project, and we never trust it without confirming the
// authenticated user actually owns it. Same multi-tenant rule applied to
// every existing dashboard endpoint.
//
// Helpers (writeJSON, writeDatasetError, datasetListFilter, …) live in
// dataset.go in this same package and are shared with the API-key handler.

// ListProjectDatasets handles GET /api/v1/dashboard/projects/{id}/datasets
func (h *DashboardHandler) ListProjectDatasets(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	page, err := h.datasetSvc.List(r.Context(), projectID, datasetListFilter(r))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// CreateProjectDataset handles POST /api/v1/dashboard/projects/{id}/datasets
func (h *DashboardHandler) CreateProjectDataset(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req dataset.CreateDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.datasetSvc.Create(r.Context(), projectID, req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// GetProjectDataset handles GET /api/v1/dashboard/projects/{id}/datasets/{datasetId}
func (h *DashboardHandler) GetProjectDataset(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	view, err := h.datasetSvc.Get(r.Context(), projectID, chi.URLParam(r, "datasetId"))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// UpdateProjectDataset handles PATCH /api/v1/dashboard/projects/{id}/datasets/{datasetId}
func (h *DashboardHandler) UpdateProjectDataset(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req dataset.UpdateDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.datasetSvc.Update(r.Context(), projectID, chi.URLParam(r, "datasetId"), req); err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// DeleteProjectDataset handles DELETE /api/v1/dashboard/projects/{id}/datasets/{datasetId}
func (h *DashboardHandler) DeleteProjectDataset(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	if err := h.datasetSvc.Delete(r.Context(), projectID, chi.URLParam(r, "datasetId")); err != nil {
		writeDatasetError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListProjectDatasetItems handles GET /api/v1/dashboard/projects/{id}/datasets/{datasetId}/items
func (h *DashboardHandler) ListProjectDatasetItems(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	page, err := h.datasetSvc.ListItems(r.Context(), projectID,
		chi.URLParam(r, "datasetId"), datasetItemListFilter(r))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// CreateProjectDatasetItem handles POST /api/v1/dashboard/projects/{id}/datasets/{datasetId}/items
func (h *DashboardHandler) CreateProjectDatasetItem(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req dataset.CreateDatasetItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.datasetSvc.CreateItem(r.Context(), projectID, chi.URLParam(r, "datasetId"), req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// AddProjectDatasetItemFromTrace handles POST /api/v1/dashboard/projects/{id}/datasets/{datasetId}/items/from-trace
func (h *DashboardHandler) AddProjectDatasetItemFromTrace(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req dataset.AddDatasetItemFromTraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.datasetSvc.AddItemFromTrace(r.Context(), projectID, chi.URLParam(r, "datasetId"), req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// ImportProjectDatasetItems handles POST /api/v1/dashboard/projects/{id}/datasets/{datasetId}/items/import
func (h *DashboardHandler) ImportProjectDatasetItems(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req dataset.ImportDatasetItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.datasetSvc.Import(r.Context(), projectID, chi.URLParam(r, "datasetId"), req)
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// GetProjectDatasetItem handles GET /api/v1/dashboard/projects/{id}/datasets/{datasetId}/items/{itemId}
func (h *DashboardHandler) GetProjectDatasetItem(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	view, err := h.datasetSvc.GetItem(r.Context(), projectID,
		chi.URLParam(r, "datasetId"), chi.URLParam(r, "itemId"))
	if err != nil {
		writeDatasetError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// DeleteProjectDatasetItem handles DELETE /api/v1/dashboard/projects/{id}/datasets/{datasetId}/items/{itemId}
func (h *DashboardHandler) DeleteProjectDatasetItem(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	if err := h.datasetSvc.DeleteItem(r.Context(), projectID,
		chi.URLParam(r, "datasetId"), chi.URLParam(r, "itemId")); err != nil {
		writeDatasetError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
