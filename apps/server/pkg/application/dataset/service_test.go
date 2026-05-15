package dataset

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ----- in-memory fakes (DatasetRepo + SpanReader) -------------------------
//
// These satisfy the narrow interfaces declared in service.go. They are
// intentionally minimal — just enough to exercise the application-layer
// behaviour without booting a real database. Persistence semantics are
// already covered by the sqlite store tests; what we want here is the
// service's own logic: validation, cross-store wiring, anti-leak checks.

type fakeRepo struct {
	datasets map[string]*entity.Dataset    // by datasetID
	items    map[string]*entity.DatasetItem // by itemID
	failNext error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		datasets: map[string]*entity.Dataset{},
		items:    map[string]*entity.DatasetItem{},
	}
}

func (f *fakeRepo) CreateDataset(_ context.Context, d *entity.Dataset) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	if d.ID == "" {
		d.ID = "ds-" + d.Name
	}
	f.datasets[d.ID] = d
	return nil
}

func (f *fakeRepo) GetDataset(_ context.Context, projectID, datasetID string) (*entity.Dataset, error) {
	d, ok := f.datasets[datasetID]
	if !ok || d.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (f *fakeRepo) ListDatasets(_ context.Context, projectID string, _ entity.DatasetFilter) (*entity.Page[entity.Dataset], error) {
	out := []entity.Dataset{}
	for _, d := range f.datasets {
		if d.ProjectID == projectID {
			out = append(out, *d)
		}
	}
	return &entity.Page[entity.Dataset]{Data: out, Total: len(out)}, nil
}

func (f *fakeRepo) UpdateDataset(_ context.Context, projectID, datasetID string, updates entity.DatasetUpdate) error {
	d, ok := f.datasets[datasetID]
	if !ok || d.ProjectID != projectID {
		return entity.ErrNotFound
	}
	if updates.Name != nil {
		d.Name = *updates.Name
	}
	if updates.Description != nil {
		d.Description = updates.Description
	}
	return nil
}

func (f *fakeRepo) DeleteDataset(_ context.Context, projectID, datasetID string) error {
	d, ok := f.datasets[datasetID]
	if !ok || d.ProjectID != projectID {
		return entity.ErrNotFound
	}
	delete(f.datasets, datasetID)
	return nil
}

func (f *fakeRepo) CreateDatasetItem(_ context.Context, it *entity.DatasetItem) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	if it.ID == "" {
		it.ID = "it-" + randSuffix(len(f.items))
	}
	f.items[it.ID] = it
	return nil
}

func (f *fakeRepo) BulkCreateDatasetItems(_ context.Context, items []entity.DatasetItem) error {
	if f.failNext != nil {
		err := f.failNext
		f.failNext = nil
		return err
	}
	for i := range items {
		if items[i].ID == "" {
			items[i].ID = "it-" + randSuffix(len(f.items))
		}
		cp := items[i]
		f.items[cp.ID] = &cp
	}
	return nil
}

func (f *fakeRepo) GetDatasetItem(_ context.Context, projectID, itemID string) (*entity.DatasetItem, error) {
	it, ok := f.items[itemID]
	if !ok || it.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *it
	return &cp, nil
}

func (f *fakeRepo) ListDatasetItems(_ context.Context, projectID, datasetID string, _ entity.DatasetItemFilter) (*entity.Page[entity.DatasetItem], error) {
	out := []entity.DatasetItem{}
	for _, it := range f.items {
		if it.ProjectID == projectID && it.DatasetID == datasetID {
			out = append(out, *it)
		}
	}
	return &entity.Page[entity.DatasetItem]{Data: out, Total: len(out)}, nil
}

func (f *fakeRepo) DeleteDatasetItem(_ context.Context, projectID, itemID string) error {
	it, ok := f.items[itemID]
	if !ok || it.ProjectID != projectID {
		return entity.ErrNotFound
	}
	delete(f.items, itemID)
	return nil
}

type fakeSpanReader struct {
	traces map[string]*entity.TraceWithSpans // keyed by projectID|traceID
}

func newFakeSpanReader() *fakeSpanReader {
	return &fakeSpanReader{traces: map[string]*entity.TraceWithSpans{}}
}

func (f *fakeSpanReader) put(projectID, traceID string, spans []entity.Span) {
	f.traces[projectID+"|"+traceID] = &entity.TraceWithSpans{
		Trace: entity.Trace{ID: traceID, ProjectID: projectID},
		Spans: spans,
	}
}

