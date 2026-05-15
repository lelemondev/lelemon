package eval

import (
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

// EvalView is the wire shape for an eval definition. Scorers travel as-is —
// their JSON tags are defined on entity.Scorer.
type EvalView struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"projectId"`
	DatasetID   string          `json:"datasetId"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Scorers     []entity.Scorer `json:"scorers"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// EvalRunView is the wire shape for one run. PassRate is a convenience —
// derived, not stored. It's nil for non-terminal runs.
type EvalRunView struct {
	ID              string                 `json:"id"`
	ProjectID       string                 `json:"projectId"`
	EvalID          string                 `json:"evalId"`
	Status          entity.EvalRunStatus   `json:"status"`
	PromptVersionID *string                `json:"promptVersionId"`
	Metadata        map[string]any         `json:"metadata"`
	TotalItems      int                    `json:"totalItems"`
	PassedItems     int                    `json:"passedItems"`
	FailedItems     int                    `json:"failedItems"`
	ErroredItems    int                    `json:"erroredItems"`
	PassRate        *float64               `json:"passRate"`
	DurationMs      *int                   `json:"durationMs"`
	CostUSD         *float64               `json:"costUsd"`
	StartedAt       time.Time              `json:"startedAt"`
	CompletedAt     *time.Time             `json:"completedAt"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// EvalRunResultView is the wire shape for one per-item result.
type EvalRunResultView struct {
	ID            string                 `json:"id"`
	ProjectID     string                 `json:"projectId"`
	EvalRunID     string                 `json:"evalRunId"`
	DatasetItemID string                 `json:"datasetItemId"`
	Actual        any                    `json:"actual"`
	Scores        []entity.ScorerResult  `json:"scores"`
	Passed        bool                   `json:"passed"`
	DurationMs    *int                   `json:"durationMs"`
	CostUSD       *float64               `json:"costUsd"`
	Error         *string                `json:"error"`
	CreatedAt     time.Time              `json:"createdAt"`
}

type EvalListResponse struct {
	Data   []EvalView `json:"data"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type EvalRunListResponse struct {
	Data   []EvalRunView `json:"data"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type EvalRunResultListResponse struct {
	Data   []EvalRunResultView `json:"data"`
	Total  int                 `json:"total"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

// ----- mappers -----------------------------------------------------------

func toEvalView(e *entity.Eval) EvalView {
	scorers := e.Scorers
	if scorers == nil {
		scorers = []entity.Scorer{}
	}
	return EvalView{
		ID:          e.ID,
		ProjectID:   e.ProjectID,
		DatasetID:   e.DatasetID,
		Name:        e.Name,
		Description: e.Description,
		Scorers:     scorers,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toEvalRunView(r *entity.EvalRun) EvalRunView {
	meta := r.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	v := EvalRunView{
		ID:              r.ID,
		ProjectID:       r.ProjectID,
		EvalID:          r.EvalID,
		Status:          r.Status,
		PromptVersionID: r.PromptVersionID,
		Metadata:        meta,
		TotalItems:      r.TotalItems,
		PassedItems:     r.PassedItems,
		FailedItems:     r.FailedItems,
		ErroredItems:    r.ErroredItems,
		DurationMs:      r.DurationMs,
		CostUSD:         r.CostUSD,
		StartedAt:       r.StartedAt,
		CompletedAt:     r.CompletedAt,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
	if r.TotalItems > 0 && (r.Status == entity.EvalRunStatusCompleted || r.Status == entity.EvalRunStatusFailed) {
		rate := float64(r.PassedItems) / float64(r.TotalItems)
		v.PassRate = &rate
	}
	return v
}

func toEvalRunResultView(r *entity.EvalRunResult) EvalRunResultView {
	scores := r.Scores
	if scores == nil {
		scores = []entity.ScorerResult{}
	}
	return EvalRunResultView{
		ID:            r.ID,
		ProjectID:     r.ProjectID,
		EvalRunID:     r.EvalRunID,
		DatasetItemID: r.DatasetItemID,
		Actual:        r.Actual,
		Scores:        scores,
		Passed:        r.Passed,
		DurationMs:    r.DurationMs,
		CostUSD:       r.CostUSD,
		Error:         r.Error,
		CreatedAt:     r.CreatedAt,
	}
}

func toEvalListResponse(p *entity.Page[entity.Eval]) EvalListResponse {
	views := make([]EvalView, len(p.Data))
	for i := range p.Data {
		views[i] = toEvalView(&p.Data[i])
	}
	return EvalListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}

func toEvalRunListResponse(p *entity.Page[entity.EvalRun]) EvalRunListResponse {
	views := make([]EvalRunView, len(p.Data))
	for i := range p.Data {
		views[i] = toEvalRunView(&p.Data[i])
	}
	return EvalRunListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}

func toEvalRunResultListResponse(p *entity.Page[entity.EvalRunResult]) EvalRunResultListResponse {
	views := make([]EvalRunResultView, len(p.Data))
	for i := range p.Data {
		views[i] = toEvalRunResultView(&p.Data[i])
	}
	return EvalRunResultListResponse{Data: views, Total: p.Total, Limit: p.Limit, Offset: p.Offset}
}
