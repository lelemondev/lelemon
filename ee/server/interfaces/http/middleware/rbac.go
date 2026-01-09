package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lelemon/ee/server/application/organization"
	"github.com/lelemon/ee/server/application/rbac"
	"github.com/lelemon/ee/server/domain/entity"
)

type contextKey string

const (
	OrganizationContextKey contextKey = "organization"
	MemberContextKey       contextKey = "member"
)

// GetOrganization retrieves the organization from context
func GetOrganization(ctx context.Context) *entity.Organization {
	if org, ok := ctx.Value(OrganizationContextKey).(*entity.Organization); ok {
		return org
	}
	return nil
}

// GetOrganizationID retrieves the organization ID from context
func GetOrganizationID(ctx context.Context) string {
	if org := GetOrganization(ctx); org != nil {
		return org.ID
	}
	return ""
}

// GetMember retrieves the team member from context
func GetMember(ctx context.Context) *entity.TeamMember {
	if m, ok := ctx.Value(MemberContextKey).(*entity.TeamMember); ok {
		return m
	}
	return nil
}

// RequirePermission creates a middleware that checks for a specific permission
func RequirePermission(rbacSvc *rbac.Service, perm entity.Permission, getUserID func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserID(r)
			if userID == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// Get orgID from URL param or context
			orgID := chi.URLParam(r, "orgId")
			if orgID == "" {
				orgID = GetOrganizationID(r.Context())
			}

			if orgID == "" {
				http.Error(w, `{"error":"organization required"}`, http.StatusBadRequest)
				return
			}

			// Check permission
			allowed, err := rbacSvc.CheckPermission(r.Context(), userID, orgID, perm)
			if err != nil {
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// InjectOrganization loads and injects the organization into the request context
func InjectOrganization(orgSvc *organization.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := chi.URLParam(r, "orgId")
			if orgID == "" {
				next.ServeHTTP(w, r)
				return
			}

			org, err := orgSvc.GetByID(r.Context(), orgID)
			if err != nil {
				http.Error(w, `{"error":"organization not found"}`, http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), OrganizationContextKey, org)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireOrgMember ensures the user is an active member of the organization
func RequireOrgMember(rbacSvc *rbac.Service, getUserID func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserID(r)
			if userID == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			orgID := chi.URLParam(r, "orgId")
			if orgID == "" {
				orgID = GetOrganizationID(r.Context())
			}

			if orgID == "" {
				http.Error(w, `{"error":"organization required"}`, http.StatusBadRequest)
				return
			}

			isMember, err := rbacSvc.IsMember(r.Context(), userID, orgID)
			if err != nil || !isMember {
				http.Error(w, `{"error":"not a member of this organization"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
