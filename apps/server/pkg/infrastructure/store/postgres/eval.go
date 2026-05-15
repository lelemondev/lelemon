package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// EVAL OPERATIONS
// ============================================

func (s *Store) CreateEval(ctx context.Context, e *entity.Eval) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now

	scorersJSON, err := json.Marshal(e.Scorers)
	if err != nil {
		return fmt.Errorf("marshal scorers: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO evals (id, project_id, dataset_id, name, description, scorers, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, e.ID, e.ProjectID, e.DatasetID, e.Name, e.Description, scorersJSON, e.CreatedAt, e.UpdatedAt); err != nil {
		return fmt.Errorf("create eval: %w", err)
	}
	return nil
}

func (s *Store) GetEval(ctx context.Context, projectID, evalID string) (*entity.Eval, error) {
	var (
		e           entity.Eval
		scorersJSON []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, dataset_id, name, description, scorers, created_at, updated_at
		FROM evals WHERE project_id = $1 AND id = $2
	`, projectID, evalID).Scan(
		&e.ID, &e.ProjectID, &e.DatasetID, &e.Name, &e.Description,
		&scorersJSON, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get eval: %w", err)
	}
	if err := json.Unmarshal(scorersJSON, &e.Scorers); err != nil {
		return nil, fmt.Errorf("unmarshal scorers: %w", err)
	}
	return &e, nil
}

func (s *Store) ListEvals(ctx context.Context, projectID string, filter entity.EvalFilter) (*entity.Page[entity.Eval], error) {
	where := []string{"project_id = $1"}
	args := []any{projectID}
	argN := 2
	if filter.DatasetID != nil && *filter.DatasetID != "" {
		where = append(where, fmt.Sprintf("dataset_id = $%d", argN))
		args = append(args, *filter.DatasetID)
		argN++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM evals WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count evals: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, project_id, dataset_id, name, description, scorers, created_at, updated_at
		FROM evals WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argN, argN+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list evals: %w", err)
	}
	defer rows.Close()

	out := make([]entity.Eval, 0)
	for rows.Next() {
		var (
			e           entity.Eval
			scorersJSON []byte
		)
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.DatasetID, &e.Name, &e.Description,
			&scorersJSON, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval: %w", err)
		}
		if err := json.Unmarshal(scorersJSON, &e.Scorers); err != nil {
			return nil, fmt.Errorf("unmarshal scorers: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evals: %w", err)
	}
	return &entity.Page[entity.Eval]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) DeleteEval(ctx context.Context, projectID, evalID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM evals WHERE project_id = $1 AND id = $2`,
		projectID, evalID,
	)
	if err != nil {
		return fmt.Errorf("delete eval: %w", err)
	}
	if tag.RowsAffected() == 0 {
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
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO eval_runs
			(id, project_id, eval_id, status, prompt_version_id, metadata,
			 total_items, passed_items, failed_items, errored_items,
			 duration_ms, cost_usd, started_at, completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, 0, 0, 0, NULL, NULL, $7, NULL, $8, $9)
	`, r.ID, r.ProjectID, r.EvalID, string(r.Status), r.PromptVersionID, metadataJSON,
		r.StartedAt, r.CreatedAt, r.UpdatedAt); err != nil {
		return fmt.Errorf("create eval run: %w", err)
	}
	return nil
}

func (s *Store) GetEvalRun(ctx context.Context, projectID, runID string) (*entity.EvalRun, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, project_id, eval_id, status, prompt_version_id, metadata,
		       total_items, passed_items, failed_items, errored_items,
		       duration_ms, cost_usd, started_at, completed_at, created_at, updated_at
		FROM eval_runs WHERE project_id = $1 AND id = $2
	`, projectID, runID)
	r, err := scanEvalRunPg(row)
	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get eval run: %w", err)
	}
	return r, nil
}

