package sqlite

import (
	"context"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

// TestListTraces_FilterByPromptVersionID exercises the JSON_EXTRACT path in
// the SQLite ListTraces SQL — the bridge that lets the "payoff" view on a
// prompt version page count + link to its traces.
//
// We seed a project with three traces: two carry metadata.prompt_version_id
// pointing at "pv-A", one at "pv-B", one with no prompt version at all.
// Filtering by "pv-A" must return exactly the first two.
func TestListTraces_FilterByPromptVersionID(t *testing.T) {
	store, projA, projB := datasetTestStore(t)
	ctx := context.Background()

	mkTrace := func(promptVersion string) *entity.Trace {
		md := map[string]any{}
		if promptVersion != "" {
			md["prompt_version_id"] = promptVersion
		}
		return &entity.Trace{
			ProjectID: projA.ID,
			Status:    entity.TraceStatusCompleted,
			Metadata:  md,
		}
	}

	t1 := mkTrace("pv-A")
	t2 := mkTrace("pv-A")
	t3 := mkTrace("pv-B")
	t4 := mkTrace("") // no prompt version
	for _, tr := range []*entity.Trace{t1, t2, t3, t4} {
		if err := store.CreateTrace(ctx, tr); err != nil {
			t.Fatalf("create trace: %v", err)
		}
	}

	pv := "pv-A"
	page, err := store.ListTraces(ctx, projA.ID, entity.TraceFilter{PromptVersionID: &pv})
	if err != nil {
		t.Fatalf("list traces with filter: %v", err)
	}
	if page.Total != 2 {
		t.Errorf("want 2 traces for pv-A, got %d", page.Total)
	}
	for _, tr := range page.Data {
		if tr.Metadata == nil || tr.Metadata["prompt_version_id"] != "pv-A" {
			t.Errorf("filter leaked unrelated trace: %+v", tr.Metadata)
		}
	}

	// Cross-tenant: project B asking with the same filter must miss everything,
	// even if project A has matching traces.
	pageB, err := store.ListTraces(ctx, projB.ID, entity.TraceFilter{PromptVersionID: &pv})
	if err != nil {
		t.Fatalf("list traces cross-tenant: %v", err)
	}
	if pageB.Total != 0 {
		t.Errorf("cross-tenant filter leaked: got %d rows", pageB.Total)
	}

	// Combining with another filter still works.
	statusErr := entity.TraceStatusError
	pageMixed, err := store.ListTraces(ctx, projA.ID, entity.TraceFilter{
		PromptVersionID: &pv,
		Status:          &statusErr,
	})
	if err != nil {
		t.Fatalf("combined filter: %v", err)
	}
	if pageMixed.Total != 0 {
		t.Errorf("combined filter (status=error AND pv=pv-A): want 0, got %d", pageMixed.Total)
	}
}