func (f *fakeSpanReader) GetTrace(_ context.Context, projectID, traceID string) (*entity.TraceWithSpans, error) {
	t, ok := f.traces[projectID+"|"+traceID]
	if !ok {
		return nil, entity.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func randSuffix(n int) string {
	// deterministic enough for tests — we just need unique strings per call.
	return string(rune('a' + n%26))
}

func newServiceForTest() (*Service, *fakeRepo, *fakeSpanReader) {
	repo := newFakeRepo()
	spans := newFakeSpanReader()
	return NewService(repo, spans), repo, spans
}

// ----- tests --------------------------------------------------------------

func TestCreate_Validation(t *testing.T) {
	svc, _, _ := newServiceForTest()
	ctx := context.Background()

	cases := []struct {
		name string
		req  CreateDatasetRequest
		want error
	}{
		{"empty name", CreateDatasetRequest{Name: ""}, entity.ErrBadRequest},
		{"whitespace name", CreateDatasetRequest{Name: "   "}, entity.ErrBadRequest},
		{"too long name", CreateDatasetRequest{Name: strings.Repeat("a", maxDatasetName+1)}, entity.ErrBadRequest},
		{"too long description", CreateDatasetRequest{
			Name:        "ok",
			Description: ptr(strings.Repeat("d", maxDatasetDescription+1)),
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

func TestCreate_TrimsName(t *testing.T) {
	svc, _, _ := newServiceForTest()
	ctx := context.Background()

	view, err := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "  vehicle-search\t"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Name != "vehicle-search" {
		t.Errorf("name not trimmed: %q", view.Name)
	}
}

func TestAddItemFromTrace_HappyPath(t *testing.T) {
	svc, repo, spans := newServiceForTest()
	ctx := context.Background()

	view, err := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "ds"})
	if err != nil {
		t.Fatalf("create dataset: %v", err)
	}

	// Seed a trace with one LLM span.
	spans.put("proj", "trace-1", []entity.Span{
		{ID: "span-1", Type: entity.SpanTypeLLM, Input: map[string]any{"q": "honda civic"}, Output: "found 3"},
		{ID: "span-2", Type: entity.SpanTypeTool, Input: "noise"},
	})

	got, err := svc.AddItemFromTrace(ctx, "proj", view.ID, AddDatasetItemFromTraceRequest{
		TraceID:  "trace-1",
		SpanID:   "span-1",
		Expected: map[string]any{"min_results": float64(1)}, // caller-supplied
		Metadata: map[string]any{"category": "natural-language"},
	})
	if err != nil {
		t.Fatalf("add from trace: %v", err)
	}

	// Input was copied from the span.
	if m, ok := got.Input.(map[string]any); !ok || m["q"] != "honda civic" {
		t.Errorf("input not seeded from span: %#v", got.Input)
	}
	// Expected is NOT auto-pulled from span.Output — only what the caller passed.
	if m, ok := got.Expected.(map[string]any); !ok || m["min_results"].(float64) != 1 {
		t.Errorf("expected not preserved: %#v", got.Expected)
	}
	// Provenance is recorded.
	if got.SourceTraceID == nil || *got.SourceTraceID != "trace-1" {
		t.Errorf("source_trace_id wrong: %v", got.SourceTraceID)
	}
	if got.SourceSpanID == nil || *got.SourceSpanID != "span-1" {
		t.Errorf("source_span_id wrong: %v", got.SourceSpanID)
	}

	// And it actually landed in the repo.
	if len(repo.items) != 1 {
		t.Errorf("want 1 item persisted, got %d", len(repo.items))
	}
}

func TestAddItemFromTrace_SpanNotFound(t *testing.T) {
	svc, _, spans := newServiceForTest()
	ctx := context.Background()

	view, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "ds"})
	spans.put("proj", "trace-1", []entity.Span{{ID: "span-1"}})

	_, err := svc.AddItemFromTrace(ctx, "proj", view.ID, AddDatasetItemFromTraceRequest{
		TraceID: "trace-1", SpanID: "ghost",
	})
	if !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("want ErrNotFound for missing span, got %v", err)
	}
}

func TestAddItemFromTrace_TraceCrossTenant(t *testing.T) {
	svc, _, spans := newServiceForTest()
	ctx := context.Background()

	view, _ := svc.Create(ctx, "proj-A", CreateDatasetRequest{Name: "ds"})
	// Trace lives under project B — must not be reachable from project A.
	spans.put("proj-B", "trace-x", []entity.Span{{ID: "span-1"}})

	_, err := svc.AddItemFromTrace(ctx, "proj-A", view.ID, AddDatasetItemFromTraceRequest{
		TraceID: "trace-x", SpanID: "span-1",
	})
	if !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("cross-tenant trace access should miss with ErrNotFound, got %v", err)
	}
}

