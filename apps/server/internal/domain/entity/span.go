package entity

import "time"

type SpanType string

const (
	SpanTypeLLM       SpanType = "llm"
	SpanTypeAgent     SpanType = "agent"
	SpanTypeTool      SpanType = "tool"
	SpanTypeRetrieval SpanType = "retrieval"
	SpanTypeEmbedding SpanType = "embedding"
	SpanTypeGuardrail SpanType = "guardrail"
	SpanTypeRerank    SpanType = "rerank"
	SpanTypeCustom    SpanType = "custom"
)

type SpanStatus string

const (
	SpanStatusPending SpanStatus = "pending"
	SpanStatusSuccess SpanStatus = "success"
	SpanStatusError   SpanStatus = "error"
)

// ToolUse represents a tool call extracted from LLM output
type ToolUse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Input    any    `json:"input"`
	Output   any    `json:"output"`
	Status   string `json:"status"`
}

type Span struct {
	ID           string
	TraceID      string
	ParentSpanID *string
	Type         SpanType
	Name         string
	Input        any
	Output       any
	InputTokens  *int
	OutputTokens *int
	CostUSD      *float64
	DurationMs   *int
	Status       SpanStatus
	ErrorMessage *string
	Model        *string
	Provider     *string
	Metadata     map[string]any
	StartedAt    time.Time
	EndedAt      *time.Time
	// Extended fields (Phase 7.1)
	StopReason       *string
	CacheReadTokens  *int
	CacheWriteTokens *int
	ReasoningTokens  *int
	FirstTokenMs     *int
	Thinking         *string
	// Pre-computed fields (calculated at ingest time)
	SubType  *string   `json:"subType,omitempty"`  // "planning" | "response" for LLM spans
	ToolUses []ToolUse `json:"toolUses,omitempty"` // Extracted tool calls from output
}

type NewSpan struct {
	TraceID      string
	ParentSpanID *string
	Type         SpanType
	Name         string
	Input        any
	Output       any
	InputTokens  *int
	OutputTokens *int
	CostUSD      *float64
	DurationMs   *int
	Status       SpanStatus
	ErrorMessage *string
	Model        *string
	Provider     *string
	Metadata     map[string]any
	StartedAt    time.Time
	EndedAt      *time.Time
	// Extended fields (Phase 7.1)
	StopReason       *string
	CacheReadTokens  *int
	CacheWriteTokens *int
	ReasoningTokens  *int
	FirstTokenMs     *int
	Thinking         *string
}
