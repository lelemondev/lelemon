package eval

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ----- in-memory fakes ---------------------------------------------------

type fakeRepo struct {
	evals   map[string]*entity.Eval
	runs    map[string]*entity.EvalRun
	results map[string]*entity.EvalRunResult
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		evals:   map[string]*entity.Eval{},
		runs:    map[string]*entity.EvalRun{},
		results: map[string]*entity.EvalRunResult{},
	}
}

func (f *fakeRepo) CreateEval(_ context.Context, e *entity.Eval) error {
	if e.ID == "" {
		e.ID = "ev-" + e.Name
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = e.CreatedAt
	f.evals[e.ID] = e
	return nil
}

func (f *fakeRepo) GetEval(_ context.Context, projectID, evalID string) (*entity.Eval, error) {
	e, ok := f.evals[evalID]
	if !ok || e.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *e
	return &cp, nil
}

func (f *fakeRepo) ListEvals(_ context.Context, projectID string, filter entity.EvalFilter) (*entity.Page[entity.Eval], error) {
	out := []entity.Eval{}
	for _, e := range f.evals {
		if e.ProjectID != projectID {
			continue
		}
		if filter.DatasetID != nil && e.DatasetID != *filter.DatasetID {
			continue
		}
		out = append(out, *e)
	}
	return &entity.Page[entity.Eval]{Data: out, Total: len(out)}, nil
}

func (f *fakeRepo) DeleteEval(_ context.Context, projectID, evalID string) error {
	e, ok := f.evals[evalID]
	if !ok || e.ProjectID != projectID {
		return entity.ErrNotFound
	}
	delete(f.evals, evalID)
	return nil
}

func (f *fakeRepo) CreateEvalRun(_ context.Context, r *entity.EvalRun) error {
	if r.ID == "" {
		r.ID = "run-" + suffix(len(f.runs))
	}
	now := time.Now()
	r.StartedAt = now
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Status == "" {
		r.Status = entity.EvalRunStatusPending
	}
	f.runs[r.ID] = r
	return nil
}

func (f *fakeRepo) GetEvalRun(_ context.Context, projectID, runID string) (*entity.EvalRun, error) {
	r, ok := f.runs[runID]
	if !ok || r.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (f *fakeRepo) ListEvalRuns(_ context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*entity.Page[entity.EvalRun], error) {
	out := []entity.EvalRun{}
	for _, r := range f.runs {
		if r.ProjectID != projectID {
			continue
		}
		if evalID != "" && r.EvalID != evalID {
			continue
		}
		if filter.Status != nil && r.Status != *filter.Status {
			continue
		}
		out = append(out, *r)
	}
	return &entity.Page[entity.EvalRun]{Data: out, Total: len(out)}, nil
}

func (f *fakeRepo) UpdateEvalRunStatus(_ context.Context, projectID, runID string, status entity.EvalRunStatus) error {
	r, ok := f.runs[runID]
	if !ok || r.ProjectID != projectID {
		return entity.ErrNotFound
	}
	r.Status = status
	r.UpdatedAt = time.Now()
	return nil
}

func (f *fakeRepo) FinalizeEvalRun(_ context.Context, projectID, runID string, in entity.FinalizeEvalRunInput) (*entity.EvalRun, error) {
	r, ok := f.runs[runID]
	if !ok || r.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	if r.Status == entity.EvalRunStatusCompleted || r.Status == entity.EvalRunStatusFailed {
		cp := *r
		return &cp, nil
	}
	var total, passed, errored int
	var sumDuration int
	var sumCost float64
	for _, res := range f.results {
		if res.ProjectID != projectID || res.EvalRunID != runID {
			continue
		}
		total++
		if res.Error != nil && *res.Error != "" {
			errored++
		} else if res.Passed {
			passed++
		}
		if res.DurationMs != nil {
			sumDuration += *res.DurationMs
		}
		if res.CostUSD != nil {
			sumCost += *res.CostUSD
		}
	}
	r.Status = in.Status
	r.TotalItems = total
	r.PassedItems = passed
	r.ErroredItems = errored
	r.FailedItems = max(total-passed-errored, 0)
	if in.DurationMs != nil {
		r.DurationMs = in.DurationMs
	} else if sumDuration > 0 {
		v := sumDuration
		r.DurationMs = &v
	}
	if in.CostUSD != nil {
		r.CostUSD = in.CostUSD
	} else if sumCost > 0 {
		v := sumCost
		r.CostUSD = &v
	}
	now := time.Now()
	r.CompletedAt = &now
	r.UpdatedAt = now
	cp := *r
	return &cp, nil
}

func (f *fakeRepo) CreateEvalRunResult(_ context.Context, res *entity.EvalRunResult) error {
	if res.ID == "" {
		res.ID = "res-" + suffix(len(f.results))
	}
	res.CreatedAt = time.Now()
	f.results[res.ID] = res
	return nil
}

func (f *fakeRepo) ListEvalRunResults(_ context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*entity.Page[entity.EvalRunResult], error) {
	out := []entity.EvalRunResult{}
	for _, r := range f.results {
		if r.ProjectID != projectID || r.EvalRunID != runID {
			continue
		}
		if filter.PassedOnly != nil && r.Passed != *filter.PassedOnly {
			continue
		}
		out = append(out, *r)
	}
	return &entity.Page[entity.EvalRunResult]{Data: out, Total: len(out)}, nil
}

type fakeDatasets struct {
	datasets map[string]*entity.Dataset    // id → row
	items    map[string]*entity.DatasetItem // id → row
}

func newFakeDatasets() *fakeDatasets {
	return &fakeDatasets{
		datasets: map[string]*entity.Dataset{},
		items:    map[string]*entity.DatasetItem{},
	}
}

func (f *fakeDatasets) putDataset(d *entity.Dataset)     { f.datasets[d.ID] = d }
func (f *fakeDatasets) putItem(it *entity.DatasetItem)   { f.items[it.ID] = it }

func (f *fakeDatasets) GetDataset(_ context.Context, projectID, datasetID string) (*entity.Dataset, error) {
	d, ok := f.datasets[datasetID]
	if !ok || d.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (f *fakeDatasets) GetDatasetItem(_ context.Context, projectID, itemID string) (*entity.DatasetItem, error) {
	it, ok := f.items[itemID]
	if !ok || it.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *it
	return &cp, nil
}

func suffix(n int) string { return string(rune('a' + n%26)) }

// makeServiceWithDataset gives back a service preloaded with one dataset and
// a couple of items, ready for run tests.
func makeServiceWithDataset(t *testing.T) (*Service, *fakeRepo, *fakeDatasets, *entity.Dataset, []*entity.DatasetItem) {
	t.Helper()
	repo := newFakeRepo()
	ds := newFakeDatasets()
	d := &entity.Dataset{ID: "ds-1", ProjectID: "proj", Name: "ds"}
	ds.putDataset(d)
	items := []*entity.DatasetItem{
		{ID: "it-1", DatasetID: d.ID, ProjectID: "proj", Input: "honda", Expected: "honda"},
		{ID: "it-2", DatasetID: d.ID, ProjectID: "proj", Input: "civic", Expected: "civic"},
	}
	for _, it := range items {
		ds.putItem(it)
	}
	return NewService(repo, ds), repo, ds, d, items
}

// ----- create eval -------------------------------------------------------

func TestCreateEval_Validation(t *testing.T) {
	svc, _, _, d, _ := makeServiceWithDataset(t)
	ctx := context.Background()

	cases := []struct {
		name string
		req  CreateEvalRequest
		want error
	}{
		{"empty name", CreateEvalRequest{DatasetID: d.ID, Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}}}, entity.ErrBadRequest},
		{"missing dataset", CreateEvalRequest{Name: "x", Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}}}, entity.ErrBadRequest},
		{"no scorers", CreateEvalRequest{DatasetID: d.ID, Name: "x"}, entity.ErrBadRequest},
		{"missing scorer id", CreateEvalRequest{
			DatasetID: d.ID, Name: "x",
			Scorers: []entity.Scorer{{Type: entity.ScorerExactMatch}},
		}, entity.ErrBadRequest},
		{"duplicate scorer id", CreateEvalRequest{
			DatasetID: d.ID, Name: "x",
			Scorers: []entity.Scorer{
				{ID: "s", Type: entity.ScorerExactMatch},
				{ID: "s", Type: entity.ScorerExactMatch},
			},
		}, entity.ErrBadRequest},
		{"contains missing value", CreateEvalRequest{
			DatasetID: d.ID, Name: "x",
			Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerContains, Config: map[string]any{}}},
		}, entity.ErrBadRequest},
		{"json_path missing op", CreateEvalRequest{
			DatasetID: d.ID, Name: "x",
			Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{"path": "a", "value": float64(1)}}},
		}, entity.ErrBadRequest},
		{"json_path bad op", CreateEvalRequest{
			DatasetID: d.ID, Name: "x",
			Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{"path": "a", "op": "bogus", "value": float64(1)}}},
		}, entity.ErrBadRequest},
		{"unknown type", CreateEvalRequest{
			DatasetID: d.ID, Name: "x",
			Scorers: []entity.Scorer{{ID: "s", Type: "bogus"}},
		}, entity.ErrBadRequest},
		{"too long name", CreateEvalRequest{
			DatasetID: d.ID, Name: strings.Repeat("a", maxEvalName+1),
			Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
		}, entity.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, "proj", tc.req)
			if !errors.Is(err, tc.want) {
				t.Errorf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestCreateEval_RejectsCrossTenantDataset(t *testing.T) {
	svc, _, _, _, _ := makeServiceWithDataset(t)
	ctx := context.Background()

	// dataset belongs to "proj"; try to create eval as "other"
	_, err := svc.Create(ctx, "other", CreateEvalRequest{
		DatasetID: "ds-1", Name: "x",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	if !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("cross-tenant dataset reference must miss with ErrNotFound, got %v", err)
	}
}

// ----- run lifecycle -----------------------------------------------------

func TestRunLifecycle_HappyPath(t *testing.T) {
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()

	ev, err := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "exact",
		Scorers: []entity.Scorer{{ID: "exact", Type: entity.ScorerExactMatch}},
	})
	if err != nil {
		t.Fatalf("create eval: %v", err)
	}

	run, err := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != entity.EvalRunStatusPending {
		t.Errorf("new run should be pending, got %s", run.Status)
	}

	// Post one passing + one failing result.
	dur1 := 120
	r1, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID, Actual: "honda", DurationMs: &dur1,
	})
	if err != nil {
		t.Fatalf("post result 1: %v", err)
	}
	if !r1.Passed {
		t.Errorf("result 1 should pass (honda == honda): %+v", r1)
	}
	if r1.Scores[0].ScorerID != "exact" || !r1.Scores[0].Passed {
		t.Errorf("result 1 scorer detail off: %+v", r1.Scores)
	}

	dur2 := 80
	r2, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[1].ID, Actual: "wrong-answer", DurationMs: &dur2,
	})
	if err != nil {
		t.Fatalf("post result 2: %v", err)
	}
	if r2.Passed {
		t.Errorf("result 2 should fail: %+v", r2)
	}

	// After first result, run should have flipped to running.
	mid, err := svc.GetRun(ctx, "proj", run.ID)
	if err != nil {
		t.Fatalf("get run mid: %v", err)
	}
	if mid.Status != entity.EvalRunStatusRunning {
		t.Errorf("run should be running mid-flight, got %s", mid.Status)
	}

	// Finalize → completed with aggregates.
	finalized, err := svc.Finalize(ctx, "proj", run.ID, FinalizeEvalRunRequest{
		Status: entity.EvalRunStatusCompleted,
	})
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if finalized.Status != entity.EvalRunStatusCompleted {
		t.Errorf("expected completed, got %s", finalized.Status)
	}
	if finalized.TotalItems != 2 || finalized.PassedItems != 1 || finalized.FailedItems != 1 {
		t.Errorf("aggregates wrong: %+v", finalized)
	}
	if finalized.PassRate == nil || *finalized.PassRate != 0.5 {
		t.Errorf("pass rate should be 0.5, got %v", finalized.PassRate)
	}
	if finalized.DurationMs == nil || *finalized.DurationMs != 200 {
		t.Errorf("duration should sum to 200, got %v", finalized.DurationMs)
	}
}

