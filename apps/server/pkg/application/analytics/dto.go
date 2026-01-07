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
