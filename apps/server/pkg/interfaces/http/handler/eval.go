package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/lelemon/server/pkg/application/eval"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/interfaces/http/middleware"
)

// EvalHandler serves the API-key (SDK) surface for evals.
//
// The project is always resolved from the authenticated API key —
// never from the request body or URL — per .claude/rules/multi-tenant.md.
// This is the surface a CI script / eval harness uses: start run, post per-
// item results, finalize, read aggregates back.
type EvalHandler struct {
	svc *eval.Service
}

// NewEvalHandler wires a new API-key eval handler.
func NewEvalHandler(svc *eval.Service) *EvalHandler {
	return &EvalHandler{svc: svc}
}

// writeEvalError maps domain errors → HTTP status. Shared between the two
// auth surfaces (the dashboard handler in dashboard_eval.go calls this too).
func writeEvalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, entity.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "not found")
	case errors.Is(err, entity.ErrBadRequest):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, entity.ErrConflict):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, entity.ErrUnsupported):
		writeJSONError(w, http.StatusNotImplemented, "evals require a SQLite or Postgres primary store")
	default:
		slog.Error("eval handler error", "err", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
	}
}

func evalListFilter(r *http.Request) entity.EvalFilter {
	f := entity.EvalFilter{}
	if v := r.URL.Query().Get("datasetId"); v != "" {
		f.DatasetID = &v
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

func evalRunListFilter(r *http.Request) entity.EvalRunFilter {
	f := entity.EvalRunFilter{}
	if v := r.URL.Query().Get("status"); v != "" {
		s := entity.EvalRunStatus(v)
		f.Status = &s
	}
	if v := r.URL.Query().Get("promptVersionId"); v != "" {
		f.PromptVersionID = &v
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

func evalRunResultListFilter(r *http.Request) entity.EvalRunResultFilter {
	f := entity.EvalRunResultFilter{}
	if v := r.URL.Query().Get("passedOnly"); v == "true" {
		yes := true
		f.PassedOnly = &yes
	} else if v == "false" {
		no := false
		f.PassedOnly = &no
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
// EVALS (API-KEY) — project from middleware
// ============================================

// CreateEval handles POST /api/v1/evals
func (h *EvalHandler) CreateEval(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req eval.CreateEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.Create(r.Context(), project.ID, req)
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// ListEvals handles GET /api/v1/evals
func (h *EvalHandler) ListEvals(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := h.svc.List(r.Context(), project.ID, evalListFilter(r))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// GetEval handles GET /api/v1/evals/{evalId}
func (h *EvalHandler) GetEval(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.Get(r.Context(), project.ID, chi.URLParam(r, "evalId"))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// DeleteEval handles DELETE /api/v1/evals/{evalId}
func (h *EvalHandler) DeleteEval(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.Delete(r.Context(), project.ID, chi.URLParam(r, "evalId")); err != nil {
		writeEvalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================
// EVAL RUNS (API-KEY)
// ============================================

// StartRun handles POST /api/v1/eval-runs
func (h *EvalHandler) StartRun(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req eval.StartEvalRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.StartRun(r.Context(), project.ID, req)
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// GetRun handles GET /api/v1/eval-runs/{runId}
func (h *EvalHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	view, err := h.svc.GetRun(r.Context(), project.ID, chi.URLParam(r, "runId"))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// ListRuns handles GET /api/v1/eval-runs?evalId=…
func (h *EvalHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	evalID := r.URL.Query().Get("evalId")
	page, err := h.svc.ListRuns(r.Context(), project.ID, evalID, evalRunListFilter(r))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// PostResult handles POST /api/v1/eval-runs/{runId}/results
func (h *EvalHandler) PostResult(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req eval.PostEvalRunResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.PostResult(r.Context(), project.ID, chi.URLParam(r, "runId"), req)
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// FinalizeRun handles POST /api/v1/eval-runs/{runId}/finalize
func (h *EvalHandler) FinalizeRun(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req eval.FinalizeEvalRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	view, err := h.svc.Finalize(r.Context(), project.ID, chi.URLParam(r, "runId"), req)
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// ListResults handles GET /api/v1/eval-runs/{runId}/results
func (h *EvalHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	project := middleware.GetProject(r.Context())
	if project == nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	page, err := h.svc.ListResults(r.Context(), project.ID, chi.URLParam(r, "runId"), evalRunResultListFilter(r))
	if err != nil {
		writeEvalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}
