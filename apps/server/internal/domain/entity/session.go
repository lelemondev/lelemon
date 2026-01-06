package entity

import "time"

// Session represents aggregated data for a sessionId
type Session struct {
	SessionID       string    `json:"SessionID"`
	UserID          *string   `json:"UserID"`
	TraceCount      int       `json:"TraceCount"`
	TotalSpans      int       `json:"TotalSpans"`
	TotalTokens     int       `json:"TotalTokens"`
	TotalCostUSD    float64   `json:"TotalCostUSD"`
	TotalDurationMs int       `json:"TotalDurationMs"`
	HasError        bool      `json:"HasError"`
	HasActive       bool      `json:"HasActive"`
	FirstTraceAt    time.Time `json:"FirstTraceAt"`
	LastTraceAt     time.Time `json:"LastTraceAt"`
}

type SessionFilter struct {
	UserID *string
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}
