package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/ee/server/domain/entity"
)

// Store implements enterprise repository interfaces
// It wraps the core store and adds enterprise-specific methods
type Store struct {
	db       *sql.DB
	coreStore repository.Store
}

// New creates a new enterprise store wrapping the core store
func New(coreStore repository.Store, db *sql.DB) *Store {
	return &Store{
		db:        db,
		coreStore: coreStore,
	}
}

// CoreStore returns the underlying core store
func (s *Store) CoreStore() repository.Store {
	return s.coreStore
}

// transactionKey is used to store transaction in context
type transactionKey struct{}

// WithTransaction executes fn within a database transaction
func (s *Store) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Store transaction in context
	txCtx := context.WithValue(ctx, transactionKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// getExecutor returns the transaction from context or the default db
func (s *Store) getExecutor(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if tx, ok := ctx.Value(transactionKey{}).(*sql.Tx); ok {
		return tx
	}
	return s.db
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// MigrateEnterprise runs enterprise-specific migrations
func (s *Store) MigrateEnterprise(ctx context.Context) error {
	migrations := []string{
		// Organizations table
		`CREATE TABLE IF NOT EXISTS organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT UNIQUE NOT NULL,
			owner_user_id TEXT NOT NULL,
			plan TEXT DEFAULT 'free',
			settings TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Team Members table
		`CREATE TABLE IF NOT EXISTS team_members (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('owner','admin','member','viewer')),
			invited_by TEXT,
			invited_at DATETIME,
			joined_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, user_id)
		)`,

		// Subscriptions table
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id TEXT PRIMARY KEY,
			organization_id TEXT UNIQUE NOT NULL REFERENCES organizations(id),
			plan TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			lemonsqueezy_id TEXT,
			customer_id TEXT,
			current_period_start DATETIME,
			current_period_end DATETIME,
			cancelled_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Usage tracking table
		`CREATE TABLE IF NOT EXISTS usage (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL REFERENCES organizations(id),
			month TEXT NOT NULL,
			traces_used INTEGER DEFAULT 0,
			spans_used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(organization_id, month)
		)`,

		// Link projects to organizations (if column doesn't exist)
		`ALTER TABLE projects ADD COLUMN organization_id TEXT REFERENCES organizations(id)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug)`,
		`CREATE INDEX IF NOT EXISTS idx_organizations_owner ON organizations(owner_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_org ON team_members(organization_id)`,
		`CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_ls_id ON subscriptions(lemonsqueezy_id)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_org ON projects(organization_id)`,
		`CREATE INDEX IF NOT EXISTS idx_usage_org_month ON usage(organization_id, month)`,
	}

	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			// Ignore "duplicate column name" errors for ALTER TABLE
			if strings.Contains(m, "ALTER TABLE") && strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			// Ignore "already exists" errors for CREATE INDEX
			if strings.Contains(m, "CREATE INDEX") && strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("enterprise migration failed: %w", err)
		}
	}

	return nil
}

// ============================================
// ORGANIZATION OPERATIONS
// ============================================

func (s *Store) CreateOrganization(ctx context.Context, org *entity.Organization) error {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	settings, err := json.Marshal(org.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO organizations (id, name, slug, owner_user_id, plan, settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, org.ID, org.Name, org.Slug, org.OwnerUserID, org.Plan, settings, org.CreatedAt, org.UpdatedAt)

	return err
}

func (s *Store) GetOrganizationByID(ctx context.Context, id string) (*entity.Organization, error) {
	var org entity.Organization
	var settings string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, owner_user_id, plan, settings, created_at, updated_at
		FROM organizations WHERE id = ?
	`, id).Scan(&org.ID, &org.Name, &org.Slug, &org.OwnerUserID, &org.Plan, &settings, &org.CreatedAt, &org.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(settings), &org.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return &org, nil
}

func (s *Store) GetOrganizationBySlug(ctx context.Context, slug string) (*entity.Organization, error) {
	var org entity.Organization
	var settings string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, owner_user_id, plan, settings, created_at, updated_at
		FROM organizations WHERE slug = ?
	`, slug).Scan(&org.ID, &org.Name, &org.Slug, &org.OwnerUserID, &org.Plan, &settings, &org.CreatedAt, &org.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(settings), &org.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return &org, nil
}

func (s *Store) UpdateOrganization(ctx context.Context, id string, updates *entity.OrganizationUpdate) error {
	var setClauses []string
	var args []interface{}

	if updates.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.Plan != nil {
		setClauses = append(setClauses, "plan = ?")
		args = append(args, *updates.Plan)
	}
	if updates.Settings != nil {
		settings, err := json.Marshal(updates.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}
		setClauses = append(setClauses, "settings = ?")
		args = append(args, settings)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE organizations SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *Store) DeleteOrganization(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM organizations WHERE id = ?", id)
	return err
}

func (s *Store) ListOrganizationsByUser(ctx context.Context, userID string) ([]entity.Organization, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.name, o.slug, o.owner_user_id, o.plan, o.settings, o.created_at, o.updated_at
		FROM organizations o
		INNER JOIN team_members tm ON tm.organization_id = o.id
		WHERE tm.user_id = ? AND tm.joined_at IS NOT NULL
		ORDER BY o.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []entity.Organization
	for rows.Next() {
		var org entity.Organization
		var settings string
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.OwnerUserID, &org.Plan, &settings, &org.CreatedAt, &org.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(settings), &org.Settings); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}

// ============================================
// TEAM MEMBER OPERATIONS
// ============================================

func (s *Store) AddMember(ctx context.Context, member *entity.TeamMember) error {
	if member.ID == "" {
		member.ID = uuid.New().String()
	}
	member.CreatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO team_members (id, organization_id, user_id, role, invited_by, invited_at, joined_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, member.ID, member.OrganizationID, member.UserID, member.Role, member.InvitedBy, member.InvitedAt, member.JoinedAt, member.CreatedAt)

	return err
}

func (s *Store) GetMember(ctx context.Context, orgID, userID string) (*entity.TeamMember, error) {
	var m entity.TeamMember
	var invitedAt, joinedAt sql.NullTime
	var invitedBy sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, organization_id, user_id, role, invited_by, invited_at, joined_at, created_at
		FROM team_members WHERE organization_id = ? AND user_id = ?
	`, orgID, userID).Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &invitedBy, &invitedAt, &joinedAt, &m.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if invitedBy.Valid {
		m.InvitedBy = invitedBy.String
	}
	if invitedAt.Valid {
		m.InvitedAt = invitedAt.Time
	}
	if joinedAt.Valid {
		m.JoinedAt = &joinedAt.Time
	}

	return &m, nil
}

