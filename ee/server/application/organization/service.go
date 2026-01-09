package organization

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lelemon/ee/server/domain/entity"
	"github.com/lelemon/ee/server/domain/repository"
)

// Service-specific errors (wrap domain errors for context)
var (
	ErrNotMember              = errors.New("not a member of this organization")
	ErrCannotInviteHigherRole = errors.New("cannot invite user with higher role than your own")
	ErrMemberLimitReached     = errors.New("member limit reached for current plan")
	ErrSlugTaken              = errors.New("organization slug already taken")
	ErrAlreadyMember          = errors.New("user is already a member")
	ErrAlreadyJoined          = errors.New("already joined")
	ErrUserNotFound           = errors.New("user not found - they must sign up first")
)

// UserStore defines what we need from the core user store (ISP)
// This matches the core repository.UserStore interface
type UserStore interface {
	GetUserByID(ctx context.Context, id string) (UserInfo, error)
	GetUserByEmail(ctx context.Context, email string) (UserInfo, error)
}

// UserInfo represents user data - interface that core entity.User implements
type UserInfo interface {
	GetID() string
	GetEmail() string
}

// Service handles organization business logic
type Service struct {
	repo      repository.OrganizationRepository // Interface, not concrete
	userStore UserStore                         // Interface, not concrete
}

// NewService creates a new organization service
func NewService(repo repository.OrganizationRepository, userStore UserStore) *Service {
	return &Service{
		repo:      repo,
		userStore: userStore,
	}
}

// CreateRequest holds data for creating an organization
type CreateRequest struct {
	Name string `json:"name"`
}

