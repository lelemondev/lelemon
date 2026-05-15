package sqlite

import (
	"context"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

// evalTestStore seeds a project + dataset + a couple of items so eval tests
// have something realistic to point at.
func evalTestStore(t *testing.T) (*Store, *entity.Project, *entity.Project, *entity.Dataset) {
	t.Helper()
	store, projA, projB := datasetTestStore(t)
	ctx := context.Background()

	d := &entity.Dataset{ProjectID: projA.ID, Name: "ds"}
	if err := store.CreateDataset(ctx, d); err != nil {
		t.Fatalf("create dataset: %v", err)
	}
	return store, projA, projB, d
}

func TestEval_CRUDAndTenantIsolation(t *testing.T) {
	store, projA, projB, dA := evalTestStore(t)
	ctx := context.Background()

	desc := "exact match regression set"
	e := &entity.Eval{
		ProjectID:   projA.ID,
		DatasetID:   dA.ID,
		Name:        "vehicle-search",
		Description: &desc,
		Scorers: []entity.Scorer{
			{ID: "s1", Type: entity.ScorerExactMatch},
			{ID: "s2", Type: entity.ScorerJSONPath, Config: map[string]any{
				"path": "results.0.id", "op": "eq", "value": "abc",
			}},
		},
	}
	if err := store.CreateEval(ctx, e); err != nil {
		t.Fatalf("create eval: %v", err)
	}

	got, err := store.GetEval(ctx, projA.ID, e.ID)
	if err != nil {
		t.Fatalf("get eval: %v", err)
	}
	if got.Name != "vehicle-search" || len(got.Scorers) != 2 || got.Scorers[1].Config["op"] != "eq" {
		t.Errorf("eval roundtrip lost data: %+v", got)
	}

	// Cross-tenant Get must miss.
	if _, err := store.GetEval(ctx, projB.ID, e.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant Get: want ErrNotFound, got %v", err)
	}

	// List scoped to project A → 1; project B → 0.
	pageA, err := store.ListEvals(ctx, projA.ID, entity.EvalFilter{})
	if err != nil || pageA.Total != 1 {
		t.Errorf("list A: want total=1, got %d (err=%v)", pageA.Total, err)
	}
	pageB, _ := store.ListEvals(ctx, projB.ID, entity.EvalFilter{})
	if pageB.Total != 0 {
		t.Errorf("list B leaked: %d", pageB.Total)
	}

	// Filter by dataset works.
	other := &entity.Dataset{ProjectID: projA.ID, Name: "other"}
	if err := store.CreateDataset(ctx, other); err != nil {
		t.Fatalf("create other dataset: %v", err)
	}
	dID := other.ID
	pageOther, _ := store.ListEvals(ctx, projA.ID, entity.EvalFilter{DatasetID: &dID})
	if pageOther.Total != 0 {
		t.Errorf("dataset filter wrong: want 0 in 'other', got %d", pageOther.Total)
	}

	// Delete (only by the owning tenant).
	if err := store.DeleteEval(ctx, projB.ID, e.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant Delete: want ErrNotFound, got %v", err)
	}
	if err := store.DeleteEval(ctx, projA.ID, e.ID); err != nil {
		t.Errorf("delete: %v", err)
	}
	if _, err := store.GetEval(ctx, projA.ID, e.ID); err != entity.ErrNotFound {
		t.Errorf("after delete: want ErrNotFound, got %v", err)
	}
}

func TestEvalRun_FinalizeAggregatesFromResults(t *testing.T) {
	// This test exercises the SQL aggregation path inside FinalizeEvalRun —
	// the place where service test fakes can lie. We post a mix of passed,
	// failed, and errored results and verify the DB-computed counts match.
	store, projA, _, d := evalTestStore(t)
	ctx := context.Background()

	e := &entity.Eval{
		ProjectID: projA.ID, DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	}
	if err := store.CreateEval(ctx, e); err != nil {
		t.Fatalf("create eval: %v", err)
	}

	run := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID}
	if err := store.CreateEvalRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	dur := 100
	cost := 0.05
	results := []entity.EvalRunResult{
		{ProjectID: projA.ID, EvalRunID: run.ID, DatasetItemID: "item-1", Passed: true, DurationMs: &dur, CostUSD: &cost},
		{ProjectID: projA.ID, EvalRunID: run.ID, DatasetItemID: "item-2", Passed: false, DurationMs: &dur, CostUSD: &cost},
		{ProjectID: projA.ID, EvalRunID: run.ID, DatasetItemID: "item-3", Passed: true, DurationMs: &dur, CostUSD: &cost},
		// One errored — should bump erroredItems even with passed=false.
		{ProjectID: projA.ID, EvalRunID: run.ID, DatasetItemID: "item-4", Passed: false,
			DurationMs: &dur, CostUSD: &cost,
			Error: strPtr("target crashed")},
	}
	for i := range results {
		if err := store.CreateEvalRunResult(ctx, &results[i]); err != nil {
			t.Fatalf("create result[%d]: %v", i, err)
		}
	}

	finalized, err := store.FinalizeEvalRun(ctx, projA.ID, run.ID, entity.FinalizeEvalRunInput{
		Status: entity.EvalRunStatusCompleted,
	})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}

	if finalized.Status != entity.EvalRunStatusCompleted {
		t.Errorf("status: %s", finalized.Status)
	}
	if finalized.TotalItems != 4 {
		t.Errorf("totalItems: want 4, got %d", finalized.TotalItems)
	}
	if finalized.PassedItems != 2 {
		t.Errorf("passedItems: want 2, got %d", finalized.PassedItems)
	}
	if finalized.ErroredItems != 1 {
		t.Errorf("erroredItems: want 1, got %d", finalized.ErroredItems)
	}
	// failed = total - passed - errored = 4 - 2 - 1 = 1
	if finalized.FailedItems != 1 {
		t.Errorf("failedItems: want 1, got %d", finalized.FailedItems)
	}
	if finalized.DurationMs == nil || *finalized.DurationMs != 400 {
		t.Errorf("duration sum: want 400, got %v", finalized.DurationMs)
	}
	if finalized.CostUSD == nil || *finalized.CostUSD < 0.199 || *finalized.CostUSD > 0.201 {
		t.Errorf("cost sum: want ~0.2, got %v", finalized.CostUSD)
	}
	if finalized.CompletedAt == nil {
		t.Errorf("completedAt should be set")
	}

	// Idempotent — re-finalize returns the same row, untouched.
	again, err := store.FinalizeEvalRun(ctx, projA.ID, run.ID, entity.FinalizeEvalRunInput{
		Status: entity.EvalRunStatusFailed, // different status — must be ignored
	})
	if err != nil {
		t.Fatalf("re-finalize: %v", err)
	}
	if again.Status != entity.EvalRunStatusCompleted {
		t.Errorf("re-finalize changed status to %s — finalize must be idempotent", again.Status)
	}
}

