package http

import (
	"github.com/go-chi/chi/v5"

	coreHttp "github.com/lelemon/server/pkg/interfaces/http"
	coreMiddleware "github.com/lelemon/server/pkg/interfaces/http/middleware"

	"github.com/lelemon/ee/server/application/billing"
	"github.com/lelemon/ee/server/application/organization"
	"github.com/lelemon/ee/server/application/rbac"
	"github.com/lelemon/ee/server/domain/entity"
	"github.com/lelemon/ee/server/infrastructure/lemonsqueezy"
	"github.com/lelemon/ee/server/interfaces/http/handler"
	"github.com/lelemon/ee/server/interfaces/http/middleware"
)

// EnterpriseExtension implements coreHttp.RouterExtension
// to add enterprise routes to the core router.
type EnterpriseExtension struct {
	orgSvc     *organization.Service
	rbacSvc    *rbac.Service
	billingSvc *billing.Service
	lsClient   *lemonsqueezy.Client
}

// NewEnterpriseExtension creates a new enterprise extension.
func NewEnterpriseExtension(
	orgSvc *organization.Service,
	rbacSvc *rbac.Service,
	billingSvc *billing.Service,
	lsClient *lemonsqueezy.Client,
) *EnterpriseExtension {
	return &EnterpriseExtension{
		orgSvc:     orgSvc,
		rbacSvc:    rbacSvc,
		billingSvc: billingSvc,
		lsClient:   lsClient,
	}
}

// MountRoutes adds enterprise routes to the router.
func (e *EnterpriseExtension) MountRoutes(r chi.Router, deps *coreHttp.RouterDeps) {
	// Create handlers
	orgHandler := handler.NewOrganizationHandler(e.orgSvc, deps.GetUserID)
	billingHandler := handler.NewBillingHandler(e.billingSvc, e.lsClient, deps.GetUserEmail)

	// Enterprise API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Webhook routes (no auth - verified by signature)
		r.Post("/webhooks/lemonsqueezy", billingHandler.HandleWebhook)

		// Organization routes (session auth required)
		r.Group(func(r chi.Router) {
			r.Use(coreMiddleware.SessionAuth(deps.JWTService))

			// Organization CRUD
			r.Post("/organizations", orgHandler.Create)
			r.Get("/organizations", orgHandler.List)

			// Organization-specific routes
			r.Route("/organizations/{orgId}", func(r chi.Router) {
				// Inject organization into context
				r.Use(middleware.InjectOrganization(e.orgSvc))
				// Require membership for all org routes
				r.Use(middleware.RequireOrgMember(e.rbacSvc, deps.GetUserID))

				r.Get("/", orgHandler.Get)

				// Team management
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermTeamRead, deps.GetUserID)).
					Get("/members", orgHandler.ListMembers)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermTeamInvite, deps.GetUserID)).
					Post("/invite", orgHandler.InviteMember)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermTeamManage, deps.GetUserID)).
					Patch("/members/{userId}", orgHandler.UpdateMember)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermTeamManage, deps.GetUserID)).
					Delete("/members/{userId}", orgHandler.RemoveMember)

				// Billing routes
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermBillingRead, deps.GetUserID)).
					Get("/billing", billingHandler.GetBilling)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermBillingWrite, deps.GetUserID)).
					Post("/billing/checkout", billingHandler.CreateCheckout)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermBillingRead, deps.GetUserID)).
					Get("/billing/portal", billingHandler.GetCustomerPortal)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermBillingWrite, deps.GetUserID)).
					Delete("/billing/subscription", billingHandler.CancelSubscription)
				r.With(middleware.RequirePermission(e.rbacSvc, entity.PermBillingRead, deps.GetUserID)).
					Get("/billing/usage", billingHandler.GetUsage)
			})
		})
	})
}
