package clickhouse

import (
	"context"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// DATASET OPERATIONS — UNSUPPORTED
// ============================================
//
// Datasets, dataset items, evals and prompts are relational data with frequent
// UPDATEs and partial reads. ClickHouse is a columnar OLAP store optimised for
// append-heavy analytical workloads — it has no real UPDATE primitive, only
// ReplacingMergeTree workarounds. Forcing these tables into ClickHouse would
// trade good defaults for a worse experience.
//
// These methods exist so *clickhouse.Store satisfies repository.Store at
// compile time. They return entity.ErrUnsupported at runtime. The application
// layer is responsible for routing dataset operations to the primary store —
// which must be SQLite or Postgres. Document this in the deployment guide.

func (s *Store) CreateDataset(ctx context.Context, d *entity.Dataset) error {
	return entity.ErrUnsupported
}

func (s *Store) GetDataset(ctx context.Context, projectID, datasetID string) (*entity.Dataset, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) ListDatasets(ctx context.Context, projectID string, filter entity.DatasetFilter) (*entity.Page[entity.Dataset], error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) UpdateDataset(ctx context.Context, projectID, datasetID string, updates entity.DatasetUpdate) error {
	return entity.ErrUnsupported
}

func (s *Store) DeleteDataset(ctx context.Context, projectID, datasetID string) error {
	return entity.ErrUnsupported
}

func (s *Store) CreateDatasetItem(ctx context.Context, item *entity.DatasetItem) error {
	return entity.ErrUnsupported
}

func (s *Store) BulkCreateDatasetItems(ctx context.Context, items []entity.DatasetItem) error {
	return entity.ErrUnsupported
}

func (s *Store) GetDatasetItem(ctx context.Context, projectID, itemID string) (*entity.DatasetItem, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) ListDatasetItems(ctx context.Context, projectID, datasetID string, filter entity.DatasetItemFilter) (*entity.Page[entity.DatasetItem], error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) DeleteDatasetItem(ctx context.Context, projectID, itemID string) error {
	return entity.ErrUnsupported
}
