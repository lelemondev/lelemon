package entity

import "time"

// CostBreakdown represents cost analytics grouped by a tag
type CostBreakdown struct {
	Tag         string  `json:"tag"`
	TotalCost   float64 `json:"totalCost"`
	TotalTokens int     `json:"totalTokens"`
	TraceCount  int     `json:"traceCount"`
	Percentage  float64 `json:"percentage"`
}

// CostBreakdownFilter defines parameters for cost breakdown queries
type CostBreakdownFilter struct {
	TagPrefix string     // Filter by prefix: "org:", "campaign:", etc.
	From      *time.Time // Start date (inclusive)
	To        *time.Time // End date (inclusive)
	Limit     int        // Max number of results (default: 10)
}

// CostBreakdownResult contains the full cost analytics response
type CostBreakdownResult struct {
	Breakdowns  []CostBreakdown `json:"breakdowns"`
	TotalCost   float64         `json:"totalCost"`
	TotalTokens int             `json:"totalTokens"`
	TotalTraces int             `json:"totalTraces"`
	From        *time.Time      `json:"from,omitempty"`
	To          *time.Time      `json:"to,omitempty"`
}

// NewCostBreakdownFilter creates a filter with sensible defaults
func NewCostBreakdownFilter() CostBreakdownFilter {
	return CostBreakdownFilter{
		Limit: 10,
	}
}

// ============================================
// ERROR ANALYTICS
// ============================================

// ErrorMetrics contains error rate analytics for a project
type ErrorMetrics struct {
	TotalTraces  int                `json:"totalTraces"`
	ErrorTraces  int                `json:"errorTraces"`
	ErrorRate    float64            `json:"errorRate"`    // Percentage (0-100)
	ByTag        []TagErrorRate     `json:"byTag"`        // Error rate per tag
	TopErrors    []ErrorSummary     `json:"topErrors"`    // Most common errors
	From         *time.Time         `json:"from,omitempty"`
	To           *time.Time         `json:"to,omitempty"`
}

// TagErrorRate represents error rate for a specific tag
type TagErrorRate struct {
	Tag         string  `json:"tag"`
	TotalTraces int     `json:"totalTraces"`
	ErrorTraces int     `json:"errorTraces"`
	ErrorRate   float64 `json:"errorRate"` // Percentage (0-100)
}

// ErrorSummary represents a summary of a specific error
type ErrorSummary struct {
	Message      string    `json:"message"`
	Count        int       `json:"count"`
	LastOccurred time.Time `json:"lastOccurred"`
	AffectedTags []string  `json:"affectedTags"`
}

// ErrorFilter defines parameters for error metrics queries
type ErrorFilter struct {
	TagPrefix string     // Filter by tag prefix
	From      *time.Time // Start date (inclusive)
	To        *time.Time // End date (inclusive)
	TopLimit  int        // Max number of top errors (default: 10)
}

// NewErrorFilter creates a filter with sensible defaults
func NewErrorFilter() ErrorFilter {
	return ErrorFilter{
		TopLimit: 10,
	}
}
