// Package eval is the application-layer service for evaluation definitions
// and their SDK-driven runs — Phase 2A of the evals & prompt management
// feature.
//
// Architectural notes:
//
//   - The service depends on narrow interfaces declared here (ISP), not on
//     repository.Store. Wiring in main.go passes the primary store, which
//     satisfies both.
//   - Scoring is server-side. The SDK posts `actual`; the server loads the
//     dataset item's `expected` plus the eval's scorers and computes pass/fail.
//     That keeps the source of truth honest — a misbehaving client cannot
//     claim "all green".
//   - Evals are immutable after creation in Phase 2A (no Update method): to
//     change scorers, create a new eval. This sidesteps the "what happens to
//     past runs when scorers change" question until there's a real need.
package eval

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lelemon/server/pkg/domain/entity"
)

// EvalRepo is the eval-persistence surface this service consumes.
type EvalRepo interface {
	CreateEval(ctx context.Context, e *entity.Eval) error
	GetEval(ctx context.Context, projectID, evalID string) (*entity.Eval, error)
	ListEvals(ctx context.Context, projectID string, filter entity.EvalFilter) (*entity.Page[entity.Eval], error)
	DeleteEval(ctx context.Context, projectID, evalID string) error

	CreateEvalRun(ctx context.Context, r *entity.EvalRun) error
	GetEvalRun(ctx context.Context, projectID, runID string) (*entity.EvalRun, error)
	ListEvalRuns(ctx context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*entity.Page[entity.EvalRun], error)
	UpdateEvalRunStatus(ctx context.Context, projectID, runID string, status entity.EvalRunStatus) error
	FinalizeEvalRun(ctx context.Context, projectID, runID string, in entity.FinalizeEvalRunInput) (*entity.EvalRun, error)

	CreateEvalRunResult(ctx context.Context, res *entity.EvalRunResult) error
	ListEvalRunResults(ctx context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*entity.Page[entity.EvalRunResult], error)
}

// DatasetReader is the slice of the dataset store this service uses.
// Reading the dataset is needed to validate references; reading items is
// needed to fetch `expected` values for scoring.
type DatasetReader interface {
	GetDataset(ctx context.Context, projectID, datasetID string) (*entity.Dataset, error)
	GetDatasetItem(ctx context.Context, projectID, itemID string) (*entity.DatasetItem, error)
}

// Validation limits.
const (
	maxEvalName        = 200
	maxEvalDescription = 2000
	maxScorers         = 20
	maxScorerName      = 200
)

// Service is the eval use-case orchestrator. Construct via NewService.
type Service struct {
	repo     EvalRepo
	datasets DatasetReader
}

// NewService wires a new eval service.
func NewService(repo EvalRepo, datasets DatasetReader) *Service {
	return &Service{repo: repo, datasets: datasets}
}

// ============================================
// EVAL DEFINITION
// ============================================

// Create validates and persists a new eval definition.
func (s *Service) Create(ctx context.Context, projectID string, req CreateEvalRequest) (*EvalView, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", entity.ErrBadRequest)
	}
	if len(name) > maxEvalName {
		return nil, fmt.Errorf("%w: name exceeds %d chars", entity.ErrBadRequest, maxEvalName)
	}
	if req.Description != nil && len(*req.Description) > maxEvalDescription {
		return nil, fmt.Errorf("%w: description exceeds %d chars", entity.ErrBadRequest, maxEvalDescription)
	}
	if req.DatasetID == "" {
		return nil, fmt.Errorf("%w: datasetId is required", entity.ErrBadRequest)
	}
	if len(req.Scorers) == 0 {
		return nil, fmt.Errorf("%w: at least one scorer is required", entity.ErrBadRequest)
	}
	if len(req.Scorers) > maxScorers {
		return nil, fmt.Errorf("%w: maximum %d scorers per eval", entity.ErrBadRequest, maxScorers)
	}
	if err := validateScorers(req.Scorers); err != nil {
		return nil, fmt.Errorf("%w: %s", entity.ErrBadRequest, err.Error())
	}

	// Verify the dataset belongs to this project — short-circuit on 404.
	if _, err := s.datasets.GetDataset(ctx, projectID, req.DatasetID); err != nil {
		return nil, err
	}

	e := &entity.Eval{
		ProjectID:   projectID,
		DatasetID:   req.DatasetID,
		Name:        name,
		Description: req.Description,
		Scorers:     req.Scorers,
	}
	if err := s.repo.CreateEval(ctx, e); err != nil {
		return nil, fmt.Errorf("create eval: %w", err)
	}
	v := toEvalView(e)
	return &v, nil
}

// Get returns a single eval by id, scoped to the project.
func (s *Service) Get(ctx context.Context, projectID, evalID string) (*EvalView, error) {
	e, err := s.repo.GetEval(ctx, projectID, evalID)
	if err != nil {
		return nil, err
	}
	v := toEvalView(e)
	return &v, nil
}

