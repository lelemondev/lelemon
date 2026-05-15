package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// PROMPT OPERATIONS
// ============================================

func (s *Store) CreatePrompt(ctx context.Context, p *entity.Prompt) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO prompts (id, project_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.ID, p.ProjectID, p.Name, p.Description, p.CreatedAt, p.UpdatedAt); err != nil {
		return fmt.Errorf("create prompt: %w", err)
	}
	return nil
}

func (s *Store) GetPrompt(ctx context.Context, projectID, promptID string) (*entity.Prompt, error) {
	var p entity.Prompt
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM prompts WHERE project_id = $1 AND id = $2
	`, projectID, promptID).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get prompt: %w", err)
	}
	return &p, nil
}

func (s *Store) ListPrompts(ctx context.Context, projectID string, filter entity.PromptFilter) (*entity.Page[entity.Prompt], error) {
	where := []string{"project_id = $1"}
	args := []any{projectID}
	argN := 2
	if filter.Name != nil && *filter.Name != "" {
		where = append(where, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+*filter.Name+"%")
		argN++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM prompts WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count prompts: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, project_id, name, description, created_at, updated_at
		FROM prompts WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argN, argN+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	defer rows.Close()

	out := make([]entity.Prompt, 0)
	for rows.Next() {
		var p entity.Prompt
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan prompt: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prompts: %w", err)
	}
	return &entity.Page[entity.Prompt]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) UpdatePrompt(ctx context.Context, projectID, promptID string, updates entity.PromptUpdate) error {
	sets := []string{}
	args := []any{}
	argN := 1
	if updates.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argN))
		args = append(args, *updates.Name)
		argN++
	}
	if updates.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argN))
		args = append(args, *updates.Description)
		argN++
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, fmt.Sprintf("updated_at = $%d", argN))
	args = append(args, time.Now())
	argN++
	args = append(args, projectID, promptID)

	tag, err := s.pool.Exec(ctx, fmt.Sprintf(
		`UPDATE prompts SET %s WHERE project_id = $%d AND id = $%d`,
		strings.Join(sets, ", "), argN, argN+1,
	), args...)
	if err != nil {
		return fmt.Errorf("update prompt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotFound
	}
	return nil
}

func (s *Store) DeletePrompt(ctx context.Context, projectID, promptID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM prompts WHERE project_id = $1 AND id = $2`,
		projectID, promptID,
	)
	if err != nil {
		return fmt.Errorf("delete prompt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// ============================================
// PROMPT VERSION OPERATIONS
// ============================================

// pgErrCodeUniqueViolation is Postgres SQLSTATE 23505 — the canonical
// "duplicate key" error. Using the typed code beats a fragile string match.
const pgErrCodeUniqueViolation = "23505"

func (s *Store) CreatePromptVersion(ctx context.Context, v *entity.PromptVersion) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	v.CreatedAt = time.Now()

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO prompt_versions
			(id, prompt_id, project_id, version, content, changelog, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, v.ID, v.PromptID, v.ProjectID, v.Version, v.Content, v.Changelog, v.CreatedBy, v.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrCodeUniqueViolation {
			return entity.ErrConflict
		}
		return fmt.Errorf("create prompt version: %w", err)
	}
	return nil
}

func (s *Store) GetPromptVersion(ctx context.Context, projectID, versionID string) (*entity.PromptVersion, error) {
	var v entity.PromptVersion
	err := s.pool.QueryRow(ctx, `
		SELECT id, prompt_id, project_id, version, content, changelog, created_by, created_at
		FROM prompt_versions WHERE project_id = $1 AND id = $2
	`, projectID, versionID).Scan(
		&v.ID, &v.PromptID, &v.ProjectID, &v.Version, &v.Content,
		&v.Changelog, &v.CreatedBy, &v.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get prompt version: %w", err)
	}
	return &v, nil
}

func (s *Store) ListPromptVersions(ctx context.Context, projectID, promptID string, filter entity.PromptVersionFilter) (*entity.Page[entity.PromptVersion], error) {
	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM prompt_versions WHERE project_id = $1 AND prompt_id = $2`,
		projectID, promptID,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count prompt versions: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	rows, err := s.pool.Query(ctx, `
		SELECT id, prompt_id, project_id, version, content, changelog, created_by, created_at
		FROM prompt_versions WHERE project_id = $1 AND prompt_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, projectID, promptID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list prompt versions: %w", err)
	}
	defer rows.Close()

	out := make([]entity.PromptVersion, 0)
	for rows.Next() {
		var v entity.PromptVersion
		if err := rows.Scan(
			&v.ID, &v.PromptID, &v.ProjectID, &v.Version, &v.Content,
			&v.Changelog, &v.CreatedBy, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan prompt version: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prompt versions: %w", err)
	}
	return &entity.Page[entity.PromptVersion]{Data: out, Total: total, Limit: limit, Offset: offset}, nil
}
