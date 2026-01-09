package entity

import (
	"regexp"
	"strings"
	"time"
)

var slugRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Organization represents a multi-tenant organization
type Organization struct {
	ID          string
	Name        string
	Slug        string // URL-friendly: "acme-corp"
	OwnerUserID string
	Plan        BillingPlan
	Settings    OrganizationSettings
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// OrganizationSettings holds configurable limits for an organization
type OrganizationSettings struct {
	MaxProjects   int `json:"maxProjects"`
	MaxMembers    int `json:"maxMembers"`
	RetentionDays int `json:"retentionDays"`
}

// OrganizationUpdate holds fields that can be updated
type OrganizationUpdate struct {
	Name     *string
	Plan     *BillingPlan
	Settings *OrganizationSettings
}

// BillingPlan represents the subscription tier
type BillingPlan string

const (
	PlanFree       BillingPlan = "free"
	PlanPro        BillingPlan = "pro"
	PlanEnterprise BillingPlan = "enterprise"
)

// PlanLimits defines limits per plan
var PlanLimits = map[BillingPlan]struct {
	MaxProjects    int
	MaxMembers     int
	MaxTracesMonth int
	RetentionDays  int
}{
	PlanFree:       {3, 1, 10_000, 7},
	PlanPro:        {20, 10, 1_000_000, 30},
	PlanEnterprise: {-1, -1, -1, 90}, // -1 = unlimited
}

// GetDefaultSettings returns default settings based on plan
func (p BillingPlan) GetDefaultSettings() OrganizationSettings {
	limits := PlanLimits[p]
	return OrganizationSettings{
		MaxProjects:   limits.MaxProjects,
		MaxMembers:    limits.MaxMembers,
		RetentionDays: limits.RetentionDays,
	}
}

// Validate checks if the organization has valid data
func (o *Organization) Validate() error {
	name := strings.TrimSpace(o.Name)
	if name == "" || len(name) > 100 {
		return ErrInvalidName
	}

	if o.Slug != "" && !slugRegex.MatchString(o.Slug) {
		return ErrInvalidSlug
	}

	return nil
}

// IsValidPlan checks if the billing plan is valid
func (p BillingPlan) IsValid() bool {
	switch p {
	case PlanFree, PlanPro, PlanEnterprise:
		return true
	}
	return false
}
