package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lelemon/ee/server/application/organization"
	"github.com/lelemon/ee/server/domain/entity"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		// Domain errors
		{
			name:           "not found",
			err:            entity.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "permission denied",
			err:            entity.ErrPermissionDenied,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "limit exceeded",
			err:            entity.ErrLimitExceeded,
			expectedStatus: http.StatusPaymentRequired,
			expectedCode:   "LIMIT_EXCEEDED",
		},
		{
			name:           "already exists",
			err:            entity.ErrAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_EXISTS",
		},
		// Validation errors
		{
			name:           "invalid name",
			err:            entity.ErrInvalidName,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "invalid slug",
			err:            entity.ErrInvalidSlug,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "invalid email",
			err:            entity.ErrInvalidEmail,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "invalid role",
			err:            entity.ErrInvalidRole,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "missing org ID",
			err:            entity.ErrMissingOrgID,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		{
			name:           "missing user ID",
			err:            entity.ErrMissingUserID,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
		// Business rule errors
		{
			name:           "cannot invite as owner",
			err:            entity.ErrCannotInviteAsOwner,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "BUSINESS_RULE",
		},
		{
			name:           "cannot remove owner",
			err:            entity.ErrCannotRemoveOwner,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "BUSINESS_RULE",
		},
		{
			name:           "cannot demote owner",
			err:            entity.ErrCannotDemoteOwner,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "BUSINESS_RULE",
		},
		{
			name:           "insufficient privilege",
			err:            entity.ErrInsufficientPrivilege,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		// Application errors
		{
			name:           "not member",
			err:            organization.ErrNotMember,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "NOT_MEMBER",
		},
		{
			name:           "cannot invite higher role",
			err:            organization.ErrCannotInviteHigherRole,
			expectedStatus: http.StatusForbidden,
			expectedCode:   "FORBIDDEN",
		},
		{
			name:           "already member",
			err:            organization.ErrAlreadyMember,
			expectedStatus: http.StatusConflict,
			expectedCode:   "ALREADY_EXISTS",
		},
		{
			name:           "user not found",
			err:            organization.ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "USER_NOT_FOUND",
		},
		// Unknown error - should return 500
		{
			name:           "unknown error",
			err:            errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteError(w, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("WriteError() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("WriteError() Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
			}

			var apiErr APIError
			if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if apiErr.Code != tt.expectedCode {
				t.Errorf("WriteError() code = %s, want %s", apiErr.Code, tt.expectedCode)
			}

			if apiErr.Error == "" {
				t.Error("WriteError() error message should not be empty")
			}
		})
	}
}

func TestWriteError_DoesNotLeakInternalDetails(t *testing.T) {
	// Simulate a database error with potentially sensitive info
	internalErr := errors.New("pq: connection to database at 192.168.1.100:5432 failed: password authentication failed for user 'admin'")

	w := httptest.NewRecorder()
	WriteError(w, internalErr)

	var apiErr APIError
	if err := json.Unmarshal(w.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should return generic message, not the actual error
	if apiErr.Error != "Internal server error" {
		t.Errorf("WriteError() should not leak internal error details, got: %s", apiErr.Error)
	}

	// Body should not contain sensitive info
	body := w.Body.String()
	sensitiveStrings := []string{"192.168", "password", "admin", "5432"}
	for _, s := range sensitiveStrings {
		if contains(body, s) {
			t.Errorf("WriteError() response contains sensitive info: %s", s)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	type TestData struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	tests := []struct {
		name     string
		status   int
		data     interface{}
	}{
		{
			name:   "success response",
			status: http.StatusOK,
			data:   TestData{ID: "123", Name: "Test"},
		},
		{
			name:   "created response",
			status: http.StatusCreated,
			data:   TestData{ID: "456", Name: "New Item"},
		},
		{
			name:   "map response",
			status: http.StatusOK,
			data:   map[string]string{"url": "https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.status, tt.data)

			if w.Code != tt.status {
				t.Errorf("WriteJSON() status = %d, want %d", w.Code, tt.status)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("WriteJSON() Content-Type = %s, want application/json", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestMapError_WrappedErrors(t *testing.T) {
	// Test that wrapped errors are correctly identified
	wrappedErr := errors.Join(errors.New("context"), entity.ErrNotFound)

	status, apiErr := mapError(wrappedErr)

	if status != http.StatusNotFound {
		t.Errorf("mapError() status = %d, want %d", status, http.StatusNotFound)
	}

	if apiErr.Code != "NOT_FOUND" {
		t.Errorf("mapError() code = %s, want NOT_FOUND", apiErr.Code)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