// Create creates a new organization and adds the creator as owner
func (s *Service) Create(ctx context.Context, userID string, req *CreateRequest) (*entity.Organization, error) {
	// Validate user exists
	if _, err := s.userStore.GetUserByID(ctx, userID); err != nil {
		return nil, entity.ErrNotFound
	}

	// Generate slug
	slug := slugify(req.Name)

	// Check if slug is taken
	if _, err := s.repo.GetOrganizationBySlug(ctx, slug); err == nil {
		// Slug exists, append random suffix
		slug = slug + "-" + uuid.New().String()[:8]
	}

	// Create organization
	now := time.Now()
	org := &entity.Organization{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Slug:        slug,
		OwnerUserID: userID,
		Plan:        entity.PlanFree,
		Settings:    entity.PlanFree.GetDefaultSettings(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Validate organization
	if err := org.Validate(); err != nil {
		return nil, err
	}

	// Create owner member
	member := &entity.TeamMember{
		ID:             uuid.New().String(),
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           entity.RoleOwner,
		JoinedAt:       &now,
		CreatedAt:      now,
	}

	// Validate member
	if err := member.Validate(); err != nil {
		return nil, err
	}

	// Create org and member (if transactional repo is available, use it)
	if txRepo, ok := s.repo.(repository.Transactional); ok {
		err := txRepo.WithTransaction(ctx, func(txCtx context.Context) error {
			if err := s.repo.CreateOrganization(txCtx, org); err != nil {
				return err
			}
			return s.repo.AddMember(txCtx, member)
		})
		if err != nil {
			return nil, err
		}
	} else {
		// Fallback: create without transaction
		if err := s.repo.CreateOrganization(ctx, org); err != nil {
			return nil, err
		}
		if err := s.repo.AddMember(ctx, member); err != nil {
			// Rollback org creation on error
			_ = s.repo.DeleteOrganization(ctx, org.ID)
			return nil, err
		}
	}

	return org, nil
}

// GetByID returns an organization by ID
func (s *Service) GetByID(ctx context.Context, id string) (*entity.Organization, error) {
	return s.repo.GetOrganizationByID(ctx, id)
}

// GetBySlug returns an organization by slug
func (s *Service) GetBySlug(ctx context.Context, slug string) (*entity.Organization, error) {
	return s.repo.GetOrganizationBySlug(ctx, slug)
}

// ListByUser returns all organizations the user is a member of
func (s *Service) ListByUser(ctx context.Context, userID string) ([]entity.Organization, error) {
	return s.repo.ListOrganizationsByUser(ctx, userID)
}

// InviteMember invites a user to join the organization
func (s *Service) InviteMember(ctx context.Context, orgID, inviterID, email string, role entity.Role) error {
	// Validate role
	if !role.IsValid() {
		return entity.ErrInvalidRole
	}

	// Cannot invite as owner
	if role == entity.RoleOwner {
		return entity.ErrCannotInviteAsOwner
	}

	// Check inviter is a member
	inviter, err := s.repo.GetMember(ctx, orgID, inviterID)
	if err != nil {
		return ErrNotMember
	}

	// Check inviter has permission to invite
	if !entity.HasPermission(inviter.Role, entity.PermTeamInvite) {
		return entity.ErrPermissionDenied
	}

	// Cannot invite higher role
	if !inviter.Role.CanManage(role) {
		return ErrCannotInviteHigherRole
	}

	// Get organization to check limits
	org, err := s.repo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}

	// Check member limit
	memberCount, err := s.repo.CountMembers(ctx, orgID)
	if err != nil {
		return err
	}

	limits := entity.PlanLimits[org.Plan]
	if limits.MaxMembers > 0 && memberCount >= limits.MaxMembers {
		return entity.ErrLimitExceeded
	}

	// Find user by email
	user, err := s.userStore.GetUserByEmail(ctx, email)
	if err != nil {
		return ErrUserNotFound
	}

	// Check if already a member
	if existing, err := s.repo.GetMember(ctx, orgID, user.GetID()); err == nil && existing != nil {
		return ErrAlreadyMember
	}

	// Create pending membership
	now := time.Now()
	member := &entity.TeamMember{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UserID:         user.GetID(),
		Role:           role,
		InvitedBy:      inviterID,
		InvitedAt:      now,
		JoinedAt:       nil, // Will be set when user accepts
		CreatedAt:      now,
	}

	// Validate member
	if err := member.Validate(); err != nil {
		return err
	}

	return s.repo.AddMember(ctx, member)
}

// AcceptInvitation marks the user as having joined the organization
func (s *Service) AcceptInvitation(ctx context.Context, orgID, userID string) error {
	member, err := s.repo.GetMember(ctx, orgID, userID)
	if err != nil {
		return ErrNotMember
	}

	if member.JoinedAt != nil {
		return ErrAlreadyJoined
	}

	// Update to joined
	now := time.Now()
	member.JoinedAt = &now

	// Re-add to update (simple approach)
	if err := s.repo.RemoveMember(ctx, orgID, userID); err != nil {
		return err
	}
	return s.repo.AddMember(ctx, member)
}

// UpdateMemberRole updates a member's role
func (s *Service) UpdateMemberRole(ctx context.Context, orgID, updaterID, targetUserID string, newRole entity.Role) error {
	// Validate role
	if !newRole.IsValid() {
		return entity.ErrInvalidRole
	}

	// Check updater permissions
	updater, err := s.repo.GetMember(ctx, orgID, updaterID)
	if err != nil {
		return ErrNotMember
	}

	if !entity.HasPermission(updater.Role, entity.PermTeamManage) {
		return entity.ErrPermissionDenied
	}

	// Get target member
	target, err := s.repo.GetMember(ctx, orgID, targetUserID)
	if err != nil {
		return ErrNotMember
	}

	// Cannot modify owner
	if target.Role == entity.RoleOwner {
		return entity.ErrCannotDemoteOwner
	}

	// Cannot promote to equal or higher role
	if !updater.Role.CanManage(newRole) {
		return entity.ErrInsufficientPrivilege
	}

	return s.repo.UpdateMember(ctx, orgID, targetUserID, newRole)
}

// RemoveMember removes a member from the organization
func (s *Service) RemoveMember(ctx context.Context, orgID, removerID, targetUserID string) error {
	// Check remover permissions
	remover, err := s.repo.GetMember(ctx, orgID, removerID)
	if err != nil {
		return ErrNotMember
	}

	if !entity.HasPermission(remover.Role, entity.PermTeamManage) {
		return entity.ErrPermissionDenied
	}

	// Get target member
	target, err := s.repo.GetMember(ctx, orgID, targetUserID)
	if err != nil {
		return ErrNotMember
	}

	// Cannot remove owner
	if target.Role == entity.RoleOwner {
		return entity.ErrCannotRemoveOwner
	}

	// Cannot remove higher or equal role
	if !remover.Role.CanManage(target.Role) {
		return entity.ErrInsufficientPrivilege
	}

	return s.repo.RemoveMember(ctx, orgID, targetUserID)
}

// ListMembers returns all members of an organization
func (s *Service) ListMembers(ctx context.Context, orgID string) ([]entity.TeamMember, error) {
	return s.repo.ListMembers(ctx, orgID)
}

// slugify converts a name to a URL-friendly slug
func slugify(name string) string {
	// Lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile("[^a-z0-9-]+")
	slug = reg.ReplaceAllString(slug, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}
