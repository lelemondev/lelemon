package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lelemon/ee/server/application/billing"
	"github.com/lelemon/ee/server/domain/entity"
	"github.com/lelemon/ee/server/infrastructure/lemonsqueezy"
)

// BillingHandler handles billing-related HTTP requests
type BillingHandler struct {
	svc          *billing.Service
	lsClient     *lemonsqueezy.Client
	getUserEmail func(r *http.Request) string
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(svc *billing.Service, lsClient *lemonsqueezy.Client, getUserEmail func(r *http.Request) string) *BillingHandler {
	return &BillingHandler{
		svc:          svc,
		lsClient:     lsClient,
		getUserEmail: getUserEmail,
	}
}

// GetBilling handles GET /organizations/{orgId}/billing
func (h *BillingHandler) GetBilling(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")

	info, err := h.svc.GetBilling(r.Context(), orgID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, info)
}

// CreateCheckout handles POST /organizations/{orgId}/billing/checkout
func (h *BillingHandler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
	userEmail := h.getUserEmail(r)
	if userEmail == "" {
		WriteError(w, entity.ErrPermissionDenied)
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	plan := entity.BillingPlan(req.Plan)
	if !plan.IsValid() || plan == entity.PlanFree {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	url, err := h.svc.CreateCheckout(r.Context(), orgID, plan, userEmail)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"url": url})
}

// GetCustomerPortal handles GET /organizations/{orgId}/billing/portal
func (h *BillingHandler) GetCustomerPortal(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")

	url, err := h.svc.GetCustomerPortal(r.Context(), orgID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"url": url})
}

// CancelSubscription handles DELETE /organizations/{orgId}/billing/subscription
func (h *BillingHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")

	err := h.svc.CancelSubscription(r.Context(), orgID)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetUsage handles GET /organizations/{orgId}/billing/usage
func (h *BillingHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")

	info, err := h.svc.GetBilling(r.Context(), orgID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"usage":       info.Usage,
		"limits":      info.Limits,
		"usageReport": info.UsageReport,
	})
}

// HandleWebhook handles POST /webhooks/lemonsqueezy
func (h *BillingHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read body"}`, http.StatusBadRequest)
		return
	}

	// Verify signature
	signature := r.Header.Get("X-Signature")
	if !h.lsClient.VerifyWebhookSignature(body, signature) {
		http.Error(w, `{"error":"invalid signature"}`, http.StatusUnauthorized)
		return
	}

	// Parse event
	event, err := h.lsClient.ParseWebhookEvent(body)
	if err != nil {
		http.Error(w, `{"error":"failed to parse event"}`, http.StatusBadRequest)
		return
	}

	// Handle event
	if err := h.svc.HandleWebhook(r.Context(), event); err != nil {
		// Log error but return 200 to prevent retries for known errors
		// In production, you'd log this properly
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}
