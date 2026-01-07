package entity

import "time"

type TraceStatus string

const (
	TraceStatusActive    TraceStatus = "active"
	TraceStatusCompleted TraceStatus = "completed"
	TraceStatusError     TraceStatus = "error"
)

type Trace struct {
	ID        string
	ProjectID string
	SessionID *string
	UserID    *string
	Status    TraceStatus
	Tags      []string
	Metadata  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TraceWithSpans includes calculated metrics from spans
type TraceWithSpans struct {
	Trace
	Spans           []Span
	TotalSpans      int
	TotalTokens     int
	TotalCostUSD    float64
	TotalDurationMs int
}

// TraceWithMetrics is a trace with calculated metrics (without spans)
type TraceWithMetrics struct {
	Trace
	TotalSpans      int
	TotalTokens     int
	TotalCostUSD    float64
	TotalDurationMs int
}

type TraceFilter struct {
	SessionID *string
	UserID    *string
	Status    *TraceStatus
	Tags      []string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

type TraceUpdate struct {
	Status   *TraceStatus
	Metadata map[string]any
	Tags     []string
}
