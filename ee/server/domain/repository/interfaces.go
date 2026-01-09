package repository

import (
	"context"

	"github.com/lelemon/ee/server/domain/entity"
)

// OrganizationStore manages organizations
type OrganizationStore interface {
	CreateOrganization(ctx context.Context, org *entity.Organization) error
	GetOrganizationByID(ctx context.Context, id string) (*entity.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*entity.Organization, error)
	UpdateOrganization(ctx context.Context, id string, updates *entity.OrganizationUpdate) error
	DeleteOrganization(ctx context.Context, id string) error
	ListOrganizationsByUser(ctx context.Context, userID string) ([]entity.Organization, error)
}

// TeamStore manages team members
type TeamStore interface {
	AddMember(ctx context.Context, member *entity.TeamMember) error
	GetMember(ctx context.Context, orgID, userID string) (*entity.TeamMember, error)
	UpdateMember(ctx context.Context, orgID, userID string, role entity.Role) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	ListMembers(ctx context.Context, orgID string) ([]entity.TeamMember, error)
	GetUserOrganizations(ctx context.Context, userID string) ([]entity.TeamMember, error)
	CountMembers(ctx context.Context, orgID string) (int, error)
}

// SubscriptionStore manages subscriptions
type SubscriptionStore interface {
	CreateSubscription(ctx context.Context, sub *entity.Subscription) error
	GetSubscriptionByOrgID(ctx context.Context, orgID string) (*entity.Subscription, error)
	GetSubscriptionByLemonSqueezyID(ctx context.Context, lsID string) (*entity.Subscription, error)
	UpdateSubscription(ctx context.Context, id string, updates *entity.SubscriptionUpdate) error
}

// UsageStore manages monthly usage tracking
type UsageStore interface {
	Increment(ctx context.Context, orgID string, traces, spans int) error
	GetCurrentMonth(ctx context.Context, orgID string) (*entity.Usage, error)
	GetByMonth(ctx context.Context, orgID, month string) (*entity.Usage, error)
}

// EnterpriseStore combines all enterprise interfaces
type EnterpriseStore interface {
	OrganizationStore
	TeamStore
	SubscriptionStore
	UsageStore

	// MigrateEnterprise runs migrations for enterprise tables
	MigrateEnterprise(ctx context.Context) error

	// Close closes the database connection
	Close() error
}

// Service-specific interfaces (ISP compliant)
// Each service depends only on what it needs

// OrganizationRepository is used by the organization service
type OrganizationRepository interface {
	OrganizationStore
	TeamStore
}

// BillingRepository is used by the billing service
type BillingRepository interface {
	GetOrganizationByID(ctx context.Context, id string) (*entity.Organization, error)
	UpdateOrganization(ctx context.Context, id string, updates *entity.OrganizationUpdate) error
	SubscriptionStore
	UsageStore
}

// RBACRepository is used by the RBAC service
type RBACRepository interface {
	TeamStore
}

// Transactional provides transaction support
type Transactional interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
