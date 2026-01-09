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
	_ "modernc.org/sqlite"
)

// Store implements repository.Store for SQLite
type Store struct {
	db *sql.DB
}

// New creates a new SQLite store
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)

	return &Store{db: db}, nil
}

// Migrate runs database migrations
func (s *Store) Migrate(ctx context.Context) error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			password_hash TEXT,
			google_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Projects table
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			api_key TEXT UNIQUE NOT NULL,
			api_key_hash TEXT NOT NULL,
			owner_email TEXT NOT NULL,
			settings TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Traces table
		`CREATE TABLE IF NOT EXISTS traces (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			session_id TEXT,
			user_id TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			tags TEXT DEFAULT '[]',
			metadata TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Spans table
		`CREATE TABLE IF NOT EXISTS spans (
			id TEXT PRIMARY KEY,
			trace_id TEXT NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
			parent_span_id TEXT,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			input TEXT,
			output TEXT,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cost_usd REAL,
			duration_ms INTEGER,
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT,
			model TEXT,
			provider TEXT,
			metadata TEXT DEFAULT '{}',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME,
			stop_reason TEXT,
			cache_read_tokens INTEGER,
			cache_write_tokens INTEGER,
			reasoning_tokens INTEGER,
			first_token_ms INTEGER,
			thinking TEXT
		)`,

		// Phase 7.1: Add extended fields to existing spans table
		`ALTER TABLE spans ADD COLUMN stop_reason TEXT`,
		`ALTER TABLE spans ADD COLUMN cache_read_tokens INTEGER`,
		`ALTER TABLE spans ADD COLUMN cache_write_tokens INTEGER`,
		`ALTER TABLE spans ADD COLUMN reasoning_tokens INTEGER`,
		`ALTER TABLE spans ADD COLUMN first_token_ms INTEGER`,
		`ALTER TABLE spans ADD COLUMN thinking TEXT`,

		// Phase 7.2: Pre-computed fields (ingest-time optimization)
		`ALTER TABLE spans ADD COLUMN sub_type TEXT`,
		`ALTER TABLE spans ADD COLUMN tool_uses TEXT`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_projects_api_key_hash ON projects(api_key_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_email)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_project_created ON traces(project_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(project_id, session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_user ON traces(project_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_trace ON spans(trace_id, started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)`,
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			// Ignore "duplicate column name" errors for ALTER TABLE (idempotent migrations)
			if strings.Contains(m, "ALTER TABLE") && strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// Ping checks the database connection
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// ============================================
// USER OPERATIONS
// ============================================

func (s *Store) CreateUser(ctx context.Context, u *entity.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, name, password_hash, google_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, u.ID, u.Email, u.Name, u.PasswordHash, u.GoogleID, u.CreatedAt, u.UpdatedAt)

	return err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	var u entity.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, google_id, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	err := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, password_hash, google_id, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, updates entity.UserUpdate) error {
	var sets []string
	var args []any

	if updates.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.PasswordHash != nil {
		sets = append(sets, "password_hash = ?")
		args = append(args, *updates.PasswordHash)
	}
	if updates.GoogleID != nil {
		sets = append(sets, "google_id = ?")
		args = append(args, *updates.GoogleID)
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ============================================
// PROJECT OPERATIONS
// ============================================

func (s *Store) CreateProject(ctx context.Context, p *entity.Project) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	settingsJSON, _ := json.Marshal(p.Settings)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.APIKey, p.APIKeyHash, p.OwnerEmail, string(settingsJSON), p.CreatedAt, p.UpdatedAt)

	return err
}

func (s *Store) GetProjectByID(ctx context.Context, id string) (*entity.Project, error) {
	var p entity.Project
	var settingsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(settingsJSON), &p.Settings)
	return &p, nil
}

