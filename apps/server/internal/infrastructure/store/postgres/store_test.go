package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/lelemon/server/internal/domain/entity"
)

// getTestDSN returns a PostgreSQL connection string for testing
// Set POSTGRES_TEST_URL env var or it will skip the test
func getTestDSN(t *testing.T) string {
	dsn := os.Getenv("POSTGRES_TEST_URL")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_URL not set, skipping PostgreSQL tests")
	}
	return dsn
}

func setupTestStore(t *testing.T) *Store {
	dsn := getTestDSN(t)

	store, err := New(dsn)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		store.Close()
		t.Fatalf("failed to migrate: %v", err)
	}

	// Clean up tables for fresh test
	store.pool.Exec(ctx, "DELETE FROM spans")
	store.pool.Exec(ctx, "DELETE FROM traces")
	store.pool.Exec(ctx, "DELETE FROM projects")
	store.pool.Exec(ctx, "DELETE FROM users")

	return store
}

func TestPostgresConnection(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestPostgresUserOperations(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("create and get user", func(t *testing.T) {
		user := &entity.User{
			Email:        fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
			Name:         "Test User",
			PasswordHash: ptrS("hashed"),
		}

		if err := store.CreateUser(ctx, user); err != nil {
			t.Fatalf("CreateUser failed: %v", err)
		}

		if user.ID == "" {
			t.Error("expected user ID to be set")
		}

		// Get by ID
		got, err := store.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetUserByID failed: %v", err)
		}
		if got.Email != user.Email {
			t.Errorf("email mismatch: got %s, want %s", got.Email, user.Email)
		}

		// Get by email
		got, err = store.GetUserByEmail(ctx, user.Email)
		if err != nil {
			t.Fatalf("GetUserByEmail failed: %v", err)
		}
		if got.ID != user.ID {
			t.Errorf("ID mismatch: got %s, want %s", got.ID, user.ID)
		}
	})

	t.Run("update user", func(t *testing.T) {
		user := &entity.User{
			Email:        fmt.Sprintf("update-%d@example.com", time.Now().UnixNano()),
			Name:         "Original Name",
			PasswordHash: ptrS("hash1"),
		}
		store.CreateUser(ctx, user)

		newName := "Updated Name"
		err := store.UpdateUser(ctx, user.ID, entity.UserUpdate{Name: &newName})
		if err != nil {
			t.Fatalf("UpdateUser failed: %v", err)
		}

		got, _ := store.GetUserByID(ctx, user.ID)
		if got.Name != newName {
			t.Errorf("name not updated: got %s, want %s", got.Name, newName)
		}
	})

	t.Run("not found error", func(t *testing.T) {
		_, err := store.GetUserByID(ctx, "00000000-0000-0000-0000-000000000000")
		if err != entity.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestPostgresProjectOperations(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	t.Run("create and get project", func(t *testing.T) {
		project := &entity.Project{
			Name:       "Test Project",
			APIKey:     fmt.Sprintf("le_test_%d", time.Now().UnixNano()),
			APIKeyHash: "hash123",
			OwnerEmail: "owner@example.com",
			Settings:   entity.ProjectSettings{RetentionDays: ptr(30)},
		}

		if err := store.CreateProject(ctx, project); err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		// Get by ID
		got, err := store.GetProjectByID(ctx, project.ID)
		if err != nil {
			t.Fatalf("GetProjectByID failed: %v", err)
		}
		if got.Name != project.Name {
			t.Errorf("name mismatch: got %s, want %s", got.Name, project.Name)
		}

		// Get by API key hash
		got, err = store.GetProjectByAPIKeyHash(ctx, project.APIKeyHash)
		if err != nil {
			t.Fatalf("GetProjectByAPIKeyHash failed: %v", err)
		}
		if got.ID != project.ID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("list projects by owner", func(t *testing.T) {
		owner := fmt.Sprintf("owner-%d@example.com", time.Now().UnixNano())

		for i := 0; i < 3; i++ {
			store.CreateProject(ctx, &entity.Project{
				Name:       fmt.Sprintf("Project %d", i),
				APIKey:     fmt.Sprintf("le_list_%d_%d", time.Now().UnixNano(), i),
				APIKeyHash: fmt.Sprintf("hash_%d_%d", time.Now().UnixNano(), i),
				OwnerEmail: owner,
			})
		}

		projects, err := store.ListProjectsByOwner(ctx, owner)
		if err != nil {
			t.Fatalf("ListProjectsByOwner failed: %v", err)
		}
		if len(projects) != 3 {
			t.Errorf("expected 3 projects, got %d", len(projects))
		}
	})

	t.Run("rotate API key", func(t *testing.T) {
		project := &entity.Project{
			Name:       "Rotate Test",
			APIKey:     fmt.Sprintf("le_rotate_%d", time.Now().UnixNano()),
			APIKeyHash: "old_hash",
			OwnerEmail: "rotate@example.com",
		}
		store.CreateProject(ctx, project)

		newKey := fmt.Sprintf("le_new_%d", time.Now().UnixNano())
		newHash := "new_hash"
		if err := store.RotateAPIKey(ctx, project.ID, newKey, newHash); err != nil {
			t.Fatalf("RotateAPIKey failed: %v", err)
		}

		got, _ := store.GetProjectByID(ctx, project.ID)
		if got.APIKey != newKey {
			t.Errorf("API key not rotated")
		}
	})
}

func TestPostgresTraceOperations(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Create a project first
	project := &entity.Project{
		Name:       "Trace Test Project",
		APIKey:     fmt.Sprintf("le_trace_%d", time.Now().UnixNano()),
		APIKeyHash: "trace_hash",
		OwnerEmail: "trace@example.com",
	}
	store.CreateProject(ctx, project)

	t.Run("create and get trace", func(t *testing.T) {
		sessionID := "session-123"
		userID := "user-456"
		trace := &entity.Trace{
			ProjectID: project.ID,
			SessionID: &sessionID,
			UserID:    &userID,
			Status:    entity.TraceStatusActive,
			Tags:      []string{"test", "postgres"},
			Metadata:  map[string]any{"source": "test"},
		}

		if err := store.CreateTrace(ctx, trace); err != nil {
			t.Fatalf("CreateTrace failed: %v", err)
		}

		got, err := store.GetTrace(ctx, project.ID, trace.ID)
		if err != nil {
			t.Fatalf("GetTrace failed: %v", err)
		}
		if got.Status != entity.TraceStatusActive {
			t.Errorf("status mismatch: got %s, want %s", got.Status, entity.TraceStatusActive)
		}
		if *got.SessionID != sessionID {
			t.Errorf("sessionID mismatch")
		}
	})

	t.Run("update trace status", func(t *testing.T) {
		trace := &entity.Trace{
			ProjectID: project.ID,
			Status:    entity.TraceStatusActive,
		}
		store.CreateTrace(ctx, trace)

		err := store.UpdateTraceStatus(ctx, project.ID, trace.ID, entity.TraceStatusCompleted)
		if err != nil {
			t.Fatalf("UpdateTraceStatus failed: %v", err)
		}

		got, _ := store.GetTrace(ctx, project.ID, trace.ID)
		if got.Status != entity.TraceStatusCompleted {
			t.Errorf("status not updated")
		}
	})

	t.Run("list traces with filters", func(t *testing.T) {
		// Create traces with different statuses
		for i := 0; i < 5; i++ {
			status := entity.TraceStatusCompleted
			if i%2 == 0 {
				status = entity.TraceStatusError
			}
			store.CreateTrace(ctx, &entity.Trace{
				ProjectID: project.ID,
				Status:    status,
			})
		}

		// List all
		page, err := store.ListTraces(ctx, project.ID, entity.TraceFilter{Limit: 100})
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if page.Total < 5 {
			t.Errorf("expected at least 5 traces, got %d", page.Total)
		}

		// Filter by status
		errorStatus := entity.TraceStatusError
		page, err = store.ListTraces(ctx, project.ID, entity.TraceFilter{
			Status: &errorStatus,
			Limit:  100,
		})
		if err != nil {
			t.Fatalf("ListTraces with filter failed: %v", err)
		}
		if page.Total < 3 {
			t.Errorf("expected at least 3 error traces")
		}
	})
}

func TestPostgresSpanOperations(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	project := &entity.Project{
		Name:       "Span Test",
		APIKey:     fmt.Sprintf("le_span_%d", time.Now().UnixNano()),
		APIKeyHash: "span_hash",
		OwnerEmail: "span@example.com",
	}
	store.CreateProject(ctx, project)

	trace := &entity.Trace{
		ProjectID: project.ID,
		Status:    entity.TraceStatusActive,
	}
	store.CreateTrace(ctx, trace)

	t.Run("create single span", func(t *testing.T) {
		span := &entity.Span{
			TraceID:      trace.ID,
			Type:         entity.SpanTypeLLM,
			Name:         "chat-completion",
			Input:        map[string]any{"prompt": "Hello"},
			Output:       map[string]any{"response": "Hi!"},
			InputTokens:  ptr(100),
			OutputTokens: ptr(50),
			CostUSD:      ptrF(0.01),
			DurationMs:   ptr(500),
			Status:       entity.SpanStatusSuccess,
			Model:        ptrS("gpt-4o"),
			Provider:     ptrS("openai"),
			StartedAt:    time.Now(),
		}

		if err := store.CreateSpan(ctx, span); err != nil {
			t.Fatalf("CreateSpan failed: %v", err)
		}

		if span.ID == "" {
			t.Error("expected span ID to be set")
		}
	})

	t.Run("create batch spans", func(t *testing.T) {
		spans := []entity.Span{
			{
				TraceID:   trace.ID,
				Type:      entity.SpanTypeLLM,
				Name:      "batch-1",
				Status:    entity.SpanStatusSuccess,
				StartedAt: time.Now(),
			},
			{
				TraceID:   trace.ID,
				Type:      entity.SpanTypeTool,
				Name:      "batch-2",
				Status:    entity.SpanStatusSuccess,
				StartedAt: time.Now(),
			},
			{
				TraceID:   trace.ID,
				Type:      entity.SpanTypeRetrieval,
				Name:      "batch-3",
				Status:    entity.SpanStatusError,
				StartedAt: time.Now(),
			},
		}

		if err := store.CreateSpans(ctx, spans); err != nil {
			t.Fatalf("CreateSpans failed: %v", err)
		}

		// Verify they were created
		got, _ := store.GetTrace(ctx, project.ID, trace.ID)
		if got.TotalSpans < 3 {
			t.Errorf("expected at least 3 spans, got %d", got.TotalSpans)
		}
	})

	t.Run("trace metrics calculated correctly", func(t *testing.T) {
		// Create a fresh trace for accurate metrics
		newTrace := &entity.Trace{
			ProjectID: project.ID,
			Status:    entity.TraceStatusActive,
		}
		store.CreateTrace(ctx, newTrace)

		spans := []entity.Span{
			{
				TraceID:      newTrace.ID,
				Type:         entity.SpanTypeLLM,
				Name:         "llm-1",
				InputTokens:  ptr(100),
				OutputTokens: ptr(50),
				CostUSD:      ptrF(0.005),
				DurationMs:   ptr(500),
				Status:       entity.SpanStatusSuccess,
				StartedAt:    time.Now(),
			},
			{
				TraceID:      newTrace.ID,
				Type:         entity.SpanTypeLLM,
				Name:         "llm-2",
				InputTokens:  ptr(200),
				OutputTokens: ptr(100),
				CostUSD:      ptrF(0.010),
				DurationMs:   ptr(800),
				Status:       entity.SpanStatusSuccess,
				StartedAt:    time.Now(),
			},
		}
		store.CreateSpans(ctx, spans)

		got, _ := store.GetTrace(ctx, project.ID, newTrace.ID)

		if got.TotalSpans != 2 {
			t.Errorf("TotalSpans: got %d, want 2", got.TotalSpans)
		}
		if got.TotalTokens != 450 { // 100+50+200+100
			t.Errorf("TotalTokens: got %d, want 450", got.TotalTokens)
		}
		if got.TotalDurationMs != 1300 { // 500+800
			t.Errorf("TotalDurationMs: got %d, want 1300", got.TotalDurationMs)
		}
		// Cost: 0.005 + 0.010 = 0.015
		if got.TotalCostUSD < 0.014 || got.TotalCostUSD > 0.016 {
			t.Errorf("TotalCostUSD: got %f, want ~0.015", got.TotalCostUSD)
		}
	})
}

func TestPostgresAnalytics(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	project := &entity.Project{
		Name:       "Analytics Test",
		APIKey:     fmt.Sprintf("le_analytics_%d", time.Now().UnixNano()),
		APIKeyHash: "analytics_hash",
		OwnerEmail: "analytics@example.com",
	}
	store.CreateProject(ctx, project)

	// Create traces and spans
	for i := 0; i < 3; i++ {
		status := entity.TraceStatusCompleted
		if i == 2 {
			status = entity.TraceStatusError
		}
		trace := &entity.Trace{
			ProjectID: project.ID,
			Status:    status,
		}
		store.CreateTrace(ctx, trace)

		store.CreateSpan(ctx, &entity.Span{
			TraceID:      trace.ID,
			Type:         entity.SpanTypeLLM,
			Name:         fmt.Sprintf("span-%d", i),
			InputTokens:  ptr(100),
			OutputTokens: ptr(50),
			CostUSD:      ptrF(0.01),
			DurationMs:   ptr(500),
			Status:       entity.SpanStatusSuccess,
			StartedAt:    time.Now(),
		})
	}

	t.Run("GetStats", func(t *testing.T) {
		period := entity.Period{
			From: time.Now().Add(-24 * time.Hour),
			To:   time.Now().Add(24 * time.Hour),
		}

		stats, err := store.GetStats(ctx, project.ID, period)
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}

		if stats.TotalTraces != 3 {
			t.Errorf("TotalTraces: got %d, want 3", stats.TotalTraces)
		}
		if stats.TotalSpans != 3 {
			t.Errorf("TotalSpans: got %d, want 3", stats.TotalSpans)
		}
		if stats.TotalTokens != 450 { // 3 * (100+50)
			t.Errorf("TotalTokens: got %d, want 450", stats.TotalTokens)
		}
		// Error rate: 1/3 = 33.33%
		if stats.ErrorRate < 33 || stats.ErrorRate > 34 {
			t.Errorf("ErrorRate: got %f, want ~33.33", stats.ErrorRate)
		}
	})

	t.Run("GetUsageTimeSeries", func(t *testing.T) {
		opts := entity.TimeSeriesOpts{
			Period: entity.Period{
				From: time.Now().Add(-24 * time.Hour),
				To:   time.Now().Add(24 * time.Hour),
			},
			Granularity: "day",
		}

		data, err := store.GetUsageTimeSeries(ctx, project.ID, opts)
		if err != nil {
			t.Fatalf("GetUsageTimeSeries failed: %v", err)
		}

		if len(data) == 0 {
			t.Error("expected at least one data point")
		}

		// Today's data point should have our 3 traces
		todayFound := false
		for _, dp := range data {
			if dp.Traces == 3 {
				todayFound = true
				if dp.Spans != 3 {
					t.Errorf("Spans: got %d, want 3", dp.Spans)
				}
				if dp.Tokens != 450 {
					t.Errorf("Tokens: got %d, want 450", dp.Tokens)
				}
			}
		}
		if !todayFound {
			t.Error("expected to find today's data point with 3 traces")
		}
	})

	t.Run("GetUsageTimeSeries hourly granularity", func(t *testing.T) {
		opts := entity.TimeSeriesOpts{
			Period: entity.Period{
				From: time.Now().Add(-24 * time.Hour),
				To:   time.Now().Add(24 * time.Hour),
			},
			Granularity: "hour",
		}

		data, err := store.GetUsageTimeSeries(ctx, project.ID, opts)
		if err != nil {
			t.Fatalf("GetUsageTimeSeries hourly failed: %v", err)
		}

		if len(data) == 0 {
			t.Error("expected at least one hourly data point")
		}
	})
}

func TestPostgresSessions(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx := context.Background()

	// Setup
	project := &entity.Project{
		Name:       "Sessions Test",
		APIKey:     fmt.Sprintf("le_sessions_%d", time.Now().UnixNano()),
		APIKeyHash: "sessions_hash",
		OwnerEmail: "sessions@example.com",
	}
	store.CreateProject(ctx, project)

	// Create traces with sessions
	sessions := []string{"session-a", "session-a", "session-b"}
	for i, sessionID := range sessions {
		sid := sessionID
		trace := &entity.Trace{
			ProjectID: project.ID,
			SessionID: &sid,
			Status:    entity.TraceStatusCompleted,
		}
		store.CreateTrace(ctx, trace)

		store.CreateSpan(ctx, &entity.Span{
			TraceID:      trace.ID,
			Type:         entity.SpanTypeLLM,
			Name:         fmt.Sprintf("span-%d", i),
			InputTokens:  ptr(100),
			OutputTokens: ptr(50),
			Status:       entity.SpanStatusSuccess,
			StartedAt:    time.Now(),
		})
	}

	t.Run("ListSessions", func(t *testing.T) {
		page, err := store.ListSessions(ctx, project.ID, entity.SessionFilter{Limit: 50})
		if err != nil {
			t.Fatalf("ListSessions failed: %v", err)
		}

		if page.Total != 2 {
			t.Errorf("Total sessions: got %d, want 2", page.Total)
		}

		// session-a should have 2 traces
		for _, sess := range page.Data {
			if sess.SessionID == "session-a" {
				if sess.TraceCount != 2 {
					t.Errorf("session-a TraceCount: got %d, want 2", sess.TraceCount)
				}
			}
		}
	})
}

// Helper functions
func ptr(i int) *int          { return &i }
func ptrS(s string) *string   { return &s }
func ptrF(f float64) *float64 { return &f }
