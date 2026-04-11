package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/lelemon/ee/server/domain/entity"
)

// Helper to setup test DB with traces and spans tables

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create traces table (minimal schema for testing)
	_, err = db.Exec(`
		CREATE TABLE traces (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			tags TEXT,
			total_cost_usd REAL DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create traces table: %v", err)
	}

	return db
}

func insertTrace(t *testing.T, db *sql.DB, id, projectID string, tags []string, cost float64, tokens int, createdAt time.Time) {
	tagsJSON, _ := json.Marshal(tags)
	_, err := db.Exec(`
		INSERT INTO traces (id, project_id, tags, total_cost_usd, total_tokens, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, projectID, string(tagsJSON), cost, tokens, createdAt)
	if err != nil {
		t.Fatalf("failed to insert trace: %v", err)
	}
}

func TestGetCostBreakdownByTags(t *testing.T) {
	ctx := context.Background()

	t.Run("returns breakdown grouped by tag", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// Insert traces with different tags
		insertTrace(t, db, "t1", projectID, []string{"org:acme", "campaign:123"}, 1.50, 1000, time.Now())
		insertTrace(t, db, "t2", projectID, []string{"org:acme", "campaign:456"}, 2.00, 1500, time.Now())
		insertTrace(t, db, "t3", projectID, []string{"org:xyz", "campaign:789"}, 0.50, 500, time.Now())

		result, err := store.GetCostBreakdownByTags(ctx, projectID, entity.NewCostBreakdownFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TotalTraces != 3 {
			t.Errorf("expected 3 total traces, got %d", result.TotalTraces)
		}

		if result.TotalCost != 4.0 {
			t.Errorf("expected total cost 4.0, got %f", result.TotalCost)
		}

		if result.TotalTokens != 3000 {
			t.Errorf("expected 3000 total tokens, got %d", result.TotalTokens)
		}

		// Should have 5 unique tags: org:acme, org:xyz, campaign:123, campaign:456, campaign:789
		if len(result.Breakdowns) != 5 {
			t.Errorf("expected 5 breakdowns, got %d", len(result.Breakdowns))
		}

		// Check org:acme has 2 traces and total of 3.50
		var acmeBreakdown *entity.CostBreakdown
		for i := range result.Breakdowns {
			if result.Breakdowns[i].Tag == "org:acme" {
				acmeBreakdown = &result.Breakdowns[i]
				break
			}
		}
		if acmeBreakdown == nil {
			t.Fatal("org:acme breakdown not found")
		}
		if acmeBreakdown.TraceCount != 2 {
			t.Errorf("expected 2 traces for org:acme, got %d", acmeBreakdown.TraceCount)
		}
		if acmeBreakdown.TotalCost != 3.50 {
			t.Errorf("expected cost 3.50 for org:acme, got %f", acmeBreakdown.TotalCost)
		}
	})

	t.Run("filters by tag prefix", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		insertTrace(t, db, "t1", projectID, []string{"org:acme", "campaign:123"}, 1.50, 1000, time.Now())
		insertTrace(t, db, "t2", projectID, []string{"org:xyz", "campaign:456"}, 2.00, 1500, time.Now())

		filter := entity.CostBreakdownFilter{TagPrefix: "org:"}
		result, err := store.GetCostBreakdownByTags(ctx, projectID, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have org: prefixed tags
		if len(result.Breakdowns) != 2 {
			t.Errorf("expected 2 breakdowns (org:acme, org:xyz), got %d", len(result.Breakdowns))
		}

		for _, bd := range result.Breakdowns {
			if bd.Tag != "org:acme" && bd.Tag != "org:xyz" {
				t.Errorf("unexpected tag: %s", bd.Tag)
			}
		}
	})

	t.Run("filters by date range", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// Insert traces at different times
		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		weekAgo := now.Add(-7 * 24 * time.Hour)

		insertTrace(t, db, "t1", projectID, []string{"recent"}, 1.00, 100, now)
		insertTrace(t, db, "t2", projectID, []string{"yesterday"}, 2.00, 200, yesterday)
		insertTrace(t, db, "t3", projectID, []string{"old"}, 3.00, 300, weekAgo)

		// Filter to last 2 days
		from := now.Add(-48 * time.Hour)
		filter := entity.CostBreakdownFilter{From: &from}
		result, err := store.GetCostBreakdownByTags(ctx, projectID, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have 2 traces (recent and yesterday)
		if result.TotalTraces != 2 {
			t.Errorf("expected 2 traces, got %d", result.TotalTraces)
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// Insert traces with many different tags
		for i := 0; i < 20; i++ {
			tag := []string{"tag:" + string(rune('a'+i))}
			insertTrace(t, db, "t"+string(rune('a'+i)), projectID, tag, 1.0, 100, time.Now())
		}

		filter := entity.CostBreakdownFilter{Limit: 5}
		result, err := store.GetCostBreakdownByTags(ctx, projectID, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Breakdowns) != 5 {
			t.Errorf("expected 5 breakdowns (limited), got %d", len(result.Breakdowns))
		}
	})

	t.Run("returns empty for project with no traces", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}

		result, err := store.GetCostBreakdownByTags(ctx, "empty-project", entity.NewCostBreakdownFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TotalTraces != 0 {
			t.Errorf("expected 0 traces, got %d", result.TotalTraces)
		}

		if len(result.Breakdowns) != 0 {
			t.Errorf("expected 0 breakdowns, got %d", len(result.Breakdowns))
		}
	})

	t.Run("handles traces without tags", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// Insert trace without tags
		insertTrace(t, db, "t1", projectID, nil, 1.00, 100, time.Now())
		insertTrace(t, db, "t2", projectID, []string{}, 2.00, 200, time.Now())
		insertTrace(t, db, "t3", projectID, []string{"has:tag"}, 3.00, 300, time.Now())

		result, err := store.GetCostBreakdownByTags(ctx, projectID, entity.NewCostBreakdownFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Total should include all traces
		if result.TotalTraces != 3 {
			t.Errorf("expected 3 total traces, got %d", result.TotalTraces)
		}

		// Only one tag from tagged trace
		if len(result.Breakdowns) != 1 {
			t.Errorf("expected 1 breakdown (has:tag), got %d", len(result.Breakdowns))
		}
	})

	t.Run("calculates percentage correctly", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// Insert traces: 75% to tag-a, 25% to tag-b
		insertTrace(t, db, "t1", projectID, []string{"tag-a"}, 3.00, 300, time.Now())
		insertTrace(t, db, "t2", projectID, []string{"tag-b"}, 1.00, 100, time.Now())

		result, err := store.GetCostBreakdownByTags(ctx, projectID, entity.NewCostBreakdownFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var tagA, tagB *entity.CostBreakdown
		for i := range result.Breakdowns {
			if result.Breakdowns[i].Tag == "tag-a" {
				tagA = &result.Breakdowns[i]
			} else if result.Breakdowns[i].Tag == "tag-b" {
				tagB = &result.Breakdowns[i]
			}
		}

		if tagA == nil || tagB == nil {
			t.Fatal("missing expected breakdowns")
		}

		// Check percentages (allowing small floating point error)
		if tagA.Percentage < 74.9 || tagA.Percentage > 75.1 {
			t.Errorf("expected tag-a percentage ~75%%, got %f%%", tagA.Percentage)
		}
		if tagB.Percentage < 24.9 || tagB.Percentage > 25.1 {
			t.Errorf("expected tag-b percentage ~25%%, got %f%%", tagB.Percentage)
		}
	})

	t.Run("sorted by cost descending", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		insertTrace(t, db, "t1", projectID, []string{"low"}, 1.00, 100, time.Now())
		insertTrace(t, db, "t2", projectID, []string{"high"}, 10.00, 1000, time.Now())
		insertTrace(t, db, "t3", projectID, []string{"medium"}, 5.00, 500, time.Now())

		result, err := store.GetCostBreakdownByTags(ctx, projectID, entity.NewCostBreakdownFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Breakdowns) < 3 {
			t.Fatalf("expected at least 3 breakdowns, got %d", len(result.Breakdowns))
		}

		// First should be highest cost
		if result.Breakdowns[0].Tag != "high" {
			t.Errorf("expected first breakdown to be 'high', got '%s'", result.Breakdowns[0].Tag)
		}
		if result.Breakdowns[1].Tag != "medium" {
			t.Errorf("expected second breakdown to be 'medium', got '%s'", result.Breakdowns[1].Tag)
		}
		if result.Breakdowns[2].Tag != "low" {
			t.Errorf("expected third breakdown to be 'low', got '%s'", result.Breakdowns[2].Tag)
		}
	})

	t.Run("isolates projects", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &Store{db: db}

		insertTrace(t, db, "t1", "proj-1", []string{"proj1-tag"}, 1.00, 100, time.Now())
		insertTrace(t, db, "t2", "proj-2", []string{"proj2-tag"}, 2.00, 200, time.Now())

		result, err := store.GetCostBreakdownByTags(ctx, "proj-1", entity.NewCostBreakdownFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TotalTraces != 1 {
			t.Errorf("expected 1 trace for proj-1, got %d", result.TotalTraces)
		}

		if len(result.Breakdowns) != 1 || result.Breakdowns[0].Tag != "proj1-tag" {
			t.Error("expected only proj1-tag breakdown")
		}
	})
}