// List returns a paginated list of evals, optionally filtered to one dataset.
func (s *Service) List(ctx context.Context, projectID string, filter entity.EvalFilter) (*EvalListResponse, error) {
	page, err := s.repo.ListEvals(ctx, projectID, filter)
	if err != nil {
		return nil, fmt.Errorf("list evals: %w", err)
	}
	resp := toEvalListResponse(page)
	return &resp, nil
}

// Delete removes the eval — cascades to runs and results at the DB layer.
func (s *Service) Delete(ctx context.Context, projectID, evalID string) error {
	return s.repo.DeleteEval(ctx, projectID, evalID)
}

// validateScorers checks every scorer's type and minimal config shape.
func validateScorers(scorers []entity.Scorer) error {
	seenIDs := make(map[string]bool, len(scorers))
	for i, sc := range scorers {
		if sc.ID == "" {
			return fmt.Errorf("scorers[%d].id is required", i)
		}
		if seenIDs[sc.ID] {
			return fmt.Errorf("scorers[%d].id %q is duplicated", i, sc.ID)
		}
		seenIDs[sc.ID] = true
		if len(sc.Name) > maxScorerName {
			return fmt.Errorf("scorers[%d].name exceeds %d chars", i, maxScorerName)
		}
		switch sc.Type {
		case entity.ScorerExactMatch:
			// config-less
		case entity.ScorerContains:
			if _, ok := sc.Config["value"]; !ok {
				return fmt.Errorf("scorers[%d] (%s): config.value is required", i, sc.Type)
			}
		case entity.ScorerJSONPath:
			if v, ok := sc.Config["path"].(string); !ok || v == "" {
				return fmt.Errorf("scorers[%d] (%s): config.path must be a non-empty string", i, sc.Type)
			}
			opStr, ok := sc.Config["op"].(string)
			if !ok || opStr == "" {
				return fmt.Errorf("scorers[%d] (%s): config.op is required", i, sc.Type)
			}
			if !isValidOp(entity.ScorerOp(opStr)) {
				return fmt.Errorf("scorers[%d] (%s): unsupported op %q", i, sc.Type, opStr)
			}
			if _, ok := sc.Config["value"]; !ok {
				return fmt.Errorf("scorers[%d] (%s): config.value is required", i, sc.Type)
			}
		case entity.ScorerRegex:
			if v, ok := sc.Config["pattern"].(string); !ok || v == "" {
				return fmt.Errorf("scorers[%d] (%s): config.pattern must be a non-empty string", i, sc.Type)
			}
		case entity.ScorerClientReported:
			// Config-less by design — the verdict comes from the caller.
		default:
			return fmt.Errorf("scorers[%d]: unknown type %q", i, sc.Type)
		}
	}
	return nil
}

func isValidOp(op entity.ScorerOp) bool {
	switch op {
	case entity.ScorerOpEq, entity.ScorerOpNe,
		entity.ScorerOpGt, entity.ScorerOpGte,
		entity.ScorerOpLt, entity.ScorerOpLte:
		return true
	}
	return false
}

// ============================================
// EVAL RUNS
// ============================================

// StartRun creates a new run for an eval and returns its ID. The run starts
// in "pending" status; the first posted result flips it to "running"; the
// finalize call freezes it as "completed" or "failed".
func (s *Service) StartRun(ctx context.Context, projectID string, req StartEvalRunRequest) (*EvalRunView, error) {
	if req.EvalID == "" {
		return nil, fmt.Errorf("%w: evalId is required", entity.ErrBadRequest)
	}
	// Verify the eval is in this project (404 propagates).
	if _, err := s.repo.GetEval(ctx, projectID, req.EvalID); err != nil {
		return nil, err
	}
	run := &entity.EvalRun{
		ProjectID:       projectID,
		EvalID:          req.EvalID,
		Status:          entity.EvalRunStatusPending,
		PromptVersionID: req.PromptVersionID,
		Metadata:        req.Metadata,
	}
	if err := s.repo.CreateEvalRun(ctx, run); err != nil {
		return nil, fmt.Errorf("create eval run: %w", err)
	}
	v := toEvalRunView(run)
	return &v, nil
}

// GetRun returns one run by id.
func (s *Service) GetRun(ctx context.Context, projectID, runID string) (*EvalRunView, error) {
	r, err := s.repo.GetEvalRun(ctx, projectID, runID)
	if err != nil {
		return nil, err
	}
	v := toEvalRunView(r)
	return &v, nil
}

