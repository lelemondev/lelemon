package entity

import (
	"testing"
	"time"
)

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role    Role
		isValid bool
	}{
		{RoleOwner, true},
		{RoleAdmin, true},
		{RoleMember, true},
		{RoleViewer, true},
		{"invalid", false},
		{"", false},
		{"OWNER", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.isValid)
			}
		})
	}
}

func TestRole_Level(t *testing.T) {
	tests := []struct {
		role  Role
		level int
	}{
		{RoleOwner, 100},
		{RoleAdmin, 75},
		{RoleMember, 50},
		{RoleViewer, 25},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.Level(); got != tt.level {
				t.Errorf("Level() = %d, want %d", got, tt.level)
			}
		})
	}
}

func TestRole_CanManage(t *testing.T) {
	tests := []struct {
		name   string
		role   Role
		target Role
		can    bool
	}{
		{"owner can manage admin", RoleOwner, RoleAdmin, true},
		{"owner can manage member", RoleOwner, RoleMember, true},
		{"owner can manage viewer", RoleOwner, RoleViewer, true},
		{"owner cannot manage owner", RoleOwner, RoleOwner, false},
		{"admin can manage member", RoleAdmin, RoleMember, true},
		{"admin can manage viewer", RoleAdmin, RoleViewer, true},
		{"admin cannot manage admin", RoleAdmin, RoleAdmin, false},
		{"admin cannot manage owner", RoleAdmin, RoleOwner, false},
		{"member can manage viewer", RoleMember, RoleViewer, true},
		{"member cannot manage member", RoleMember, RoleMember, false},
		{"member cannot manage admin", RoleMember, RoleAdmin, false},
		{"viewer cannot manage anyone", RoleViewer, RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.CanManage(tt.target); got != tt.can {
				t.Errorf("CanManage(%s) = %v, want %v", tt.target, got, tt.can)
			}
		})
	}
}

func TestTeamMember_Validate(t *testing.T) {
	tests := []struct {
		name    string
		member  TeamMember
		wantErr error
	}{
		{
			name: "valid member",
			member: TeamMember{
				OrganizationID: "org-123",
				UserID:         "user-456",
				Role:           RoleMember,
			},
			wantErr: nil,
		},
		{
			name: "valid owner",
			member: TeamMember{
				OrganizationID: "org-123",
				UserID:         "user-456",
				Role:           RoleOwner,
			},
			wantErr: nil,
		},
		{
			name: "missing org ID",
			member: TeamMember{
				UserID: "user-456",
				Role:   RoleMember,
			},
			wantErr: ErrMissingOrgID,
		},
		{
			name: "missing user ID",
			member: TeamMember{
				OrganizationID: "org-123",
				Role:           RoleMember,
			},
			wantErr: ErrMissingUserID,
		},
		{
			name: "invalid role",
			member: TeamMember{
				OrganizationID: "org-123",
				UserID:         "user-456",
				Role:           "superadmin",
			},
			wantErr: ErrInvalidRole,
		},
		{
			name: "empty role",
			member: TeamMember{
				OrganizationID: "org-123",
				UserID:         "user-456",
				Role:           "",
			},
			wantErr: ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.member.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTeamMember_IsPending(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		member   TeamMember
		isPending bool
	}{
		{
			name: "pending - no JoinedAt",
			member: TeamMember{
				JoinedAt: nil,
			},
			isPending: true,
		},
		{
			name: "not pending - has JoinedAt",
			member: TeamMember{
				JoinedAt: &now,
			},
			isPending: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.member.IsPending(); got != tt.isPending {
				t.Errorf("IsPending() = %v, want %v", got, tt.isPending)
			}
		})
	}
}
