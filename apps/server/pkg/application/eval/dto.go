package eval

import "github.com/lelemon/server/pkg/domain/entity"

// CreateEvalRequest is the body of POST .../evals.
// Scorers must be a non-empty list — an eval with no scorers is a non-eval.
type CreateEvalRequest struct {
	DatasetID   string           `json:"datasetId"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Scorers     []entity.Scorer  `json:"scorers"`
}

// StartEvalRunRequest is the body of POST .../eval-runs.
// PromptVersionID is free-form text for now — Phase 3 makes it typed.
type StartEvalRunRequest struct {
	EvalID          string         `json:"evalId"`
	PromptVersionID *string        `json:"promptVersionId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// PostEvalRunResultRequest is the body of POST .../eval-runs/{id}/results.
//
// The SDK reports `actual` (what the target produced) plus optional resource
// telemetry. Scoring happens server-side from this payload for built-in
// scorers — the client does NOT control the verdict for those. That keeps
// the source of truth honest.
//
// `ClientScores` is the explicit escape hatch: for scorers declared as
// `ScorerClientReported`, the client provides the verdict verbatim (typically
// because the customer ran their own LLM-as-judge or domain-specific check).
// Stray entries (scorer ids not declared on the eval) are silently dropped.
// A missing entry for a declared client-reported scorer produces an error
// result, not a silent pass.
type PostEvalRunResultRequest struct {
	DatasetItemID string                 `json:"datasetItemId"`
	Actual        any                    `json:"actual,omitempty"`
	DurationMs    *int                   `json:"durationMs,omitempty"`
	CostUSD       *float64               `json:"costUsd,omitempty"`
	ClientScores  []entity.ScorerResult  `json:"clientScores,omitempty"`
	// Error, when set, marks this case as a hard execution failure (the target
	// blew up). Scoring is skipped, the item counts as errored.
	Error *string `json:"error,omitempty"`
}

// FinalizeEvalRunRequest is the body of POST .../eval-runs/{id}/finalize.
type FinalizeEvalRunRequest struct {
	// Status must be "completed" or "failed". "failed" is for caller-initiated
	// aborts (e.g. CI couldn't finish the run); a run with passing+failing
	// results is still "completed" — pass/fail is derived from the aggregates.
	Status     entity.EvalRunStatus `json:"status"`
	DurationMs *int                 `json:"durationMs,omitempty"`
	CostUSD    *float64             `json:"costUsd,omitempty"`
}
