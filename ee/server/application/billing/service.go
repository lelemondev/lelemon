package billing

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lelemon/ee/server/domain/entity"
	"github.com/lelemon/ee/server/domain/repository"
	"github.com/lelemon/ee/server/infrastructure/lemonsqueezy"
)

var (
	ErrAlreadyOnPlan = errors.New("already on this plan")
	ErrOwnerNotFound = errors.New("organization owner not found")
	ErrMissingOrgID  = errors.New("missing org_id in webhook data")
)

// UserRepository defines what we need from the core user store (ISP)
type UserRepository interface {
	GetUserByID(ctx context.Context, id string) (interface{ GetEmail() string }, error)
}

// Service handles billing operations
type Service struct {
	repo     repository.BillingRepository // Interface, not concrete
	lsClient *lemonsqueezy.Client
	config   *Config
}

// Config holds billing configuration
type Config struct {
	ProVariantID        string
	EnterpriseVariantID string
}

// LoadConfigFromEnv loads billing config from environment
func LoadConfigFromEnv() *Config {
	return &Config{
		ProVariantID:        os.Getenv("LEMONSQUEEZY_PRO_VARIANT_ID"),
		EnterpriseVariantID: os.Getenv("LEMONSQUEEZY_ENTERPRISE_VARIANT_ID"),
	}
}

// NewService creates a new billing service
func NewService(repo repository.BillingRepository, lsClient *lemonsqueezy.Client, config *Config) *Service {
	if config == nil {
		config = LoadConfigFromEnv()
	}
	return &Service{
		repo:     repo,
		lsClient: lsClient,
		config:   config,
	}
}

// GetBilling returns the current billing info for an organization
func (s *Service) GetBilling(ctx context.Context, orgID string) (*BillingInfo, error) {
	org, err := s.repo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, entity.ErrNotFound
	}

	sub, _ := s.repo.GetSubscriptionByOrgID(ctx, orgID)

	usage, err := s.repo.GetCurrentMonth(ctx, orgID)
	if err != nil {
		return nil, err
	}

	limits := entity.PlanLimits[org.Plan]
	daysRemaining := daysUntilEndOfMonth()

	return &BillingInfo{
		Organization: org,
		Subscription: sub,
		Usage:        usage,
		Limits:       &limits,
		UsageReport:  entity.CalculateUsageReport(usage, org.Plan, daysRemaining),
	}, nil
}

// CreateCheckout generates a checkout URL for upgrading to a plan
func (s *Service) CreateCheckout(ctx context.Context, orgID string, plan entity.BillingPlan, ownerEmail string) (string, error) {
	org, err := s.repo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return "", entity.ErrNotFound
	}

	// Check if already subscribed to this plan
	if org.Plan == plan {
		return "", ErrAlreadyOnPlan
	}

	// Create checkout in Lemon Squeezy
	checkoutURL, err := s.lsClient.CreateCheckout(ctx, lemonsqueezy.CheckoutRequest{
		Email:     ownerEmail,
		VariantID: s.getVariantID(plan),
		CustomData: map[string]string{
			"org_id": orgID,
		},
	})
	if err != nil {
		return "", err
	}

	return checkoutURL, nil
}

// GetCustomerPortal returns the Lemon Squeezy customer portal URL
func (s *Service) GetCustomerPortal(ctx context.Context, orgID string) (string, error) {
	sub, err := s.repo.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		return "", entity.ErrNotFound
	}

	return s.lsClient.GetCustomerPortalURL(ctx, sub.CustomerID)
}

// CancelSubscription cancels the current subscription
func (s *Service) CancelSubscription(ctx context.Context, orgID string) error {
	sub, err := s.repo.GetSubscriptionByOrgID(ctx, orgID)
	if err != nil {
		return entity.ErrNotFound
	}

	if err := s.lsClient.CancelSubscription(ctx, sub.LemonSqueezyID); err != nil {
		return err
	}

	// Update local status
	status := entity.SubStatusCancelled
	now := time.Now()
	return s.repo.UpdateSubscription(ctx, sub.ID, &entity.SubscriptionUpdate{
		Status:      &status,
		CancelledAt: &now,
	})
}

// HandleWebhook processes incoming webhooks from Lemon Squeezy
func (s *Service) HandleWebhook(ctx context.Context, event *lemonsqueezy.WebhookEvent) error {
	switch event.EventName {
	case "subscription_created":
		return s.handleSubscriptionCreated(ctx, event)
	case "subscription_updated":
		return s.handleSubscriptionUpdated(ctx, event)
	case "subscription_cancelled":
		return s.handleSubscriptionCancelled(ctx, event)
	case "subscription_payment_success":
		return s.handlePaymentSuccess(ctx, event)
	case "subscription_payment_failed":
		return s.handlePaymentFailed(ctx, event)
	default:
		// Ignore unknown events
		return nil
	}
}

