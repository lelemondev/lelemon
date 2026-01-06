package handler_test

import (
	"net/http"
	"testing"
)

func TestRegister(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("successful registration", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
			"email":    "test@example.com",
			"password": "password123",
			"name":     "Test User",
		}, nil)

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}

		var auth AuthResponse
		ParseJSON(t, resp, &auth)

		if auth.Token == "" {
			t.Error("expected token to be non-empty")
		}
		if auth.User.Email != "test@example.com" {
			t.Errorf("expected email test@example.com, got %s", auth.User.Email)
		}
		if auth.User.Name != "Test User" {
			t.Errorf("expected name Test User, got %s", auth.User.Name)
		}
	})

	t.Run("duplicate email fails", func(t *testing.T) {
		// First registration
		ts.Request("POST", "/api/v1/auth/register", map[string]string{
			"email":    "duplicate@example.com",
			"password": "password123",
			"name":     "User 1",
		}, nil)

		// Second registration with same email
		resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
			"email":    "duplicate@example.com",
			"password": "password456",
			"name":     "User 2",
		}, nil)

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("expected status 409, got %d", resp.StatusCode)
		}
	})

	t.Run("weak password fails", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
			"email":    "weak@example.com",
			"password": "short",
			"name":     "Weak User",
		}, nil)

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("missing fields fails", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
			"email": "incomplete@example.com",
		}, nil)

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestLogin(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create a user first
	ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email":    "login@example.com",
		"password": "password123",
		"name":     "Login User",
	}, nil)

	t.Run("successful login", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"email":    "login@example.com",
			"password": "password123",
		}, nil)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var auth AuthResponse
		ParseJSON(t, resp, &auth)

		if auth.Token == "" {
			t.Error("expected token to be non-empty")
		}
	})

	t.Run("wrong password fails", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"email":    "login@example.com",
			"password": "wrongpassword",
		}, nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("unknown email fails", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/auth/login", map[string]string{
			"email":    "unknown@example.com",
			"password": "password123",
		}, nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})
}

func TestMe(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create a user and get token
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email":    "me@example.com",
		"password": "password123",
		"name":     "Me User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	t.Run("get current user with valid token", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/auth/me", nil, map[string]string{
			"Authorization": "Bearer " + auth.Token,
		})

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("no token fails", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/auth/me", nil, nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid token fails", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/auth/me", nil, map[string]string{
			"Authorization": "Bearer invalid-token",
		})

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})
}
