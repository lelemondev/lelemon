package prompt

import (
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

// PromptView is the wire shape returned by prompt endpoints. camelCase JSON
// tags — matches the convention from trace/response.go.
type PromptView struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PromptVersionView is the wire shape for one version.
type PromptVersionView struct {
	ID        string    `json:"id"`
	PromptID  string    `json:"promptId"`
	ProjectID string    `json:"projectId"`
	Version   string    `json:"version"`
	Content   string    `json:"content"`
	Changelog *string   `json:"changelog"`
	CreatedBy *string   `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

// PromptListResponse / PromptVersionListResponse are camelCased pagination
// envelopes (the domain Page[T] uses PascalCase tags — not exposed).
type PromptListResponse struct {
	Data   []PromptView `json:"data"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type PromptVersionListResponse struct {
	Data   []PromptVersionView `json:"data"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// ----- mappers -----------------------------------------------------------

func toPromptView(p *entity.Prompt) PromptView {
	return PromptView{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toPromptVersionView(v *entity.PromptVersion) PromptVersionView {
	return PromptVersionView{
		ID:        v.ID,
		PromptID:  v.PromptID,
		ProjectID: v.ProjectID,
		Version:   v.Version,
		Content:   v.Content,
		Changelog: v.Changelog,
		CreatedBy: v.CreatedBy,
		CreatedAt: v.CreatedAt,
	}
}

func toPromptListResponse(p *entity.Page[entity.Prompt]) PromptListResponse {
	views := make([]PromptView, len(p.Data))
	for i := range p.Data {
		views[i] = toPromptView(&p.Data[i])
	}
	return PromptListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}

func toPromptVersionListResponse(p *entity.Page[entity.PromptVersion]) PromptVersionListResponse {
	views := make([]PromptVersionView, len(p.Data))
	for i := range p.Data {
		views[i] = toPromptVersionView(&p.Data[i])
	}
	return PromptVersionListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}
