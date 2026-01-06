package handler_test

import (
	"net/http"
	"testing"
)

func TestAnalytics(t *testing.T) {
	ts := setupTestServer(t)

	// Setup
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "analytics@example.com", "password": "password123", "name": "Analytics User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Analytics Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	// Ingest varied events for analytics testing
	ts.Request("POST", "/api/v1/ingest", map[string]any{
		"events": []map[string]any{
			{
				"spanType": "llm", "provider": "openai", "model": "gpt-4o",
				"inputTokens": 100, "outputTokens": 50, "durationMs": 500, "status": "success",
			},
			{
				"spanType": "llm", "provider": "openai", "model": "gpt-4o",
				"inputTokens": 200, "outputTokens": 100, "durationMs": 800, "status": "success",
			},
			{
				"spanType": "llm", "provider": "anthropic", "model": "claude-3-5-sonnet-20241022",
				"inputTokens": 300, "outputTokens": 150, "durationMs": 600, "status": "error",
			},
		},
	}, apiKeyHeaders)

	t.Run("get analytics summary", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/analytics/summary", nil, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var stats StatsResponse
		ParseJSON(t, resp, &stats)

		// We created 3 events, but they're grouped by session
		// Empty sessionId means they go into different traces
		if stats.TotalSpans != 3 {
			t.Errorf("expected 3 spans, got %d", stats.TotalSpans)
		}

		// Total tokens: 100+50 + 200+100 + 300+150 = 900
		if stats.TotalTokens != 900 {
			t.Errorf("expected 900 tokens, got %d", stats.TotalTokens)
		}

		// Cost should be calculated
		if stats.TotalCostUSD == 0 {
			t.Error("expected non-zero cost")
		}
	})

	t.Run("get usage time series", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/analytics/usage", nil, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		ParseJSON(t, resp, &result)

		data, ok := result["data"].([]any)
		if !ok {
			t.Error("expected data array in response")
		}

		// Should have at least one data point for today
		if len(data) == 0 {
			t.Error("expected at least one data point")
		}
	})

	t.Run("analytics with date range", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/analytics/summary?from=2020-01-01T00:00:00Z&to=2020-12-31T23:59:59Z", nil, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var stats StatsResponse
		ParseJSON(t, resp, &stats)

		// No data in 2020, should be zeros
		if stats.TotalTraces != 0 {
			t.Errorf("expected 0 traces for 2020, got %d", stats.TotalTraces)
		}
	})
}

func TestCostCalculation(t *testing.T) {
	ts := setupTestServer(t)

	// Setup
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "cost@example.com", "password": "password123", "name": "Cost User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Cost Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("gpt-4o cost calculation", func(t *testing.T) {
		// gpt-4o: $5/1M input, $15/1M output
		// 1000 input tokens = $0.005
		// 500 output tokens = $0.0075
		// Total = $0.0125
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"spanType": "llm", "provider": "openai", "model": "gpt-4o",
				"inputTokens": 1000, "outputTokens": 500, "status": "success",
			}},
		}, apiKeyHeaders)

		resp := ts.Request("GET", "/api/v1/analytics/summary", nil, apiKeyHeaders)
		var stats StatsResponse
		ParseJSON(t, resp, &stats)

		expectedCost := 0.0125
		tolerance := 0.0001
		if stats.TotalCostUSD < expectedCost-tolerance || stats.TotalCostUSD > expectedCost+tolerance {
			t.Errorf("expected cost ~$0.0125, got $%f", stats.TotalCostUSD)
		}
	})
}

func TestHealth(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("health check", func(t *testing.T) {
		resp := ts.Request("GET", "/health", nil, nil)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		ParseJSON(t, resp, &result)

		if result["status"] != "ok" {
			t.Errorf("expected status 'ok', got '%v'", result["status"])
		}

		checks, ok := result["checks"].(map[string]any)
		if !ok {
			t.Fatal("expected checks object in response")
		}
		db, ok := checks["database"].(map[string]any)
		if !ok {
			t.Fatal("expected database object in checks")
		}
		if db["status"] != "ok" {
			t.Errorf("expected database status 'ok', got '%v'", db["status"])
		}
	})

	t.Run("health verbose", func(t *testing.T) {
		resp := ts.Request("GET", "/health?verbose=true", nil, nil)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result map[string]any
		ParseJSON(t, resp, &result)

		system, ok := result["system"].(map[string]any)
		if !ok {
			t.Fatal("expected system object in verbose response")
		}
		if system["go_version"] == "" {
			t.Error("expected go_version in system info")
		}
	})

	t.Run("liveness probe", func(t *testing.T) {
		resp := ts.Request("GET", "/health/live", nil, nil)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("readiness probe", func(t *testing.T) {
		resp := ts.Request("GET", "/health/ready", nil, nil)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}
