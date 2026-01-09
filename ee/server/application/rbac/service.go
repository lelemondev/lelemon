package rbac

import (
	"context"

	"github.com/lelemon/ee/server/domain/entity"
	"github.com/lelemon/ee/server/domain/repository"
)

// Service handles RBAC (Role-Based Access Control) logic
type Service struct {
	repo repository.RBACRepository // Interface, not concrete
}

// NewService creates a new RBAC service
func NewService(repo repository.RBACRepository) *Service {
	return &Service{repo: repo}
}

// CheckPermission verifies if a user has a specific permission in an organization
func (s *Service) CheckPermission(ctx context.Context, userID, orgID string, perm entity.Permission) (bool, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, nil // Not a member = no permission
	}

	// Check if invitation is accepted
	if member.IsPending() {
		return false, nil
	}

	return entity.HasPermission(member.Role, perm), nil
}

// GetUserRole returns the user's role in an organization
func (s *Service) GetUserRole(ctx context.Context, userID, orgID string) (entity.Role, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return "", err
	}
	return member.Role, nil
}

// GetUserPermissions returns all permissions for a user in an organization
func (s *Service) GetUserPermissions(ctx context.Context, userID, orgID string) ([]entity.Permission, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	if member.IsPending() {
		return nil, nil
	}

	return entity.RolePermissions[member.Role], nil
}

// IsMember checks if a user is an active member of an organization
func (s *Service) IsMember(ctx context.Context, userID, orgID string) (bool, error) {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return false, nil
	}
	return !member.IsPending(), nil
}

// CanAccessProject checks if a user can access a project
// Projects belong to organizations, so we check org membership
func (s *Service) CanAccessProject(ctx context.Context, userID, orgID string) (bool, error) {
	return s.CheckPermission(ctx, userID, orgID, entity.PermProjectRead)
}

// RequirePermission is a helper that returns an error if permission is denied
func (s *Service) RequirePermission(ctx context.Context, userID, orgID string, perm entity.Permission) error {
	allowed, err := s.CheckPermission(ctx, userID, orgID, perm)
	if err != nil {
		return err
	}
	if !allowed {
		return entity.ErrPermissionDenied
	}
	return nil
}
