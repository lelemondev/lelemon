package entity

import "time"

// TeamMember represents a user's membership in an organization
type TeamMember struct {
	ID             string
	OrganizationID string
	UserID         string
	Role           Role
	InvitedBy      string     // UserID of who invited them
	InvitedAt      time.Time  // When invitation was sent
	JoinedAt       *time.Time // nil = pending invitation
	CreatedAt      time.Time
}

// Role represents the permission level within an organization
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

// RoleLevel returns numeric level for role comparison
func (r Role) Level() int {
	switch r {
	case RoleOwner:
		return 100
	case RoleAdmin:
		return 75
	case RoleMember:
		return 50
	case RoleViewer:
		return 25
	default:
		return 0
	}
}

// CanManage returns true if this role can manage the target role
func (r Role) CanManage(target Role) bool {
	return r.Level() > target.Level()
}

// IsPending returns true if the invitation hasn't been accepted
func (m *TeamMember) IsPending() bool {
	return m.JoinedAt == nil
}

// IsValid checks if the role is a valid enum value
func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	}
	return false
}

// Validate checks if the team member has valid data
func (m *TeamMember) Validate() error {
	if m.OrganizationID == "" {
		return ErrMissingOrgID
	}
	if m.UserID == "" {
		return ErrMissingUserID
	}
	if !m.Role.IsValid() {
		return ErrInvalidRole
	}
	return nil
}
