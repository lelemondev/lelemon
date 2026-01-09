package handler_test

import (
	"net/http"
	"testing"
)

// FeaturesResponse for parsing /api/v1/features
type FeaturesResponse struct {
	Edition  string          `json:"edition"`
	Features map[string]bool `json:"features"`
}

func TestFeatures(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("returns features configuration", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/features", nil, nil)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var features FeaturesResponse
		ParseJSON(t, resp, &features)

		// Verify edition is set
		if features.Edition == "" {
			t.Error("expected edition to be set")
		}

		// Verify features map exists
		if features.Features == nil {
			t.Error("expected features map to exist")
		}
	})

	t.Run("community edition has EE features disabled", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/features", nil, nil)

		var features FeaturesResponse
		ParseJSON(t, resp, &features)

		// In test setup (community), EE features should be disabled
		if features.Edition != "community" {
			t.Skipf("skipping community test, edition is %s", features.Edition)
		}

		expectedDisabled := []string{"organizations", "rbac", "billing", "sso"}
		for _, feature := range expectedDisabled {
			if enabled, exists := features.Features[feature]; exists && enabled {
				t.Errorf("expected %s to be disabled in community edition", feature)
			}
		}
	})

	t.Run("no authentication required", func(t *testing.T) {
		// Features endpoint should work without auth
		resp := ts.Request("GET", "/api/v1/features", nil, nil)

		if resp.StatusCode == http.StatusUnauthorized {
			t.Error("features endpoint should not require authentication")
		}
	})

	t.Run("returns JSON content type", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/features", nil, nil)
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})
}
