// Package dataset is the application-layer service for curating and
// managing evaluation datasets — Phase 1 of the evals & prompt management
// feature (see specs/20260515-evals-and-prompt-management).
//
// The service follows the Interface Segregation Principle: instead of
// depending on the full repository.Store, it declares the narrow interfaces
// it consumes (DatasetRepo, SpanReader). Wiring passes the primary store
// (for dataset persistence) and the analytics store (for reading the spans
// that seed a dataset item from a real trace) — both happen to satisfy the
// narrow interfaces.
package dataset

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lelemon/server/pkg/domain/entity"
)

// DatasetRepo is the dataset-persistence surface this service needs.
// Satisfied by repository.DatasetStore (and therefore by *sqlite.Store and
// *postgres.Store). *clickhouse.Store also satisfies it, but every method
// returns entity.ErrUnsupported — datasets must live on a relational primary.
type DatasetRepo interface {
	CreateDataset(ctx context.Context, d *entity.Dataset) error
	GetDataset(ctx context.Context, projectID, datasetID string) (*entity.Dataset, error)
	ListDatasets(ctx context.Context, projectID string, filter entity.DatasetFilter) (*entity.Page[entity.Dataset], error)
	UpdateDataset(ctx context.Context, projectID, datasetID string, updates entity.DatasetUpdate) error
	DeleteDataset(ctx context.Context, projectID, datasetID string) error

	CreateDatasetItem(ctx context.Context, item *entity.DatasetItem) error
	BulkCreateDatasetItems(ctx context.Context, items []entity.DatasetItem) error
	GetDatasetItem(ctx context.Context, projectID, itemID string) (*entity.DatasetItem, error)
	ListDatasetItems(ctx context.Context, projectID, datasetID string, filter entity.DatasetItemFilter) (*entity.Page[entity.DatasetItem], error)
	DeleteDatasetItem(ctx context.Context, projectID, itemID string) error
}

// SpanReader is the slice of the trace store this service uses to seed a
// dataset item from a real trace. The reader is project-scoped — GetTrace
// already filters by projectID, so cross-tenant access is impossible.
type SpanReader interface {
	GetTrace(ctx context.Context, projectID, traceID string) (*entity.TraceWithSpans, error)
}

// Validation limits — chosen conservatively; tighten/loosen via real usage.
const (
	maxDatasetName        = 200
	maxDatasetDescription = 2000
	maxItemsPerImport     = 1000
)

// Service is the dataset use-case orchestrator. Construct via NewService.
type Service struct {
	repo  DatasetRepo
	spans SpanReader
}

// NewService wires a dataset service. Pass the primary store as `repo` and
// the analytics store as `spans` — when both are the same store, just pass
// it twice.
func NewService(repo DatasetRepo, spans SpanReader) *Service {
	return &Service{repo: repo, spans: spans}
}

// ============================================
// DATASETS
// ============================================

// Create validates and persists a new dataset under the given project.
func (s *Service) Create(ctx context.Context, projectID string, req CreateDatasetRequest) (*DatasetView, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", entity.ErrBadRequest)
	}
	if len(name) > maxDatasetName {
		return nil, fmt.Errorf("%w: name exceeds %d chars", entity.ErrBadRequest, maxDatasetName)
	}
	if req.Description != nil && len(*req.Description) > maxDatasetDescription {
		return nil, fmt.Errorf("%w: description exceeds %d chars", entity.ErrBadRequest, maxDatasetDescription)
	}

	d := &entity.Dataset{
		ProjectID:   projectID,
		Name:        name,
		Description: req.Description,
	}
	if err := s.repo.CreateDataset(ctx, d); err != nil {
		return nil, fmt.Errorf("create dataset: %w", err)
	}
	v := toDatasetView(d)
	return &v, nil
}

// Get returns a dataset by id, scoped to the project.
func (s *Service) Get(ctx context.Context, projectID, datasetID string) (*DatasetView, error) {
	d, err := s.repo.GetDataset(ctx, projectID, datasetID)
	if err != nil {
		return nil, err // ErrNotFound propagates unchanged
	}
	v := toDatasetView(d)
	return &v, nil
}