func (s *Store) UpdateMember(ctx context.Context, orgID, userID string, role entity.Role) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE team_members SET role = ? WHERE organization_id = ? AND user_id = ?
	`, role, orgID, userID)
	return err
}

func (s *Store) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM team_members WHERE organization_id = ? AND user_id = ?
	`, orgID, userID)
	return err
}

func (s *Store) ListMembers(ctx context.Context, orgID string) ([]entity.TeamMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, organization_id, user_id, role, invited_by, invited_at, joined_at, created_at
		FROM team_members WHERE organization_id = ?
		ORDER BY created_at ASC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []entity.TeamMember
	for rows.Next() {
		var m entity.TeamMember
		var invitedAt, joinedAt sql.NullTime
		var invitedBy sql.NullString

		if err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &invitedBy, &invitedAt, &joinedAt, &m.CreatedAt); err != nil {
			return nil, err
		}

		if invitedBy.Valid {
			m.InvitedBy = invitedBy.String
		}
		if invitedAt.Valid {
			m.InvitedAt = invitedAt.Time
		}
		if joinedAt.Valid {
			m.JoinedAt = &joinedAt.Time
		}

		members = append(members, m)
	}

	return members, rows.Err()
}

func (s *Store) GetUserOrganizations(ctx context.Context, userID string) ([]entity.TeamMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, organization_id, user_id, role, invited_by, invited_at, joined_at, created_at
		FROM team_members WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []entity.TeamMember
	for rows.Next() {
		var m entity.TeamMember
		var invitedAt, joinedAt sql.NullTime
		var invitedBy sql.NullString

		if err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.Role, &invitedBy, &invitedAt, &joinedAt, &m.CreatedAt); err != nil {
			return nil, err
		}

		if invitedBy.Valid {
			m.InvitedBy = invitedBy.String
		}
		if invitedAt.Valid {
			m.InvitedAt = invitedAt.Time
		}
		if joinedAt.Valid {
			m.JoinedAt = &joinedAt.Time
		}

		members = append(members, m)
	}

	return members, rows.Err()
}