// ============================================
// ERROR ANALYTICS TESTS
// ============================================

func setupTestDBWithSpans(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create traces and spans tables
	_, err = db.Exec(`
		CREATE TABLE traces (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			tags TEXT,
			status TEXT DEFAULT 'completed',
			total_cost_usd REAL DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create traces table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE spans (
			id TEXT PRIMARY KEY,
			trace_id TEXT NOT NULL,
			status TEXT DEFAULT 'success',
			error_message TEXT,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (trace_id) REFERENCES traces(id)
		)
	`)
	if err != nil {
		t.Fatalf("failed to create spans table: %v", err)
	}

	return db
}

func insertTraceWithStatus(t *testing.T, db *sql.DB, id, projectID string, tags []string, status string, createdAt time.Time) {
	tagsJSON, _ := json.Marshal(tags)
	_, err := db.Exec(`
		INSERT INTO traces (id, project_id, tags, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, projectID, string(tagsJSON), status, createdAt)
	if err != nil {
		t.Fatalf("failed to insert trace: %v", err)
	}
}

func insertSpan(t *testing.T, db *sql.DB, id, traceID, status string, errorMsg *string, startedAt time.Time) {
	_, err := db.Exec(`
		INSERT INTO spans (id, trace_id, status, error_message, started_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, traceID, status, errorMsg, startedAt)
	if err != nil {
		t.Fatalf("failed to insert span: %v", err)
	}
}

func TestGetErrorMetrics(t *testing.T) {
	ctx := context.Background()

	t.Run("calculates error rate correctly", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// Insert 4 traces: 1 error, 3 completed = 25% error rate
		insertTraceWithStatus(t, db, "t1", projectID, []string{"tag1"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t2", projectID, []string{"tag1"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t3", projectID, []string{"tag1"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t4", projectID, []string{"tag1"}, "error", time.Now())

		result, err := store.GetErrorMetrics(ctx, projectID, entity.NewErrorFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TotalTraces != 4 {
			t.Errorf("expected 4 total traces, got %d", result.TotalTraces)
		}

		if result.ErrorTraces != 1 {
			t.Errorf("expected 1 error trace, got %d", result.ErrorTraces)
		}

		if result.ErrorRate < 24.9 || result.ErrorRate > 25.1 {
			t.Errorf("expected 25%% error rate, got %f%%", result.ErrorRate)
		}
	})

	t.Run("calculates error rate by tag", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		// tag-a: 2 completed, 0 errors = 0% error rate
		insertTraceWithStatus(t, db, "t1", projectID, []string{"tag-a"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t2", projectID, []string{"tag-a"}, "completed", time.Now())

		// tag-b: 1 completed, 1 error = 50% error rate
		insertTraceWithStatus(t, db, "t3", projectID, []string{"tag-b"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t4", projectID, []string{"tag-b"}, "error", time.Now())

		result, err := store.GetErrorMetrics(ctx, projectID, entity.NewErrorFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.ByTag) != 2 {
			t.Fatalf("expected 2 tag rates, got %d", len(result.ByTag))
		}

		// Results should be sorted by error rate descending
		// tag-b should be first (50% error rate)
		if result.ByTag[0].Tag != "tag-b" {
			t.Errorf("expected tag-b first (highest error rate), got %s", result.ByTag[0].Tag)
		}
		if result.ByTag[0].ErrorRate < 49.9 || result.ByTag[0].ErrorRate > 50.1 {
			t.Errorf("expected 50%% error rate for tag-b, got %f%%", result.ByTag[0].ErrorRate)
		}

		// tag-a should be second (0% error rate)
		if result.ByTag[1].Tag != "tag-a" {
			t.Errorf("expected tag-a second, got %s", result.ByTag[1].Tag)
		}
		if result.ByTag[1].ErrorRate != 0 {
			t.Errorf("expected 0%% error rate for tag-a, got %f%%", result.ByTag[1].ErrorRate)
		}
	})

	t.Run("filters by tag prefix", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		insertTraceWithStatus(t, db, "t1", projectID, []string{"org:acme"}, "error", time.Now())
		insertTraceWithStatus(t, db, "t2", projectID, []string{"org:xyz"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t3", projectID, []string{"campaign:123"}, "error", time.Now())

		filter := entity.ErrorFilter{TagPrefix: "org:"}
		result, err := store.GetErrorMetrics(ctx, projectID, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have org: tags
		if len(result.ByTag) != 2 {
			t.Errorf("expected 2 org: tags, got %d", len(result.ByTag))
		}

		for _, tagRate := range result.ByTag {
			if tagRate.Tag != "org:acme" && tagRate.Tag != "org:xyz" {
				t.Errorf("unexpected tag: %s", tagRate.Tag)
			}
		}
	})

	t.Run("returns top error messages", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		insertTraceWithStatus(t, db, "t1", projectID, []string{"tag1"}, "error", time.Now())
		insertTraceWithStatus(t, db, "t2", projectID, []string{"tag2"}, "error", time.Now())

		// Insert error spans
		errMsg1 := "Connection refused"
		errMsg2 := "Rate limit exceeded"
		insertSpan(t, db, "s1", "t1", "error", &errMsg1, time.Now())
		insertSpan(t, db, "s2", "t1", "error", &errMsg1, time.Now()) // Same error twice
		insertSpan(t, db, "s3", "t2", "error", &errMsg2, time.Now())

		result, err := store.GetErrorMetrics(ctx, projectID, entity.NewErrorFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.TopErrors) != 2 {
			t.Fatalf("expected 2 top errors, got %d", len(result.TopErrors))
		}

		// "Connection refused" should be first (count = 2)
		if result.TopErrors[0].Message != "Connection refused" {
			t.Errorf("expected 'Connection refused' first, got '%s'", result.TopErrors[0].Message)
		}
		if result.TopErrors[0].Count != 2 {
			t.Errorf("expected count 2 for first error, got %d", result.TopErrors[0].Count)
		}
	})

	t.Run("returns empty for project with no errors", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		insertTraceWithStatus(t, db, "t1", projectID, []string{"tag1"}, "completed", time.Now())
		insertTraceWithStatus(t, db, "t2", projectID, []string{"tag1"}, "completed", time.Now())

		result, err := store.GetErrorMetrics(ctx, projectID, entity.NewErrorFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ErrorRate != 0 {
			t.Errorf("expected 0%% error rate, got %f%%", result.ErrorRate)
		}

		if result.ErrorTraces != 0 {
			t.Errorf("expected 0 error traces, got %d", result.ErrorTraces)
		}
	})

	t.Run("isolates projects", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}

		insertTraceWithStatus(t, db, "t1", "proj-1", []string{"tag1"}, "error", time.Now())
		insertTraceWithStatus(t, db, "t2", "proj-2", []string{"tag2"}, "completed", time.Now())

		result, err := store.GetErrorMetrics(ctx, "proj-1", entity.NewErrorFilter())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TotalTraces != 1 {
			t.Errorf("expected 1 trace for proj-1, got %d", result.TotalTraces)
		}

		if result.ErrorRate != 100 {
			t.Errorf("expected 100%% error rate for proj-1, got %f%%", result.ErrorRate)
		}
	})

	t.Run("filters by date range", func(t *testing.T) {
		db := setupTestDBWithSpans(t)
		defer db.Close()

		store := &Store{db: db}
		projectID := "proj-1"

		now := time.Now()
		yesterday := now.Add(-24 * time.Hour)
		weekAgo := now.Add(-7 * 24 * time.Hour)

		insertTraceWithStatus(t, db, "t1", projectID, []string{"recent"}, "error", now)
		insertTraceWithStatus(t, db, "t2", projectID, []string{"old"}, "error", weekAgo)
		insertTraceWithStatus(t, db, "t3", projectID, []string{"yesterday"}, "completed", yesterday)

		from := now.Add(-48 * time.Hour)
		filter := entity.ErrorFilter{From: &from}
		result, err := store.GetErrorMetrics(ctx, projectID, filter)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have 2 traces (recent and yesterday)
		if result.TotalTraces != 2 {
			t.Errorf("expected 2 traces, got %d", result.TotalTraces)
		}

		// 1 error out of 2 = 50%
		if result.ErrorRate < 49.9 || result.ErrorRate > 50.1 {
			t.Errorf("expected 50%% error rate, got %f%%", result.ErrorRate)
		}
	})
}
