package entity

import "time"

// ScorerType identifies which scoring strategy to apply.
//
// Phase 2A ships four built-in deterministic scorers; LLM-as-judge and webhook
// scorers come later. The string values are part of the wire contract —
// renaming one is a breaking change.
type ScorerType string

const (
	// ScorerExactMatch deep-equals the actual value against `dataset_item.expected`.
	ScorerExactMatch ScorerType = "exact_match"

	// ScorerContains checks substring (string actual) or membership (array actual).
	// Config: {"value": <string|any>}.
	ScorerContains ScorerType = "contains"

	// ScorerJSONPath extracts a dotted path from the actual value and compares it
	// to a target with an operator. Covers JSON-field assertions and numeric
	// compares in one scorer.
	// Config: {"path": "results.0.id", "op": "eq|ne|gt|gte|lt|lte", "value": <any>}.
	ScorerJSONPath ScorerType = "json_path"

	// ScorerRegex tests a regular expression against a string actual.
	// Config: {"pattern": "^honda \\w+$"}.
	ScorerRegex ScorerType = "regex"

	// ScorerClientReported defers the verdict to the SDK/CI caller. The
	// caller sends a `clientScores` entry keyed by this scorer's id alongside
	// the result; the platform stores that entry verbatim and ANDs it with
	// built-in scorers. Used to wire custom logic — LLM-as-judge against the
	// customer's own provider key, domain-specific assertions — without the
	// platform invoking outbound code.
	//
	// Missing clientScores for a declared ScorerClientReported is an error
	// (recorded on the per-scorer result), NOT a silent pass — a CI gate that
	// can pretend a case ran is a dangerous CI gate.
	ScorerClientReported ScorerType = "client_reported"
)

// ScorerOp is the comparison operator used by ScorerJSONPath.
type ScorerOp string

const (
	ScorerOpEq  ScorerOp = "eq"
	ScorerOpNe  ScorerOp = "ne"
	ScorerOpGt  ScorerOp = "gt"
	ScorerOpGte ScorerOp = "gte"
	ScorerOpLt  ScorerOp = "lt"
	ScorerOpLte ScorerOp = "lte"
)

// Scorer is one entry in an Eval's scorer list. ID is stable within the eval
// (used as the per-result key); Config holds type-specific parameters.
type Scorer struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   ScorerType     `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

// Eval is a definition of *how* to score the rows of a Dataset.
//
// Phase 2A evals are immutable after creation — to change scorers, create a
// new eval. That sidesteps "what happens to old runs when scorers change?"
// until there's a real reason to revisit it.
type Eval struct {
	ID          string
	ProjectID   string
	DatasetID   string
	Name        string
	Description *string
	Scorers     []Scorer
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EvalFilter narrows a ListEvals query.
type EvalFilter struct {
	// DatasetID, when set, scopes the list to one dataset.
	DatasetID *string
	Limit     int
	Offset    int
}

// EvalRunStatus is the lifecycle state of an EvalRun.
//
// pending  — created, no results yet
// running  — at least one result posted
// completed — finalized; aggregates frozen
// failed   — finalized with an explicit failure (e.g. caller error, abort)
type EvalRunStatus string

const (
	EvalRunStatusPending   EvalRunStatus = "pending"
	EvalRunStatusRunning   EvalRunStatus = "running"
	EvalRunStatusCompleted EvalRunStatus = "completed"
	EvalRunStatusFailed    EvalRunStatus = "failed"
)

// EvalRun is one execution of an Eval against its dataset.
//
// Two write phases: results come in incrementally via PostResult, then a
// single Finalize freezes the aggregates and marks the run completed/failed.
// PromptVersionID is the free-form metadata convention from Phase 3 of the
// spec (§6) — promoted to a typed reference once Prompt entities exist.
type EvalRun struct {
	ID              string
	ProjectID       string
	EvalID          string
	Status          EvalRunStatus
	PromptVersionID *string
	Metadata        map[string]any

	// Aggregates — computed on Finalize; nil pointers until then.
	TotalItems   int
	PassedItems  int
	FailedItems  int
	ErroredItems int
	DurationMs   *int
	CostUSD      *float64

	StartedAt   time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ScorerResult is one scorer's verdict on one item. Score is in [0,1]; a
// scorer is boolean today, so Passed is the source of truth and Score
// mirrors it (0 or 1). LLM-as-judge will produce a real fractional score.
type ScorerResult struct {
	ScorerID string  `json:"scorerId"`
	Passed   bool    `json:"passed"`
	Score    float64 `json:"score"`
	Details  string  `json:"details,omitempty"`
	Error    string  `json:"error,omitempty"`
}

// EvalRunResult is the per-dataset-item outcome of a run.
//
// Passed is the AND of all ScorerResult.Passed — if any scorer fails, the
// item fails. A run-level execution error (Error != nil) does not pass.
type EvalRunResult struct {
	ID            string
	ProjectID     string
	EvalRunID     string
	DatasetItemID string
	Actual        any
	Scores        []ScorerResult
	Passed        bool
	DurationMs    *int
	CostUSD       *float64
	Error         *string
	CreatedAt     time.Time
}

// EvalRunFilter narrows a ListEvalRuns query.
type EvalRunFilter struct {
	EvalID          *string
	Status          *EvalRunStatus
	PromptVersionID *string // payoff view: "runs that tested this prompt version"
	Limit           int
	Offset          int
}

// EvalRunResultFilter narrows a ListEvalRunResults query.
type EvalRunResultFilter struct {
	// PassedOnly / FailedOnly narrow to one verdict. Both nil means "all".
	PassedOnly *bool
	Limit      int
	Offset     int
}

// FinalizeEvalRunInput is the payload for Finalize (called once at end-of-run).
//
// Status must be one of EvalRunStatusCompleted or EvalRunStatusFailed.
// Aggregates are computed from the posted results; DurationMs/CostUSD here
// are the run-level totals reported by the caller (the SDK adds them up).
type FinalizeEvalRunInput struct {
	Status     EvalRunStatus
	DurationMs *int
	CostUSD    *float64
}