func (s *Store) CountMembers(ctx context.Context, orgID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM team_members WHERE organization_id = ? AND joined_at IS NOT NULL
	`, orgID).Scan(&count)
	return count, err
}

// ============================================
// SUBSCRIPTION OPERATIONS
// ============================================

func (s *Store) CreateSubscription(ctx context.Context, sub *entity.Subscription) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO subscriptions (id, organization_id, plan, status, lemonsqueezy_id, customer_id,
			current_period_start, current_period_end, cancelled_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sub.ID, sub.OrganizationID, sub.Plan, sub.Status, sub.LemonSqueezyID, sub.CustomerID,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CancelledAt, sub.CreatedAt, sub.UpdatedAt)

	return err
}

func (s *Store) GetSubscriptionByOrgID(ctx context.Context, orgID string) (*entity.Subscription, error) {
	var sub entity.Subscription
	var cancelledAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, organization_id, plan, status, lemonsqueezy_id, customer_id,
			current_period_start, current_period_end, cancelled_at, created_at, updated_at
		FROM subscriptions WHERE organization_id = ?
	`, orgID).Scan(&sub.ID, &sub.OrganizationID, &sub.Plan, &sub.Status, &sub.LemonSqueezyID,
		&sub.CustomerID, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &cancelledAt,
		&sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if cancelledAt.Valid {
		sub.CancelledAt = &cancelledAt.Time
	}

	return &sub, nil
}

