package clickhouse

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/lelemon/server/pkg/domain/entity"
)

// Store implements repository.Store for ClickHouse
// Optimized for high-volume analytics workloads
type Store struct {
	conn driver.Conn
}

// New creates a new ClickHouse store
func New(connString string) (*Store, error) {
	// Parse connection string: clickhouse://user:pass@host:9000/database
	u, err := url.Parse(connString)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	password, _ := u.User.Password()
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		database = "lelemon"
	}

	opts := &clickhouse.Options{
		Addr: []string{u.Host},
		Auth: clickhouse.Auth{
			Database: database,
			Username: u.User.Username(),
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     10 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}

	// Enable TLS if scheme is clickhouses://
	if u.Scheme == "clickhouses" {
		opts.TLS = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &Store{conn: conn}, nil
}

// Migrate creates the ClickHouse tables
func (s *Store) Migrate(ctx context.Context) error {
	migrations := []string{
		// Users table - ReplacingMergeTree for updates
		`CREATE TABLE IF NOT EXISTS users (
			id UUID,
			email String,
			name String,
			password_hash Nullable(String),
			google_id Nullable(String),
			created_at DateTime64(3),
			updated_at DateTime64(3)
		) ENGINE = ReplacingMergeTree(updated_at)
		ORDER BY id`,

		// Projects table - ReplacingMergeTree for updates
		`CREATE TABLE IF NOT EXISTS projects (
			id UUID,
			name String,
			api_key String,
			api_key_hash String,
			owner_email String,
			settings String DEFAULT '{}',
			created_at DateTime64(3),
			updated_at DateTime64(3)
		) ENGINE = ReplacingMergeTree(updated_at)
		ORDER BY id`,

		// Traces table - ReplacingMergeTree for status updates
		`CREATE TABLE IF NOT EXISTS traces (
			id UUID,
			project_id UUID,
			session_id Nullable(String),
			user_id Nullable(String),
			status String DEFAULT 'active',
			tags Array(String) DEFAULT [],
			metadata String DEFAULT '{}',
			created_at DateTime64(3),
			updated_at DateTime64(3)
		) ENGINE = ReplacingMergeTree(updated_at)
		PARTITION BY toYYYYMM(created_at)
		ORDER BY (project_id, created_at, id)`,

		// Spans table - MergeTree (append-only, no updates)
		`CREATE TABLE IF NOT EXISTS spans (
			id UUID,
			trace_id UUID,
			parent_span_id Nullable(UUID),
			type String,
			name String,
			input Nullable(String),
			output Nullable(String),
			input_tokens Nullable(UInt32),
			output_tokens Nullable(UInt32),
			cost_usd Nullable(Float64),
			duration_ms Nullable(UInt32),
			status String DEFAULT 'pending',
			error_message Nullable(String),
			model Nullable(String),
			provider Nullable(String),
			metadata String DEFAULT '{}',
			started_at DateTime64(3),
			ended_at Nullable(DateTime64(3)),
			stop_reason Nullable(String),
			cache_read_tokens Nullable(UInt32),
			cache_write_tokens Nullable(UInt32),
			reasoning_tokens Nullable(UInt32),
			first_token_ms Nullable(UInt32),
			thinking Nullable(String)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(started_at)
		ORDER BY (trace_id, started_at, id)`,

		// Phase 7.1: Add extended fields to existing spans table
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS stop_reason Nullable(String)`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS cache_read_tokens Nullable(UInt32)`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS cache_write_tokens Nullable(UInt32)`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS reasoning_tokens Nullable(UInt32)`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS first_token_ms Nullable(UInt32)`,
		`ALTER TABLE spans ADD COLUMN IF NOT EXISTS thinking Nullable(String)`,

		// Indexes for common queries
		`ALTER TABLE projects ADD INDEX IF NOT EXISTS idx_api_key_hash api_key_hash TYPE bloom_filter GRANULARITY 1`,
		`ALTER TABLE projects ADD INDEX IF NOT EXISTS idx_owner_email owner_email TYPE bloom_filter GRANULARITY 1`,
		`ALTER TABLE traces ADD INDEX IF NOT EXISTS idx_session session_id TYPE bloom_filter GRANULARITY 1`,
		`ALTER TABLE traces ADD INDEX IF NOT EXISTS idx_user user_id TYPE bloom_filter GRANULARITY 1`,
		`ALTER TABLE users ADD INDEX IF NOT EXISTS idx_email email TYPE bloom_filter GRANULARITY 1`,
	}

	for _, m := range migrations {
		if err := s.conn.Exec(ctx, m); err != nil {
			// Ignore "already exists" errors for indexes
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	return nil
}

// Ping checks the database connection
func (s *Store) Ping(ctx context.Context) error {
	return s.conn.Ping(ctx)
}

// Close closes the connection
func (s *Store) Close() error {
	return s.conn.Close()
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

	return s.conn.Exec(ctx, `
		INSERT INTO users (id, email, name, password_hash, google_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(u.ID), u.Email, u.Name, u.PasswordHash, u.GoogleID, u.CreatedAt, u.UpdatedAt)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*entity.User, error) {
	var u entity.User
	var uid uuid.UUID

	// FINAL ensures we get the latest version from ReplacingMergeTree
	row := s.conn.QueryRow(ctx, `
		SELECT id, email, name, password_hash, google_id, created_at, updated_at
		FROM users FINAL WHERE id = ?
	`, uuid.MustParse(id))

	err := row.Scan(&uid, &u.Email, &u.Name, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	u.ID = uid.String()
	return &u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	var uid uuid.UUID

	row := s.conn.QueryRow(ctx, `
		SELECT id, email, name, password_hash, google_id, created_at, updated_at
		FROM users FINAL WHERE email = ?
	`, email)

	err := row.Scan(&uid, &u.Email, &u.Name, &u.PasswordHash, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	u.ID = uid.String()
	return &u, nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, updates entity.UserUpdate) error {
	// In ClickHouse with ReplacingMergeTree, we insert a new row with updated values
	existing, err := s.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	if updates.Name != nil {
		existing.Name = *updates.Name
	}
	if updates.PasswordHash != nil {
		existing.PasswordHash = updates.PasswordHash
	}
	if updates.GoogleID != nil {
		existing.GoogleID = updates.GoogleID
	}
	existing.UpdatedAt = time.Now()

	return s.conn.Exec(ctx, `
		INSERT INTO users (id, email, name, password_hash, google_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(existing.ID), existing.Email, existing.Name, existing.PasswordHash, existing.GoogleID, existing.CreatedAt, existing.UpdatedAt)
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

	return s.conn.Exec(ctx, `
		INSERT INTO projects (id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(p.ID), p.Name, p.APIKey, p.APIKeyHash, p.OwnerEmail, string(settingsJSON), p.CreatedAt, p.UpdatedAt)
}

func (s *Store) GetProjectByID(ctx context.Context, id string) (*entity.Project, error) {
	var p entity.Project
	var pid uuid.UUID
	var settingsJSON string

	row := s.conn.QueryRow(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects FINAL WHERE id = ?
	`, uuid.MustParse(id))

	err := row.Scan(&pid, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	p.ID = pid.String()
	json.Unmarshal([]byte(settingsJSON), &p.Settings)
	return &p, nil
}

func (s *Store) GetProjectByAPIKeyHash(ctx context.Context, hash string) (*entity.Project, error) {
	var p entity.Project
	var pid uuid.UUID
	var settingsJSON string

	row := s.conn.QueryRow(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects FINAL WHERE api_key_hash = ?
	`, hash)

	err := row.Scan(&pid, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	p.ID = pid.String()
	json.Unmarshal([]byte(settingsJSON), &p.Settings)
	return &p, nil
}

func (s *Store) UpdateProject(ctx context.Context, id string, updates entity.ProjectUpdate) error {
	existing, err := s.GetProjectByID(ctx, id)
	if err != nil {
		return err
	}

	if updates.Name != nil {
		existing.Name = *updates.Name
	}
	if updates.Settings != nil {
		existing.Settings = *updates.Settings
	}
	existing.UpdatedAt = time.Now()

	settingsJSON, _ := json.Marshal(existing.Settings)

	return s.conn.Exec(ctx, `
		INSERT INTO projects (id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(existing.ID), existing.Name, existing.APIKey, existing.APIKeyHash, existing.OwnerEmail, string(settingsJSON), existing.CreatedAt, existing.UpdatedAt)
}

func (s *Store) DeleteProject(ctx context.Context, id string) error {
	// ClickHouse doesn't support DELETE directly, use ALTER TABLE DELETE
	return s.conn.Exec(ctx, "ALTER TABLE projects DELETE WHERE id = ?", uuid.MustParse(id))
}

func (s *Store) ListProjectsByOwner(ctx context.Context, email string) ([]entity.Project, error) {
	rows, err := s.conn.Query(ctx, `
		SELECT id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at
		FROM projects FINAL WHERE owner_email = ? ORDER BY created_at DESC
	`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []entity.Project
	for rows.Next() {
		var p entity.Project
		var pid uuid.UUID
		var settingsJSON string
		if err := rows.Scan(&pid, &p.Name, &p.APIKey, &p.APIKeyHash, &p.OwnerEmail, &settingsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.ID = pid.String()
		json.Unmarshal([]byte(settingsJSON), &p.Settings)
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) RotateAPIKey(ctx context.Context, id string, newKey, newHash string) error {
	existing, err := s.GetProjectByID(ctx, id)
	if err != nil {
		return err
	}

	existing.APIKey = newKey
	existing.APIKeyHash = newHash
	existing.UpdatedAt = time.Now()

	settingsJSON, _ := json.Marshal(existing.Settings)

	return s.conn.Exec(ctx, `
		INSERT INTO projects (id, name, api_key, api_key_hash, owner_email, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(existing.ID), existing.Name, existing.APIKey, existing.APIKeyHash, existing.OwnerEmail, string(settingsJSON), existing.CreatedAt, existing.UpdatedAt)
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

	metadataJSON, _ := json.Marshal(t.Metadata)

	// Convert tags to ClickHouse array
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}

	return s.conn.Exec(ctx, `
		INSERT INTO traces (id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(t.ID), uuid.MustParse(t.ProjectID), t.SessionID, t.UserID, string(t.Status), tags, string(metadataJSON), t.CreatedAt, t.UpdatedAt)
}

func (s *Store) UpdateTrace(ctx context.Context, projectID, traceID string, updates entity.TraceUpdate) error {
	existing, err := s.GetTrace(ctx, projectID, traceID)
	if err != nil {
		return err
	}

	if updates.Status != nil {
		existing.Status = *updates.Status
	}
	if updates.Metadata != nil {
		existing.Metadata = updates.Metadata
	}
	if updates.Tags != nil {
		existing.Tags = updates.Tags
	}
	existing.UpdatedAt = time.Now()

	metadataJSON, _ := json.Marshal(existing.Metadata)
	tags := existing.Tags
	if tags == nil {
		tags = []string{}
	}

	return s.conn.Exec(ctx, `
		INSERT INTO traces (id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(existing.ID), uuid.MustParse(existing.ProjectID), existing.SessionID, existing.UserID, string(existing.Status), tags, string(metadataJSON), existing.CreatedAt, existing.UpdatedAt)
}

func (s *Store) UpdateTraceStatus(ctx context.Context, projectID, traceID string, status entity.TraceStatus) error {
	return s.UpdateTrace(ctx, projectID, traceID, entity.TraceUpdate{Status: &status})
}

func (s *Store) DeleteAllTraces(ctx context.Context, projectID string) (int64, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return 0, fmt.Errorf("invalid project ID: %w", err)
	}

	// First delete spans for all traces in this project
	if err := s.conn.Exec(ctx, `
		ALTER TABLE spans DELETE WHERE trace_id IN (
			SELECT id FROM traces WHERE project_id = ?
		)
	`, pid); err != nil {
		return 0, err
	}

	// Then delete traces
	if err := s.conn.Exec(ctx, `ALTER TABLE traces DELETE WHERE project_id = ?`, pid); err != nil {
		return 0, err
	}

	// ClickHouse doesn't return affected rows for ALTER TABLE DELETE
	return 0, nil
}

func (s *Store) GetTrace(ctx context.Context, projectID, traceID string) (*entity.TraceWithSpans, error) {
	var t entity.Trace
	var tid, pid uuid.UUID
	var tags []string
	var metadataJSON string

	row := s.conn.QueryRow(ctx, `
		SELECT id, project_id, session_id, user_id, status, tags, metadata, created_at, updated_at
		FROM traces FINAL WHERE project_id = ? AND id = ?
	`, uuid.MustParse(projectID), uuid.MustParse(traceID))

	err := row.Scan(&tid, &pid, &t.SessionID, &t.UserID, &t.Status, &tags, &metadataJSON, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, entity.ErrNotFound
		}
		return nil, err
	}

	t.ID = tid.String()
	t.ProjectID = pid.String()
	t.Tags = tags
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
	rows, err := s.conn.Query(ctx, `
		SELECT id, trace_id, parent_span_id, type, name, input, output,
		       input_tokens, output_tokens, cost_usd, duration_ms, status,
		       error_message, model, provider, metadata, started_at, ended_at,
		       stop_reason, cache_read_tokens, cache_write_tokens,
		       reasoning_tokens, first_token_ms, thinking
		FROM spans WHERE trace_id = ? ORDER BY started_at
	`, uuid.MustParse(traceID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []entity.Span
	for rows.Next() {
		var sp entity.Span
		var spid, traceid uuid.UUID
		var parentSpanID *uuid.UUID
		var inputJSON, outputJSON, metadataJSON *string
		var stopReason, thinking *string
		var endedAt *time.Time

		err := rows.Scan(&spid, &traceid, &parentSpanID, &sp.Type, &sp.Name,
			&inputJSON, &outputJSON, &sp.InputTokens, &sp.OutputTokens, &sp.CostUSD,
			&sp.DurationMs, &sp.Status, &sp.ErrorMessage, &sp.Model, &sp.Provider, &metadataJSON,
			&sp.StartedAt, &endedAt,
			&stopReason, &sp.CacheReadTokens, &sp.CacheWriteTokens,
			&sp.ReasoningTokens, &sp.FirstTokenMs, &thinking)
		if err != nil {
			return nil, err
		}

		sp.ID = spid.String()
		sp.TraceID = traceid.String()
		if parentSpanID != nil {
			s := parentSpanID.String()
			sp.ParentSpanID = &s
		}
		if inputJSON != nil {
			json.Unmarshal([]byte(*inputJSON), &sp.Input)
		}
		if outputJSON != nil {
			json.Unmarshal([]byte(*outputJSON), &sp.Output)
		}
		if metadataJSON != nil {
			json.Unmarshal([]byte(*metadataJSON), &sp.Metadata)
		}
		sp.EndedAt = endedAt
		// Extended fields (Phase 7.1)
		sp.StopReason = stopReason
		sp.Thinking = thinking

		spans = append(spans, sp)
	}

	return spans, nil
}

func (s *Store) ListTraces(ctx context.Context, projectID string, filter entity.TraceFilter) (*entity.Page[entity.TraceWithMetrics], error) {
	// Build query
	where := []string{"t.project_id = ?"}
	args := []any{uuid.MustParse(projectID)}

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
	var total uint64
	countQuery := fmt.Sprintf("SELECT count() FROM traces FINAL AS t WHERE %s", whereClause)
	if err := s.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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
		       count(s.id) as total_spans,
		       sum(coalesce(s.input_tokens, 0) + coalesce(s.output_tokens, 0)) as total_tokens,
		       sum(coalesce(s.cost_usd, 0)) as total_cost,
		       sum(coalesce(s.duration_ms, 0)) as total_duration
		FROM traces FINAL AS t
		LEFT JOIN spans AS s ON s.trace_id = t.id
		WHERE %s
		GROUP BY t.id, t.project_id, t.session_id, t.user_id, t.status, t.tags, t.metadata, t.created_at, t.updated_at
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []entity.TraceWithMetrics
	for rows.Next() {
		var t entity.TraceWithMetrics
		var tid, pid uuid.UUID
		var tags []string
		var metadataJSON string

		err := rows.Scan(&tid, &pid, &t.SessionID, &t.UserID, &t.Status, &tags, &metadataJSON,
			&t.CreatedAt, &t.UpdatedAt, &t.TotalSpans, &t.TotalTokens, &t.TotalCostUSD, &t.TotalDurationMs)
		if err != nil {
			return nil, err
		}

		t.ID = tid.String()
		t.ProjectID = pid.String()
		t.Tags = tags
		json.Unmarshal([]byte(metadataJSON), &t.Metadata)

		traces = append(traces, t)
	}

	return &entity.Page[entity.TraceWithMetrics]{
		Data:   traces,
		Total:  int(total),
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

	var parentSpanID *uuid.UUID
	if span.ParentSpanID != nil {
		pid := uuid.MustParse(*span.ParentSpanID)
		parentSpanID = &pid
	}

	return s.conn.Exec(ctx, `
		INSERT INTO spans (id, trace_id, parent_span_id, type, name, input, output,
		                   input_tokens, output_tokens, cost_usd, duration_ms, status,
		                   error_message, model, provider, metadata, started_at, ended_at,
		                   stop_reason, cache_read_tokens, cache_write_tokens,
		                   reasoning_tokens, first_token_ms, thinking)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.MustParse(span.ID), uuid.MustParse(span.TraceID), parentSpanID, string(span.Type), span.Name,
		string(inputJSON), string(outputJSON), span.InputTokens, span.OutputTokens,
		span.CostUSD, span.DurationMs, string(span.Status), span.ErrorMessage, span.Model,
		span.Provider, string(metadataJSON), span.StartedAt, span.EndedAt,
		span.StopReason, span.CacheReadTokens, span.CacheWriteTokens,
		span.ReasoningTokens, span.FirstTokenMs, span.Thinking)
}

func (s *Store) CreateSpans(ctx context.Context, spans []entity.Span) error {
	if len(spans) == 0 {
		return nil
	}

	// Use batch insert for efficiency
	batch, err := s.conn.PrepareBatch(ctx, `
		INSERT INTO spans (id, trace_id, parent_span_id, type, name, input, output,
		                   input_tokens, output_tokens, cost_usd, duration_ms, status,
		                   error_message, model, provider, metadata, started_at, ended_at,
		                   stop_reason, cache_read_tokens, cache_write_tokens,
		                   reasoning_tokens, first_token_ms, thinking)
	`)
	if err != nil {
		return err
	}

	for i := range spans {
		span := &spans[i]
		if span.ID == "" {
			span.ID = uuid.New().String()
		}

		inputJSON, _ := json.Marshal(span.Input)
		outputJSON, _ := json.Marshal(span.Output)
		metadataJSON, _ := json.Marshal(span.Metadata)

		var parentSpanID *uuid.UUID
		if span.ParentSpanID != nil {
			pid := uuid.MustParse(*span.ParentSpanID)
			parentSpanID = &pid
		}

		err := batch.Append(
			uuid.MustParse(span.ID), uuid.MustParse(span.TraceID), parentSpanID,
			string(span.Type), span.Name, string(inputJSON), string(outputJSON),
			span.InputTokens, span.OutputTokens, span.CostUSD, span.DurationMs,
			string(span.Status), span.ErrorMessage, span.Model, span.Provider,
			string(metadataJSON), span.StartedAt, span.EndedAt,
			span.StopReason, span.CacheReadTokens, span.CacheWriteTokens,
			span.ReasoningTokens, span.FirstTokenMs, span.Thinking,
		)
		if err != nil {
			return err
		}
	}

	return batch.Send()
}

// ============================================
// SESSION OPERATIONS
// ============================================

func (s *Store) ListSessions(ctx context.Context, projectID string, filter entity.SessionFilter) (*entity.Page[entity.Session], error) {
	where := []string{"t.project_id = ?", "t.session_id IS NOT NULL"}
	args := []any{uuid.MustParse(projectID)}

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
	var total uint64
	countQuery := fmt.Sprintf(`
		SELECT count(DISTINCT t.session_id) FROM traces FINAL AS t WHERE %s
	`, whereClause)
	if err := s.conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
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
			max(t.user_id) as user_id,
			count(DISTINCT t.id) as trace_count,
			count(s.id) as total_spans,
			sum(coalesce(s.input_tokens, 0) + coalesce(s.output_tokens, 0)) as total_tokens,
			sum(coalesce(s.cost_usd, 0)) as total_cost,
			sum(coalesce(s.duration_ms, 0)) as total_duration,
			max(if(t.status = 'error', 1, 0)) as has_error,
			max(if(t.status = 'active', 1, 0)) as has_active,
			min(t.created_at) as first_trace_at,
			max(t.created_at) as last_trace_at
		FROM traces FINAL AS t
		LEFT JOIN spans AS s ON s.trace_id = t.id
		WHERE %s
		GROUP BY t.session_id
		ORDER BY max(t.created_at) DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := s.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := make([]entity.Session, 0) // Initialize as empty slice, not nil
	for rows.Next() {
		var sess entity.Session
		var hasError, hasActive uint8

		err := rows.Scan(&sess.SessionID, &sess.UserID, &sess.TraceCount, &sess.TotalSpans,
			&sess.TotalTokens, &sess.TotalCostUSD, &sess.TotalDurationMs,
			&hasError, &hasActive, &sess.FirstTraceAt, &sess.LastTraceAt)
		if err != nil {
			return nil, err
		}

		sess.HasError = hasError == 1
		sess.HasActive = hasActive == 1

		sessions = append(sessions, sess)
	}

	return &entity.Page[entity.Session]{
		Data:   sessions,
		Total:  int(total),
		Limit:  limit,
		Offset: offset,
	}, nil
}

// ============================================
// ANALYTICS OPERATIONS (Optimized for ClickHouse)
// ============================================

func (s *Store) GetStats(ctx context.Context, projectID string, period entity.Period) (*entity.Stats, error) {
	// ClickHouse is optimized for these aggregate queries
	query := `
		SELECT
			count(DISTINCT t.id) as total_traces,
			count(s.id) as total_spans,
			sum(coalesce(s.input_tokens, 0) + coalesce(s.output_tokens, 0)) as total_tokens,
			sum(coalesce(s.cost_usd, 0)) as total_cost,
			avg(s.duration_ms) as avg_duration,
			countIf(t.status = 'error') as error_count
		FROM traces FINAL AS t
		LEFT JOIN spans AS s ON s.trace_id = t.id
		WHERE t.project_id = ? AND t.created_at >= ? AND t.created_at <= ?
	`

	var stats entity.Stats
	var errorCount uint64
	var avgDuration float64

	err := s.conn.QueryRow(ctx, query, uuid.MustParse(projectID), period.From, period.To).Scan(
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
	// ClickHouse has specialized time functions
	var dateExpr string
	switch opts.Granularity {
	case "hour":
		dateExpr = "toStartOfHour(t.created_at)"
	case "week":
		dateExpr = "toStartOfWeek(t.created_at)"
	default: // day
		dateExpr = "toDate(t.created_at)"
	}

	query := fmt.Sprintf(`
		SELECT
			%s as date,
			count(DISTINCT t.id) as traces,
			count(s.id) as spans,
			sum(coalesce(s.input_tokens, 0) + coalesce(s.output_tokens, 0)) as tokens,
			sum(coalesce(s.cost_usd, 0)) as cost
		FROM traces FINAL AS t
		LEFT JOIN spans AS s ON s.trace_id = t.id
		WHERE t.project_id = ? AND t.created_at >= ? AND t.created_at <= ?
		GROUP BY %s
		ORDER BY date
	`, dateExpr, dateExpr)

	rows, err := s.conn.Query(ctx, query, uuid.MustParse(projectID), opts.From, opts.To)
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
