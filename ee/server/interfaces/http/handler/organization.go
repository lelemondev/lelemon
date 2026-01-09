package handler

import (
	"encoding/json"
	"net/http"
	"net/mail"

	"github.com/go-chi/chi/v5"
	"github.com/lelemon/ee/server/application/organization"
	"github.com/lelemon/ee/server/domain/entity"
)

// OrganizationHandler handles organization-related HTTP requests
type OrganizationHandler struct {
	svc       *organization.Service
	getUserID func(r *http.Request) string
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(svc *organization.Service, getUserID func(r *http.Request) string) *OrganizationHandler {
	return &OrganizationHandler{
		svc:       svc,
		getUserID: getUserID,
	}
}

// Create handles POST /organizations
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		WriteError(w, entity.ErrPermissionDenied)
		return
	}

	var req organization.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	if req.Name == "" {
		WriteError(w, entity.ErrInvalidName)
		return
	}

	org, err := h.svc.Create(r.Context(), userID, &req)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, org)
}

// List handles GET /organizations
func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID == "" {
		WriteError(w, entity.ErrPermissionDenied)
		return
	}

	orgs, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"data": orgs})
}

// Get handles GET /organizations/{orgId}
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")

	org, err := h.svc.GetByID(r.Context(), orgID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, org)
}

// ListMembers handles GET /organizations/{orgId}/members
func (h *OrganizationHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")

	members, err := h.svc.ListMembers(r.Context(), orgID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"data": members})
}

// InviteMember handles POST /organizations/{orgId}/invite
func (h *OrganizationHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	orgID := chi.URLParam(r, "orgId")

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	// Validate email format
	if _, err := mail.ParseAddress(req.Email); err != nil {
		WriteError(w, entity.ErrInvalidEmail)
		return
	}

	// Validate and default role
	role := entity.Role(req.Role)
	if role == "" {
		role = entity.RoleMember
	}
	if !role.IsValid() {
		WriteError(w, entity.ErrInvalidRole)
		return
	}

	// Cannot invite as owner
	if role == entity.RoleOwner {
		WriteError(w, entity.ErrCannotInviteAsOwner)
		return
	}

	err := h.svc.InviteMember(r.Context(), orgID, userID, req.Email, role)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateMember handles PATCH /organizations/{orgId}/members/{userId}
func (h *OrganizationHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	updaterID := h.getUserID(r)
	orgID := chi.URLParam(r, "orgId")
	targetUserID := chi.URLParam(r, "userId")

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, entity.ErrInvalidInput)
		return
	}

	role := entity.Role(req.Role)
	if !role.IsValid() {
		WriteError(w, entity.ErrInvalidRole)
		return
	}

	err := h.svc.UpdateMemberRole(r.Context(), orgID, updaterID, targetUserID, role)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember handles DELETE /organizations/{orgId}/members/{userId}
func (h *OrganizationHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	removerID := h.getUserID(r)
	orgID := chi.URLParam(r, "orgId")
	targetUserID := chi.URLParam(r, "userId")

	err := h.svc.RemoveMember(r.Context(), orgID, removerID, targetUserID)
	if err != nil {
		WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