// List returns a paginated list of datasets in the project.
func (s *Service) List(ctx context.Context, projectID string, filter entity.DatasetFilter) (*DatasetListResponse, error) {
	page, err := s.repo.ListDatasets(ctx, projectID, filter)
	if err != nil {
		return nil, fmt.Errorf("list datasets: %w", err)
	}
	resp := toDatasetListResponse(page)
	return &resp, nil
}

// Update applies a partial patch. Empty updates are a no-op (no DB call).
func (s *Service) Update(ctx context.Context, projectID, datasetID string, req UpdateDatasetRequest) error {
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return fmt.Errorf("%w: name cannot be empty", entity.ErrBadRequest)
		}
		if len(trimmed) > maxDatasetName {
			return fmt.Errorf("%w: name exceeds %d chars", entity.ErrBadRequest, maxDatasetName)
		}
		req.Name = &trimmed
	}
	if req.Description != nil && len(*req.Description) > maxDatasetDescription {
		return fmt.Errorf("%w: description exceeds %d chars", entity.ErrBadRequest, maxDatasetDescription)
	}
	return s.repo.UpdateDataset(ctx, projectID, datasetID, entity.DatasetUpdate{
		Name:        req.Name,
		Description: req.Description,
	})
}

// Delete removes the dataset and (via DB cascade) all its items.
func (s *Service) Delete(ctx context.Context, projectID, datasetID string) error {
	return s.repo.DeleteDataset(ctx, projectID, datasetID)
}

// ============================================
// DATASET ITEMS
// ============================================

// CreateItem authors a new item manually inside a dataset. The dataset must
// belong to the project (verified up front so the caller gets a clean 404
// instead of a foreign-key failure later).
func (s *Service) CreateItem(ctx context.Context, projectID, datasetID string, req CreateDatasetItemRequest) (*DatasetItemView, error) {
	if _, err := s.repo.GetDataset(ctx, projectID, datasetID); err != nil {
		return nil, err
	}
	if req.Input == nil {
		return nil, fmt.Errorf("%w: input is required", entity.ErrBadRequest)
	}

	item := &entity.DatasetItem{
		DatasetID: datasetID,
		ProjectID: projectID,
		Input:     req.Input,
		Expected:  req.Expected,
		Metadata:  req.Metadata,
	}
	if err := s.repo.CreateDatasetItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create dataset item: %w", err)
	}
	v := toDatasetItemView(item)
	return &v, nil
}

// AddItemFromTrace seeds a new item from a real production span.
//
// The trace is fetched through the SpanReader (which is project-scoped, so
// cross-tenant access is impossible at the DB layer). We locate the target
// span by id and copy its `input` into the item. We deliberately do NOT
// pre-fill `expected` from `span.output` — see dto.go for the reasoning.
// SourceTraceID / SourceSpanID are persisted for provenance.
func (s *Service) AddItemFromTrace(ctx context.Context, projectID, datasetID string, req AddDatasetItemFromTraceRequest) (*DatasetItemView, error) {
	if req.TraceID == "" || req.SpanID == "" {
		return nil, fmt.Errorf("%w: traceId and spanId are required", entity.ErrBadRequest)
	}
	if _, err := s.repo.GetDataset(ctx, projectID, datasetID); err != nil {
		return nil, err
	}

	trace, err := s.spans.GetTrace(ctx, projectID, req.TraceID)
	if err != nil {
		return nil, err // ErrNotFound propagates (incl. tenant mismatch)
	}

	var src *entity.Span
	for i := range trace.Spans {
		if trace.Spans[i].ID == req.SpanID {
			src = &trace.Spans[i]
			break
		}
	}
	if src == nil {
		return nil, fmt.Errorf("span %s not found in trace %s: %w", req.SpanID, req.TraceID, entity.ErrNotFound)
	}

	traceID, spanID := req.TraceID, req.SpanID
	item := &entity.DatasetItem{
		DatasetID:     datasetID,
		ProjectID:     projectID,
		Input:         src.Input,
		Expected:      req.Expected,
		Metadata:      req.Metadata,
		SourceTraceID: &traceID,
		SourceSpanID:  &spanID,
	}
	if err := s.repo.CreateDatasetItem(ctx, item); err != nil {
		return nil, fmt.Errorf("create dataset item from trace: %w", err)
	}
	v := toDatasetItemView(item)
	return &v, nil
}

