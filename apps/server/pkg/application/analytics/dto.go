package analytics

import "time"

// SummaryRequest is the request for analytics summary
type SummaryRequest struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

// UsageRequest is the request for usage time series
type UsageRequest struct {
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
	Granularity string     `json:"granularity,omitempty"` // "hour" | "day" | "week"
}

// PeriodRequest holds from/to with optional fields
type PeriodRequest struct {
	From   *time.Time
	To     *time.Time
	Prefix string // for tag filtering
	Limit  int    // for top-N queries
}