// ListRuns returns runs for an eval (or for all evals when evalID is "").
func (s *Service) ListRuns(ctx context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*EvalRunListResponse, error) {
	page, err := s.repo.ListEvalRuns(ctx, projectID, evalID, filter)
	if err != nil {
		return nil, fmt.Errorf("list eval runs: %w", err)
	}
	resp := toEvalRunListResponse(page)
	return &resp, nil
}

// PostResult is the heart of Phase 2A. The SDK reports `actual` per dataset
// item; the server loads the dataset item's `expected` and the eval's
// scorers, runs them, persists the verdict.
//
// Validation: the run must exist, must not be terminal, the item must belong
// to the eval's dataset. Hard execution errors reported by the caller (req.Error
// set) skip scoring — the item counts as errored.
func (s *Service) PostResult(ctx context.Context, projectID, runID string, req PostEvalRunResultRequest) (*EvalRunResultView, error) {
	if req.DatasetItemID == "" {
		return nil, fmt.Errorf("%w: datasetItemId is required", entity.ErrBadRequest)
	}

	run, err := s.repo.GetEvalRun(ctx, projectID, runID)
	if err != nil {
		return nil, err
	}
	if run.Status == entity.EvalRunStatusCompleted || run.Status == entity.EvalRunStatusFailed {
		return nil, fmt.Errorf("%w: run is already finalized (%s)", entity.ErrConflict, run.Status)
	}

	ev, err := s.repo.GetEval(ctx, projectID, run.EvalID)
	if err != nil {
		return nil, err
	}

	item, err := s.datasets.GetDatasetItem(ctx, projectID, req.DatasetItemID)
	if err != nil {
		return nil, err
	}
	if item.DatasetID != ev.DatasetID {
		// Anti-leak: the item exists in the project but in a different dataset.
		// Don't reveal that — return 404, same shape as a non-existent item.
		return nil, entity.ErrNotFound
	}

	result := &entity.EvalRunResult{
		ProjectID:     projectID,
		EvalRunID:     runID,
		DatasetItemID: req.DatasetItemID,
		Actual:        req.Actual,
		DurationMs:    req.DurationMs,
		CostUSD:       req.CostUSD,
		Error:         req.Error,
	}
	if req.Error != nil && *req.Error != "" {
		// Execution failure — record without scoring.
		result.Scores = []entity.ScorerResult{}
		result.Passed = false
	} else {
		// ScoreAllWithClient honours built-in scorers server-side and lets
		// client_reported scorers take their verdict from req.ClientScores.
		scores, passed := ScoreAllWithClient(ev.Scorers, item.Expected, req.Actual, req.ClientScores)
		result.Scores = scores
		result.Passed = passed
	}

	if err := s.repo.CreateEvalRunResult(ctx, result); err != nil {
		return nil, fmt.Errorf("create eval run result: %w", err)
	}

	// First result flips the run from pending → running. Subsequent results
	// re-issue the (cheap) update; cleaner than reading-before-writing.
	if run.Status == entity.EvalRunStatusPending {
		if err := s.repo.UpdateEvalRunStatus(ctx, projectID, runID, entity.EvalRunStatusRunning); err != nil {
			// Non-fatal: the result is already persisted. Surface for observability.
			return nil, fmt.Errorf("advance run status: %w", err)
		}
	}

	v := toEvalRunResultView(result)
	return &v, nil
}

// Finalize freezes a run's aggregates from its posted results.
func (s *Service) Finalize(ctx context.Context, projectID, runID string, req FinalizeEvalRunRequest) (*EvalRunView, error) {
	if req.Status != entity.EvalRunStatusCompleted && req.Status != entity.EvalRunStatusFailed {
		return nil, fmt.Errorf("%w: status must be completed or failed", entity.ErrBadRequest)
	}
	r, err := s.repo.FinalizeEvalRun(ctx, projectID, runID, entity.FinalizeEvalRunInput{
		Status:     req.Status,
		DurationMs: req.DurationMs,
		CostUSD:    req.CostUSD,
	})
	if err != nil {
		return nil, err
	}
	v := toEvalRunView(r)
	return &v, nil
}

// ListResults returns per-item results for a run, optionally filtered.
func (s *Service) ListResults(ctx context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*EvalRunResultListResponse, error) {
	// Verify the run is in this project — better than leaking 200/empty for a
	// run that exists in another tenant.
	if _, err := s.repo.GetEvalRun(ctx, projectID, runID); err != nil {
		return nil, err
	}
	page, err := s.repo.ListEvalRunResults(ctx, projectID, runID, filter)
	if err != nil {
		return nil, fmt.Errorf("list eval run results: %w", err)
	}
	resp := toEvalRunResultListResponse(page)
	return &resp, nil
}

// IsUnsupported mirrors the dataset package — surfaces ClickHouse-as-primary
// errors so handlers can map to a clear 501.
func IsUnsupported(err error) bool {
	return errors.Is(err, entity.ErrUnsupported)
}