func TestEvalRun_CascadesWithEval(t *testing.T) {
	// Deleting an eval should cascade-delete its runs and results.
	store, projA, _, d := evalTestStore(t)
	ctx := context.Background()

	e := &entity.Eval{
		ProjectID: projA.ID, DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	}
	_ = store.CreateEval(ctx, e)
	run := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID}
	_ = store.CreateEvalRun(ctx, run)
	_ = store.CreateEvalRunResult(ctx, &entity.EvalRunResult{
		ProjectID: projA.ID, EvalRunID: run.ID, DatasetItemID: "i", Passed: true,
	})

	if err := store.DeleteEval(ctx, projA.ID, e.ID); err != nil {
		t.Fatalf("delete eval: %v", err)
	}
	if _, err := store.GetEvalRun(ctx, projA.ID, run.ID); err != entity.ErrNotFound {
		t.Errorf("run should cascade with eval, got %v", err)
	}
	page, err := store.ListEvalRunResults(ctx, projA.ID, run.ID, entity.EvalRunResultFilter{})
	if err != nil {
		t.Fatalf("list results: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("results should cascade with run, got %d rows", page.Total)
	}
}

func TestEvalRunResult_ListAndPassedFilter(t *testing.T) {
	store, projA, _, d := evalTestStore(t)
	ctx := context.Background()
	e := &entity.Eval{
		ProjectID: projA.ID, DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	}
	_ = store.CreateEval(ctx, e)
	run := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID}
	_ = store.CreateEvalRun(ctx, run)
	for i, passed := range []bool{true, false, true, false, true} {
		err := store.CreateEvalRunResult(ctx, &entity.EvalRunResult{
			ProjectID: projA.ID, EvalRunID: run.ID,
			DatasetItemID: "i" + string(rune('a'+i)), Passed: passed,
			Actual: map[string]any{"echo": float64(i)},
			Scores: []entity.ScorerResult{{ScorerID: "s", Passed: passed, Score: boolScore(passed)}},
		})
		if err != nil {
			t.Fatalf("create result[%d]: %v", i, err)
		}
	}

	all, err := store.ListEvalRunResults(ctx, projA.ID, run.ID, entity.EvalRunResultFilter{})
	if err != nil || all.Total != 5 {
		t.Errorf("list all: total=%d err=%v", all.Total, err)
	}
	// Check JSON roundtrip on first result.
	if m, ok := all.Data[0].Actual.(map[string]any); !ok || m["echo"] != float64(0) {
		t.Errorf("actual roundtrip lost: %#v", all.Data[0].Actual)
	}

	yes := true
	passedOnly, _ := store.ListEvalRunResults(ctx, projA.ID, run.ID, entity.EvalRunResultFilter{PassedOnly: &yes})
	if passedOnly.Total != 3 {
		t.Errorf("passedOnly: want 3, got %d", passedOnly.Total)
	}
	no := false
	failedOnly, _ := store.ListEvalRunResults(ctx, projA.ID, run.ID, entity.EvalRunResultFilter{PassedOnly: &no})
	if failedOnly.Total != 2 {
		t.Errorf("failedOnly: want 2, got %d", failedOnly.Total)
	}
}

func strPtr(s string) *string { return &s }
func boolScore(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