func TestPostResult_AfterFinalize(t *testing.T) {
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})
	_, _ = svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID, Actual: "honda",
	})
	_, _ = svc.Finalize(ctx, "proj", run.ID, FinalizeEvalRunRequest{Status: entity.EvalRunStatusCompleted})

	_, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[1].ID, Actual: "x",
	})
	if !errors.Is(err, entity.ErrConflict) {
		t.Errorf("posting after finalize must conflict, got %v", err)
	}
}

func TestPostResult_HardExecutionError(t *testing.T) {
	svc, repo, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	errStr := "target threw"
	res, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID, Error: &errStr,
	})
	if err != nil {
		t.Fatalf("post errored result: %v", err)
	}
	if res.Passed {
		t.Errorf("hard error must not pass")
	}
	if len(res.Scores) != 0 {
		t.Errorf("scoring should be skipped on hard error, got %d scores", len(res.Scores))
	}

	// Aggregates after finalize: 1 total, 0 passed, 1 errored, 0 failed.
	final, _ := svc.Finalize(ctx, "proj", run.ID, FinalizeEvalRunRequest{Status: entity.EvalRunStatusCompleted})
	if final.TotalItems != 1 || final.ErroredItems != 1 || final.PassedItems != 0 || final.FailedItems != 0 {
		t.Errorf("errored item aggregation off: %+v", final)
	}
	// Sanity: the result is actually persisted.
	if len(repo.results) != 1 {
		t.Errorf("want 1 result persisted, got %d", len(repo.results))
	}
}

