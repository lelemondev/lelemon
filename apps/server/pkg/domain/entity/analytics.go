package entity

import "time"

type Stats struct {
	TotalTraces   int
	TotalSpans    int
	TotalTokens   int
	TotalCostUSD  float64
	AvgDurationMs int
	ErrorRate     float64 // 0-100 percentage
}

type DataPoint struct {
	Time    time.Time
	Traces  int
	Spans   int
	Tokens  int
	CostUSD float64
}

type Period struct {
	From time.Time
	To   time.Time
}

type TimeSeriesOpts struct {
	Period
	Granularity string // "hour" | "day" | "week"
	Filter      AnalyticsFilter
}

// AnalyticsQuery combines a period with dimensional filters
type AnalyticsQuery struct {
	Period
	Filter AnalyticsFilter
}

// ModelStats represents analytics grouped by model
type ModelStats struct {
	Model        string
	Provider     string
	Requests     int
	TotalTokens  int
	InputTokens  int
	OutputTokens int
	TotalCostUSD float64
	AvgLatencyMs int
	P50LatencyMs int
	P95LatencyMs int
	P99LatencyMs int
}

// TagStats represents analytics grouped by tag
type TagStats struct {
	Tag          string
	Traces       int
	TotalTokens  int
	TotalCostUSD float64
	AvgLatencyMs int
}

// UserStats represents analytics grouped by user
type UserStats struct {
	UserID       string
	Traces       int
	TotalTokens  int
	TotalCostUSD float64
	AvgLatencyMs int
	LastActive   time.Time
}

// HourlyHeatmap represents usage by hour of day and day of week
type HourlyHeatmap struct {
	Hour    int     // 0-23
	Day     int     // 0=Sun, 6=Sat
	Traces  int
	Tokens  int
	CostUSD float64
}

// LatencyBucket represents a latency histogram bucket
type LatencyBucket struct {
	Bucket string // e.g. "0-100ms", "100-500ms"
	MinMs  int
	MaxMs  int
	Count  int
}

// LatencyPoint represents percentile latency at a point in time
type LatencyPoint struct {
	Time time.Time
	P50  int
	P95  int
	P99  int
}

// AnalyticsFilter holds optional dimensional filters for analytics queries
type AnalyticsFilter struct {
	Tag       string // exact tag match
	SessionID string // filter by session
	UserID    string // filter by user
	Name      string // filter by trace name
}

// HasFilters returns true if any dimensional filter is set
func (f AnalyticsFilter) HasFilters() bool {
	return f.Tag != "" || f.SessionID != "" || f.UserID != "" || f.Name != ""
}

// ValidGranularity checks if a granularity string is allowed
func ValidGranularity(g string) bool {
	return g == "hour" || g == "day" || g == "week"
}
