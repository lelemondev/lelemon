package dataset

import (
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

// DatasetView is the wire shape returned by dataset endpoints.
// Fields use camelCase tags — matching the convention in trace/response.go,
// not the (stale) PascalCase guidance in .claude/rules/go-patterns.md.
type DatasetView struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// DatasetItemView is the wire shape for one item.
type DatasetItemView struct {
	ID            string         `json:"id"`
	DatasetID     string         `json:"datasetId"`
	ProjectID     string         `json:"projectId"`
	Input         any            `json:"input"`
	Expected      any            `json:"expected"`
	Metadata      map[string]any `json:"metadata"`
	SourceTraceID *string        `json:"sourceTraceId"`
	SourceSpanID  *string        `json:"sourceSpanId"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// DatasetListResponse and DatasetItemListResponse are camelCased pagination
// envelopes. The domain Page[T] uses PascalCase tags; we don't propagate that
// to the wire for new endpoints.
type DatasetListResponse struct {
	Data   []DatasetView `json:"data"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type DatasetItemListResponse struct {
	Data   []DatasetItemView `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

// ImportResponse is returned by the bulk-import endpoint.
type ImportResponse struct {
	Created int `json:"created"`
}

// ----- mappers ------------------------------------------------------------

func toDatasetView(d *entity.Dataset) DatasetView {
	return DatasetView{
		ID:          d.ID,
		ProjectID:   d.ProjectID,
		Name:        d.Name,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func toDatasetItemView(it *entity.DatasetItem) DatasetItemView {
	meta := it.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	return DatasetItemView{
		ID:            it.ID,
		DatasetID:     it.DatasetID,
		ProjectID:     it.ProjectID,
		Input:         it.Input,
		Expected:      it.Expected,
		Metadata:      meta,
		SourceTraceID: it.SourceTraceID,
		SourceSpanID:  it.SourceSpanID,
		CreatedAt:     it.CreatedAt,
		UpdatedAt:     it.UpdatedAt,
	}
}

func toDatasetListResponse(p *entity.Page[entity.Dataset]) DatasetListResponse {
	views := make([]DatasetView, len(p.Data))
	for i := range p.Data {
		views[i] = toDatasetView(&p.Data[i])
	}
	return DatasetListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}

func toDatasetItemListResponse(p *entity.Page[entity.DatasetItem]) DatasetItemListResponse {
	views := make([]DatasetItemView, len(p.Data))
	for i := range p.Data {
		views[i] = toDatasetItemView(&p.Data[i])
	}
	return DatasetItemListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}
