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
}
