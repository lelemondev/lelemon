package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// DATASET OPERATIONS
// ============================================
//
// Datasets and items are relational and project-scoped. All queries filter by
// project_id (multi-tenant rule). source_trace_id / source_span_id are stored
// but NOT FK-checked — traces may live in a separate analytics store.

func (s *Store) CreateDataset(ctx context.Context, d *entity.Dataset) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	_, err := s.pool.Exec(ctx, `
		INSERT INTO datasets (id, project_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, d.ID, d.ProjectID, d.Name, d.Description, d.CreatedAt, d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create dataset: %w", err)
	}
	return nil
}

func (s *Store) GetDataset(ctx context.Context, projectID, datasetID string) (*entity.Dataset, error) {
	var d entity.Dataset
	err := s.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM datasets WHERE project_id = $1 AND id = $2
	`, projectID, datasetID).Scan(&d.ID, &d.ProjectID, &d.Name, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	return &d, nil
}

func (s *Store) ListDatasets(ctx context.Context, projectID string, filter entity.DatasetFilter) (*entity.Page[entity.Dataset], error) {
	where := []string{"project_id = $1"}
	args := []any{projectID}
	argN := 2

	if filter.Name != nil && *filter.Name != "" {
		where = append(where, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+*filter.Name+"%")
		argN++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM datasets WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count datasets: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	limitN, offsetN := argN, argN+1

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, project_id, name, description, created_at, updated_at
		FROM datasets WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitN, offsetN), args...)
	if err != nil {
		return nil, fmt.Errorf("list datasets: %w", err)
	}
	defer rows.Close()

	out := make([]entity.Dataset, 0)
	for rows.Next() {
		var d entity.Dataset
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.Name, &d.Description, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan dataset: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate datasets: %w", err)
	}

	return &entity.Page[entity.Dataset]{
		Data:   out,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Store) UpdateDataset(ctx context.Context, projectID, datasetID string, updates entity.DatasetUpdate) error {
	sets := []string{}
	args := []any{}
	argN := 1

	if updates.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argN))
		args = append(args, *updates.Name)
		argN++
	}
	if updates.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argN))
		args = append(args, *updates.Description)
		argN++
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, fmt.Sprintf("updated_at = $%d", argN))
	args = append(args, time.Now())
	argN++

	args = append(args, projectID, datasetID)
	query := fmt.Sprintf(
		`UPDATE datasets SET %s WHERE project_id = $%d AND id = $%d`,
		strings.Join(sets, ", "), argN, argN+1,
	)
	tag, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update dataset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteDataset(ctx context.Context, projectID, datasetID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM datasets WHERE project_id = $1 AND id = $2`,
		projectID, datasetID,
	)
	if err != nil {
		return fmt.Errorf("delete dataset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// ============================================
// DATASET ITEM OPERATIONS
// ============================================

func (s *Store) CreateDatasetItem(ctx context.Context, item *entity.DatasetItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	inputJSON, err := json.Marshal(item.Input)
	if err != nil {
		return fmt.Errorf("marshal item input: %w", err)
	}
	expectedJSON, err := marshalOptionalJSON(item.Expected)
	if err != nil {
		return fmt.Errorf("marshal item expected: %w", err)
	}
	metadataJSON, err := marshalMetadata(item.Metadata)
	if err != nil {
		return fmt.Errorf("marshal item metadata: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO dataset_items
			(id, dataset_id, project_id, input, expected, metadata, source_trace_id, source_span_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, item.ID, item.DatasetID, item.ProjectID, inputJSON, expectedJSON, metadataJSON,
		item.SourceTraceID, item.SourceSpanID, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create dataset item: %w", err)
	}
	return nil
}

func (s *Store) BulkCreateDatasetItems(ctx context.Context, items []entity.DatasetItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bulk insert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // safe after Commit

	now := time.Now()
	batch := &pgx.Batch{}
	for i := range items {
		it := &items[i]
		if it.ID == "" {
			it.ID = uuid.New().String()
		}
		it.CreatedAt = now
		it.UpdatedAt = now

		inputJSON, err := json.Marshal(it.Input)
		if err != nil {
			return fmt.Errorf("marshal item[%d] input: %w", i, err)
		}
		expectedJSON, err := marshalOptionalJSON(it.Expected)
		if err != nil {
			return fmt.Errorf("marshal item[%d] expected: %w", i, err)
		}
		metadataJSON, err := marshalMetadata(it.Metadata)
		if err != nil {
			return fmt.Errorf("marshal item[%d] metadata: %w", i, err)
		}

		batch.Queue(`
			INSERT INTO dataset_items
				(id, dataset_id, project_id, input, expected, metadata, source_trace_id, source_span_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, it.ID, it.DatasetID, it.ProjectID, inputJSON, expectedJSON, metadataJSON,
			it.SourceTraceID, it.SourceSpanID, it.CreatedAt, it.UpdatedAt)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return fmt.Errorf("bulk insert item[%d]: %w", i, err)
		}
	}
	if err := br.Close(); err != nil {
		return fmt.Errorf("close bulk batch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bulk insert: %w", err)
	}
	return nil
}

