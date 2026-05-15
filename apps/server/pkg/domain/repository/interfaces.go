package repository

import (
	"context"

	"github.com/lelemon/server/pkg/domain/entity"
)

// Store is the main interface for all database operations.
// It composes sub-interfaces for different domains.
type Store interface {
	// Lifecycle
	Migrate(ctx context.Context) error
	Ping(ctx context.Context) error
	Close() error

	// Composed interfaces
	ProjectStore
	TraceStore
	AnalyticsStore
	UserStore
	DatasetStore
	EvalStore
	PromptStore
}

// ProjectStore handles project-related operations
type ProjectStore interface {
	CreateProject(ctx context.Context, p *entity.Project) error
	GetProjectByID(ctx context.Context, id string) (*entity.Project, error)
	GetProjectByAPIKeyHash(ctx context.Context, hash string) (*entity.Project, error)
	UpdateProject(ctx context.Context, id string, updates entity.ProjectUpdate) error
	DeleteProject(ctx context.Context, id string) error
	ListProjectsByOwner(ctx context.Context, email string) ([]entity.Project, error)
	IsProjectOwner(ctx context.Context, projectID, ownerEmail string) (bool, error)
	RotateAPIKey(ctx context.Context, id string, newKey, newHash string) error
}

// TraceStore handles trace and span operations
type TraceStore interface {
	// Trace writes
	CreateTrace(ctx context.Context, t *entity.Trace) error
	UpdateTrace(ctx context.Context, projectID, traceID string, updates entity.TraceUpdate) error
	UpdateTraceStatus(ctx context.Context, projectID, traceID string, status entity.TraceStatus) error
	DeleteAllTraces(ctx context.Context, projectID string) (int64, error)

	// Span writes
	CreateSpan(ctx context.Context, span *entity.Span) error
	CreateSpans(ctx context.Context, spans []entity.Span) error

	// Trace reads
	GetTrace(ctx context.Context, projectID, traceID string) (*entity.TraceWithSpans, error)
	ListTraces(ctx context.Context, projectID string, filter entity.TraceFilter) (*entity.Page[entity.TraceWithMetrics], error)

	// Session reads
	ListSessions(ctx context.Context, projectID string, filter entity.SessionFilter) (*entity.Page[entity.Session], error)
}

// AnalyticsStore handles analytics queries
type AnalyticsStore interface {
	GetStats(ctx context.Context, projectID string, q entity.AnalyticsQuery) (*entity.Stats, error)
	GetUsageTimeSeries(ctx context.Context, projectID string, opts entity.TimeSeriesOpts) ([]entity.DataPoint, error)
	GetModelStats(ctx context.Context, projectID string, q entity.AnalyticsQuery) ([]entity.ModelStats, error)
	GetTagStats(ctx context.Context, projectID string, q entity.AnalyticsQuery, prefix string) ([]entity.TagStats, error)
	GetTopUsers(ctx context.Context, projectID string, q entity.AnalyticsQuery, limit int) ([]entity.UserStats, error)
	GetHourlyHeatmap(ctx context.Context, projectID string, q entity.AnalyticsQuery) ([]entity.HourlyHeatmap, error)
	GetLatencyDistribution(ctx context.Context, projectID string, q entity.AnalyticsQuery) ([]entity.LatencyBucket, error)
	GetLatencyTimeSeries(ctx context.Context, projectID string, opts entity.TimeSeriesOpts) ([]entity.LatencyPoint, error)
}

// UserStore handles user operations (for dashboard auth)
type UserStore interface {
	CreateUser(ctx context.Context, u *entity.User) error
	GetUserByID(ctx context.Context, id string) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	UpdateUser(ctx context.Context, id string, updates entity.UserUpdate) error
}

