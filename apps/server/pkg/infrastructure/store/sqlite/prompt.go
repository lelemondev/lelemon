package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// PROMPT OPERATIONS
// ============================================
//
// Prompts and their versions are relational, project-scoped, append-only on
// the version side. UNIQUE(prompt_id, version) is enforced at the DB layer;
// we map the resulting constraint violation to entity.ErrConflict.

func (s *Store) CreatePrompt(ctx context.Context, p *entity.Prompt) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO prompts (id, project_id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, p.ID, p.ProjectID, p.Name, p.Description, p.CreatedAt, p.UpdatedAt); err != nil {
		return fmt.Errorf("create prompt: %w", err)
	}
	return nil
}

func (s *Store) GetPrompt(ctx context.Context, projectID, promptID string) (*entity.Prompt, error) {
	var p entity.Prompt
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM prompts WHERE project_id = ? AND id = ?
	`, projectID, promptID).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get prompt: %w", err)
	}
	return &p, nil
}

func (s *Store) ListPrompts(ctx context.Context, projectID string, filter entity.PromptFilter) (*entity.Page[entity.Prompt], error) {
	where := []string{"project_id = ?"}
	args := []any{projectID}
	if filter.Name != nil && *filter.Name != "" {
		where = append(where, "name LIKE ?")
		args = append(args, "%"+*filter.Name+"%")
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM prompts WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count prompts: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM prompts WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, args...)
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
	if updates.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *updates.Description)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, projectID, promptID)

	res, err := s.db.ExecContext(ctx, `
		UPDATE prompts SET `+strings.Join(sets, ", ")+`
		WHERE project_id = ? AND id = ?
	`, args...)
	if err != nil {
		return fmt.Errorf("update prompt: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update prompt rows: %w", err)
	}
	if affected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

func (s *Store) DeletePrompt(ctx context.Context, projectID, promptID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM prompts WHERE project_id = ? AND id = ?`,
		projectID, promptID,
	)
	if err != nil {
		return fmt.Errorf("delete prompt: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete prompt rows: %w", err)
	}
	if affected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// ============================================
// PROMPT VERSION OPERATIONS
// ============================================

func (s *Store) CreatePromptVersion(ctx context.Context, v *entity.PromptVersion) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	v.CreatedAt = time.Now()

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO prompt_versions
			(id, prompt_id, project_id, version, content, changelog, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, v.ID, v.PromptID, v.ProjectID, v.Version, v.Content, v.Changelog, v.CreatedBy, v.CreatedAt); err != nil {
		// modernc.org/sqlite reports UNIQUE violations as
		// "constraint failed: UNIQUE constraint failed: ...". Map to ErrConflict
		// so handlers can return 409 instead of leaking a generic 500.
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return entity.ErrConflict
		}
		return fmt.Errorf("create prompt version: %w", err)
	}
	return nil
}

func (s *Store) GetPromptVersion(ctx context.Context, projectID, versionID string) (*entity.PromptVersion, error) {
	var v entity.PromptVersion
	err := s.db.QueryRowContext(ctx, `
		SELECT id, prompt_id, project_id, version, content, changelog, created_by, created_at
		FROM prompt_versions WHERE project_id = ? AND id = ?
	`, projectID, versionID).Scan(
		&v.ID, &v.PromptID, &v.ProjectID, &v.Version, &v.Content,
		&v.Changelog, &v.CreatedBy, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get prompt version: %w", err)
	}
	return &v, nil
}

func (s *Store) ListPromptVersions(ctx context.Context, projectID, promptID string, filter entity.PromptVersionFilter) (*entity.Page[entity.PromptVersion], error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM prompt_versions WHERE project_id = ? AND prompt_id = ?`,
		projectID, promptID,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count prompt versions: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, prompt_id, project_id, version, content, changelog, created_by, created_at
		FROM prompt_versions WHERE project_id = ? AND prompt_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
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
