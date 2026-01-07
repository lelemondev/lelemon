package ingest

import "time"

// IngestRequest is the request payload for the ingest endpoint
type IngestRequest struct {
	Events []IngestEvent `json:"events"`
}

// IngestEvent represents a single LLM event
type IngestEvent struct {
	// Span metadata
	SpanType string `json:"spanType"` // "llm" | "tool" | "retrieval" | "custom"
	Provider string `json:"provider"` // "openai" | "anthropic" | "gemini" | "bedrock" | "openrouter" | "unknown"
	Model    string `json:"model"`
	Name     string `json:"name,omitempty"`

	// Input/Output
	Input       any `json:"input,omitempty"`
	RawResponse any `json:"rawResponse,omitempty"` // Raw LLM response - server extracts tokens, output, etc.

	// Legacy fields (used if RawResponse is nil)
	Output       any  `json:"output,omitempty"`
	InputTokens  *int `json:"inputTokens,omitempty"`
	OutputTokens *int `json:"outputTokens,omitempty"`

	// Execution
	DurationMs   *int   `json:"durationMs,omitempty"`
	Status       string `json:"status"` // "success" | "error"
	ErrorMessage string `json:"errorMessage,omitempty"`
	ErrorStack   string `json:"errorStack,omitempty"`
	Streaming    bool   `json:"streaming,omitempty"`

	// Context
	SessionID string `json:"sessionId,omitempty"`
	UserID    string `json:"userId,omitempty"`

	// Relationships (Hierarchy)
	TraceID      string `json:"traceId,omitempty"`
	SpanID       string `json:"spanId,omitempty"`
	ParentSpanID string `json:"parentSpanId,omitempty"`
	ToolCallID   string `json:"toolCallId,omitempty"`

	// Extended fields (legacy - extracted from RawResponse when available)
	StopReason       string `json:"stopReason,omitempty"`
	CacheReadTokens  *int   `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens *int   `json:"cacheWriteTokens,omitempty"`
	ReasoningTokens  *int   `json:"reasoningTokens,omitempty"`
	FirstTokenMs     *int   `json:"firstTokenMs,omitempty"` // SDK must still provide this (timing)
	Thinking         string `json:"thinking,omitempty"`

	// Custom data
	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`

	// Timestamp
	Timestamp *time.Time `json:"timestamp,omitempty"`
}

// IngestResponse is the response payload for the ingest endpoint
type IngestResponse struct {
	Success   bool          `json:"success"`
	Processed int           `json:"processed"`
	Errors    []IngestError `json:"errors,omitempty"`
}

// IngestError represents an error for a specific event
type IngestError struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}
