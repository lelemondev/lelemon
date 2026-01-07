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
}

// ProjectStore handles project-related operations
type ProjectStore interface {
	CreateProject(ctx context.Context, p *entity.Project) error
	GetProjectByID(ctx context.Context, id string) (*entity.Project, error)
	GetProjectByAPIKeyHash(ctx context.Context, hash string) (*entity.Project, error)
	UpdateProject(ctx context.Context, id string, updates entity.ProjectUpdate) error
	DeleteProject(ctx context.Context, id string) error
	ListProjectsByOwner(ctx context.Context, email string) ([]entity.Project, error)
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
	GetStats(ctx context.Context, projectID string, period entity.Period) (*entity.Stats, error)
	GetUsageTimeSeries(ctx context.Context, projectID string, opts entity.TimeSeriesOpts) ([]entity.DataPoint, error)
}

// UserStore handles user operations (for dashboard auth)
type UserStore interface {
	CreateUser(ctx context.Context, u *entity.User) error
	GetUserByID(ctx context.Context, id string) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	UpdateUser(ctx context.Context, id string, updates entity.UserUpdate) error
}
