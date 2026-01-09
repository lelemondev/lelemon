package entity

import "errors"

// Domain errors - safe to expose to clients
var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrInvalidInput     = errors.New("invalid input")
	ErrPermissionDenied = errors.New("permission denied")
	ErrLimitExceeded    = errors.New("limit exceeded")
)

// Validation errors
var (
	ErrInvalidName   = errors.New("name must be 1-100 characters")
	ErrInvalidSlug   = errors.New("invalid slug format")
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrInvalidRole   = errors.New("invalid role")
	ErrMissingOrgID  = errors.New("organization ID required")
	ErrMissingUserID = errors.New("user ID required")
)

// Business rule errors
var (
	ErrCannotInviteAsOwner   = errors.New("cannot invite as owner")
	ErrCannotRemoveOwner     = errors.New("cannot remove organization owner")
	ErrCannotDemoteOwner     = errors.New("cannot demote organization owner")
	ErrInsufficientPrivilege = errors.New("insufficient privilege for this action")
)
