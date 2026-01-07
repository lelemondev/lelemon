package trace

// CreateTraceRequest is the request to create a trace
type CreateTraceRequest struct {
	SessionID string         `json:"sessionId,omitempty"`
	UserID    string         `json:"userId,omitempty"`
	Tags      []string       `json:"tags,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// UpdateTraceRequest is the request to update a trace
type UpdateTraceRequest struct {
	Status   *string        `json:"status,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateSpanRequest is the request to create a span
type CreateSpanRequest struct {
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Input        any            `json:"input,omitempty"`
	Output       any            `json:"output,omitempty"`
	InputTokens  *int           `json:"inputTokens,omitempty"`
	OutputTokens *int           `json:"outputTokens,omitempty"`
	DurationMs   *int           `json:"durationMs,omitempty"`
	Status       string         `json:"status"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	Model        string         `json:"model,omitempty"`
	Provider     string         `json:"provider,omitempty"`
	ParentSpanID string         `json:"parentSpanId,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}
