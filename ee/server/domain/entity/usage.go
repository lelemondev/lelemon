package entity

import "time"

// Usage tracks monthly usage for billing enforcement
type Usage struct {
	ID             string
	OrganizationID string
	Month          string // Format: "2025-01"
	TracesUsed     int
	SpansUsed      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UsageReport provides a summary of usage for display
type UsageReport struct {
	CurrentMonth   *Usage
	Plan           BillingPlan
	TracesLimit    int
	TracesPercent  float64
	IsOverLimit    bool
	DaysRemaining  int
}

// CalculateUsageReport creates a usage report with percentage and limits
func CalculateUsageReport(usage *Usage, plan BillingPlan, daysRemaining int) *UsageReport {
	limits := PlanLimits[plan]

	var tracesPercent float64
	var isOverLimit bool

	if limits.MaxTracesMonth > 0 {
		tracesPercent = float64(usage.TracesUsed) / float64(limits.MaxTracesMonth) * 100
		isOverLimit = usage.TracesUsed >= limits.MaxTracesMonth
	}

	return &UsageReport{
		CurrentMonth:   usage,
		Plan:           plan,
		TracesLimit:    limits.MaxTracesMonth,
		TracesPercent:  tracesPercent,
		IsOverLimit:    isOverLimit,
		DaysRemaining:  daysRemaining,
	}
}