func TestPostResult_AntiLeakAcrossDatasets(t *testing.T) {
	// An item that belongs to dataset B must NOT be accepted as a result for
	// an eval defined on dataset A — even though both are in the same project.
	svc, _, ds, dA, _ := makeServiceWithDataset(t)
	ctx := context.Background()
	// Create a sibling dataset + an item in it under the same project.
	dB := &entity.Dataset{ID: "ds-2", ProjectID: "proj", Name: "other"}
	ds.putDataset(dB)
	itemB := &entity.DatasetItem{ID: "it-B", DatasetID: dB.ID, ProjectID: "proj", Input: "x", Expected: "x"}
	ds.putItem(itemB)

	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: dA.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	_, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: itemB.ID, Actual: "x",
	})
	if !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("posting an item from another dataset must miss with ErrNotFound, got %v", err)
	}
}

func TestPostResult_CrossTenantRun(t *testing.T) {
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	_, err := svc.PostResult(ctx, "other-proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID, Actual: "x",
	})
	if !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("cross-tenant run access must miss with ErrNotFound, got %v", err)
	}
}

func TestFinalize_Idempotent(t *testing.T) {
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})
	_, _ = svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID, Actual: "honda",
	})
	first, _ := svc.Finalize(ctx, "proj", run.ID, FinalizeEvalRunRequest{Status: entity.EvalRunStatusCompleted})
	second, err := svc.Finalize(ctx, "proj", run.ID, FinalizeEvalRunRequest{Status: entity.EvalRunStatusCompleted})
	if err != nil {
		t.Fatalf("second finalize should be a no-op, got %v", err)
	}
	if first.TotalItems != second.TotalItems || first.PassedItems != second.PassedItems {
		t.Errorf("aggregates changed between idempotent finalizes: %+v vs %+v", first, second)
	}
}

func TestFinalize_RejectsNonTerminalStatus(t *testing.T) {
	svc, _, _, d, _ := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})
	_, err := svc.Finalize(ctx, "proj", run.ID, FinalizeEvalRunRequest{Status: entity.EvalRunStatusRunning})
	if !errors.Is(err, entity.ErrBadRequest) {
		t.Errorf("finalize with non-terminal status must fail with ErrBadRequest, got %v", err)
	}
}
