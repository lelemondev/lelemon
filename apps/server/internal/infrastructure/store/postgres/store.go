package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lelemon/server/internal/domain/entity"
)

// Store implements repository.Store for PostgreSQL
type Store struct {
	pool *pgxpool.Pool
}

// New creates a new PostgreSQL store with connection pooling
func New(connString string) (*Store, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure pool
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Migrate runs database migrations
func (s *Store) Migrate(ctx context.Context) error {
	migrations := []string{
		// Enable UUID extension
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,

		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			password_hash TEXT,
			google_id TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Projects table
		`CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name TEXT NOT NULL,
			api_key TEXT UNIQUE NOT NULL,
			api_key_hash TEXT NOT NULL,
			owner_email TEXT NOT NULL,
			settings JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Traces table
		`CREATE TABLE IF NOT EXISTS traces (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			session_id TEXT,
			user_id TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			tags JSONB DEFAULT '[]',
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Spans table
		`CREATE TABLE IF NOT EXISTS spans (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			trace_id UUID NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
			parent_span_id UUID,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			input JSONB,
			output JSONB,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cost_usd DOUBLE PRECISION,
			duration_ms INTEGER,
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT,
			model TEXT,
			provider TEXT,
			metadata JSONB DEFAULT '{}',
			started_at TIMESTAMPTZ DEFAULT NOW(),
			ended_at TIMESTAMPTZ,
			stop_reason TEXT,
			cache_read_tokens INTEGER,
			cache_write_tokens INTEGER,
			reasoning_tokens INTEGER,
			first_token_ms INTEGER,
			thinking TEXT
		)`,

		// Phase 7.1: Add extended fields to existing spans table
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS stop_reason TEXT`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS cache_read_tokens INTEGER`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS cache_write_tokens INTEGER`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS reasoning_tokens INTEGER`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS first_token_ms INTEGER`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS thinking TEXT`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_projects_api_key_hash ON projects(api_key_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_email)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_project_created ON traces(project_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(project_id, session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_user ON traces(project_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_trace ON spans(trace_id, started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id)`,
	}

	for _, m := range migrations {
		if _, err := s.pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// Ping checks the database connection
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close closes the connection pool
func (s *Store) Close() error {
	s.pool.Close()
	return nil
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

	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, password_hash, google_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, u.ID, u.Email, u.Name, u.PasswordHash, u.GoogleID, u.CreatedAt, u.UpdatedAt)

	return err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	var u entity.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, google_id, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, google_id, created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)

	if err == pgx.ErrNoRows {
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
	argNum := 1

	if updates.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *updates.Name)
		argNum++
	}
	if updates.PasswordHash != nil {
		sets = append(sets, fmt.Sprintf("password_hash = $%d", argNum))
		args = append(args, *updates.PasswordHash)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.pool.Exec(ctx, query, args...)
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

	_, err := s.pool.Exec(ctx, `
		INSERT INTO projects (id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, p.ID, p.Name, p.APIKey, p.APIKeyHash, p.OwnerEmail, settingsJSON, p.CreatedAt, p.UpdatedAt)

	return err
}

func (s *Store) GetProjectByID(ctx context.Context, id string) (*entity.Project, error) {
	var p entity.Project
	var settingsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsJSON, &p.Settings)
	return &p, nil
}

func (s *Store) GetProjectByAPIKeyHash(ctx context.Context, hash string) (*entity.Project, error) {
	var p entity.Project
	var settingsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects WHERE api_key_hash = $1
	`, hash).Scan(&p.ID, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(settingsJSON, &p.Settings)
	return &p, nil
}

func (s *Store) UpdateProject(ctx context.Context, id string, updates entity.ProjectUpdate) error {
	var sets []string
	var args []any
	argNum := 1

	if updates.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *updates.Name)
		argNum++
	}
	if updates.Settings != nil {
		settingsJSON, _ := json.Marshal(updates.Settings)
		sets = append(sets, fmt.Sprintf("settings = $%d", argNum))
		args = append(args, settingsJSON)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM projects WHERE id = $1", id)
	return err
}

func (s *Store) ListProjectsByOwner(ctx context.Context, email string) ([]entity.Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects WHERE owner_email = $1 ORDER BY created_at DESC
	`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []entity.Project
	for rows.Next() {
		var p entity.Project
		var settingsJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(settingsJSON, &p.Settings)
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) RotateAPIKey(ctx context.Context, id string, newKey, newHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE projects SET api_key = $1, api_key_hash = $2, updated_at = $3 WHERE id = $4
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

	_, err := s.pool.Exec(ctx, `
		INSERT INTO traces (id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, t.ID, t.ProjectID, t.SessionID, t.UserID, t.Status, tagsJSON, metadataJSON, t.CreatedAt, t.UpdatedAt)

	return err
}

func (s *Store) UpdateTrace(ctx context.Context, projectID, traceID string, updates entity.TraceUpdate) error {
	var sets []string
	var args []any
	argNum := 1

	if updates.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argNum))
		args = append(args, string(*updates.Status))
		argNum++
	}
	if updates.Metadata != nil {
		metadataJSON, _ := json.Marshal(updates.Metadata)
		sets = append(sets, fmt.Sprintf("metadata = $%d", argNum))
		args = append(args, metadataJSON)
		argNum++
	}
	if updates.Tags != nil {
		tagsJSON, _ := json.Marshal(updates.Tags)
		sets = append(sets, fmt.Sprintf("tags = $%d", argNum))
		args = append(args, tagsJSON)
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++
	args = append(args, projectID, traceID)

	query := fmt.Sprintf("UPDATE traces SET %s WHERE project_id = $%d AND id = $%d", strings.Join(sets, ", "), argNum, argNum+1)
	_, err := s.pool.Exec(ctx, query, args...)
	return err
}

func (s *Store) UpdateTraceStatus(ctx context.Context, projectID, traceID string, status entity.TraceStatus) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE traces SET status = $1, updated_at = $2 WHERE project_id = $3 AND id = $4
	`, string(status), time.Now(), projectID, traceID)
	return err
}

func (s *Store) DeleteAllTraces(ctx context.Context, projectID string) (int64, error) {
	// Spans are deleted via CASCADE when traces are deleted
	result, err := s.pool.Exec(ctx, `DELETE FROM traces WHERE project_id = $1`, projectID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func (s *Store) GetTrace(ctx context.Context, projectID, traceID string) (*entity.TraceWithSpans, error) {
	var t entity.Trace
	var tagsJSON, metadataJSON []byte
	var sessionID, userID *string

	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at
		FROM traces WHERE project_id = $1 AND id = $2
	`, projectID, traceID).Scan(&t.ID, &t.ProjectID, &sessionID, &userID, &t.Status, &tagsJSON, &metadataJSON, &t.CreatedAt, &t.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	t.SessionID = sessionID
	t.UserID = userID
	json.Unmarshal(tagsJSON, &t.Tags)
	json.Unmarshal(metadataJSON, &t.Metadata)

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
	rows, err := s.pool.Query(ctx, `
		SELECT id, trace_id, parent_span_id, type, name, input, output,
		       input_tokens, output_tokens, cost_usd, duration_ms, status,
		       error_message, model, provider, metadata, started_at, ended_at,
		       stop_reason, cache_read_tokens, cache_write_tokens,
		       reasoning_tokens, first_token_ms, thinking
		FROM spans WHERE trace_id = $1 ORDER BY started_at
	`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []entity.Span
	for rows.Next() {
		var sp entity.Span
		var parentSpanID *string
		var inputJSON, outputJSON, metadataJSON []byte
		var errorMsg, model, provider *string
		var stopReason, thinking *string
		var inputTokens, outputTokens, durationMs *int
		var cacheReadTokens, cacheWriteTokens, reasoningTokens, firstTokenMs *int
		var costUSD *float64
		var endedAt *time.Time

		err := rows.Scan(&sp.ID, &sp.TraceID, &parentSpanID, &sp.Type, &sp.Name,
			&inputJSON, &outputJSON, &inputTokens, &outputTokens, &costUSD,
			&durationMs, &sp.Status, &errorMsg, &model, &provider, &metadataJSON,
			&sp.StartedAt, &endedAt,
			&stopReason, &cacheReadTokens, &cacheWriteTokens,
			&reasoningTokens, &firstTokenMs, &thinking)
		if err != nil {
			return nil, err
		}

		sp.ParentSpanID = parentSpanID
		if inputJSON != nil {
			json.Unmarshal(inputJSON, &sp.Input)
		}
		if outputJSON != nil {
			json.Unmarshal(outputJSON, &sp.Output)
		}
		sp.InputTokens = inputTokens
		sp.OutputTokens = outputTokens
		sp.CostUSD = costUSD
		sp.DurationMs = durationMs
		sp.ErrorMessage = errorMsg
		sp.Model = model
		sp.Provider = provider
		sp.EndedAt = endedAt
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &sp.Metadata)
		}
		// Extended fields (Phase 7.1)
		sp.StopReason = stopReason
		sp.CacheReadTokens = cacheReadTokens
		sp.CacheWriteTokens = cacheWriteTokens
		sp.ReasoningTokens = reasoningTokens
		sp.FirstTokenMs = firstTokenMs
		sp.Thinking = thinking

		spans = append(spans, sp)
	}

	return spans, nil
}

func (s *Store) ListTraces(ctx context.Context, projectID string, filter entity.TraceFilter) (*entity.Page[entity.TraceWithMetrics], error) {
	// Build query with positional parameters
	where := []string{"t.project_id = $1"}
	args := []any{projectID}
	argNum := 2

	if filter.SessionID != nil {
		where = append(where, fmt.Sprintf("t.session_id = $%d", argNum))
		args = append(args, *filter.SessionID)
		argNum++
	}
	if filter.UserID != nil {
		where = append(where, fmt.Sprintf("t.user_id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}
	if filter.Status != nil {
		where = append(where, fmt.Sprintf("t.status = $%d", argNum))
		args = append(args, string(*filter.Status))
		argNum++
	}
	if filter.From != nil {
		where = append(where, fmt.Sprintf("t.created_at >= $%d", argNum))
		args = append(args, *filter.From)
		argNum++
	}
	if filter.To != nil {
		where = append(where, fmt.Sprintf("t.created_at <= $%d", argNum))
		args = append(args, *filter.To)
		argNum++
	}

	whereClause := strings.Join(where, " AND ")

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM traces t WHERE %s", whereClause)
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []entity.TraceWithMetrics
	for rows.Next() {
		var t entity.TraceWithMetrics
		var tagsJSON, metadataJSON []byte
		var sessionID, userID *string

		err := rows.Scan(&t.ID, &t.ProjectID, &sessionID, &userID, &t.Status, &tagsJSON, &metadataJSON,
			&t.CreatedAt, &t.UpdatedAt, &t.TotalSpans, &t.TotalTokens, &t.TotalCostUSD, &t.TotalDurationMs)
		if err != nil {
			return nil, err
		}

		t.SessionID = sessionID
		t.UserID = userID
		json.Unmarshal(tagsJSON, &t.Tags)
		json.Unmarshal(metadataJSON, &t.Metadata)

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

	_, err := s.pool.Exec(ctx, `
		INSERT INTO spans (id, trace_id, parent_span_id, type, name, input, output,
		                   input_tokens, output_tokens, cost_usd, duration_ms, status,
		                   error_message, model, provider, metadata, started_at, ended_at,
		                   stop_reason, cache_read_tokens, cache_write_tokens,
		                   reasoning_tokens, first_token_ms, thinking)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`, span.ID, span.TraceID, span.ParentSpanID, span.Type, span.Name,
		inputJSON, outputJSON, span.InputTokens, span.OutputTokens,
		span.CostUSD, span.DurationMs, span.Status, span.ErrorMessage, span.Model,
		span.Provider, metadataJSON, span.StartedAt, span.EndedAt,
		span.StopReason, span.CacheReadTokens, span.CacheWriteTokens,
		span.ReasoningTokens, span.FirstTokenMs, span.Thinking)

	return err
}

func (s *Store) CreateSpans(ctx context.Context, spans []entity.Span) error {
	if len(spans) == 0 {
		return nil
	}

	batch := &pgx.Batch{}

	for i := range spans {
		span := &spans[i]
		if span.ID == "" {
			span.ID = uuid.New().String()
		}

		inputJSON, _ := json.Marshal(span.Input)
		outputJSON, _ := json.Marshal(span.Output)
		metadataJSON, _ := json.Marshal(span.Metadata)

		batch.Queue(`
			INSERT INTO spans (id, trace_id, parent_span_id, type, name, input, output,
			                   input_tokens, output_tokens, cost_usd, duration_ms, status,
			                   error_message, model, provider, metadata, started_at, ended_at,
			                   stop_reason, cache_read_tokens, cache_write_tokens,
			                   reasoning_tokens, first_token_ms, thinking)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		`, span.ID, span.TraceID, span.ParentSpanID, span.Type, span.Name,
			inputJSON, outputJSON, span.InputTokens, span.OutputTokens,
			span.CostUSD, span.DurationMs, span.Status, span.ErrorMessage, span.Model,
			span.Provider, metadataJSON, span.StartedAt, span.EndedAt,
			span.StopReason, span.CacheReadTokens, span.CacheWriteTokens,
			span.ReasoningTokens, span.FirstTokenMs, span.Thinking)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range spans {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

// ============================================
// SESSION OPERATIONS
// ============================================

func (s *Store) ListSessions(ctx context.Context, projectID string, filter entity.SessionFilter) (*entity.Page[entity.Session], error) {
	where := []string{"t.project_id = $1", "t.session_id IS NOT NULL"}
	args := []any{projectID}
	argNum := 2

	if filter.UserID != nil {
		where = append(where, fmt.Sprintf("t.user_id = $%d", argNum))
		args = append(args, *filter.UserID)
		argNum++
	}
	if filter.From != nil {
		where = append(where, fmt.Sprintf("t.created_at >= $%d", argNum))
		args = append(args, *filter.From)
		argNum++
	}
	if filter.To != nil {
		where = append(where, fmt.Sprintf("t.created_at <= $%d", argNum))
		args = append(args, *filter.To)
		argNum++
	}

	whereClause := strings.Join(where, " AND ")

	// Get total
	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT t.session_id) FROM traces t WHERE %s
	`, whereClause)
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, offset)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]entity.Session, 0) // Initialize as empty slice, not nil
	for rows.Next() {
		var sess entity.Session
		var userID *string
		var hasError, hasActive int

		err := rows.Scan(&sess.SessionID, &userID, &sess.TraceCount, &sess.TotalSpans,
			&sess.TotalTokens, &sess.TotalCostUSD, &sess.TotalDurationMs,
			&hasError, &hasActive, &sess.FirstTraceAt, &sess.LastTraceAt)
		if err != nil {
			return nil, err
		}

		sess.UserID = userID
		sess.HasError = hasError == 1
		sess.HasActive = hasActive == 1

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
		WHERE t.project_id = $1 AND t.created_at >= $2 AND t.created_at <= $3
	`

	var stats entity.Stats
	var errorCount int
	var avgDuration float64

	err := s.pool.QueryRow(ctx, query, projectID, period.From, period.To).Scan(
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
	// Determine date truncation based on granularity
	truncTo := "day"
	if opts.Granularity == "hour" {
		truncTo = "hour"
	} else if opts.Granularity == "week" {
		truncTo = "week"
	}

	query := fmt.Sprintf(`
		SELECT
			date_trunc('%s', t.created_at) as date,
			COUNT(DISTINCT t.id) as traces,
			COALESCE(COUNT(s.id), 0) as spans,
			COALESCE(SUM(COALESCE(s.input_tokens, 0) + COALESCE(s.output_tokens, 0)), 0) as tokens,
			COALESCE(SUM(COALESCE(s.cost_usd, 0)), 0) as cost
		FROM traces t
		LEFT JOIN spans s ON s.trace_id = t.id
		WHERE t.project_id = $1 AND t.created_at >= $2 AND t.created_at <= $3
		GROUP BY date_trunc('%s', t.created_at)
		ORDER BY date
	`, truncTo, truncTo)

	rows, err := s.pool.Query(ctx, query, projectID, opts.From, opts.To)
	if err != nil {
		return nil, fmt.Errorf("GetUsageTimeSeries query error: %w", err)
	}
	defer rows.Close()

	var dataPoints []entity.DataPoint
	for rows.Next() {
		var dp entity.DataPoint

		if err := rows.Scan(&dp.Time, &dp.Traces, &dp.Spans, &dp.Tokens, &dp.CostUSD); err != nil {
			return nil, fmt.Errorf("GetUsageTimeSeries scan error: %w", err)
		}

		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}
