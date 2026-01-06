package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lelemon/server/internal/domain/entity"
)

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
