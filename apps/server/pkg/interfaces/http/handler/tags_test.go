package handler_test

import (
	"net/http"
	"testing"
)

// TestTagsE2E tests the complete tags flow:
// 1. Ingest events with tags in agent span
// 2. Verify tags are stored in trace
// 3. Filter traces by tags
func TestTagsE2E(t *testing.T) {
	ts := setupTestServer(t)

	// Setup: create user, get JWT, create project, get API key
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "tags@example.com", "password": "SecurePass123", "name": "Tags User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Tags Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}
	jwtHeaders := map[string]string{"Authorization": "Bearer " + auth.Token}

	t.Run("agent span tags are saved to trace", func(t *testing.T) {
		// Simulate SDK trace() with tags - agent span comes with LLM spans
		traceID := "tags-test-trace-001"
		agentSpanID := "agent-with-tags-001"
		llmSpanID := "llm-span-001"

		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				// LLM span (typically sent first by SDK)
				{
					"traceId":      traceID,
					"spanId":       llmSpanID,
					"parentSpanId": agentSpanID,
					"spanType":     "llm",
					"name":         "bedrock-call",
					"provider":     "bedrock",
					"model":        "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
					"inputTokens":  100,
					"outputTokens": 50,
					"durationMs":   1000,
					"status":       "success",
					"sessionId":    "session-tags-001",
				},
				// Agent span (root span with tags - sent last by SDK)
				{
					"traceId":    traceID,
					"spanId":     agentSpanID,
					"spanType":   "agent",
					"name":       "bedrock-playground-agent",
					"provider":   "agent",
					"model":      "bedrock-playground-agent",
					"input":      map[string]any{"message": "Hello"},
					"output":     "Hi there!",
					"durationMs": 1500,
					"status":     "success",
					"sessionId":  "session-tags-001",
					"tags":       []string{"provider:bedrock", "type:agent", "env:playground"},
				},
			},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		// Verify the trace was created with tags
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200 for trace, got %d", traceResp.StatusCode)
		}

		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Check trace has tags from agent span
		tags, ok := trace["Tags"].([]any)
		if !ok {
			t.Fatalf("expected Tags array in trace, got %T", trace["Tags"])
		}

		if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d: %v", len(tags), tags)
		}

		// Verify specific tags
		tagStrings := make([]string, len(tags))
		for i, tag := range tags {
			tagStrings[i] = tag.(string)
		}

		expectedTags := []string{"provider:bedrock", "type:agent", "env:playground"}
		for _, expected := range expectedTags {
			found := false
			for _, tag := range tagStrings {
				if tag == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected tag '%s' not found in %v", expected, tagStrings)
			}
		}

		// Verify trace name is from agent span
		if trace["Name"] == nil || trace["Name"].(string) != "bedrock-playground-agent" {
			t.Errorf("expected trace Name 'bedrock-playground-agent', got '%v'", trace["Name"])
		}
	})

	t.Run("filter traces by single tag", func(t *testing.T) {
		// Create another trace with different tags
		traceID2 := "tags-test-trace-002"
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":    traceID2,
					"spanId":     "agent-002",
					"spanType":   "agent",
					"name":       "openai-playground-agent",
					"provider":   "agent",
					"model":      "openai-playground-agent",
					"durationMs": 500,
					"status":     "success",
					"tags":       []string{"provider:openai", "type:agent", "env:playground"},
				},
			},
		}, apiKeyHeaders)

		// Filter by provider:bedrock - should get only first trace
		resp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces?tags=provider:bedrock", nil, jwtHeaders)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, resp, &result)

		if result.Total != 1 {
			t.Errorf("expected 1 trace with provider:bedrock tag, got %d", result.Total)
		}

		if len(result.Data) > 0 {
			tags := result.Data[0].Tags
			if len(tags) > 0 {
				foundBedrock := false
				for _, tag := range tags {
					if tag.(string) == "provider:bedrock" {
						foundBedrock = true
						break
					}
				}
				if !foundBedrock {
					t.Error("filtered trace should have provider:bedrock tag")
				}
			}
		}
	})

	t.Run("filter traces by multiple tags (OR logic)", func(t *testing.T) {
		// Filter by both providers - should get both traces
		resp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces?tags=provider:bedrock&tags=provider:openai", nil, jwtHeaders)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, resp, &result)

		if result.Total < 2 {
			t.Errorf("expected at least 2 traces with OR filter, got %d", result.Total)
		}
	})

	t.Run("filter traces by common tag", func(t *testing.T) {
		// Filter by type:agent - should get all agent traces
		resp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces?tags=type:agent", nil, jwtHeaders)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, resp, &result)

		if result.Total < 2 {
			t.Errorf("expected at least 2 traces with type:agent tag, got %d", result.Total)
		}
	})

	t.Run("filter traces by non-existent tag returns empty", func(t *testing.T) {
		resp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces?tags=nonexistent:tag", nil, jwtHeaders)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, resp, &result)

		if result.Total != 0 {
			t.Errorf("expected 0 traces with non-existent tag, got %d", result.Total)
		}
	})

	t.Run("tags from first event as fallback", func(t *testing.T) {
		// When no agent span, tags should come from first event
		traceID3 := "tags-test-trace-003"
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID3,
					"spanId":       "llm-only-001",
					"spanType":     "llm",
					"name":         "simple-llm-call",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  50,
					"outputTokens": 25,
					"durationMs":   300,
					"status":       "success",
					"tags":         []string{"fallback:test"},
				},
			},
		}, apiKeyHeaders)

		// Verify tags were saved
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID3, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		tags, ok := trace["Tags"].([]any)
		if !ok || len(tags) == 0 {
			t.Error("expected tags from first event as fallback")
		} else if tags[0].(string) != "fallback:test" {
			t.Errorf("expected fallback:test tag, got %v", tags)
		}
	})
}

// TestTagsWithDateRange tests combining tag filters with date range
func TestTagsWithDateRange(t *testing.T) {
	ts := setupTestServer(t)

	// Setup
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "tags-date@example.com", "password": "SecurePass123", "name": "Tags Date User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Tags Date Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}
	jwtHeaders := map[string]string{"Authorization": "Bearer " + auth.Token}

	// Create trace with tags
	ts.Request("POST", "/api/v1/ingest", map[string]any{
		"events": []map[string]any{
			{
				"traceId":    "date-test-001",
				"spanId":     "agent-date-001",
				"spanType":   "agent",
				"name":       "date-test-agent",
				"provider":   "agent",
				"durationMs": 100,
				"status":     "success",
				"tags":       []string{"test:date-range"},
			},
		},
	}, apiKeyHeaders)

	t.Run("filter by tags and date range combined", func(t *testing.T) {
		// Use a wide date range that includes today
		resp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces?tags=test:date-range&from=2020-01-01T00:00:00Z", nil, jwtHeaders)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, resp, &result)

		if result.Total != 1 {
			t.Errorf("expected 1 trace with combined filters, got %d", result.Total)
		}
	})
}
