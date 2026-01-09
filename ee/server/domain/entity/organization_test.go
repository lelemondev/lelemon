package entity

import (
	"strings"
	"testing"
)

func TestOrganization_Validate(t *testing.T) {
	tests := []struct {
		name    string
		org     Organization
		wantErr error
	}{
		{
			name: "valid organization",
			org: Organization{
				Name: "Acme Corp",
				Slug: "acme-corp",
			},
			wantErr: nil,
		},
		{
			name: "valid with single word slug",
			org: Organization{
				Name: "Acme",
				Slug: "acme",
			},
			wantErr: nil,
		},
		{
			name: "valid without slug",
			org: Organization{
				Name: "Acme Corp",
			},
			wantErr: nil,
		},
		{
			name: "empty name",
			org: Organization{
				Name: "",
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "whitespace only name",
			org: Organization{
				Name: "   ",
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "name too long",
			org: Organization{
				Name: strings.Repeat("a", 101),
			},
			wantErr: ErrInvalidName,
		},
		{
			name:    "name exactly 100 chars",
			org:     Organization{Name: strings.Repeat("a", 100)},
			wantErr: nil,
		},
		{
			name: "invalid slug with uppercase",
			org: Organization{
				Name: "Acme",
				Slug: "Acme-Corp",
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid slug with spaces",
			org: Organization{
				Name: "Acme",
				Slug: "acme corp",
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid slug with underscore",
			org: Organization{
				Name: "Acme",
				Slug: "acme_corp",
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid slug starting with hyphen",
			org: Organization{
				Name: "Acme",
				Slug: "-acme",
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid slug ending with hyphen",
			org: Organization{
				Name: "Acme",
				Slug: "acme-",
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "invalid slug with consecutive hyphens",
			org: Organization{
				Name: "Acme",
				Slug: "acme--corp",
			},
			wantErr: ErrInvalidSlug,
		},
		{
			name: "valid slug with numbers",
			org: Organization{
				Name: "Team 42",
				Slug: "team-42",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.org.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBillingPlan_IsValid(t *testing.T) {
	tests := []struct {
		plan    BillingPlan
		isValid bool
	}{
		{PlanFree, true},
		{PlanPro, true},
		{PlanEnterprise, true},
		{"invalid", false},
		{"", false},
		{"FREE", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			if got := tt.plan.IsValid(); got != tt.isValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.isValid)
			}
		})
	}
}

func TestBillingPlan_GetDefaultSettings(t *testing.T) {
	tests := []struct {
		plan               BillingPlan
		expectedProjects   int
		expectedMembers    int
		expectedRetention  int
	}{
		{PlanFree, 3, 1, 7},
		{PlanPro, 20, 10, 30},
		{PlanEnterprise, -1, -1, 90},
	}

	for _, tt := range tests {
		t.Run(string(tt.plan), func(t *testing.T) {
			settings := tt.plan.GetDefaultSettings()
			if settings.MaxProjects != tt.expectedProjects {
				t.Errorf("MaxProjects = %d, want %d", settings.MaxProjects, tt.expectedProjects)
			}
			if settings.MaxMembers != tt.expectedMembers {
				t.Errorf("MaxMembers = %d, want %d", settings.MaxMembers, tt.expectedMembers)
			}
			if settings.RetentionDays != tt.expectedRetention {
				t.Errorf("RetentionDays = %d, want %d", settings.RetentionDays, tt.expectedRetention)
			}
		})
	}
}
