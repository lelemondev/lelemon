package handler_test

import (
	"net/http"
	"testing"
)

func TestIngest(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create user, get JWT, create project, get API key
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "ingest@example.com", "password": "password123", "name": "Ingest User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Ingest Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("ingest single event", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"name":         "chat-completion",
				"input":        map[string]any{"prompt": "Hello"},
				"output":       map[string]any{"response": "Hi!"},
				"inputTokens":  10,
				"outputTokens": 5,
				"durationMs":   500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result IngestResponse
		ParseJSON(t, resp, &result)

		if !result.Success {
			t.Error("expected success to be true")
		}
		if result.Processed != 1 {
			t.Errorf("expected processed 1, got %d", result.Processed)
		}
	})

	t.Run("ingest batch events", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"spanType": "llm", "provider": "openai", "model": "gpt-4o",
					"inputTokens": 100, "outputTokens": 50, "status": "success",
				},
				{
					"spanType": "tool", "name": "search",
					"durationMs": 200, "status": "success",
				},
				{
					"spanType": "llm", "provider": "anthropic", "model": "claude-3-5-sonnet-20241022",
					"inputTokens": 500, "outputTokens": 200, "status": "success",
				},
			},
		}, apiKeyHeaders)

		var result IngestResponse
		ParseJSON(t, resp, &result)

		if result.Processed != 3 {
			t.Errorf("expected processed 3, got %d", result.Processed)
		}
	})

	t.Run("ingest with session groups events", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"spanType": "llm", "sessionId": "session-1", "status": "success"},
				{"spanType": "llm", "sessionId": "session-1", "status": "success"},
				{"spanType": "llm", "sessionId": "session-2", "status": "success"},
			},
		}, apiKeyHeaders)

		var result IngestResponse
		ParseJSON(t, resp, &result)

		if result.Processed != 3 {
			t.Errorf("expected processed 3, got %d", result.Processed)
		}
	})

	t.Run("ingest empty events", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{},
		}, apiKeyHeaders)

		var result IngestResponse
		ParseJSON(t, resp, &result)

		if !result.Success {
			t.Error("expected success for empty events")
		}
		if result.Processed != 0 {
			t.Errorf("expected processed 0, got %d", result.Processed)
		}
	})

	t.Run("ingest without auth fails", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{"spanType": "llm", "status": "success"}},
		}, nil)

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("ingest with invalid API key fails", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{"spanType": "llm", "status": "success"}},
		}, map[string]string{"Authorization": "Bearer le_invalid"})

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.StatusCode)
		}
	})
}

func TestTraces(t *testing.T) {
	ts := setupTestServer(t)

	// Setup
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "traces@example.com", "password": "password123", "name": "Traces User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Traces Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	// Ingest some events first
	ts.Request("POST", "/api/v1/ingest", map[string]any{
		"events": []map[string]any{
			{
				"spanType": "llm", "provider": "openai", "model": "gpt-4o",
				"inputTokens": 100, "outputTokens": 50, "durationMs": 500, "status": "success",
			},
		},
	}, apiKeyHeaders)

	t.Run("list traces", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/traces", nil, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, resp, &result)

		if result.Total != 1 {
			t.Errorf("expected 1 trace, got %d", result.Total)
		}
		if len(result.Data) != 1 {
			t.Errorf("expected 1 trace in data, got %d", len(result.Data))
		}

		trace := result.Data[0]
		if trace.TotalSpans != 1 {
			t.Errorf("expected 1 span, got %d", trace.TotalSpans)
		}
		if trace.TotalTokens != 150 {
			t.Errorf("expected 150 tokens, got %d", trace.TotalTokens)
		}
	})

	t.Run("get single trace", func(t *testing.T) {
		// First get the list to get the trace ID
		listResp := ts.Request("GET", "/api/v1/traces", nil, apiKeyHeaders)
		var traces TracesResponse
		ParseJSON(t, listResp, &traces)
		traceID := traces.Data[0].ID

		resp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("create trace manually", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/traces", map[string]any{
			"sessionId": "manual-session",
			"userId":    "user-123",
			"tags":      []string{"test", "manual"},
			"metadata":  map[string]any{"source": "test"},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}
	})

	t.Run("update trace status", func(t *testing.T) {
		// Create a trace
		createResp := ts.Request("POST", "/api/v1/traces", map[string]any{
			"sessionId": "update-test",
		}, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, createResp, &trace)
		traceID := trace["ID"].(string)

		// Update it
		resp := ts.Request("PATCH", "/api/v1/traces/"+traceID, map[string]any{
			"status": "completed",
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}
