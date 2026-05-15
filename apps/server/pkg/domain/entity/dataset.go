package entity

import "time"

// Dataset is a named, project-scoped collection of eval cases.
//
// Datasets are the raw material of evals: each item is one test case with an
// `input` (what the LLM/tool gets) and an optional `expected` (what scorers
// compare against). Items are seeded from real production spans
// (see DatasetItem.SourceTraceID / SourceSpanID) or authored manually.
type Dataset struct {
	ID          string
	ProjectID   string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DatasetItem is one eval case inside a Dataset.
//
// `Input` and `Expected` are arbitrary JSON values — they carry whatever the
// target under test consumes/produces (chat messages, tool args, plain text…).
// Provenance fields point at the span this case was curated from, when any.
type DatasetItem struct {
	ID            string
	DatasetID     string
	ProjectID     string // denormalized for 1-hop tenant scoping
	Input         any
	Expected      any
	Metadata      map[string]any
	SourceTraceID *string
	SourceSpanID  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewDatasetItem is the payload passed into the store for creating an item.
// IDs and timestamps are filled by the store; ProjectID is set from the auth
// context, never from the client (multi-tenant rule).
type NewDatasetItem struct {
	DatasetID     string
	ProjectID     string
	Input         any
	Expected      any
	Metadata      map[string]any
	SourceTraceID *string
	SourceSpanID  *string
}

// DatasetFilter narrows a ListDatasets query. All fields are optional.
type DatasetFilter struct {
	Name   *string // substring match
	Limit  int
	Offset int
}

// DatasetItemFilter narrows a ListDatasetItems query.
type DatasetItemFilter struct {
	// SourceTraceID, when set, returns only items curated from this trace.
	// Useful for "is this trace already in any dataset?" checks.
	SourceTraceID *string
	Limit         int
	Offset        int
}

// DatasetUpdate carries optional fields to patch on an existing dataset.
// Nil fields are left untouched (same convention as TraceUpdate).
type DatasetUpdate struct {
	Name        *string
	Description *string
}

// Items are immutable in Phase 1: callers create + delete, no in-place edit.
// Mutating an `any` JSON value with "nil = leave alone" semantics is ambiguous
// (untyped nil vs JSON null), so we sidestep it until there's a real need.
