package handler_test

import (
	"net/http"
	"testing"
)

func TestProjectCRUD(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create user and get JWT token
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email":    "projectuser@example.com",
		"password": "password123",
		"name":     "Project User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)
	jwtHeaders := map[string]string{"Authorization": "Bearer " + auth.Token}

	var createdProject ProjectResponse

	t.Run("create project", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
			"name": "My Test Project",
		}, jwtHeaders)

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}

		ParseJSON(t, resp, &createdProject)

		if createdProject.Name != "My Test Project" {
			t.Errorf("expected name 'My Test Project', got '%s'", createdProject.Name)
		}
		if createdProject.APIKey == "" {
			t.Error("expected APIKey to be non-empty")
		}
		if createdProject.APIKey[:3] != "le_" {
			t.Errorf("expected APIKey to start with 'le_', got '%s'", createdProject.APIKey[:3])
		}
	})

	t.Run("list projects", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/dashboard/projects", nil, jwtHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var projects []map[string]any
		ParseJSON(t, resp, &projects)

		if len(projects) != 1 {
			t.Errorf("expected 1 project, got %d", len(projects))
		}
	})

	t.Run("update project", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/dashboard/projects/"+createdProject.ID, map[string]string{
			"name": "Updated Project Name",
		}, jwtHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("rotate API key", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/dashboard/projects/"+createdProject.ID+"/api-key", nil, jwtHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]string
		ParseJSON(t, resp, &result)

		newKey := result["apiKey"]
		if newKey == "" {
			t.Error("expected new API key")
		}
		if newKey == createdProject.APIKey {
			t.Error("expected new API key to be different from old one")
		}
	})

	t.Run("delete project", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/dashboard/projects/"+createdProject.ID, nil, jwtHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Verify deleted
		listResp := ts.Request("GET", "/api/v1/dashboard/projects", nil, jwtHeaders)
		var projects []map[string]any
		ParseJSON(t, listResp, &projects)

		if len(projects) != 0 {
			t.Errorf("expected 0 projects after delete, got %d", len(projects))
		}
	})
}

func TestProjectIsolation(t *testing.T) {
	ts := setupTestServer(t)

	// Create two users
	reg1 := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "user1@example.com", "password": "password123", "name": "User 1",
	}, nil)
	var auth1 AuthResponse
	ParseJSON(t, reg1, &auth1)

	reg2 := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "user2@example.com", "password": "password123", "name": "User 2",
	}, nil)
	var auth2 AuthResponse
	ParseJSON(t, reg2, &auth2)

	// User 1 creates a project
	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "User 1 Project",
	}, map[string]string{"Authorization": "Bearer " + auth1.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	t.Run("user cannot see other user's projects", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/dashboard/projects", nil, map[string]string{
			"Authorization": "Bearer " + auth2.Token,
		})

		var projects []map[string]any
		ParseJSON(t, resp, &projects)

		if len(projects) != 0 {
			t.Errorf("user2 should not see user1's projects, got %d", len(projects))
		}
	})

	t.Run("user cannot update other user's project", func(t *testing.T) {
		resp := ts.Request("PATCH", "/api/v1/dashboard/projects/"+project.ID, map[string]string{
			"name": "Hacked!",
		}, map[string]string{"Authorization": "Bearer " + auth2.Token})

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})

	t.Run("user cannot delete other user's project", func(t *testing.T) {
		resp := ts.Request("DELETE", "/api/v1/dashboard/projects/"+project.ID, nil, map[string]string{
			"Authorization": "Bearer " + auth2.Token,
		})

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.StatusCode)
		}
	})
}
