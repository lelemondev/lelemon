package handler_test

import (
	"net/http"
	"testing"
)

func TestIngest(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create user, get JWT, create project, get API key
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "ingest@example.com", "password": "SecurePass123", "name": "Ingest User",
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

func TestIngestHierarchy(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create user, get JWT, create project, get API key
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "hierarchy@example.com", "password": "SecurePass123", "name": "Hierarchy User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Hierarchy Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("agent span sets trace name", func(t *testing.T) {
		// Simulate SDK trace() + observe() pattern:
		// 1. Agent span (root) with trace name
		// 2. LLM span as child
		traceID := "test-trace-001"
		agentSpanID := "agent-span-001"
		llmSpanID := "llm-span-001"

		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":    traceID,
					"spanId":     agentSpanID,
					"spanType":   "agent",
					"name":       "sales-conversation", // This should become trace.Name
					"provider":   "agent",
					"model":      "sales-conversation",
					"input":      map[string]any{"message": "Hello"},
					"output":     "Hi there!",
					"durationMs": 1500,
					"status":     "success",
					"sessionId":  "session-123",
				},
				{
					"traceId":      traceID,
					"spanId":       llmSpanID,
					"parentSpanId": agentSpanID, // LLM is child of agent
					"spanType":     "llm",
					"name":         "bedrock-call",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"input":        []map[string]any{{"role": "user", "content": "Hello"}},
					"inputTokens":  50,
					"outputTokens": 100,
					"durationMs":   800,
					"status":       "success",
					"sessionId":    "session-123",
				},
			},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var result IngestResponse
		ParseJSON(t, resp, &result)
		if result.Processed != 2 {
			t.Errorf("expected processed 2, got %d", result.Processed)
		}

		// Verify the trace was created with correct name
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 for trace, got %d", traceResp.StatusCode)
		}

		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Check trace name is set from agent span
		if trace["Name"] == nil {
			t.Error("expected trace to have Name set from agent span")
		} else if trace["Name"].(string) != "sales-conversation" {
			t.Errorf("expected trace Name 'sales-conversation', got '%v'", trace["Name"])
		}

		// Check we have 2 spans
		spans, ok := trace["Spans"].([]any)
		if !ok {
			t.Fatal("expected Spans array in trace response")
		}
		if len(spans) != 2 {
			t.Errorf("expected 2 spans, got %d", len(spans))
		}

		// Find agent span and verify it has no parent
		var agentSpan, llmSpan map[string]any
		for _, s := range spans {
			span := s.(map[string]any)
			if span["Type"] == "agent" {
				agentSpan = span
			} else if span["Type"] == "llm" {
				llmSpan = span
			}
		}

		if agentSpan == nil {
			t.Fatal("agent span not found")
		}
		if llmSpan == nil {
			t.Fatal("llm span not found")
		}

		// Agent span should have no parent
		if agentSpan["ParentSpanID"] != nil {
			t.Errorf("agent span should have no parent, got %v", agentSpan["ParentSpanID"])
		}

		// Verify span IDs are preserved (not generated by server)
		actualAgentID := agentSpan["ID"].(string)
		if actualAgentID != agentSpanID {
			t.Errorf("agent span ID should be preserved as '%s', got '%s'", agentSpanID, actualAgentID)
		}

		actualLLMID := llmSpan["ID"].(string)
		if actualLLMID != llmSpanID {
			t.Errorf("llm span ID should be preserved as '%s', got '%s'", llmSpanID, actualLLMID)
		}

		// LLM span should have agent as parent (verify against ACTUAL span ID, not constant)
		if llmSpan["ParentSpanID"] == nil {
			t.Error("llm span should have parent")
		} else {
			llmParentID := llmSpan["ParentSpanID"].(string)
			// Check it matches the constant we sent
			if llmParentID != agentSpanID {
				t.Errorf("llm span parent should be '%s', got '%v'", agentSpanID, llmParentID)
			}
			// CRITICAL: Check the parent ID actually matches the agent span's real ID
			if llmParentID != actualAgentID {
				t.Errorf("llm span parent '%s' doesn't match actual agent span ID '%s' - parent-child relationship broken!", llmParentID, actualAgentID)
			}
		}
	})

	t.Run("trace name from metadata._traceName fallback", func(t *testing.T) {
		// When no agent span, trace name comes from metadata._traceName
		traceID := "test-trace-002"

		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "llm-only-001",
					"spanType":     "llm",
					"name":         "bedrock-call",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"inputTokens":  50,
					"outputTokens": 100,
					"durationMs":   800,
					"status":       "success",
					"metadata": map[string]any{
						"_traceName": "fallback-trace-name",
					},
				},
			},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Verify the trace was created with name from metadata
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		if trace["Name"] == nil {
			t.Error("expected trace to have Name set from metadata._traceName")
		} else if trace["Name"].(string) != "fallback-trace-name" {
			t.Errorf("expected trace Name 'fallback-trace-name', got '%v'", trace["Name"])
		}
	})
}

func TestTraces(t *testing.T) {
	ts := setupTestServer(t)

	// Setup
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "traces@example.com", "password": "SecurePass123", "name": "Traces User",
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
