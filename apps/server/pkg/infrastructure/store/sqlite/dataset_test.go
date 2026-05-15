package sqlite

import (
	"context"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

// datasetTestStore spins up an in-memory SQLite store with two projects so we
// can exercise multi-tenant isolation as the headline property.
func datasetTestStore(t *testing.T) (*Store, *entity.Project, *entity.Project) {
	t.Helper()

	store, err := New(t.TempDir() + "/datasets.db")
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	a := &entity.Project{Name: "A", APIKey: "le_a", APIKeyHash: "hash_a", OwnerEmail: "a@test"}
	b := &entity.Project{Name: "B", APIKey: "le_b", APIKeyHash: "hash_b", OwnerEmail: "b@test"}
	if err := store.CreateProject(ctx, a); err != nil {
		t.Fatalf("create project A: %v", err)
	}
	if err := store.CreateProject(ctx, b); err != nil {
		t.Fatalf("create project B: %v", err)
	}
	return store, a, b
}

func TestDataset_CRUD(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	desc := "venpu agent regression cases"
	d := &entity.Dataset{
		ProjectID:   projA.ID,
		Name:        "vehicle-search",
		Description: &desc,
	}
	if err := store.CreateDataset(ctx, d); err != nil {
		t.Fatalf("create dataset: %v", err)
	}
	if d.ID == "" {
		t.Fatal("expected store to fill ID")
	}
	if d.CreatedAt.IsZero() {
		t.Fatal("expected store to fill CreatedAt")
	}

	got, err := store.GetDataset(ctx, projA.ID, d.ID)
	if err != nil {
		t.Fatalf("get dataset: %v", err)
	}
	if got.Name != "vehicle-search" || got.Description == nil || *got.Description != desc {
		t.Errorf("get returned wrong values: %+v", got)
	}

	newName := "vehicle-search-v2"
	if err := store.UpdateDataset(ctx, projA.ID, d.ID, entity.DatasetUpdate{Name: &newName}); err != nil {
		t.Fatalf("update dataset: %v", err)
	}
	got, err = store.GetDataset(ctx, projA.ID, d.ID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Name != "vehicle-search-v2" {
		t.Errorf("update did not persist: got name %q", got.Name)
	}
	if got.UpdatedAt.Before(got.CreatedAt) {
		t.Errorf("UpdatedAt regressed before CreatedAt: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}

	if err := store.DeleteDataset(ctx, projA.ID, d.ID); err != nil {
		t.Fatalf("delete dataset: %v", err)
	}
	if _, err := store.GetDataset(ctx, projA.ID, d.ID); err != entity.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDataset_NotFoundErrors(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	if _, err := store.GetDataset(ctx, projA.ID, "00000000-0000-0000-0000-000000000000"); err != entity.ErrNotFound {
		t.Errorf("Get on missing dataset: want ErrNotFound, got %v", err)
	}
	if err := store.UpdateDataset(ctx, projA.ID, "missing", entity.DatasetUpdate{Name: ptr("x")}); err != entity.ErrNotFound {
		t.Errorf("Update on missing dataset: want ErrNotFound, got %v", err)
	}
	if err := store.DeleteDataset(ctx, projA.ID, "missing"); err != entity.ErrNotFound {
		t.Errorf("Delete on missing dataset: want ErrNotFound, got %v", err)
	}
}

func TestDataset_TenantIsolation(t *testing.T) {
	store, projA, projB := datasetTestStore(t)
	ctx := context.Background()

	dA := &entity.Dataset{ProjectID: projA.ID, Name: "A-set"}
	dB := &entity.Dataset{ProjectID: projB.ID, Name: "B-set"}
	if err := store.CreateDataset(ctx, dA); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if err := store.CreateDataset(ctx, dB); err != nil {
		t.Fatalf("create B: %v", err)
	}

	// Cross-tenant reads must miss, not bleed.
	if _, err := store.GetDataset(ctx, projB.ID, dA.ID); err != entity.ErrNotFound {
		t.Errorf("project B should not see project A's dataset: got %v", err)
	}
	if err := store.UpdateDataset(ctx, projB.ID, dA.ID, entity.DatasetUpdate{Name: ptr("hijack")}); err != entity.ErrNotFound {
		t.Errorf("project B should not be able to update project A's dataset: got %v", err)
	}
	if err := store.DeleteDataset(ctx, projB.ID, dA.ID); err != entity.ErrNotFound {
		t.Errorf("project B should not be able to delete project A's dataset: got %v", err)
	}

	// List must be project-scoped.
	pageA, err := store.ListDatasets(ctx, projA.ID, entity.DatasetFilter{})
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if pageA.Total != 1 || len(pageA.Data) != 1 || pageA.Data[0].Name != "A-set" {
		t.Errorf("project A list leaked or missed rows: %+v", pageA)
	}

	pageB, err := store.ListDatasets(ctx, projB.ID, entity.DatasetFilter{})
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if pageB.Total != 1 || pageB.Data[0].Name != "B-set" {
		t.Errorf("project B list leaked or missed rows: %+v", pageB)
	}
}

func TestDataset_ListFilterAndPaging(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	for _, name := range []string{"vehicle-search", "agent-greeting", "vehicle-recall"} {
		if err := store.CreateDataset(ctx, &entity.Dataset{ProjectID: projA.ID, Name: name}); err != nil {
			t.Fatalf("seed dataset %q: %v", name, err)
		}
	}

	// Name substring filter.
	filter := entity.DatasetFilter{Name: ptr("vehicle"), Limit: 50}
	page, err := store.ListDatasets(ctx, projA.ID, filter)
	if err != nil {
		t.Fatalf("filtered list: %v", err)
	}
	if page.Total != 2 || len(page.Data) != 2 {
		t.Errorf("name filter: want 2, got total=%d data=%d", page.Total, len(page.Data))
	}

	// Paging — limit 1 should return the most recent first (DESC order).
	page, err = store.ListDatasets(ctx, projA.ID, entity.DatasetFilter{Limit: 1})
	if err != nil {
		t.Fatalf("paged list: %v", err)
	}
	if page.Total != 3 || len(page.Data) != 1 {
		t.Errorf("paging: want total=3 data=1, got total=%d data=%d", page.Total, len(page.Data))
	}
}

func TestDatasetItem_CRUDAndCascade(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	ds := &entity.Dataset{ProjectID: projA.ID, Name: "ds"}
	if err := store.CreateDataset(ctx, ds); err != nil {
		t.Fatalf("create dataset: %v", err)
	}

	traceID := "trace-1"
	spanID := "span-1"
	item := &entity.DatasetItem{
		DatasetID: ds.ID,
		ProjectID: projA.ID,
		Input:     map[string]any{"query": "honda civic"},
		Expected:  map[string]any{"min_results": float64(1)},
		Metadata:  map[string]any{"category": "natural-language"},

		SourceTraceID: &traceID,
		SourceSpanID:  &spanID,
	}
	if err := store.CreateDatasetItem(ctx, item); err != nil {
		t.Fatalf("create item: %v", err)
	}

	got, err := store.GetDatasetItem(ctx, projA.ID, item.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if m, ok := got.Input.(map[string]any); !ok || m["query"] != "honda civic" {
		t.Errorf("input roundtrip failed: %#v", got.Input)
	}
	if got.SourceTraceID == nil || *got.SourceTraceID != traceID {
		t.Errorf("source_trace_id roundtrip failed: %v", got.SourceTraceID)
	}
	if got.Metadata["category"] != "natural-language" {
		t.Errorf("metadata roundtrip failed: %#v", got.Metadata)
	}

	// Cascade: deleting the parent dataset must remove its items.
	if err := store.DeleteDataset(ctx, projA.ID, ds.ID); err != nil {
		t.Fatalf("delete dataset: %v", err)
	}
	if _, err := store.GetDatasetItem(ctx, projA.ID, item.ID); err != entity.ErrNotFound {
		t.Errorf("expected item to cascade-delete with dataset, got %v", err)
	}
}

func TestDatasetItem_BulkCreateAndSourceTraceFilter(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	ds := &entity.Dataset{ProjectID: projA.ID, Name: "ds"}
	if err := store.CreateDataset(ctx, ds); err != nil {
		t.Fatalf("create dataset: %v", err)
	}

	traceX := "trace-X"
	items := []entity.DatasetItem{
		{DatasetID: ds.ID, ProjectID: projA.ID, Input: "a", SourceTraceID: &traceX},
		{DatasetID: ds.ID, ProjectID: projA.ID, Input: "b", SourceTraceID: &traceX},
		{DatasetID: ds.ID, ProjectID: projA.ID, Input: "c"}, // manual, no source
	}
	if err := store.BulkCreateDatasetItems(ctx, items); err != nil {
		t.Fatalf("bulk create: %v", err)
	}
	for i, it := range items {
		if it.ID == "" {
			t.Errorf("bulk: item[%d] missing ID after insert", i)
		}
	}

	all, err := store.ListDatasetItems(ctx, projA.ID, ds.ID, entity.DatasetItemFilter{})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if all.Total != 3 {
		t.Errorf("want 3 items, got %d", all.Total)
	}

	fromTrace, err := store.ListDatasetItems(ctx, projA.ID, ds.ID, entity.DatasetItemFilter{SourceTraceID: &traceX})
	if err != nil {
		t.Fatalf("list items filtered: %v", err)
	}
	if fromTrace.Total != 2 {
		t.Errorf("source_trace_id filter: want 2, got %d", fromTrace.Total)
	}
}

func TestDatasetItem_TenantIsolation(t *testing.T) {
	store, projA, projB := datasetTestStore(t)
	ctx := context.Background()

	dsA := &entity.Dataset{ProjectID: projA.ID, Name: "A"}
	if err := store.CreateDataset(ctx, dsA); err != nil {
		t.Fatalf("create A dataset: %v", err)
	}
	itemA := &entity.DatasetItem{DatasetID: dsA.ID, ProjectID: projA.ID, Input: "secret"}
	if err := store.CreateDatasetItem(ctx, itemA); err != nil {
		t.Fatalf("create A item: %v", err)
	}

	// Project B must not see Project A's item.
	if _, err := store.GetDatasetItem(ctx, projB.ID, itemA.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant Get item should miss: got %v", err)
	}
	if err := store.DeleteDatasetItem(ctx, projB.ID, itemA.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant Delete item should miss: got %v", err)
	}
	// Listing under project B with project A's dataset id must return nothing.
	page, err := store.ListDatasetItems(ctx, projB.ID, dsA.ID, entity.DatasetItemFilter{})
	if err != nil {
		t.Fatalf("cross-tenant list: %v", err)
	}
	if page.Total != 0 {
		t.Errorf("cross-tenant list should be empty, got %d rows", page.Total)
	}
}

// ptr is a tiny generic helper for taking the address of a literal.
func ptr[T any](v T) *T { return &v }
