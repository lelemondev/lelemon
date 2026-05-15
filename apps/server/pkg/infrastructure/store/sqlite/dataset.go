package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// DATASET OPERATIONS
// ============================================
//
// Datasets and items are relational, project-scoped, and always filtered by
// project_id (multi-tenant rule). source_trace_id / source_span_id are stored
// but NOT FK-checked — traces may live in a separate analytics store.

func (s *Store) CreateDataset(ctx context.Context, d *entity.Dataset) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO datasets (id, project_id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, d.ID, d.ProjectID, d.Name, d.Description, d.CreatedAt, d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create dataset: %w", err)
	}
	return nil
}

func (s *Store) GetDataset(ctx context.Context, projectID, datasetID string) (*entity.Dataset, error) {
	var d entity.Dataset
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM datasets WHERE project_id = ? AND id = ?
	`, projectID, datasetID).Scan(&d.ID, &d.ProjectID, &d.Name, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, entity.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	return &d, nil
}

func (s *Store) ListDatasets(ctx context.Context, projectID string, filter entity.DatasetFilter) (*entity.Page[entity.Dataset], error) {
	where := []string{"project_id = ?"}
	args := []any{projectID}

	if filter.Name != nil && *filter.Name != "" {
		where = append(where, "name LIKE ?")
		args = append(args, "%"+*filter.Name+"%")
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM datasets WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count datasets: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, name, description, created_at, updated_at
		FROM datasets WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, args...)
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

	if updates.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *updates.Description)
	}
	if len(sets) == 0 {
		return nil // no-op
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now())

	args = append(args, projectID, datasetID)
	res, err := s.db.ExecContext(ctx, `
		UPDATE datasets SET `+strings.Join(sets, ", ")+`
		WHERE project_id = ? AND id = ?
	`, args...)
	if err != nil {
		return fmt.Errorf("update dataset: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update dataset rows affected: %w", err)
	}
	if affected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteDataset(ctx context.Context, projectID, datasetID string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM datasets WHERE project_id = ? AND id = ?`,
		projectID, datasetID,
	)
	if err != nil {
		return fmt.Errorf("delete dataset: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete dataset rows affected: %w", err)
	}
	if affected == 0 {
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

	inputJSON, err := marshalJSON(item.Input)
	if err != nil {
		return fmt.Errorf("marshal item input: %w", err)
	}
	expectedJSON, err := marshalJSONOptional(item.Expected)
	if err != nil {
		return fmt.Errorf("marshal item expected: %w", err)
	}
	metadataJSON, err := marshalMetadata(item.Metadata)
	if err != nil {
		return fmt.Errorf("marshal item metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO dataset_items
			(id, dataset_id, project_id, input, expected, metadata, source_trace_id, source_span_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bulk insert: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // safe after Commit

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO dataset_items
			(id, dataset_id, project_id, input, expected, metadata, source_trace_id, source_span_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare bulk insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for i := range items {
		it := &items[i]
		if it.ID == "" {
			it.ID = uuid.New().String()
		}
		it.CreatedAt = now
		it.UpdatedAt = now

		inputJSON, err := marshalJSON(it.Input)
		if err != nil {
			return fmt.Errorf("marshal item[%d] input: %w", i, err)
		}
		expectedJSON, err := marshalJSONOptional(it.Expected)
		if err != nil {
			return fmt.Errorf("marshal item[%d] expected: %w", i, err)
		}
		metadataJSON, err := marshalMetadata(it.Metadata)
		if err != nil {
			return fmt.Errorf("marshal item[%d] metadata: %w", i, err)
		}

		if _, err := stmt.ExecContext(ctx,
			it.ID, it.DatasetID, it.ProjectID, inputJSON, expectedJSON, metadataJSON,
			it.SourceTraceID, it.SourceSpanID, it.CreatedAt, it.UpdatedAt,
		); err != nil {
			return fmt.Errorf("bulk insert item[%d]: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bulk insert: %w", err)
	}
	return nil
}

func (s *Store) GetDatasetItem(ctx context.Context, projectID, itemID string) (*entity.DatasetItem, error) {
	var (
		it           entity.DatasetItem
		inputJSON    string
		expectedJSON sql.NullString
		metadataJSON string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, dataset_id, project_id, input, expected, metadata,
		       source_trace_id, source_span_id, created_at, updated_at
		FROM dataset_items WHERE project_id = ? AND id = ?
	`, projectID, itemID).Scan(
		&it.ID, &it.DatasetID, &it.ProjectID,
		&inputJSON, &expectedJSON, &metadataJSON,
		&it.SourceTraceID, &it.SourceSpanID,
		&it.CreatedAt, &it.UpdatedAt,
	)
	if err == sql.ErrNoRows {
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
	where := []string{"project_id = ?", "dataset_id = ?"}
	args := []any{projectID, datasetID}

	if filter.SourceTraceID != nil && *filter.SourceTraceID != "" {
		where = append(where, "source_trace_id = ?")
		args = append(args, *filter.SourceTraceID)
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dataset_items WHERE `+whereClause, args...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count dataset items: %w", err)
	}

	limit, offset := pageBounds(filter.Limit, filter.Offset)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dataset_id, project_id, input, expected, metadata,
		       source_trace_id, source_span_id, created_at, updated_at
		FROM dataset_items WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list dataset items: %w", err)
	}
	defer rows.Close()

	out := make([]entity.DatasetItem, 0)
	for rows.Next() {
		var (
			it           entity.DatasetItem
			inputJSON    string
			expectedJSON sql.NullString
			metadataJSON string
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
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM dataset_items WHERE project_id = ? AND id = ?`,
		projectID, itemID,
	)
	if err != nil {
		return fmt.Errorf("delete dataset item: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete dataset item rows affected: %w", err)
	}
	if affected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// ----- helpers (package-local; reusable across dataset.go) ----------------

// pageBounds clamps Limit/Offset to sane values and applies defaults.
func pageBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return limit, max(offset, 0)
}

// marshalJSON serializes any value as a JSON string, even nil → "null".
func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// marshalJSONOptional returns sql.NullString — nil interface → NULL in the column.
// Use for `expected` where "no expectation" is meaningfully distinct from "expected: null".
func marshalJSONOptional(v any) (sql.NullString, error) {
	if v == nil {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

// marshalMetadata serializes map metadata to JSON; nil/empty → "{}".
func marshalMetadata(m map[string]any) (string, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalItemJSON decodes the three JSON columns into the entity.
func unmarshalItemJSON(it *entity.DatasetItem, inputJSON string, expectedJSON sql.NullString, metadataJSON string) error {
	if inputJSON != "" {
		if err := json.Unmarshal([]byte(inputJSON), &it.Input); err != nil {
			return fmt.Errorf("unmarshal input: %w", err)
		}
	}
	if expectedJSON.Valid && expectedJSON.String != "" {
		if err := json.Unmarshal([]byte(expectedJSON.String), &it.Expected); err != nil {
			return fmt.Errorf("unmarshal expected: %w", err)
		}
	}
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &it.Metadata); err != nil {
			return fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return nil
}