func (s *Store) GetProjectByAPIKeyHash(ctx context.Context, hash string) (*entity.Project, error) {
	var p entity.Project
	var settingsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects WHERE api_key_hash = ?
	`, hash).Scan(&p.ID, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(settingsJSON), &p.Settings)
	return &p, nil
}

func (s *Store) UpdateProject(ctx context.Context, id string, updates entity.ProjectUpdate) error {
	var sets []string
	var args []any

	if updates.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.Settings != nil {
		settingsJSON, _ := json.Marshal(updates.Settings)
		sets = append(sets, "settings = ?")
		args = append(args, string(settingsJSON))
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE id = ?", id)
	return err
}

func (s *Store) ListProjectsByOwner(ctx context.Context, email string) ([]entity.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects WHERE owner_email = ? ORDER BY created_at DESC
	`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []entity.Project
	for rows.Next() {
		var p entity.Project
		var settingsJSON string
		if err := rows.Scan(&p.ID, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(settingsJSON), &p.Settings)
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) RotateAPIKey(ctx context.Context, id string, newKey, newHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects SET api_key = ?, api_key_hash = ?, updated_at = ? WHERE id = ?
	`, newKey, newHash, time.Now(), id)
	return err
}

// ============================================
// TRACE OPERATIONS
// ============================================

func (s *Store) CreateTrace(ctx context.Context, t *entity.Trace) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	tagsJSON, _ := json.Marshal(t.Tags)
	metadataJSON, _ := json.Marshal(t.Metadata)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO traces (id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ProjectID, t.SessionID, t.UserID, t.Status, string(tagsJSON), string(metadataJSON), t.CreatedAt, t.UpdatedAt)

	return err
}

func (s *Store) UpdateTrace(ctx context.Context, projectID, traceID string, updates entity.TraceUpdate) error {
	var sets []string
	var args []any

	if updates.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, string(*updates.Status))
	}
	if updates.Metadata != nil {
		metadataJSON, _ := json.Marshal(updates.Metadata)
		sets = append(sets, "metadata = ?")
		args = append(args, string(metadataJSON))
	}
	if updates.Tags != nil {
		tagsJSON, _ := json.Marshal(updates.Tags)
		sets = append(sets, "tags = ?")
		args = append(args, string(tagsJSON))
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, projectID, traceID)

	query := fmt.Sprintf("UPDATE traces SET %s WHERE project_id = ? AND id = ?", strings.Join(sets, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) UpdateTraceStatus(ctx context.Context, projectID, traceID string, status entity.TraceStatus) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE traces SET status = ?, updated_at = ? WHERE project_id = ? AND id = ?
	`, string(status), time.Now(), projectID, traceID)
	return err
}

