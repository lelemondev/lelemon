package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

func TestListTraces_Filters(t *testing.T) {
	// Create temp database
	tmpFile := t.TempDir() + "/test_filters.db"
	store, err := New(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()
	defer os.Remove(tmpFile)

	ctx := context.Background()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create a project
	project := &entity.Project{
		Name:       "Test",
		APIKey:     "le_test",
		APIKeyHash: "hash",
		OwnerEmail: "test@test.com",
	}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Create test traces with different attributes
	name1 := "sales-agent"
	name2 := "support-agent"
	name3 := "billing-agent"

	trace1 := &entity.Trace{
		ProjectID: project.ID,
		Name:      &name1,
		Status:    entity.TraceStatusCompleted,
		Tags:      []string{"org:abc", "campaign:123"},
	}
	trace2 := &entity.Trace{
		ProjectID: project.ID,
		Name:      &name2,
		Status:    entity.TraceStatusCompleted,
		Tags:      []string{"org:abc", "campaign:456"},
	}
	trace3 := &entity.Trace{
		ProjectID: project.ID,
		Name:      &name3,
		Status:    entity.TraceStatusError,
		Tags:      []string{"org:xyz", "campaign:789"},
	}

	for _, tr := range []*entity.Trace{trace1, trace2, trace3} {
		if err := store.CreateTrace(ctx, tr); err != nil {
			t.Fatalf("failed to create trace: %v", err)
		}
	}

	t.Run("FilterByTags_SingleTag", func(t *testing.T) {
		filter := entity.TraceFilter{
			Tags:  []string{"org:abc"},
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 traces with org:abc, got %d", len(result.Data))
		}
	})

	t.Run("FilterByTags_MultipleTags_ORLogic", func(t *testing.T) {
		filter := entity.TraceFilter{
			Tags:  []string{"campaign:123", "campaign:789"},
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 2 {
			t.Errorf("expected 2 traces (campaign:123 OR campaign:789), got %d", len(result.Data))
		}
	})

	t.Run("FilterByTags_NoMatches", func(t *testing.T) {
		filter := entity.TraceFilter{
			Tags:  []string{"nonexistent"},
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 0 {
			t.Errorf("expected 0 traces, got %d", len(result.Data))
		}
	})

	t.Run("FilterByName_PartialMatch", func(t *testing.T) {
		name := "agent"
		filter := entity.TraceFilter{
			Name:  &name,
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 3 {
			t.Errorf("expected 3 traces containing 'agent', got %d", len(result.Data))
		}
	})

	t.Run("FilterByName_ExactPrefix", func(t *testing.T) {
		name := "sales"
		filter := entity.TraceFilter{
			Name:  &name,
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 1 {
			t.Errorf("expected 1 trace containing 'sales', got %d", len(result.Data))
		}
	})

	t.Run("FilterByDateRange", func(t *testing.T) {
		from := time.Now().Add(-1 * time.Hour)
		to := time.Now().Add(1 * time.Hour)
		filter := entity.TraceFilter{
			From:  &from,
			To:    &to,
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 3 {
			t.Errorf("expected 3 traces in date range, got %d", len(result.Data))
		}
	})

	t.Run("FilterByDateRange_NoMatches", func(t *testing.T) {
		from := time.Now().Add(-48 * time.Hour)
		to := time.Now().Add(-24 * time.Hour)
		filter := entity.TraceFilter{
			From:  &from,
			To:    &to,
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 0 {
			t.Errorf("expected 0 traces in old date range, got %d", len(result.Data))
		}
	})

	t.Run("CombinedFilters_TagsAndStatus", func(t *testing.T) {
		status := entity.TraceStatusError
		filter := entity.TraceFilter{
			Tags:   []string{"org:xyz"},
			Status: &status,
			Limit:  50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 1 {
			t.Errorf("expected 1 trace (org:xyz AND error), got %d", len(result.Data))
		}
	})

	t.Run("CombinedFilters_NameAndTags", func(t *testing.T) {
		name := "support"
		filter := entity.TraceFilter{
			Name:  &name,
			Tags:  []string{"org:abc"},
			Limit: 50,
		}
		result, err := store.ListTraces(ctx, project.ID, filter)
		if err != nil {
			t.Fatalf("ListTraces failed: %v", err)
		}
		if len(result.Data) != 1 {
			t.Errorf("expected 1 trace (support AND org:abc), got %d", len(result.Data))
		}
	})
}

func TestAnalyticsQueries(t *testing.T) {
	// Create temp database
	tmpFile := t.TempDir() + "/test.db"
	store, err := New(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()
	defer os.Remove(tmpFile)

	ctx := context.Background()

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create a project
	project := &entity.Project{
		Name:       "Test",
		APIKey:     "le_test",
		APIKeyHash: "hash",
		OwnerEmail: "test@test.com",
	}
	if err := store.CreateProject(ctx, project); err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Create a trace
	trace := &entity.Trace{
		ProjectID: project.ID,
		Status:    entity.TraceStatusCompleted,
	}
	if err := store.CreateTrace(ctx, trace); err != nil {
		t.Fatalf("failed to create trace: %v", err)
	}

	// Create a span
	span := entity.Span{
		TraceID:      trace.ID,
		Type:         entity.SpanTypeLLM,
		Name:         "test",
		Status:       entity.SpanStatusSuccess,
		StartedAt:    time.Now(),
	}
	if err := store.CreateSpan(ctx, &span); err != nil {
		t.Fatalf("failed to create span: %v", err)
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
		if stats.TotalTraces != 1 {
			t.Errorf("expected 1 trace, got %d", stats.TotalTraces)
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
		t.Logf("Data points: %d", len(data))
	})
}
