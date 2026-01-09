package entity

// Permission represents a specific action that can be performed
type Permission string

const (
	// Projects
	PermProjectCreate Permission = "project:create"
	PermProjectRead   Permission = "project:read"
	PermProjectUpdate Permission = "project:update"
	PermProjectDelete Permission = "project:delete"

	// Traces
	PermTraceRead   Permission = "trace:read"
	PermTraceDelete Permission = "trace:delete"

	// Team
	PermTeamRead   Permission = "team:read"
	PermTeamInvite Permission = "team:invite"
	PermTeamManage Permission = "team:manage"

	// Billing
	PermBillingRead  Permission = "billing:read"
	PermBillingWrite Permission = "billing:write"

	// API Keys
	PermAPIKeyCreate Permission = "apikey:create"
	PermAPIKeyRotate Permission = "apikey:rotate"

	// Organization
	PermOrgRead   Permission = "org:read"
	PermOrgUpdate Permission = "org:update"
	PermOrgDelete Permission = "org:delete"
)

// RolePermissions defines what each role can do
var RolePermissions = map[Role][]Permission{
	RoleOwner: {
		PermProjectCreate, PermProjectRead, PermProjectUpdate, PermProjectDelete,
		PermTraceRead, PermTraceDelete,
		PermTeamRead, PermTeamInvite, PermTeamManage,
		PermBillingRead, PermBillingWrite,
		PermAPIKeyCreate, PermAPIKeyRotate,
		PermOrgRead, PermOrgUpdate, PermOrgDelete,
	},
	RoleAdmin: {
		PermProjectCreate, PermProjectRead, PermProjectUpdate, PermProjectDelete,
		PermTraceRead, PermTraceDelete,
		PermTeamRead, PermTeamInvite,
		PermAPIKeyCreate, PermAPIKeyRotate,
		PermOrgRead, PermOrgUpdate,
	},
	RoleMember: {
		PermProjectRead, PermProjectUpdate,
		PermTraceRead,
		PermTeamRead,
		PermAPIKeyCreate,
		PermOrgRead,
	},
	RoleViewer: {
		PermProjectRead,
		PermTraceRead,
		PermTeamRead,
		PermOrgRead,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role Role, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}
