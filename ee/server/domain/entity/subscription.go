package entity

import "time"

// Subscription represents a billing subscription from Lemon Squeezy
type Subscription struct {
	ID                 string
	OrganizationID     string
	Plan               BillingPlan
	Status             SubscriptionStatus
	LemonSqueezyID     string // Subscription ID in Lemon Squeezy
	CustomerID         string // Customer ID in Lemon Squeezy
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SubscriptionUpdate holds fields that can be updated
type SubscriptionUpdate struct {
	Plan               *BillingPlan
	Status             *SubscriptionStatus
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	CancelledAt        *time.Time
}

// SubscriptionStatus represents the current state of a subscription
type SubscriptionStatus string

const (
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusPastDue   SubscriptionStatus = "past_due"
	SubStatusCancelled SubscriptionStatus = "cancelled"
	SubStatusExpired   SubscriptionStatus = "expired"
	SubStatusPaused    SubscriptionStatus = "paused"
)

// IsActive returns true if the subscription allows access
func (s *Subscription) IsActive() bool {
	return s.Status == SubStatusActive || s.Status == SubStatusPastDue
}

// IsExpired returns true if the subscription has ended
func (s *Subscription) IsExpired() bool {
	return s.Status == SubStatusExpired || s.Status == SubStatusCancelled
}
