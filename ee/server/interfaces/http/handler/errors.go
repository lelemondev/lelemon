package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lelemon/ee/server/application/organization"
	"github.com/lelemon/ee/server/domain/entity"
)

// APIError represents a structured API error response
type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// WriteError writes a JSON error response with appropriate status code
func WriteError(w http.ResponseWriter, err error) {
	status, apiErr := mapError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiErr)
}

// mapError converts domain/application errors to HTTP status and safe messages
func mapError(err error) (int, APIError) {
	switch {
	// Domain errors - safe to expose
	case errors.Is(err, entity.ErrNotFound):
		return http.StatusNotFound, APIError{Error: "Resource not found", Code: "NOT_FOUND"}
	case errors.Is(err, entity.ErrPermissionDenied):
		return http.StatusForbidden, APIError{Error: "Permission denied", Code: "FORBIDDEN"}
	case errors.Is(err, entity.ErrLimitExceeded):
		return http.StatusPaymentRequired, APIError{Error: "Plan limit exceeded", Code: "LIMIT_EXCEEDED"}
	case errors.Is(err, entity.ErrAlreadyExists):
		return http.StatusConflict, APIError{Error: "Resource already exists", Code: "ALREADY_EXISTS"}

	// Validation errors - safe to expose
	case errors.Is(err, entity.ErrInvalidName):
		return http.StatusBadRequest, APIError{Error: err.Error(), Code: "VALIDATION_ERROR"}
	case errors.Is(err, entity.ErrInvalidSlug):
		return http.StatusBadRequest, APIError{Error: err.Error(), Code: "VALIDATION_ERROR"}
	case errors.Is(err, entity.ErrInvalidEmail):
		return http.StatusBadRequest, APIError{Error: err.Error(), Code: "VALIDATION_ERROR"}
	case errors.Is(err, entity.ErrInvalidRole):
		return http.StatusBadRequest, APIError{Error: err.Error(), Code: "VALIDATION_ERROR"}
	case errors.Is(err, entity.ErrMissingOrgID):
		return http.StatusBadRequest, APIError{Error: err.Error(), Code: "VALIDATION_ERROR"}
	case errors.Is(err, entity.ErrMissingUserID):
		return http.StatusBadRequest, APIError{Error: err.Error(), Code: "VALIDATION_ERROR"}

	// Business rule errors - safe to expose
	case errors.Is(err, entity.ErrCannotInviteAsOwner):
		return http.StatusBadRequest, APIError{Error: "Cannot invite as owner", Code: "BUSINESS_RULE"}
	case errors.Is(err, entity.ErrCannotRemoveOwner):
		return http.StatusBadRequest, APIError{Error: "Cannot remove organization owner", Code: "BUSINESS_RULE"}
	case errors.Is(err, entity.ErrCannotDemoteOwner):
		return http.StatusBadRequest, APIError{Error: "Cannot demote organization owner", Code: "BUSINESS_RULE"}
	case errors.Is(err, entity.ErrInsufficientPrivilege):
		return http.StatusForbidden, APIError{Error: "Insufficient privilege", Code: "FORBIDDEN"}

	// Application errors - safe to expose
	case errors.Is(err, organization.ErrNotMember):
		return http.StatusForbidden, APIError{Error: "Not a member of this organization", Code: "NOT_MEMBER"}
	case errors.Is(err, organization.ErrCannotInviteHigherRole):
		return http.StatusForbidden, APIError{Error: "Cannot invite user with higher role", Code: "FORBIDDEN"}
	case errors.Is(err, organization.ErrAlreadyMember):
		return http.StatusConflict, APIError{Error: "User is already a member", Code: "ALREADY_EXISTS"}
	case errors.Is(err, organization.ErrUserNotFound):
		return http.StatusNotFound, APIError{Error: "User not found - they must sign up first", Code: "USER_NOT_FOUND"}

	// Default: log internal error, return generic message
	default:
		slog.Error("internal error", "error", err)
		return http.StatusInternalServerError, APIError{Error: "Internal server error", Code: "INTERNAL_ERROR"}
	}
}

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
