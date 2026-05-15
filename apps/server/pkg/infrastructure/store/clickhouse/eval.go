package clickhouse

import (
	"context"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// EVAL OPERATIONS — UNSUPPORTED
// ============================================
//
// Like datasets, evals are relational data with frequent UPDATEs and partial
// reads — a bad fit for ClickHouse's columnar OLAP shape. These methods
// satisfy repository.Store at compile time and return entity.ErrUnsupported
// at runtime. The deployment guide documents the requirement.

func (s *Store) CreateEval(ctx context.Context, e *entity.Eval) error {
	return entity.ErrUnsupported
}

func (s *Store) GetEval(ctx context.Context, projectID, evalID string) (*entity.Eval, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) ListEvals(ctx context.Context, projectID string, filter entity.EvalFilter) (*entity.Page[entity.Eval], error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) DeleteEval(ctx context.Context, projectID, evalID string) error {
	return entity.ErrUnsupported
}

func (s *Store) CreateEvalRun(ctx context.Context, r *entity.EvalRun) error {
	return entity.ErrUnsupported
}

func (s *Store) GetEvalRun(ctx context.Context, projectID, runID string) (*entity.EvalRun, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) ListEvalRuns(ctx context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*entity.Page[entity.EvalRun], error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) UpdateEvalRunStatus(ctx context.Context, projectID, runID string, status entity.EvalRunStatus) error {
	return entity.ErrUnsupported
}

func (s *Store) FinalizeEvalRun(ctx context.Context, projectID, runID string, in entity.FinalizeEvalRunInput) (*entity.EvalRun, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) CreateEvalRunResult(ctx context.Context, res *entity.EvalRunResult) error {
	return entity.ErrUnsupported
}

func (s *Store) ListEvalRunResults(ctx context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*entity.Page[entity.EvalRunResult], error) {
	return nil, entity.ErrUnsupported
}