func (s *Store) ListEvalRuns(ctx context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*entity.Page[entity.EvalRun], error) {
	where := []string{"project_id = $1"}
	args := []any{projectID}
	argN := 2
	if evalID != "" {
		where = append(where, fmt.Sprintf("eval_id = $%d", argN))
		args = append(args, evalID)
		argN++
	}
	if filter.EvalID != nil && *filter.EvalID != "" && evalID == "" {
		where = append(where, fmt.Sprintf("eval_id = $%d", argN))
		args = append(args, *filter.EvalID)
		argN++
	}
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", argN))
		args = append(args, string(*filter.Status))
		argN++
	}
	if filter.PromptVersionID != nil && *filter.PromptVersionID != "" {
		where = append(where, fmt.Sprintf("prompt_version_id = $%d", argN))
		args = append(args, *filter.PromptVersionID)
		argN++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM eval_runs WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count eval runs: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, project_id, eval_id, status, prompt_version_id, metadata,
		       total_items, passed_items, failed_items, errored_items,
		       duration_ms, cost_usd, started_at, completed_at, created_at, updated_at
		FROM eval_runs WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argN, argN+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list eval runs: %w", err)
	}
	defer rows.Close()

	out := make([]entity.EvalRun, 0)
	for rows.Next() {
		r, err := scanEvalRunPg(rows)
		if err != nil {
			return nil, fmt.Errorf("scan eval run row: %w", err)
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval runs: %w", err)
	}
	return &entity.Page[entity.EvalRun]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) UpdateEvalRunStatus(ctx context.Context, projectID, runID string, status entity.EvalRunStatus) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE eval_runs SET status = $1, updated_at = $2 WHERE project_id = $3 AND id = $4`,
		string(status), time.Now(), projectID, runID,
	)
	if err != nil {
		return fmt.Errorf("update eval run status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotFound
	}
	return nil
}

func (s *Store) FinalizeEvalRun(ctx context.Context, projectID, runID string, in entity.FinalizeEvalRunInput) (*entity.EvalRun, error) {
	if in.Status != entity.EvalRunStatusCompleted && in.Status != entity.EvalRunStatusFailed {
		return nil, fmt.Errorf("finalize requires completed or failed status, got %q", in.Status)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin finalize tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	if err := tx.QueryRow(ctx,
		`SELECT status FROM eval_runs WHERE project_id = $1 AND id = $2 FOR UPDATE`,
		projectID, runID,
	).Scan(&currentStatus); err != nil {
		if err == pgx.ErrNoRows {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("read current status: %w", err)
	}
	if currentStatus == string(entity.EvalRunStatusCompleted) || currentStatus == string(entity.EvalRunStatusFailed) {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit no-op finalize: %w", err)
		}
		return s.GetEvalRun(ctx, projectID, runID)
	}

	var (
		total, passed, errored int
		actualDuration         *int
		actualCost             *float64
	)
	err = tx.QueryRow(ctx, `
		SELECT
		  COUNT(*),
		  COALESCE(SUM(CASE WHEN passed THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END), 0),
		  SUM(duration_ms),
		  SUM(cost_usd)
		FROM eval_run_results WHERE project_id = $1 AND eval_run_id = $2
	`, projectID, runID).Scan(&total, &passed, &errored, &actualDuration, &actualCost)
	if err != nil {
		return nil, fmt.Errorf("aggregate results: %w", err)
	}

	duration := in.DurationMs
	if duration == nil {
		duration = actualDuration
	}
	cost := in.CostUSD
	if cost == nil {
		cost = actualCost
	}

	// Defensive clamp — see sqlite/eval.go for the rationale.
	failed := max(total-passed-errored, 0)
	now := time.Now()
	if _, err := tx.Exec(ctx, `
		UPDATE eval_runs
		SET status = $1, total_items = $2, passed_items = $3, failed_items = $4, errored_items = $5,
		    duration_ms = $6, cost_usd = $7, completed_at = $8, updated_at = $9
		WHERE project_id = $10 AND id = $11
	`, string(in.Status), total, passed, failed, errored,
		duration, cost, now, now, projectID, runID); err != nil {
		return nil, fmt.Errorf("update eval run finalize: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
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

	actualJSON, err := marshalOptionalJSON(res.Actual)
	if err != nil {
		return fmt.Errorf("marshal actual: %w", err)
	}
	scoresJSON, err := json.Marshal(res.Scores)
	if err != nil {
		return fmt.Errorf("marshal scores: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO eval_run_results
			(id, project_id, eval_run_id, dataset_item_id, actual, scores, passed,
			 duration_ms, cost_usd, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, res.ID, res.ProjectID, res.EvalRunID, res.DatasetItemID,
		actualJSON, scoresJSON, res.Passed,
		res.DurationMs, res.CostUSD, res.Error, res.CreatedAt); err != nil {
		return fmt.Errorf("create eval run result: %w", err)
	}
	return nil
}

func (s *Store) ListEvalRunResults(ctx context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*entity.Page[entity.EvalRunResult], error) {
	where := []string{"project_id = $1", "eval_run_id = $2"}
	args := []any{projectID, runID}
	argN := 3
	if filter.PassedOnly != nil {
		where = append(where, fmt.Sprintf("passed = $%d", argN))
		args = append(args, *filter.PassedOnly)
		argN++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM eval_run_results WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count eval run results: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, project_id, eval_run_id, dataset_item_id, actual, scores, passed,
		       duration_ms, cost_usd, error, created_at
		FROM eval_run_results WHERE %s
		ORDER BY created_at ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argN, argN+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list eval run results: %w", err)
	}
	defer rows.Close()

	out := make([]entity.EvalRunResult, 0)
	for rows.Next() {
		var (
			r          entity.EvalRunResult
			actualJSON []byte
			scoresJSON []byte
		)
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.EvalRunID, &r.DatasetItemID,
			&actualJSON, &scoresJSON, &r.Passed,
			&r.DurationMs, &r.CostUSD, &r.Error, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval run result: %w", err)
		}
		if len(actualJSON) > 0 {
			if err := json.Unmarshal(actualJSON, &r.Actual); err != nil {
				return nil, fmt.Errorf("unmarshal actual: %w", err)
			}
		}
		if len(scoresJSON) > 0 {
			if err := json.Unmarshal(scoresJSON, &r.Scores); err != nil {
				return nil, fmt.Errorf("unmarshal scores: %w", err)
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eval run results: %w", err)
	}
	return &entity.Page[entity.EvalRunResult]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

// ----- helpers (eval-specific) --------------------------------------------

// pgxRowScanner abstracts over pgx.Row and pgx.Rows.
type pgxRowScanner interface {
	Scan(dest ...any) error
}

func scanEvalRunPg(s pgxRowScanner) (*entity.EvalRun, error) {
	var (
		r            entity.EvalRun
		statusStr    string
		metadataJSON []byte
		completedAt  *time.Time
	)
	if err := s.Scan(
		&r.ID, &r.ProjectID, &r.EvalID, &statusStr, &r.PromptVersionID, &metadataJSON,
		&r.TotalItems, &r.PassedItems, &r.FailedItems, &r.ErroredItems,
		&r.DurationMs, &r.CostUSD, &r.StartedAt, &completedAt, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return nil, err
	}
	r.Status = entity.EvalRunStatus(statusStr)
	r.CompletedAt = completedAt
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &r.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal run metadata: %w", err)
		}
	}
	return &r, nil
}