func (s *Store) DeleteAllTraces(ctx context.Context, projectID string) (int64, error) {
	// Spans are deleted via CASCADE when traces are deleted
	result, err := s.db.ExecContext(ctx, `DELETE FROM traces WHERE project_id = ?`, projectID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) GetTrace(ctx context.Context, projectID, traceID string) (*entity.TraceWithSpans, error) {
	// Get trace
	var t entity.Trace
	var tagsJSON, metadataJSON string
	var sessionID, userID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at
		FROM traces WHERE project_id = ? AND id = ?
	`, projectID, traceID).Scan(&t.ID, &t.ProjectID, &sessionID, &userID, &t.Status, &tagsJSON, &metadataJSON, &t.CreatedAt, &t.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if sessionID.Valid {
		t.SessionID = &sessionID.String
	}
	if userID.Valid {
		t.UserID = &userID.String
	}
	json.Unmarshal([]byte(tagsJSON), &t.Tags)
	json.Unmarshal([]byte(metadataJSON), &t.Metadata)

	// Get spans
	spans, err := s.getSpansForTrace(ctx, traceID)
	if err != nil {
		return nil, err
	}

	// Calculate metrics
	result := &entity.TraceWithSpans{Trace: t, Spans: spans}
	for _, span := range spans {
		result.TotalSpans++
		if span.InputTokens != nil {
			result.TotalTokens += *span.InputTokens
		}
		if span.OutputTokens != nil {
			result.TotalTokens += *span.OutputTokens
		}
		if span.CostUSD != nil {
			result.TotalCostUSD += *span.CostUSD
		}
		if span.DurationMs != nil {
			result.TotalDurationMs += *span.DurationMs
		}
	}

	return result, nil
}

func (s *Store) getSpansForTrace(ctx context.Context, traceID string) ([]entity.Span, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trace_id, parent_span_id, type, name, input, output,
		       input_tokens, output_tokens, cost_usd, duration_ms, status,
		       error_message, model, provider, metadata, started_at, ended_at,
		       stop_reason, cache_read_tokens, cache_write_tokens,
		       reasoning_tokens, first_token_ms, thinking,
		       sub_type, tool_uses
		FROM spans WHERE trace_id = ? ORDER BY started_at
	`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []entity.Span
	for rows.Next() {
		var sp entity.Span
		var parentSpanID, inputJSON, outputJSON, errorMsg, model, provider sql.NullString
		var stopReason, thinking sql.NullString
		var subType, toolUsesJSON sql.NullString
		var metadataJSON string
		var inputTokens, outputTokens, durationMs sql.NullInt64
		var cacheReadTokens, cacheWriteTokens, reasoningTokens, firstTokenMs sql.NullInt64
		var costUSD sql.NullFloat64
		var endedAt sql.NullTime

		err := rows.Scan(&sp.ID, &sp.TraceID, &parentSpanID, &sp.Type, &sp.Name,
			&inputJSON, &outputJSON, &inputTokens, &outputTokens, &costUSD,
			&durationMs, &sp.Status, &errorMsg, &model, &provider, &metadataJSON,
			&sp.StartedAt, &endedAt,
			&stopReason, &cacheReadTokens, &cacheWriteTokens,
			&reasoningTokens, &firstTokenMs, &thinking,
			&subType, &toolUsesJSON)
		if err != nil {
			return nil, err
		}

		if parentSpanID.Valid {
			sp.ParentSpanID = &parentSpanID.String
		}
		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &sp.Input)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &sp.Output)
		}
		if inputTokens.Valid {
			v := int(inputTokens.Int64)
			sp.InputTokens = &v
		}
		if outputTokens.Valid {
			v := int(outputTokens.Int64)
			sp.OutputTokens = &v
		}
		if costUSD.Valid {
			sp.CostUSD = &costUSD.Float64
		}
		if durationMs.Valid {
			v := int(durationMs.Int64)
			sp.DurationMs = &v
		}
		if errorMsg.Valid {
			sp.ErrorMessage = &errorMsg.String
		}
		if model.Valid {
			sp.Model = &model.String
		}
		if provider.Valid {
			sp.Provider = &provider.String
		}
		if endedAt.Valid {
			sp.EndedAt = &endedAt.Time
		}
		// Extended fields (Phase 7.1)
		if stopReason.Valid {
			sp.StopReason = &stopReason.String
		}
		if cacheReadTokens.Valid {
			v := int(cacheReadTokens.Int64)
			sp.CacheReadTokens = &v
		}
		if cacheWriteTokens.Valid {
			v := int(cacheWriteTokens.Int64)
			sp.CacheWriteTokens = &v
		}
		if reasoningTokens.Valid {
			v := int(reasoningTokens.Int64)
			sp.ReasoningTokens = &v
		}
		if firstTokenMs.Valid {
			v := int(firstTokenMs.Int64)
			sp.FirstTokenMs = &v
		}
		if thinking.Valid {
			sp.Thinking = &thinking.String
		}
		// Pre-computed fields (Phase 7.2)
		if subType.Valid {
			sp.SubType = &subType.String
		}
		if toolUsesJSON.Valid && toolUsesJSON.String != "" {
			json.Unmarshal([]byte(toolUsesJSON.String), &sp.ToolUses)
		}
		json.Unmarshal([]byte(metadataJSON), &sp.Metadata)

		spans = append(spans, sp)
	}

	return spans, nil
}

