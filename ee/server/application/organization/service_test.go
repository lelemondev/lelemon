package organization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lelemon/ee/server/domain/entity"
)

// Mock implementations

type mockOrgRepo struct {
	orgs    map[string]*entity.Organization
	members map[string]map[string]*entity.TeamMember // orgID -> userID -> member
}

func newMockOrgRepo() *mockOrgRepo {
	return &mockOrgRepo{
		orgs:    make(map[string]*entity.Organization),
		members: make(map[string]map[string]*entity.TeamMember),
	}
}

func (m *mockOrgRepo) CreateOrganization(ctx context.Context, org *entity.Organization) error {
	// For tests, use Pro plan settings to allow more members
	org.Plan = entity.PlanPro
	org.Settings = entity.PlanPro.GetDefaultSettings()
	m.orgs[org.ID] = org
	return nil
}

func (m *mockOrgRepo) GetOrganizationByID(ctx context.Context, id string) (*entity.Organization, error) {
	if org, ok := m.orgs[id]; ok {
		return org, nil
	}
	return nil, entity.ErrNotFound
}

func (m *mockOrgRepo) GetOrganizationBySlug(ctx context.Context, slug string) (*entity.Organization, error) {
	for _, org := range m.orgs {
		if org.Slug == slug {
			return org, nil
		}
	}
	return nil, entity.ErrNotFound
}

func (m *mockOrgRepo) UpdateOrganization(ctx context.Context, id string, updates *entity.OrganizationUpdate) error {
	return nil
}

func (m *mockOrgRepo) DeleteOrganization(ctx context.Context, id string) error {
	delete(m.orgs, id)
	return nil
}

func (m *mockOrgRepo) ListOrganizationsByUser(ctx context.Context, userID string) ([]entity.Organization, error) {
	var result []entity.Organization
	for orgID, members := range m.members {
		if _, ok := members[userID]; ok {
			if org, ok := m.orgs[orgID]; ok {
				result = append(result, *org)
			}
		}
	}
	return result, nil
}

func (m *mockOrgRepo) AddMember(ctx context.Context, member *entity.TeamMember) error {
	if m.members[member.OrganizationID] == nil {
		m.members[member.OrganizationID] = make(map[string]*entity.TeamMember)
	}
	m.members[member.OrganizationID][member.UserID] = member
	return nil
}

func (m *mockOrgRepo) GetMember(ctx context.Context, orgID, userID string) (*entity.TeamMember, error) {
	if members, ok := m.members[orgID]; ok {
		if member, ok := members[userID]; ok {
			return member, nil
		}
	}
	return nil, entity.ErrNotFound
}

func (m *mockOrgRepo) UpdateMember(ctx context.Context, orgID, userID string, role entity.Role) error {
	if members, ok := m.members[orgID]; ok {
		if member, ok := members[userID]; ok {
			member.Role = role
			return nil
		}
	}
	return entity.ErrNotFound
}

func (m *mockOrgRepo) RemoveMember(ctx context.Context, orgID, userID string) error {
	if members, ok := m.members[orgID]; ok {
		delete(members, userID)
	}
	return nil
}

func (m *mockOrgRepo) ListMembers(ctx context.Context, orgID string) ([]entity.TeamMember, error) {
	var result []entity.TeamMember
	if members, ok := m.members[orgID]; ok {
		for _, m := range members {
			result = append(result, *m)
		}
	}
	return result, nil
}

func (m *mockOrgRepo) GetUserOrganizations(ctx context.Context, userID string) ([]entity.TeamMember, error) {
	return nil, nil
}

func (m *mockOrgRepo) CountMembers(ctx context.Context, orgID string) (int, error) {
	if members, ok := m.members[orgID]; ok {
		return len(members), nil
	}
	return 0, nil
}

type mockUserStore struct {
	users map[string]*mockUser // email -> user
}

type mockUser struct {
	id    string
	email string
}

func (u *mockUser) GetID() string    { return u.id }
func (u *mockUser) GetEmail() string { return u.email }

func newMockUserStore() *mockUserStore {
	return &mockUserStore{users: make(map[string]*mockUser)}
}

func (m *mockUserStore) AddUser(id, email string) {
	m.users[email] = &mockUser{id: id, email: email}
}