// Import bulk-creates items inside a dataset. All-or-nothing: the underlying
// store wraps the insert in a transaction, so a single bad row aborts the
// whole batch and returns the error.
func (s *Service) Import(ctx context.Context, projectID, datasetID string, req ImportDatasetItemsRequest) (*ImportResponse, error) {
	if len(req.Items) == 0 {
		return nil, fmt.Errorf("%w: items is empty", entity.ErrBadRequest)
	}
	if len(req.Items) > maxItemsPerImport {
		return nil, fmt.Errorf("%w: import capped at %d items per request", entity.ErrBadRequest, maxItemsPerImport)
	}
	if _, err := s.repo.GetDataset(ctx, projectID, datasetID); err != nil {
		return nil, err
	}

	items := make([]entity.DatasetItem, 0, len(req.Items))
	for i, r := range req.Items {
		if r.Input == nil {
			return nil, fmt.Errorf("%w: items[%d].input is required", entity.ErrBadRequest, i)
		}
		items = append(items, entity.DatasetItem{
			DatasetID: datasetID,
			ProjectID: projectID,
			Input:     r.Input,
			Expected:  r.Expected,
			Metadata:  r.Metadata,
		})
	}
	if err := s.repo.BulkCreateDatasetItems(ctx, items); err != nil {
		return nil, fmt.Errorf("import dataset items: %w", err)
	}
	return &ImportResponse{Created: len(items)}, nil
}

// ListItems returns a paginated list of items in a dataset.
func (s *Service) ListItems(ctx context.Context, projectID, datasetID string, filter entity.DatasetItemFilter) (*DatasetItemListResponse, error) {
	if _, err := s.repo.GetDataset(ctx, projectID, datasetID); err != nil {
		return nil, err
	}
	page, err := s.repo.ListDatasetItems(ctx, projectID, datasetID, filter)
	if err != nil {
		return nil, fmt.Errorf("list dataset items: %w", err)
	}
	resp := toDatasetItemListResponse(page)
	return &resp, nil
}

// GetItem returns a single item, verifying it belongs to the URL's dataset.
// The repo lookup is by (projectID, itemID); the dataset id in the URL is an
// extra contract we enforce here — if the item exists but under a different
// dataset, we return ErrNotFound (don't leak its real parent).
func (s *Service) GetItem(ctx context.Context, projectID, datasetID, itemID string) (*DatasetItemView, error) {
	item, err := s.repo.GetDatasetItem(ctx, projectID, itemID)
	if err != nil {
		return nil, err
	}
	if item.DatasetID != datasetID {
		return nil, entity.ErrNotFound
	}
	v := toDatasetItemView(item)
	return &v, nil
}

// DeleteItem removes one item. Same dataset-id contract as GetItem.
func (s *Service) DeleteItem(ctx context.Context, projectID, datasetID, itemID string) error {
	item, err := s.repo.GetDatasetItem(ctx, projectID, itemID)
	if err != nil {
		return err
	}
	if item.DatasetID != datasetID {
		return entity.ErrNotFound
	}
	return s.repo.DeleteDatasetItem(ctx, projectID, itemID)
}

// IsUnsupported reports whether err comes from a backend that doesn't support
// dataset operations (i.e. ClickHouse-as-primary). Exposed so handlers can map
// it to a clear 501 Not Implemented instead of a generic 500.
func IsUnsupported(err error) bool {
	return errors.Is(err, entity.ErrUnsupported)
}