func TestAddItemFromTrace_MissingIDs(t *testing.T) {
	svc, _, _ := newServiceForTest()
	ctx := context.Background()
	view, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "ds"})

	_, err := svc.AddItemFromTrace(ctx, "proj", view.ID, AddDatasetItemFromTraceRequest{
		TraceID: "", SpanID: "span-1",
	})
	if !errors.Is(err, entity.ErrBadRequest) {
		t.Errorf("missing traceId: want ErrBadRequest, got %v", err)
	}
}

func TestGetItem_AntiLeakAcrossDatasets(t *testing.T) {
	// An item that lives in dataset A must NOT be reachable via dataset B's
	// URL, even though both datasets belong to the same project.
	svc, repo, _ := newServiceForTest()
	ctx := context.Background()

	dsA, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "A"})
	dsB, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "B"})

	itemA, err := svc.CreateItem(ctx, "proj", dsA.ID, CreateDatasetItemRequest{Input: "leak?"})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	// Sanity: the item is actually under dataset A.
	if repo.items[itemA.ID].DatasetID != dsA.ID {
		t.Fatalf("test setup wrong: item parent is %s, want %s", repo.items[itemA.ID].DatasetID, dsA.ID)
	}

	// Reading itemA through dataset B's URL must miss.
	if _, err := svc.GetItem(ctx, "proj", dsB.ID, itemA.ID); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("anti-leak Get: want ErrNotFound, got %v", err)
	}
	// Same for delete — must miss, and the item must still exist after.
	if err := svc.DeleteItem(ctx, "proj", dsB.ID, itemA.ID); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("anti-leak Delete: want ErrNotFound, got %v", err)
	}
	if _, ok := repo.items[itemA.ID]; !ok {
		t.Errorf("anti-leak Delete: item was actually removed (leak)")
	}
}

func TestCreateItem_RequiresInput(t *testing.T) {
	svc, _, _ := newServiceForTest()
	ctx := context.Background()
	ds, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "ds"})

	_, err := svc.CreateItem(ctx, "proj", ds.ID, CreateDatasetItemRequest{Input: nil})
	if !errors.Is(err, entity.ErrBadRequest) {
		t.Errorf("want ErrBadRequest when input is nil, got %v", err)
	}
}

func TestImport_Validation(t *testing.T) {
	svc, repo, _ := newServiceForTest()
	ctx := context.Background()
	ds, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "ds"})

	// Empty payload → BadRequest, no inserts.
	if _, err := svc.Import(ctx, "proj", ds.ID, ImportDatasetItemsRequest{}); !errors.Is(err, entity.ErrBadRequest) {
		t.Errorf("empty import: want ErrBadRequest, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Errorf("empty import should not insert anything, got %d", len(repo.items))
	}

	// One bad row → fail-fast, no inserts (the service validates BEFORE handing
	// to the store, so partial-success is impossible).
	bad := ImportDatasetItemsRequest{Items: []CreateDatasetItemRequest{
		{Input: "ok"},
		{Input: nil}, // <-- bad
	}}
	if _, err := svc.Import(ctx, "proj", ds.ID, bad); !errors.Is(err, entity.ErrBadRequest) {
		t.Errorf("partial-bad import: want ErrBadRequest, got %v", err)
	}
	if len(repo.items) != 0 {
		t.Errorf("partial-bad import must not have inserted anything, got %d", len(repo.items))
	}

	// All-good import works.
	good := ImportDatasetItemsRequest{Items: []CreateDatasetItemRequest{
		{Input: "a"},
		{Input: "b"},
	}}
	resp, err := svc.Import(ctx, "proj", ds.ID, good)
	if err != nil {
		t.Fatalf("good import: %v", err)
	}
	if resp.Created != 2 {
		t.Errorf("want Created=2, got %d", resp.Created)
	}
}

func TestImport_RespectsCap(t *testing.T) {
	svc, _, _ := newServiceForTest()
	ctx := context.Background()
	ds, _ := svc.Create(ctx, "proj", CreateDatasetRequest{Name: "ds"})

	items := make([]CreateDatasetItemRequest, maxItemsPerImport+1)
	for i := range items {
		items[i] = CreateDatasetItemRequest{Input: i}
	}
	if _, err := svc.Import(ctx, "proj", ds.ID, ImportDatasetItemsRequest{Items: items}); !errors.Is(err, entity.ErrBadRequest) {
		t.Errorf("over-cap import: want ErrBadRequest, got %v", err)
	}
}

func TestIsUnsupported_WrapsClickHouse(t *testing.T) {
	// Sanity: when the repo returns ErrUnsupported (i.e. ClickHouse-as-primary),
	// the service surfaces it intact so the handler can map to 501.
	svc, repo, _ := newServiceForTest()
	repo.failNext = entity.ErrUnsupported

	_, err := svc.Create(context.Background(), "proj", CreateDatasetRequest{Name: "ds"})
	if !IsUnsupported(err) {
		t.Errorf("expected unsupported error to propagate, got %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
