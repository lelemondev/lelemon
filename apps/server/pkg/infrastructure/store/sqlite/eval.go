package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// EVAL OPERATIONS
// ============================================
//
// Same multi-tenant invariant as datasets: project_id is the outer filter on
// every read and the source of truth on every write. Eval data is relational
// and lives only in the primary store.

func (s *Store) CreateEval(ctx context.Context, e *entity.Eval) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	scorersJSON, err := marshalScorers(e.Scorers)
	if err != nil {
		return fmt.Errorf("marshal scorers: %w", err)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO evals (id, project_id, dataset_id, name, description, scorers, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.ProjectID, e.DatasetID, e.Name, e.Description, scorersJSON, e.CreatedAt, e.UpdatedAt); err != nil {
		return fmt.Errorf("create eval: %w", err)
	}
	return nil
}

func (s *Store) GetEval(ctx context.Context, projectID, evalID string) (*entity.Eval, error) {
	var (
		e           entity.Eval
		scorersJSON string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, dataset_id, name, description, scorers, created_at, updated_at
		FROM evals WHERE project_id = ? AND id = ?
	`, projectID, evalID).Scan(
		&e.ID, &e.ProjectID, &e.DatasetID, &e.Name, &e.Description,
		&scorersJSON, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get eval: %w", err)
	}
	e.Scorers, err = unmarshalScorers(scorersJSON)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) ListEvals(ctx context.Context, projectID string, filter entity.EvalFilter) (*entity.Page[entity.Eval], error) {
	where := []string{"project_id = ?"}
	args := []any{projectID}
	if filter.DatasetID != nil && *filter.DatasetID != "" {
		where = append(where, "dataset_id = ?")
		args = append(args, *filter.DatasetID)
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM evals WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count evals: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, dataset_id, name, description, scorers, created_at, updated_at
		FROM evals WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list evals: %w", err)
	}
	defer rows.Close()

	out := make([]entity.Eval, 0)
	for rows.Next() {
		var (
			e           entity.Eval
			scorersJSON string
		)
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.DatasetID, &e.Name, &e.Description,
			&scorersJSON, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval: %w", err)
		}
		e.Scorers, err = unmarshalScorers(scorersJSON)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evals: %w", err)
	}

	return &entity.Page[entity.Eval]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) DeleteEval(ctx context.Context, projectID, evalID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM evals WHERE project_id = ? AND id = ?`,
		projectID, evalID,
	)
	if err != nil {
		return fmt.Errorf("delete eval: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete eval rows: %w", err)
	}
	if affected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// ============================================
// EVAL RUN OPERATIONS
// ============================================

func (s *Store) CreateEvalRun(ctx context.Context, r *entity.EvalRun) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	now := time.Now()
	r.StartedAt = now
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Status == "" {
		r.Status = entity.EvalRunStatusPending
	}

	metadataJSON, err := marshalMetadata(r.Metadata)
	if err != nil {
		return fmt.Errorf("marshal run metadata: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO eval_runs
			(id, project_id, eval_id, status, prompt_version_id, metadata,
			 total_items, passed_items, failed_items, errored_items,
			 duration_ms, cost_usd, started_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0, 0, NULL, NULL, ?, NULL, ?, ?)
	`, r.ID, r.ProjectID, r.EvalID, string(r.Status), r.PromptVersionID, metadataJSON,
		r.StartedAt, r.CreatedAt, r.UpdatedAt); err != nil {
		return fmt.Errorf("create eval run: %w", err)
	}
	return nil
}

func (s *Store) GetEvalRun(ctx context.Context, projectID, runID string) (*entity.EvalRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, eval_id, status, prompt_version_id, metadata,
		       total_items, passed_items, failed_items, errored_items,
		       duration_ms, cost_usd, started_at, completed_at, created_at, updated_at
		FROM eval_runs WHERE project_id = ? AND id = ?
	`, projectID, runID)
	return scanEvalRun(row)
}

func (s *Store) ListEvalRuns(ctx context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*entity.Page[entity.EvalRun], error) {
	where := []string{"project_id = ?"}
	args := []any{projectID}
	if evalID != "" {
		where = append(where, "eval_id = ?")
		args = append(args, evalID)
	}
	if filter.EvalID != nil && *filter.EvalID != "" && evalID == "" {
		where = append(where, "eval_id = ?")
		args = append(args, *filter.EvalID)
	}
	if filter.Status != nil {
		where = append(where, "status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.PromptVersionID != nil && *filter.PromptVersionID != "" {
		where = append(where, "prompt_version_id = ?")
		args = append(args, *filter.PromptVersionID)
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM eval_runs WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count eval runs: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, eval_id, status, prompt_version_id, metadata,
		       total_items, passed_items, failed_items, errored_items,
		       duration_ms, cost_usd, started_at, completed_at, created_at, updated_at
		FROM eval_runs WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list eval runs: %w", err)
	}
	defer rows.Close()

	out := make([]entity.EvalRun, 0)
	for rows.Next() {
		r, err := scanEvalRunRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval runs: %w", err)
	}

	return &entity.Page[entity.EvalRun]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) UpdateEvalRunStatus(ctx context.Context, projectID, runID string, status entity.EvalRunStatus) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE eval_runs SET status = ?, updated_at = ? WHERE project_id = ? AND id = ?`,
		string(status), time.Now(), projectID, runID,
	)
	if err != nil {
		return fmt.Errorf("update eval run status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update eval run status rows: %w", err)
	}
	if affected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// FinalizeEvalRun computes aggregates from posted results and freezes them
// transactionally. Idempotent: re-finalizing returns the current state.
func (s *Store) FinalizeEvalRun(ctx context.Context, projectID, runID string, in entity.FinalizeEvalRunInput) (*entity.EvalRun, error) {
	if in.Status != entity.EvalRunStatusCompleted && in.Status != entity.EvalRunStatusFailed {
		return nil, fmt.Errorf("finalize requires completed or failed status, got %q", in.Status)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin finalize tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Check current state, idempotent on already-terminal.
	var currentStatus string
	if err := tx.QueryRowContext(ctx,
		`SELECT status FROM eval_runs WHERE project_id = ? AND id = ?`,
		projectID, runID,
	).Scan(&currentStatus); err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("read current status: %w", err)
	}
	if currentStatus == string(entity.EvalRunStatusCompleted) || currentStatus == string(entity.EvalRunStatusFailed) {
		// Already finalized — just return whatever we have.
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit no-op finalize: %w", err)
		}
		return s.GetEvalRun(ctx, projectID, runID)
	}

	// Aggregate from results.
	var (
		total, passed, errored int
		actualDuration         sql.NullInt64
		actualCost             sql.NullFloat64
	)
	err = tx.QueryRowContext(ctx, `
		SELECT
		  COUNT(*),
		  COALESCE(SUM(CASE WHEN passed = 1 THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END), 0),
		  SUM(duration_ms),
		  SUM(cost_usd)
		FROM eval_run_results WHERE project_id = ? AND eval_run_id = ?
	`, projectID, runID).Scan(&total, &passed, &errored, &actualDuration, &actualCost)
	if err != nil {
		return nil, fmt.Errorf("aggregate results: %w", err)
	}

	// Prefer caller-supplied totals when present, fall back to row-sum.
	var (
		duration *int
		cost     *float64
	)
	switch {
	case in.DurationMs != nil:
		duration = in.DurationMs
	case actualDuration.Valid:
		v := int(actualDuration.Int64)
		duration = &v
	}
	switch {
	case in.CostUSD != nil:
		cost = in.CostUSD
	case actualCost.Valid:
		v := actualCost.Float64
		cost = &v
	}

	now := time.Now()
	// Clamp defensively — passed + errored should never exceed total, but a
	// future migration that backfills one column without the other could trip
	// this. max() keeps the row sane.
	failed := max(total-passed-errored, 0)
	if _, err := tx.ExecContext(ctx, `
		UPDATE eval_runs
		SET status = ?, total_items = ?, passed_items = ?, failed_items = ?, errored_items = ?,
		    duration_ms = ?, cost_usd = ?, completed_at = ?, updated_at = ?
		WHERE project_id = ? AND id = ?
	`, string(in.Status), total, passed, failed, errored,
		duration, cost, now, now, projectID, runID); err != nil {
		return nil, fmt.Errorf("update eval run finalize: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit finalize: %w", err)
	}
	return s.GetEvalRun(ctx, projectID, runID)
}

// ============================================
// EVAL RUN RESULT OPERATIONS
// ============================================

func (s *Store) CreateEvalRunResult(ctx context.Context, res *entity.EvalRunResult) error {
	if res.ID == "" {
		res.ID = uuid.New().String()
	}
	res.CreatedAt = time.Now()

	actualJSON, err := marshalJSONOptional(res.Actual)
	if err != nil {
		return fmt.Errorf("marshal actual: %w", err)
	}
	scoresJSON, err := marshalScoresJSON(res.Scores)
	if err != nil {
		return fmt.Errorf("marshal scores: %w", err)
	}

	passedInt := 0
	if res.Passed {
		passedInt = 1
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO eval_run_results
			(id, project_id, eval_run_id, dataset_item_id, actual, scores, passed,
			 duration_ms, cost_usd, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, res.ID, res.ProjectID, res.EvalRunID, res.DatasetItemID,
		actualJSON, scoresJSON, passedInt,
		res.DurationMs, res.CostUSD, res.Error, res.CreatedAt); err != nil {
		return fmt.Errorf("create eval run result: %w", err)
	}
	return nil
}

func (s *Store) ListEvalRunResults(ctx context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*entity.Page[entity.EvalRunResult], error) {
	where := []string{"project_id = ?", "eval_run_id = ?"}
	args := []any{projectID, runID}
	if filter.PassedOnly != nil {
		if *filter.PassedOnly {
			where = append(where, "passed = 1")
		} else {
			where = append(where, "passed = 0")
		}
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM eval_run_results WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count eval run results: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, eval_run_id, dataset_item_id, actual, scores, passed,
		       duration_ms, cost_usd, error, created_at
		FROM eval_run_results WHERE `+whereClause+`
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list eval run results: %w", err)
	}
	defer rows.Close()

	out := make([]entity.EvalRunResult, 0)
	for rows.Next() {
		var (
			r             entity.EvalRunResult
			actualJSON    sql.NullString
			scoresJSON    string
			passedInt     int
		)
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.EvalRunID, &r.DatasetItemID,
			&actualJSON, &scoresJSON, &passedInt,
			&r.DurationMs, &r.CostUSD, &r.Error, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval run result: %w", err)
		}
		if actualJSON.Valid && actualJSON.String != "" {
			if err := json.Unmarshal([]byte(actualJSON.String), &r.Actual); err != nil {
				return nil, fmt.Errorf("unmarshal actual: %w", err)
			}
		}
		r.Scores, err = unmarshalScoresJSON(scoresJSON)
		if err != nil {
			return nil, err
		}
		r.Passed = passedInt == 1
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval run results: %w", err)
	}

	return &entity.Page[entity.EvalRunResult]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

// ----- helpers (eval-specific) --------------------------------------------

func marshalScorers(scorers []entity.Scorer) (string, error) {
	if scorers == nil {
		return "[]", nil
	}
	b, err := json.Marshal(scorers)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalScorers(raw string) ([]entity.Scorer, error) {
	if raw == "" {
		return []entity.Scorer{}, nil
	}
	out := []entity.Scorer{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("unmarshal scorers: %w", err)
	}
	return out, nil
}

func marshalScoresJSON(scores []entity.ScorerResult) (string, error) {
	if scores == nil {
		return "[]", nil
	}
	b, err := json.Marshal(scores)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalScoresJSON(raw string) ([]entity.ScorerResult, error) {
	if raw == "" {
		return []entity.ScorerResult{}, nil
	}
	out := []entity.ScorerResult{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("unmarshal scores: %w", err)
	}
	return out, nil
}

// evalRunScanner abstracts over *sql.Row (single) and *sql.Rows (iterating).
type evalRunScanner interface {
	Scan(dest ...any) error
}

func scanEvalRunCommon(s evalRunScanner) (*entity.EvalRun, error) {
	var (
		r            entity.EvalRun
		statusStr    string
		metadataJSON string
		completedAt  sql.NullTime
	)
	err := s.Scan(
		&r.ID, &r.ProjectID, &r.EvalID, &statusStr, &r.PromptVersionID, &metadataJSON,
		&r.TotalItems, &r.PassedItems, &r.FailedItems, &r.ErroredItems,
		&r.DurationMs, &r.CostUSD, &r.StartedAt, &completedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.Status = entity.EvalRunStatus(statusStr)
	if completedAt.Valid {
		t := completedAt.Time
		r.CompletedAt = &t
	}
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &r.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal run metadata: %w", err)
		}
	}
	return &r, nil
}

func scanEvalRun(row *sql.Row) (*entity.EvalRun, error) {
	r, err := scanEvalRunCommon(row)
	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan eval run: %w", err)
	}
	return r, nil
}

func scanEvalRunRows(rows *sql.Rows) (*entity.EvalRun, error) {
	r, err := scanEvalRunCommon(rows)
	if err != nil {
		return nil, fmt.Errorf("scan eval run row: %w", err)
	}
	return r, nil
}