func (s *Service) handleSubscriptionCreated(ctx context.Context, event *lemonsqueezy.WebhookEvent) error {
	orgID := event.CustomData["org_id"]
	if orgID == "" {
		return ErrMissingOrgID
	}

	plan := s.planFromVariantID(event.VariantID)

	sub := &entity.Subscription{
		ID:                 uuid.New().String(),
		OrganizationID:     orgID,
		Plan:               plan,
		Status:             entity.SubStatusActive,
		LemonSqueezyID:     event.SubscriptionID,
		CustomerID:         event.CustomerID,
		CurrentPeriodStart: event.CurrentPeriodStart,
		CurrentPeriodEnd:   event.CurrentPeriodEnd,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.repo.CreateSubscription(ctx, sub); err != nil {
		return err
	}

	// Update organization plan
	return s.repo.UpdateOrganization(ctx, orgID, &entity.OrganizationUpdate{
		Plan: &plan,
	})
}

func (s *Service) handleSubscriptionUpdated(ctx context.Context, event *lemonsqueezy.WebhookEvent) error {
	sub, err := s.repo.GetSubscriptionByLemonSqueezyID(ctx, event.SubscriptionID)
	if err != nil {
		return err
	}

	plan := s.planFromVariantID(event.VariantID)
	status := mapLemonSqueezyStatus(event.Status)

	updates := &entity.SubscriptionUpdate{
		Plan:               &plan,
		Status:             &status,
		CurrentPeriodStart: &event.CurrentPeriodStart,
		CurrentPeriodEnd:   &event.CurrentPeriodEnd,
	}

	if err := s.repo.UpdateSubscription(ctx, sub.ID, updates); err != nil {
		return err
	}

	// Update organization plan
	return s.repo.UpdateOrganization(ctx, sub.OrganizationID, &entity.OrganizationUpdate{
		Plan: &plan,
	})
}

func (s *Service) handleSubscriptionCancelled(ctx context.Context, event *lemonsqueezy.WebhookEvent) error {
	sub, err := s.repo.GetSubscriptionByLemonSqueezyID(ctx, event.SubscriptionID)
	if err != nil {
		return err
	}

	status := entity.SubStatusCancelled
	now := time.Now()

	if err := s.repo.UpdateSubscription(ctx, sub.ID, &entity.SubscriptionUpdate{
		Status:      &status,
		CancelledAt: &now,
	}); err != nil {
		return err
	}

	// Downgrade to free plan
	plan := entity.PlanFree
	return s.repo.UpdateOrganization(ctx, sub.OrganizationID, &entity.OrganizationUpdate{
		Plan: &plan,
	})
}

func (s *Service) handlePaymentSuccess(ctx context.Context, event *lemonsqueezy.WebhookEvent) error {
	sub, err := s.repo.GetSubscriptionByLemonSqueezyID(ctx, event.SubscriptionID)
	if err != nil {
		return err
	}

	status := entity.SubStatusActive
	return s.repo.UpdateSubscription(ctx, sub.ID, &entity.SubscriptionUpdate{
		Status:             &status,
		CurrentPeriodStart: &event.CurrentPeriodStart,
		CurrentPeriodEnd:   &event.CurrentPeriodEnd,
	})
}

func (s *Service) handlePaymentFailed(ctx context.Context, event *lemonsqueezy.WebhookEvent) error {
	sub, err := s.repo.GetSubscriptionByLemonSqueezyID(ctx, event.SubscriptionID)
	if err != nil {
		return err
	}

	status := entity.SubStatusPastDue
	return s.repo.UpdateSubscription(ctx, sub.ID, &entity.SubscriptionUpdate{
		Status: &status,
	})
}

// getVariantID returns the Lemon Squeezy variant ID for a plan
func (s *Service) getVariantID(plan entity.BillingPlan) string {
	switch plan {
	case entity.PlanPro:
		return s.config.ProVariantID
	case entity.PlanEnterprise:
		return s.config.EnterpriseVariantID
	default:
		return ""
	}
}

// planFromVariantID returns the plan for a variant ID
func (s *Service) planFromVariantID(variantID string) entity.BillingPlan {
	switch variantID {
	case s.config.ProVariantID:
		return entity.PlanPro
	case s.config.EnterpriseVariantID:
		return entity.PlanEnterprise
	default:
		return entity.PlanFree
	}
}

func mapLemonSqueezyStatus(status string) entity.SubscriptionStatus {
	switch status {
	case "active":
		return entity.SubStatusActive
	case "past_due":
		return entity.SubStatusPastDue
	case "cancelled":
		return entity.SubStatusCancelled
	case "expired":
		return entity.SubStatusExpired
	case "paused":
		return entity.SubStatusPaused
	default:
		return entity.SubStatusActive
	}
}

func daysUntilEndOfMonth() int {
	now := time.Now()
	firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	return int(firstOfNextMonth.Sub(now).Hours() / 24)
}

// BillingInfo holds complete billing information for display
type BillingInfo struct {
	Organization *entity.Organization
	Subscription *entity.Subscription
	Usage        *entity.Usage
	Limits       *struct {
		MaxProjects    int
		MaxMembers     int
		MaxTracesMonth int
		RetentionDays  int
	}
	UsageReport *entity.UsageReport
}