func (s *Store) ListTraces(ctx context.Context, projectID string, filter entity.TraceFilter) (*entity.Page[entity.TraceWithMetrics], error) {
	// Build query
	where := []string{"t.project_id = ?"}
	args := []any{projectID}

	if filter.SessionID != nil {
		where = append(where, "t.session_id = ?")
		args = append(args, *filter.SessionID)
	}
	if filter.UserID != nil {
		where = append(where, "t.user_id = ?")
		args = append(args, *filter.UserID)
	}
	if filter.Status != nil {
		where = append(where, "t.status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.From != nil {
		where = append(where, "t.created_at >= ?")
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		where = append(where, "t.created_at <= ?")
		args = append(args, *filter.To)
	}

	whereClause := strings.Join(where, " AND ")

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM traces t WHERE %s", whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Get traces with metrics
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT t.id, t.project_id, t.session_id, t.user_id, t.status, t.tags, t.metadata, t.created_at, t.updated_at,
		       COALESCE(COUNT(s.id), 0) as total_spans,
		       COALESCE(SUM(COALESCE(s.input_tokens, 0) + COALESCE(s.output_tokens, 0)), 0) as total_tokens,
		       COALESCE(SUM(COALESCE(s.cost_usd, 0)), 0) as total_cost,
		       COALESCE(SUM(COALESCE(s.duration_ms, 0)), 0) as total_duration
		FROM traces t
		LEFT JOIN spans s ON s.trace_id = t.id
		WHERE %s
		GROUP BY t.id
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []entity.TraceWithMetrics
	for rows.Next() {
		var t entity.TraceWithMetrics
		var tagsJSON, metadataJSON string
		var sessionID, userID sql.NullString

		err := rows.Scan(&t.ID, &t.ProjectID, &sessionID, &userID, &t.Status, &tagsJSON, &metadataJSON,
			&t.CreatedAt, &t.UpdatedAt, &t.TotalSpans, &t.TotalTokens, &t.TotalCostUSD, &t.TotalDurationMs)
		if err != nil {
			return nil, err
		}

		if sessionID.Valid {
			t.SessionID = &sessionID.String
		}
		if userID.Valid {
			t.UserID = &userID.String
		}
		json.Unmarshal([]byte(tagsJSON), &t.Tags)
		json.Unmarshal([]byte(metadataJSON), &t.Metadata)

		traces = append(traces, t)
	}

	return &entity.Page[entity.TraceWithMetrics]{
		Data:   traces,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ============================================
// SPAN OPERATIONS
// ============================================

func (s *Store) CreateSpan(ctx context.Context, span *entity.Span) error {
	if span.ID == "" {
		span.ID = uuid.New().String()
	}

	inputJSON, _ := json.Marshal(span.Input)
	outputJSON, _ := json.Marshal(span.Output)
	metadataJSON, _ := json.Marshal(span.Metadata)

	var toolUsesJSON *string
	if len(span.ToolUses) > 0 {
		b, _ := json.Marshal(span.ToolUses)
		s := string(b)
		toolUsesJSON = &s
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO spans (id, trace_id, parent_span_id, type, name, input, output,
		                   input_tokens, output_tokens, cost_usd, duration_ms, status,
		                   error_message, model, provider, metadata, started_at, ended_at,
		                   stop_reason, cache_read_tokens, cache_write_tokens,
		                   reasoning_tokens, first_token_ms, thinking,
		                   sub_type, tool_uses)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, span.ID, span.TraceID, span.ParentSpanID, span.Type, span.Name,
		string(inputJSON), string(outputJSON), span.InputTokens, span.OutputTokens,
		span.CostUSD, span.DurationMs, span.Status, span.ErrorMessage, span.Model,
		span.Provider, string(metadataJSON), span.StartedAt, span.EndedAt,
		span.StopReason, span.CacheReadTokens, span.CacheWriteTokens,
		span.ReasoningTokens, span.FirstTokenMs, span.Thinking,
		span.SubType, toolUsesJSON)

	return err
}

func (s *Store) CreateSpans(ctx context.Context, spans []entity.Span) error {
	if len(spans) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO spans (id, trace_id, parent_span_id, type, name, input, output,
		                   input_tokens, output_tokens, cost_usd, duration_ms, status,
		                   error_message, model, provider, metadata, started_at, ended_at,
		                   stop_reason, cache_read_tokens, cache_write_tokens,
		                   reasoning_tokens, first_token_ms, thinking,
		                   sub_type, tool_uses)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range spans {
		span := &spans[i]
		if span.ID == "" {
			span.ID = uuid.New().String()
		}

		inputJSON, _ := json.Marshal(span.Input)
		outputJSON, _ := json.Marshal(span.Output)
		metadataJSON, _ := json.Marshal(span.Metadata)

		var toolUsesJSON *string
		if len(span.ToolUses) > 0 {
			b, _ := json.Marshal(span.ToolUses)
			str := string(b)
			toolUsesJSON = &str
		}

		_, err := stmt.ExecContext(ctx, span.ID, span.TraceID, span.ParentSpanID, span.Type, span.Name,
			string(inputJSON), string(outputJSON), span.InputTokens, span.OutputTokens,
			span.CostUSD, span.DurationMs, span.Status, span.ErrorMessage, span.Model,
			span.Provider, string(metadataJSON), span.StartedAt, span.EndedAt,
			span.StopReason, span.CacheReadTokens, span.CacheWriteTokens,
			span.ReasoningTokens, span.FirstTokenMs, span.Thinking,
			span.SubType, toolUsesJSON)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ============================================
// SESSION OPERATIONS
// ============================================

func (s *Store) ListSessions(ctx context.Context, projectID string, filter entity.SessionFilter) (*entity.Page[entity.Session], error) {
	where := []string{"t.project_id = ?", "t.session_id IS NOT NULL"}
	args := []any{projectID}

	if filter.UserID != nil {
		where = append(where, "t.user_id = ?")
		args = append(args, *filter.UserID)
	}
	if filter.From != nil {
		where = append(where, "t.created_at >= ?")
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		where = append(where, "t.created_at <= ?")
		args = append(args, *filter.To)
	}

	whereClause := strings.Join(where, " AND ")

	// Get total
	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT t.session_id) FROM traces t WHERE %s
	`, whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Get sessions
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset

	query := fmt.Sprintf(`
		SELECT
			t.session_id,
			MAX(t.user_id) as user_id,
			COUNT(DISTINCT t.id) as trace_count,
			COALESCE(COUNT(s.id), 0) as total_spans,
			COALESCE(SUM(COALESCE(s.input_tokens, 0) + COALESCE(s.output_tokens, 0)), 0) as total_tokens,
			COALESCE(SUM(COALESCE(s.cost_usd, 0)), 0) as total_cost,
			COALESCE(SUM(COALESCE(s.duration_ms, 0)), 0) as total_duration,
			MAX(CASE WHEN t.status = 'error' THEN 1 ELSE 0 END) as has_error,
			MAX(CASE WHEN t.status = 'active' THEN 1 ELSE 0 END) as has_active,
			MIN(t.created_at) as first_trace_at,
			MAX(t.created_at) as last_trace_at
		FROM traces t
		LEFT JOIN spans s ON s.trace_id = t.id
		WHERE %s
		GROUP BY t.session_id
		ORDER BY MAX(t.created_at) DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]entity.Session, 0) // Initialize as empty slice, not nil
	for rows.Next() {
		var sess entity.Session
		var userID sql.NullString
		var hasError, hasActive int
		var firstTraceAt, lastTraceAt string

		err := rows.Scan(&sess.SessionID, &userID, &sess.TraceCount, &sess.TotalSpans,
			&sess.TotalTokens, &sess.TotalCostUSD, &sess.TotalDurationMs,
			&hasError, &hasActive, &firstTraceAt, &lastTraceAt)
		if err != nil {
			return nil, err
		}

		if userID.Valid {
			sess.UserID = &userID.String
		}
		sess.HasError = hasError == 1
		sess.HasActive = hasActive == 1
		sess.FirstTraceAt, _ = time.Parse(time.RFC3339, firstTraceAt)
		sess.LastTraceAt, _ = time.Parse(time.RFC3339, lastTraceAt)

		sessions = append(sessions, sess)
	}

	return &entity.Page[entity.Session]{
		Data:   sessions,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ============================================
// ANALYTICS OPERATIONS
// ============================================

func (s *Store) GetStats(ctx context.Context, projectID string, period entity.Period) (*entity.Stats, error) {
	query := `
		SELECT
			COUNT(DISTINCT t.id) as total_traces,
			COALESCE(COUNT(s.id), 0) as total_spans,
			COALESCE(SUM(COALESCE(s.input_tokens, 0) + COALESCE(s.output_tokens, 0)), 0) as total_tokens,
			COALESCE(SUM(COALESCE(s.cost_usd, 0)), 0) as total_cost,
			COALESCE(AVG(s.duration_ms), 0) as avg_duration,
			COUNT(DISTINCT CASE WHEN t.status = 'error' THEN t.id END) as error_count
		FROM traces t
		LEFT JOIN spans s ON s.trace_id = t.id
		WHERE t.project_id = ? AND t.created_at >= ? AND t.created_at <= ?
	`

	var stats entity.Stats
	var errorCount int

	var avgDuration float64
	err := s.db.QueryRowContext(ctx, query, projectID, period.From.Format(time.RFC3339), period.To.Format(time.RFC3339)).Scan(
		&stats.TotalTraces, &stats.TotalSpans, &stats.TotalTokens,
		&stats.TotalCostUSD, &avgDuration, &errorCount)
	if err != nil {
		return nil, fmt.Errorf("GetStats query error: %w", err)
	}
	stats.AvgDurationMs = int(avgDuration)

	if stats.TotalTraces > 0 {
		stats.ErrorRate = (float64(errorCount) / float64(stats.TotalTraces)) * 100
	}

	return &stats, nil
}

func (s *Store) GetUsageTimeSeries(ctx context.Context, projectID string, opts entity.TimeSeriesOpts) ([]entity.DataPoint, error) {
	// Determine date grouping length based on granularity
	// For day: substr(1,10) = "YYYY-MM-DD"
	// For hour: substr(1,13) = "YYYY-MM-DDTHH"
	// For week: we'll use substr(1,10) and group by date
	dateLen := 10 // default: day
	if opts.Granularity == "hour" {
		dateLen = 13
	}

	// Use substr to extract ISO8601 date part since SQLite strftime doesn't parse Go's RFC3339
	query := fmt.Sprintf(`
		SELECT
			substr(t.created_at, 1, %d) as date,
			COUNT(DISTINCT t.id) as traces,
			COALESCE(COUNT(s.id), 0) as spans,
			COALESCE(SUM(COALESCE(s.input_tokens, 0) + COALESCE(s.output_tokens, 0)), 0) as tokens,
			COALESCE(SUM(COALESCE(s.cost_usd, 0)), 0) as cost
		FROM traces t
		LEFT JOIN spans s ON s.trace_id = t.id
		WHERE t.project_id = ? AND t.created_at >= ? AND t.created_at <= ?
		GROUP BY substr(t.created_at, 1, %d)
		ORDER BY date
	`, dateLen, dateLen)

	rows, err := s.db.QueryContext(ctx, query, projectID, opts.From.Format(time.RFC3339), opts.To.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("GetUsageTimeSeries query error: %w", err)
	}
	defer rows.Close()

	var dataPoints []entity.DataPoint
	for rows.Next() {
		var dp entity.DataPoint
		var dateStr string

		if err := rows.Scan(&dateStr, &dp.Traces, &dp.Spans, &dp.Tokens, &dp.CostUSD); err != nil {
			return nil, fmt.Errorf("GetUsageTimeSeries scan error: %w", err)
		}

		// Parse date string back to time.Time
		dp.Time, _ = time.Parse("2006-01-02", dateStr)
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}