func (m *mockUserStore) GetUserByID(ctx context.Context, id string) (UserInfo, error) {
	for _, u := range m.users {
		if u.id == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserStore) GetUserByEmail(ctx context.Context, email string) (UserInfo, error) {
	if u, ok := m.users[email]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

// Tests

func TestService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("creates organization with owner", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("user-1", "owner@example.com")

		svc := NewService(repo, userStore)

		org, err := svc.Create(ctx, "user-1", &CreateRequest{Name: "Acme Corp"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if org.Name != "Acme Corp" {
			t.Errorf("expected name 'Acme Corp', got '%s'", org.Name)
		}
		if org.Slug != "acme-corp" {
			t.Errorf("expected slug 'acme-corp', got '%s'", org.Slug)
		}
		if org.OwnerUserID != "user-1" {
			t.Errorf("expected owner 'user-1', got '%s'", org.OwnerUserID)
		}
		// Note: mock upgrades to Pro for testing member limits
		if org.Plan != entity.PlanPro {
			t.Errorf("expected plan 'pro', got '%s'", org.Plan)
		}

		// Verify owner was added as member
		member, err := repo.GetMember(ctx, org.ID, "user-1")
		if err != nil {
			t.Fatalf("owner not added as member: %v", err)
		}
		if member.Role != entity.RoleOwner {
			t.Errorf("expected role 'owner', got '%s'", member.Role)
		}
	})

	t.Run("fails with invalid name", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("user-1", "owner@example.com")

		svc := NewService(repo, userStore)

		_, err := svc.Create(ctx, "user-1", &CreateRequest{Name: ""})

		if err != entity.ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("fails with non-existent user", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()

		svc := NewService(repo, userStore)

		_, err := svc.Create(ctx, "non-existent", &CreateRequest{Name: "Acme"})

		if err != entity.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestService_InviteMember(t *testing.T) {
	ctx := context.Background()

	setup := func() (*Service, *mockOrgRepo, string) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")
		userStore.AddUser("invitee-1", "invitee@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		return svc, repo, org.ID
	}

	t.Run("owner invites member", func(t *testing.T) {
		svc, repo, orgID := setup()

		err := svc.InviteMember(ctx, orgID, "owner-1", "invitee@example.com", entity.RoleMember)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		member, err := repo.GetMember(ctx, orgID, "invitee-1")
		if err != nil {
			t.Fatalf("member not found: %v", err)
		}
		if member.Role != entity.RoleMember {
			t.Errorf("expected role 'member', got '%s'", member.Role)
		}
		if member.JoinedAt != nil {
			t.Error("expected pending invitation (JoinedAt nil)")
		}
	})

	t.Run("cannot invite as owner", func(t *testing.T) {
		svc, _, orgID := setup()

		err := svc.InviteMember(ctx, orgID, "owner-1", "invitee@example.com", entity.RoleOwner)

		if err != entity.ErrCannotInviteAsOwner {
			t.Errorf("expected ErrCannotInviteAsOwner, got %v", err)
		}
	})

	t.Run("non-member cannot invite", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")
		userStore.AddUser("random-1", "random@example.com")
		userStore.AddUser("invitee-1", "invitee@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		err := svc.InviteMember(ctx, org.ID, "random-1", "invitee@example.com", entity.RoleMember)

		if err != ErrNotMember {
			t.Errorf("expected ErrNotMember, got %v", err)
		}
	})

	t.Run("cannot invite non-existent user", func(t *testing.T) {
		svc, _, orgID := setup()

		err := svc.InviteMember(ctx, orgID, "owner-1", "nonexistent@example.com", entity.RoleMember)

		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("cannot invite already member", func(t *testing.T) {
		svc, _, orgID := setup()

		// First invite
		svc.InviteMember(ctx, orgID, "owner-1", "invitee@example.com", entity.RoleMember)

		// Second invite
		err := svc.InviteMember(ctx, orgID, "owner-1", "invitee@example.com", entity.RoleMember)

		if err != ErrAlreadyMember {
			t.Errorf("expected ErrAlreadyMember, got %v", err)
		}
	})
}

func TestService_UpdateMemberRole(t *testing.T) {
	ctx := context.Background()

	setup := func() (*Service, *mockOrgRepo, string) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")
		userStore.AddUser("member-1", "member@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		// Add member
		svc.InviteMember(ctx, org.ID, "owner-1", "member@example.com", entity.RoleMember)

		return svc, repo, org.ID
	}

	t.Run("owner promotes member to admin", func(t *testing.T) {
		svc, repo, orgID := setup()

		err := svc.UpdateMemberRole(ctx, orgID, "owner-1", "member-1", entity.RoleAdmin)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		member, _ := repo.GetMember(ctx, orgID, "member-1")
		if member.Role != entity.RoleAdmin {
			t.Errorf("expected role 'admin', got '%s'", member.Role)
		}
	})

	t.Run("cannot demote owner", func(t *testing.T) {
		svc, _, orgID := setup()

		err := svc.UpdateMemberRole(ctx, orgID, "owner-1", "owner-1", entity.RoleAdmin)

		if err != entity.ErrCannotDemoteOwner {
			t.Errorf("expected ErrCannotDemoteOwner, got %v", err)
		}
	})

	t.Run("non-member cannot update roles", func(t *testing.T) {
		svc, _, orgID := setup()

		err := svc.UpdateMemberRole(ctx, orgID, "random-user", "member-1", entity.RoleAdmin)

		if err != ErrNotMember {
			t.Errorf("expected ErrNotMember, got %v", err)
		}
	})
}

func TestService_RemoveMember(t *testing.T) {
	ctx := context.Background()

	setup := func() (*Service, *mockOrgRepo, string) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")
		userStore.AddUser("member-1", "member@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		// Add member
		svc.InviteMember(ctx, org.ID, "owner-1", "member@example.com", entity.RoleMember)

		return svc, repo, org.ID
	}

	t.Run("owner removes member", func(t *testing.T) {
		svc, repo, orgID := setup()

		err := svc.RemoveMember(ctx, orgID, "owner-1", "member-1")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = repo.GetMember(ctx, orgID, "member-1")
		if err == nil {
			t.Error("expected member to be removed")
		}
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		svc, _, orgID := setup()

		err := svc.RemoveMember(ctx, orgID, "owner-1", "owner-1")

		if err != entity.ErrCannotRemoveOwner {
			t.Errorf("expected ErrCannotRemoveOwner, got %v", err)
		}
	})

	t.Run("member cannot remove others", func(t *testing.T) {
		svc, _, orgID := setup()

		err := svc.RemoveMember(ctx, orgID, "member-1", "owner-1")

		if err != entity.ErrPermissionDenied {
			t.Errorf("expected ErrPermissionDenied, got %v", err)
		}
	})
}

func TestService_AcceptInvitation(t *testing.T) {
	ctx := context.Background()

	t.Run("accepts pending invitation", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")
		userStore.AddUser("invitee-1", "invitee@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		// Invite
		svc.InviteMember(ctx, org.ID, "owner-1", "invitee@example.com", entity.RoleMember)

		// Accept
		err := svc.AcceptInvitation(ctx, org.ID, "invitee-1")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		member, _ := repo.GetMember(ctx, org.ID, "invitee-1")
		if member.JoinedAt == nil {
			t.Error("expected JoinedAt to be set")
		}
	})

	t.Run("fails if not invited", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		err := svc.AcceptInvitation(ctx, org.ID, "random-user")

		if err != ErrNotMember {
			t.Errorf("expected ErrNotMember, got %v", err)
		}
	})

	t.Run("fails if already joined", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("owner-1", "owner@example.com")
		userStore.AddUser("invitee-1", "invitee@example.com")

		svc := NewService(repo, userStore)
		org, _ := svc.Create(ctx, "owner-1", &CreateRequest{Name: "Test Org"})

		svc.InviteMember(ctx, org.ID, "owner-1", "invitee@example.com", entity.RoleMember)
		svc.AcceptInvitation(ctx, org.ID, "invitee-1")

		// Try to accept again
		err := svc.AcceptInvitation(ctx, org.ID, "invitee-1")

		if err != ErrAlreadyJoined {
			t.Errorf("expected ErrAlreadyJoined, got %v", err)
		}
	})
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"Acme Corp", "acme-corp"},
		{"My   Company", "my-company"},
		{"Test 123", "test-123"},
		{"Hello World!", "hello-world"},
		{"  Leading Trailing  ", "leading-trailing"},
		{"Special@#$Characters", "specialcharacters"},
		{"MixedCASE", "mixedcase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.name)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestService_ListByUser(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user organizations", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()
		userStore.AddUser("user-1", "user@example.com")

		svc := NewService(repo, userStore)

		// Create multiple orgs
		org1, _ := svc.Create(ctx, "user-1", &CreateRequest{Name: "Org 1"})
		org2, _ := svc.Create(ctx, "user-1", &CreateRequest{Name: "Org 2"})

		orgs, err := svc.ListByUser(ctx, "user-1")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(orgs) != 2 {
			t.Errorf("expected 2 orgs, got %d", len(orgs))
		}

		// Verify both orgs are returned
		ids := make(map[string]bool)
		for _, o := range orgs {
			ids[o.ID] = true
		}
		if !ids[org1.ID] || !ids[org2.ID] {
			t.Error("missing expected organizations")
		}
	})

	t.Run("returns empty for new user", func(t *testing.T) {
		repo := newMockOrgRepo()
		userStore := newMockUserStore()

		svc := NewService(repo, userStore)

		orgs, err := svc.ListByUser(ctx, "new-user")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(orgs) != 0 {
			t.Errorf("expected 0 orgs, got %d", len(orgs))
		}
	})
}

// Test helper to avoid unused import
var _ = time.Now
