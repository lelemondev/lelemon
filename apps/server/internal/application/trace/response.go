package trace

import "time"

// TraceDetailResponse is the optimized response for GET /traces/{id}
// Pre-processes spans into a tree structure ready for frontend rendering
type TraceDetailResponse struct {
	// Basic trace info
	ID        string         `json:"id"`
	ProjectID string         `json:"projectId"`
	SessionID *string        `json:"sessionId"`
	UserID    *string        `json:"userId"`
	Status    string         `json:"status"`
	Tags      []string       `json:"tags"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`

	// Aggregate metrics
	TotalSpans      int     `json:"totalSpans"`
	TotalTokens     int     `json:"totalTokens"`
	TotalCostUSD    float64 `json:"totalCostUsd"`
	TotalDurationMs int     `json:"totalDurationMs"`

	// Pre-processed span tree (hierarchical structure)
	SpanTree []SpanNode `json:"spanTree"`

	// Timeline context for visualization
	Timeline TimelineContext `json:"timeline"`
}

// SpanNode represents a node in the span tree
type SpanNode struct {
	Span          ProcessedSpan `json:"span"`
	Children      []SpanNode    `json:"children"`
	Depth         int           `json:"depth"`
	TimelineStart float64       `json:"timelineStart"` // 0-100%
	TimelineWidth float64       `json:"timelineWidth"` // 0-100%
}

// ProcessedSpan is a span with computed fields for visualization
type ProcessedSpan struct {
	// Original span fields
	ID           string     `json:"id"`
	TraceID      string     `json:"traceId"`
	ParentSpanID *string    `json:"parentSpanId"`
	Type         string     `json:"type"`
	Name         string     `json:"name"`
	Input        any        `json:"input"`
	Output       any        `json:"output"`
	InputTokens  *int       `json:"inputTokens"`
	OutputTokens *int       `json:"outputTokens"`
	CostUSD      *float64   `json:"costUsd"`
	DurationMs   *int       `json:"durationMs"`
	Status       string     `json:"status"`
	ErrorMessage *string    `json:"errorMessage"`
	Model        *string    `json:"model"`
	Provider     *string    `json:"provider"`
	Metadata     any        `json:"metadata"`
	StartedAt    time.Time  `json:"startedAt"`
	EndedAt      *time.Time `json:"endedAt"`

	// Extended fields
	StopReason       *string `json:"stopReason"`
	CacheReadTokens  *int    `json:"cacheReadTokens"`
	CacheWriteTokens *int    `json:"cacheWriteTokens"`
	ReasoningTokens  *int    `json:"reasoningTokens"`
	FirstTokenMs     *int    `json:"firstTokenMs"`
	Thinking         *string `json:"thinking"`

	// Computed fields (calculated by backend)
	SubType     *string   `json:"subType,omitempty"`     // "planning" | "response" for LLM spans
	ToolUses    []ToolUse `json:"toolUses,omitempty"`    // Extracted tool calls from output
	UserInput   *string   `json:"userInput,omitempty"`   // Extracted user message for agent spans
	IsToolUse   bool      `json:"isToolUse,omitempty"`   // True if this is a synthetic tool use node
	ToolUseData *ToolUse  `json:"toolUseData,omitempty"` // Tool use data if IsToolUse is true
}

// ToolUse represents a tool call extracted from LLM output
type ToolUse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Input      any    `json:"input"`
	Output     any    `json:"output"`
	Status     string `json:"status"` // "success" | "error" | "pending"
	DurationMs *int   `json:"durationMs"`
}

// TimelineContext provides timing information for timeline visualization
type TimelineContext struct {
	MinTime       int64 `json:"minTime"`       // Unix timestamp in milliseconds
	MaxTime       int64 `json:"maxTime"`       // Unix timestamp in milliseconds
	TotalDuration int   `json:"totalDuration"` // Duration in milliseconds
}
