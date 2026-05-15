package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/eval"
)

// Dashboard eval endpoints (session auth, JWT). Same surface as the API-key
// handler in eval.go, but resolves project from the URL and verifies the
// authenticated user owns it via verifyProjectOwnership.

// ============================================
// EVAL DEFINITIONS
// ============================================

// ListProjectEvals handles GET /api/v1/dashboard/projects/{id}/evals
func (h *DashboardHandler) ListProjectEvals(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	page, err := h.evalSvc.List(r.Context(), projectID, evalListFilter(r))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// CreateProjectEval handles POST /api/v1/dashboard/projects/{id}/evals
func (h *DashboardHandler) CreateProjectEval(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	var req eval.CreateEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.evalSvc.Create(r.Context(), projectID, req)
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// GetProjectEval handles GET /api/v1/dashboard/projects/{id}/evals/{evalId}
func (h *DashboardHandler) GetProjectEval(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	view, err := h.evalSvc.Get(r.Context(), projectID, chi.URLParam(r, "evalId"))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// DeleteProjectEval handles DELETE /api/v1/dashboard/projects/{id}/evals/{evalId}
func (h *DashboardHandler) DeleteProjectEval(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	if err := h.evalSvc.Delete(r.Context(), projectID, chi.URLParam(r, "evalId")); err != nil {
		writeEvalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================
// EVAL RUNS (dashboard read mostly — runs are SDK-started in Phase 2A)
// ============================================

// ListProjectEvalRuns handles GET /api/v1/dashboard/projects/{id}/eval-runs?evalId=
func (h *DashboardHandler) ListProjectEvalRuns(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	evalID := r.URL.Query().Get("evalId")
	page, err := h.evalSvc.ListRuns(r.Context(), projectID, evalID, evalRunListFilter(r))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// GetProjectEvalRun handles GET /api/v1/dashboard/projects/{id}/eval-runs/{runId}
func (h *DashboardHandler) GetProjectEvalRun(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	view, err := h.evalSvc.GetRun(r.Context(), projectID, chi.URLParam(r, "runId"))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// ListProjectEvalRunResults handles GET /api/v1/dashboard/projects/{id}/eval-runs/{runId}/results
func (h *DashboardHandler) ListProjectEvalRunResults(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.verifyProjectOwnership(w, r)
	if !ok {
		return
	}
	page, err := h.evalSvc.ListResults(r.Context(), projectID,
		chi.URLParam(r, "runId"), evalRunResultListFilter(r))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}