// DatasetStore handles datasets and their items.
//
// Every method takes projectID and uses it as the outer tenant filter
// (multi-tenant rule, see .claude/rules/multi-tenant.md). The ClickHouse
// implementation returns entity.ErrUnsupported for these methods — datasets
// are relational data, they live in the SQLite/Postgres primary store.
type DatasetStore interface {
	// Datasets
	CreateDataset(ctx context.Context, d *entity.Dataset) error
	GetDataset(ctx context.Context, projectID, datasetID string) (*entity.Dataset, error)
	ListDatasets(ctx context.Context, projectID string, filter entity.DatasetFilter) (*entity.Page[entity.Dataset], error)
	UpdateDataset(ctx context.Context, projectID, datasetID string, updates entity.DatasetUpdate) error
	DeleteDataset(ctx context.Context, projectID, datasetID string) error

	// Dataset items
	CreateDatasetItem(ctx context.Context, item *entity.DatasetItem) error
	BulkCreateDatasetItems(ctx context.Context, items []entity.DatasetItem) error
	GetDatasetItem(ctx context.Context, projectID, itemID string) (*entity.DatasetItem, error)
	ListDatasetItems(ctx context.Context, projectID, datasetID string, filter entity.DatasetItemFilter) (*entity.Page[entity.DatasetItem], error)
	DeleteDatasetItem(ctx context.Context, projectID, itemID string) error
}

// EvalStore handles evals, their runs, and per-item run results.
//
// Same multi-tenant rule as DatasetStore: every method takes projectID and
// uses it as the outer filter. ClickHouse returns entity.ErrUnsupported — eval
// data is relational and needs SQLite/Postgres as the primary store.
type EvalStore interface {
	// Evals
	CreateEval(ctx context.Context, e *entity.Eval) error
	GetEval(ctx context.Context, projectID, evalID string) (*entity.Eval, error)
	ListEvals(ctx context.Context, projectID string, filter entity.EvalFilter) (*entity.Page[entity.Eval], error)
	DeleteEval(ctx context.Context, projectID, evalID string) error

	// Eval runs
	CreateEvalRun(ctx context.Context, r *entity.EvalRun) error
	GetEvalRun(ctx context.Context, projectID, runID string) (*entity.EvalRun, error)
	ListEvalRuns(ctx context.Context, projectID, evalID string, filter entity.EvalRunFilter) (*entity.Page[entity.EvalRun], error)
	// UpdateEvalRunStatus is used by the service to move a pending run to
	// running on the first posted result. Returns ErrNotFound if the row is
	// missing in the tenant.
	UpdateEvalRunStatus(ctx context.Context, projectID, runID string, status entity.EvalRunStatus) error
	// FinalizeEvalRun computes aggregates from posted results, freezes them on
	// the row, and sets status + completed_at. Idempotent: re-finalizing
	// returns the current state without touching it.
	FinalizeEvalRun(ctx context.Context, projectID, runID string, in entity.FinalizeEvalRunInput) (*entity.EvalRun, error)

	// Eval run results
	CreateEvalRunResult(ctx context.Context, res *entity.EvalRunResult) error
	ListEvalRunResults(ctx context.Context, projectID, runID string, filter entity.EvalRunResultFilter) (*entity.Page[entity.EvalRunResult], error)
}

// PromptStore handles prompts and their immutable versions.
//
// Same multi-tenant invariant: every method takes projectID. ClickHouse
// returns entity.ErrUnsupported — relational data lives in the primary store.
// Versions enforce UNIQUE(prompt_id, version) at the DB layer; the store
// surfaces a conflict as entity.ErrConflict so the handler can map to 409.
type PromptStore interface {
	CreatePrompt(ctx context.Context, p *entity.Prompt) error
	GetPrompt(ctx context.Context, projectID, promptID string) (*entity.Prompt, error)
	ListPrompts(ctx context.Context, projectID string, filter entity.PromptFilter) (*entity.Page[entity.Prompt], error)
	UpdatePrompt(ctx context.Context, projectID, promptID string, updates entity.PromptUpdate) error
	DeletePrompt(ctx context.Context, projectID, promptID string) error

	CreatePromptVersion(ctx context.Context, v *entity.PromptVersion) error
	GetPromptVersion(ctx context.Context, projectID, versionID string) (*entity.PromptVersion, error)
	ListPromptVersions(ctx context.Context, projectID, promptID string, filter entity.PromptVersionFilter) (*entity.Page[entity.PromptVersion], error)
}