func (s *Store) GetSubscriptionByLemonSqueezyID(ctx context.Context, lsID string) (*entity.Subscription, error) {
	var sub entity.Subscription
	var cancelledAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, organization_id, plan, status, lemonsqueezy_id, customer_id,
			current_period_start, current_period_end, cancelled_at, created_at, updated_at
		FROM subscriptions WHERE lemonsqueezy_id = ?
	`, lsID).Scan(&sub.ID, &sub.OrganizationID, &sub.Plan, &sub.Status, &sub.LemonSqueezyID,
		&sub.CustomerID, &sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &cancelledAt,
		&sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if cancelledAt.Valid {
		sub.CancelledAt = &cancelledAt.Time
	}

	return &sub, nil
}

func (s *Store) UpdateSubscription(ctx context.Context, id string, updates *entity.SubscriptionUpdate) error {
	var setClauses []string
	var args []interface{}

	if updates.Plan != nil {
		setClauses = append(setClauses, "plan = ?")
		args = append(args, *updates.Plan)
	}
	if updates.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, *updates.Status)
	}
	if updates.CurrentPeriodStart != nil {
		setClauses = append(setClauses, "current_period_start = ?")
		args = append(args, *updates.CurrentPeriodStart)
	}
	if updates.CurrentPeriodEnd != nil {
		setClauses = append(setClauses, "current_period_end = ?")
		args = append(args, *updates.CurrentPeriodEnd)
	}
	if updates.CancelledAt != nil {
		setClauses = append(setClauses, "cancelled_at = ?")
		args = append(args, *updates.CancelledAt)
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// ============================================
// USAGE OPERATIONS
// ============================================

func (s *Store) Increment(ctx context.Context, orgID string, traces, spans int) error {
	month := time.Now().Format("2006-01")
	now := time.Now()
	db := s.getExecutor(ctx)

	// SQLite UPSERT - atomic operation, no race condition
	_, err := db.ExecContext(ctx, `
		INSERT INTO usage (id, organization_id, month, traces_used, spans_used, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(organization_id, month) DO UPDATE SET
			traces_used = traces_used + excluded.traces_used,
			spans_used = spans_used + excluded.spans_used,
			updated_at = excluded.updated_at
	`, uuid.New().String(), orgID, month, traces, spans, now, now)

	return err
}

func (s *Store) GetCurrentMonth(ctx context.Context, orgID string) (*entity.Usage, error) {
	month := time.Now().Format("2006-01")
	return s.GetByMonth(ctx, orgID, month)
}

func (s *Store) GetByMonth(ctx context.Context, orgID, month string) (*entity.Usage, error) {
	var u entity.Usage

	err := s.db.QueryRowContext(ctx, `
		SELECT id, organization_id, month, traces_used, spans_used, created_at, updated_at
		FROM usage WHERE organization_id = ? AND month = ?
	`, orgID, month).Scan(&u.ID, &u.OrganizationID, &u.Month, &u.TracesUsed, &u.SpansUsed, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		// Return empty usage if not found
		return &entity.Usage{
			OrganizationID: orgID,
			Month:          month,
			TracesUsed:     0,
			SpansUsed:      0,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// ============================================
// ANALYTICS OPERATIONS
// ============================================

// GetCostBreakdownByTags returns cost analytics grouped by tag for a project
func (s *Store) GetCostBreakdownByTags(ctx context.Context, projectID string, filter entity.CostBreakdownFilter) (*entity.CostBreakdownResult, error) {
	// Build query for traces with date filters
	query := `
		SELECT tags, total_cost_usd, total_tokens
		FROM traces
		WHERE project_id = ?
	`
	args := []interface{}{projectID}

	if filter.From != nil {
		query += " AND created_at >= ?"
		args = append(args, filter.From)
	}
	if filter.To != nil {
		query += " AND created_at <= ?"
		args = append(args, filter.To)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}
	defer rows.Close()

	// Aggregate costs by tag
	tagCosts := make(map[string]*entity.CostBreakdown)
	var totalCost float64
	var totalTokens int
	var totalTraces int

	for rows.Next() {
		var tagsJSON string
		var costUSD float64
		var tokens int

		if err := rows.Scan(&tagsJSON, &costUSD, &tokens); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		totalCost += costUSD
		totalTokens += tokens
		totalTraces++

		// Parse tags JSON
		var tags []string
		if tagsJSON != "" {
			if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
				continue // Skip malformed tags
			}
		}

		// Aggregate by each tag
		for _, tag := range tags {
			// Apply tag prefix filter if specified
			if filter.TagPrefix != "" && !strings.HasPrefix(tag, filter.TagPrefix) {
				continue
			}

			if _, ok := tagCosts[tag]; !ok {
				tagCosts[tag] = &entity.CostBreakdown{Tag: tag}
			}
			tagCosts[tag].TotalCost += costUSD
			tagCosts[tag].TotalTokens += tokens
			tagCosts[tag].TraceCount++
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Convert map to slice and calculate percentages
	breakdowns := make([]entity.CostBreakdown, 0, len(tagCosts))
	for _, bd := range tagCosts {
		if totalCost > 0 {
			bd.Percentage = (bd.TotalCost / totalCost) * 100
		}
		breakdowns = append(breakdowns, *bd)
	}

	// Sort by cost descending
	sortCostBreakdowns(breakdowns)

	// Apply limit
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	if len(breakdowns) > limit {
		breakdowns = breakdowns[:limit]
	}

	return &entity.CostBreakdownResult{
		Breakdowns:  breakdowns,
		TotalCost:   totalCost,
		TotalTokens: totalTokens,
		TotalTraces: totalTraces,
		From:        filter.From,
		To:          filter.To,
	}, nil
}

// sortCostBreakdowns sorts breakdowns by TotalCost descending
func sortCostBreakdowns(breakdowns []entity.CostBreakdown) {
	for i := 0; i < len(breakdowns)-1; i++ {
		for j := i + 1; j < len(breakdowns); j++ {
			if breakdowns[j].TotalCost > breakdowns[i].TotalCost {
				breakdowns[i], breakdowns[j] = breakdowns[j], breakdowns[i]
			}
		}
	}
}

// GetErrorMetrics returns error rate analytics for a project
func (s *Store) GetErrorMetrics(ctx context.Context, projectID string, filter entity.ErrorFilter) (*entity.ErrorMetrics, error) {
	// Build base query for traces
	baseWhere := "WHERE project_id = ?"
	args := []interface{}{projectID}

	if filter.From != nil {
		baseWhere += " AND created_at >= ?"
		args = append(args, filter.From)
	}
	if filter.To != nil {
		baseWhere += " AND created_at <= ?"
		args = append(args, filter.To)
	}

	// 1. Get total and error trace counts
	var totalTraces, errorTraces int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as errors
		FROM traces `+baseWhere, args...).Scan(&totalTraces, &errorTraces)
	if err != nil {
		return nil, fmt.Errorf("failed to count traces: %w", err)
	}

	// Calculate overall error rate
	var errorRate float64
	if totalTraces > 0 {
		errorRate = float64(errorTraces) / float64(totalTraces) * 100
	}

	// 2. Get error rate by tag
	rows, err := s.db.QueryContext(ctx, `
		SELECT tags, status
		FROM traces `+baseWhere, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces for tags: %w", err)
	}
	defer rows.Close()

	// Aggregate by tag
	type tagStats struct {
		total  int
		errors int
	}
	tagMap := make(map[string]*tagStats)

	for rows.Next() {
		var tagsJSON string
		var status string
		if err := rows.Scan(&tagsJSON, &status); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var tags []string
		if tagsJSON != "" {
			if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
				continue
			}
		}

		isError := status == "error"

		for _, tag := range tags {
			if filter.TagPrefix != "" && !strings.HasPrefix(tag, filter.TagPrefix) {
				continue
			}
			if _, ok := tagMap[tag]; !ok {
				tagMap[tag] = &tagStats{}
			}
			tagMap[tag].total++
			if isError {
				tagMap[tag].errors++
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Convert to slice and calculate rates
	byTag := make([]entity.TagErrorRate, 0, len(tagMap))
	for tag, stats := range tagMap {
		rate := 0.0
		if stats.total > 0 {
			rate = float64(stats.errors) / float64(stats.total) * 100
		}
		byTag = append(byTag, entity.TagErrorRate{
			Tag:         tag,
			TotalTraces: stats.total,
			ErrorTraces: stats.errors,
			ErrorRate:   rate,
		})
	}

	// Sort by error rate descending
	sortTagErrorRates(byTag)

	// 3. Get top error messages from spans
	topLimit := filter.TopLimit
	if topLimit <= 0 {
		topLimit = 10
	}

	// Query spans with error messages, joined with traces for tags
	topErrors, err := s.getTopErrors(ctx, projectID, filter, topLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top errors: %w", err)
	}

	return &entity.ErrorMetrics{
		TotalTraces: totalTraces,
		ErrorTraces: errorTraces,
		ErrorRate:   errorRate,
		ByTag:       byTag,
		TopErrors:   topErrors,
		From:        filter.From,
		To:          filter.To,
	}, nil
}

// getTopErrors retrieves the most common error messages
func (s *Store) getTopErrors(ctx context.Context, projectID string, filter entity.ErrorFilter, limit int) ([]entity.ErrorSummary, error) {
	query := `
		SELECT s.error_message, COUNT(*) as count, MAX(s.started_at) as last_occurred, t.tags
		FROM spans s
		INNER JOIN traces t ON s.trace_id = t.id
		WHERE t.project_id = ? AND s.status = 'error' AND s.error_message IS NOT NULL AND s.error_message != ''
	`
	args := []interface{}{projectID}

	if filter.From != nil {
		query += " AND s.started_at >= ?"
		args = append(args, filter.From)
	}
	if filter.To != nil {
		query += " AND s.started_at <= ?"
		args = append(args, filter.To)
	}

	query += " GROUP BY s.error_message ORDER BY count DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Aggregate errors with their tags
	errorMap := make(map[string]*entity.ErrorSummary)
	var orderedMessages []string

	for rows.Next() {
		var message string
		var count int
		var lastOccurredStr string
		var tagsJSON sql.NullString

		if err := rows.Scan(&message, &count, &lastOccurredStr, &tagsJSON); err != nil {
			return nil, err
		}

		// Parse time from string (SQLite stores as string)
		lastOccurred, _ := time.Parse("2006-01-02 15:04:05", lastOccurredStr)
		if lastOccurred.IsZero() {
			// Try RFC3339 format
			lastOccurred, _ = time.Parse(time.RFC3339, lastOccurredStr)
		}

		if existing, ok := errorMap[message]; ok {
			existing.Count += count
			if lastOccurred.After(existing.LastOccurred) {
				existing.LastOccurred = lastOccurred
			}
			// Add tags
			if tagsJSON.Valid && tagsJSON.String != "" {
				var tags []string
				if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err == nil {
					for _, tag := range tags {
						found := false
						for _, t := range existing.AffectedTags {
							if t == tag {
								found = true
								break
							}
						}
						if !found && len(existing.AffectedTags) < 5 {
							existing.AffectedTags = append(existing.AffectedTags, tag)
						}
					}
				}
			}
		} else {
			summary := &entity.ErrorSummary{
				Message:      message,
				Count:        count,
				LastOccurred: lastOccurred,
				AffectedTags: []string{},
			}
			if tagsJSON.Valid && tagsJSON.String != "" {
				var tags []string
				if err := json.Unmarshal([]byte(tagsJSON.String), &tags); err == nil {
					if len(tags) > 5 {
						tags = tags[:5]
					}
					summary.AffectedTags = tags
				}
			}
			errorMap[message] = summary
			orderedMessages = append(orderedMessages, message)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build result in order
	result := make([]entity.ErrorSummary, 0, len(orderedMessages))
	for _, msg := range orderedMessages {
		if summary, ok := errorMap[msg]; ok {
			result = append(result, *summary)
		}
	}

	return result, nil
}

// sortTagErrorRates sorts by error rate descending
func sortTagErrorRates(rates []entity.TagErrorRate) {
	for i := 0; i < len(rates)-1; i++ {
		for j := i + 1; j < len(rates); j++ {
			if rates[j].ErrorRate > rates[i].ErrorRate {
				rates[i], rates[j] = rates[j], rates[i]
			}
		}
	}
}