func (s *Store) GetDatasetItem(ctx context.Context, projectID, itemID string) (*entity.DatasetItem, error) {
	var (
		it           entity.DatasetItem
		inputJSON    []byte
		expectedJSON []byte // nil when SQL NULL
		metadataJSON []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT id, dataset_id, project_id, input, expected, metadata,
		       source_trace_id, source_span_id, created_at, updated_at
		FROM dataset_items WHERE project_id = $1 AND id = $2
	`, projectID, itemID).Scan(
		&it.ID, &it.DatasetID, &it.ProjectID,
		&inputJSON, &expectedJSON, &metadataJSON,
		&it.SourceTraceID, &it.SourceSpanID,
		&it.CreatedAt, &it.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dataset item: %w", err)
	}
	if err := unmarshalItemJSON(&it, inputJSON, expectedJSON, metadataJSON); err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Store) ListDatasetItems(ctx context.Context, projectID, datasetID string, filter entity.DatasetItemFilter) (*entity.Page[entity.DatasetItem], error) {
	where := []string{"project_id = $1", "dataset_id = $2"}
	args := []any{projectID, datasetID}
	argN := 3

	if filter.SourceTraceID != nil && *filter.SourceTraceID != "" {
		where = append(where, fmt.Sprintf("source_trace_id = $%d", argN))
		args = append(args, *filter.SourceTraceID)
		argN++
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM dataset_items WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count dataset items: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)
	limitN, offsetN := argN, argN+1

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, dataset_id, project_id, input, expected, metadata,
		       source_trace_id, source_span_id, created_at, updated_at
		FROM dataset_items WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitN, offsetN), args...)
	if err != nil {
		return nil, fmt.Errorf("list dataset items: %w", err)
	}
	defer rows.Close()

	out := make([]entity.DatasetItem, 0)
	for rows.Next() {
		var (
			it           entity.DatasetItem
			inputJSON    []byte
			expectedJSON []byte
			metadataJSON []byte
		)
		if err := rows.Scan(
			&it.ID, &it.DatasetID, &it.ProjectID,
			&inputJSON, &expectedJSON, &metadataJSON,
			&it.SourceTraceID, &it.SourceSpanID,
			&it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan dataset item: %w", err)
		}
		if err := unmarshalItemJSON(&it, inputJSON, expectedJSON, metadataJSON); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dataset items: %w", err)
	}

	return &entity.Page[entity.DatasetItem]{
		Data:   out,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *Store) DeleteDatasetItem(ctx context.Context, projectID, itemID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM dataset_items WHERE project_id = $1 AND id = $2`,
		projectID, itemID,
	)
	if err != nil {
		return fmt.Errorf("delete dataset item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// ----- helpers (package-local; reusable across dataset.go) ----------------

func pageBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return limit, max(offset, 0)
}

// marshalOptionalJSON returns nil bytes when the value is a nil interface, so the
// column ends up as SQL NULL (meaningfully distinct from JSON null).
func marshalOptionalJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// marshalMetadata serializes a metadata map; nil → "{}" to match the column default.
func marshalMetadata(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

// unmarshalItemJSON decodes the three JSON columns into the entity.
func unmarshalItemJSON(it *entity.DatasetItem, inputJSON, expectedJSON, metadataJSON []byte) error {
	if len(inputJSON) > 0 {
		if err := json.Unmarshal(inputJSON, &it.Input); err != nil {
			return fmt.Errorf("unmarshal input: %w", err)
		}
	}
	if len(expectedJSON) > 0 {
		if err := json.Unmarshal(expectedJSON, &it.Expected); err != nil {
			return fmt.Errorf("unmarshal expected: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &it.Metadata); err != nil {
			return fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return nil
}
