package handler_test

import (
	"net/http"
	"testing"
)

// =============================================================================
// TOKEN TESTS
// Verify all token fields are correctly preserved and aggregated
// =============================================================================

// TestTokensBasic verifies basic input/output tokens are preserved
func TestTokensBasic(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "tokens@example.com", "password": "SecurePass123", "name": "Tokens User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Tokens Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("inputTokens and outputTokens are preserved", func(t *testing.T) {
		traceID := "tokens-basic-001"
		spanID := "tokens-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"inputTokens":  100,
				"outputTokens": 50,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		inputTokens := int(span["InputTokens"].(float64))
		outputTokens := int(span["OutputTokens"].(float64))

		if inputTokens != 100 {
			t.Errorf("inputTokens: expected 100, got %d", inputTokens)
		}
		if outputTokens != 50 {
			t.Errorf("outputTokens: expected 50, got %d", outputTokens)
		}
	})

	t.Run("zero tokens are preserved (not null)", func(t *testing.T) {
		traceID := "tokens-basic-002"
		spanID := "tokens-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"inputTokens":  0,
				"outputTokens": 0,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Zero should be preserved, not nil
		if span["InputTokens"] == nil {
			t.Error("inputTokens should be 0, not nil")
		} else {
			inputTokens := int(span["InputTokens"].(float64))
			if inputTokens != 0 {
				t.Errorf("inputTokens: expected 0, got %d", inputTokens)
			}
		}
	})

	t.Run("large token counts are preserved", func(t *testing.T) {
		traceID := "tokens-basic-003"
		spanID := "tokens-span-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "llm",
				"provider":     "anthropic",
				"model":        "claude-3-5-sonnet-20241022",
				"inputTokens":  128000, // Max context
				"outputTokens": 8192,   // Max output
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		inputTokens := int(span["InputTokens"].(float64))
		outputTokens := int(span["OutputTokens"].(float64))

		if inputTokens != 128000 {
			t.Errorf("inputTokens: expected 128000, got %d", inputTokens)
		}
		if outputTokens != 8192 {
			t.Errorf("outputTokens: expected 8192, got %d", outputTokens)
		}
	})
}

// TestTokensCache verifies cache tokens (Anthropic/Bedrock feature)
func TestTokensCache(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "cache@example.com", "password": "SecurePass123", "name": "Cache User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Cache Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("cacheReadTokens is preserved", func(t *testing.T) {
		traceID := "cache-001"
		spanID := "cache-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":         traceID,
				"spanId":          spanID,
				"spanType":        "llm",
				"provider":        "anthropic",
				"model":           "claude-3-5-sonnet-20241022",
				"inputTokens":     100,
				"outputTokens":    50,
				"cacheReadTokens": 500,
				"status":          "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["CacheReadTokens"] == nil {
			t.Error("cacheReadTokens should not be nil")
		} else {
			cacheRead := int(span["CacheReadTokens"].(float64))
			if cacheRead != 500 {
				t.Errorf("cacheReadTokens: expected 500, got %d", cacheRead)
			}
		}
	})

	t.Run("cacheWriteTokens is preserved", func(t *testing.T) {
		traceID := "cache-002"
		spanID := "cache-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":          traceID,
				"spanId":           spanID,
				"spanType":         "llm",
				"provider":         "anthropic",
				"model":            "claude-3-5-sonnet-20241022",
				"inputTokens":      100,
				"outputTokens":     50,
				"cacheWriteTokens": 200,
				"status":           "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["CacheWriteTokens"] == nil {
			t.Error("cacheWriteTokens should not be nil")
		} else {
			cacheWrite := int(span["CacheWriteTokens"].(float64))
			if cacheWrite != 200 {
				t.Errorf("cacheWriteTokens: expected 200, got %d", cacheWrite)
			}
		}
	})

	t.Run("both cache tokens together", func(t *testing.T) {
		traceID := "cache-003"
		spanID := "cache-span-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":          traceID,
				"spanId":           spanID,
				"spanType":         "llm",
				"provider":         "bedrock",
				"model":            "anthropic.claude-3-haiku-20240307-v1:0",
				"inputTokens":      1000,
				"outputTokens":     500,
				"cacheReadTokens":  5000,
				"cacheWriteTokens": 1000,
				"status":           "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		cacheRead := int(span["CacheReadTokens"].(float64))
		cacheWrite := int(span["CacheWriteTokens"].(float64))

		if cacheRead != 5000 {
			t.Errorf("cacheReadTokens: expected 5000, got %d", cacheRead)
		}
		if cacheWrite != 1000 {
			t.Errorf("cacheWriteTokens: expected 1000, got %d", cacheWrite)
		}
	})
}

// TestTokensReasoning verifies reasoning tokens (o1, Claude thinking)
func TestTokensReasoning(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "reasoning@example.com", "password": "SecurePass123", "name": "Reasoning User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Reasoning Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("reasoningTokens is preserved", func(t *testing.T) {
		traceID := "reasoning-001"
		spanID := "reasoning-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":         traceID,
				"spanId":          spanID,
				"spanType":        "llm",
				"provider":        "openai",
				"model":           "o1-preview",
				"inputTokens":     500,
				"outputTokens":    200,
				"reasoningTokens": 10000,
				"status":          "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["ReasoningTokens"] == nil {
			t.Error("reasoningTokens should not be nil")
		} else {
			reasoning := int(span["ReasoningTokens"].(float64))
			if reasoning != 10000 {
				t.Errorf("reasoningTokens: expected 10000, got %d", reasoning)
			}
		}
	})
}

// TestTokensAggregation verifies TotalTokens is calculated correctly
func TestTokensAggregation(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "aggregation@example.com", "password": "SecurePass123", "name": "Aggregation User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Aggregation Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("TotalTokens sums all spans", func(t *testing.T) {
		traceID := "agg-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "agg-span-001",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  100,
					"outputTokens": 50,
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       "agg-span-002",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  200,
					"outputTokens": 100,
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       "agg-span-003",
					"spanType":     "llm",
					"provider":     "anthropic",
					"model":        "claude-3-haiku-20240307",
					"inputTokens":  300,
					"outputTokens": 150,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Expected: (100+50) + (200+100) + (300+150) = 900
		totalTokens := int(trace["TotalTokens"].(float64))
		if totalTokens != 900 {
			t.Errorf("TotalTokens: expected 900, got %d", totalTokens)
		}
	})

	t.Run("TotalTokens excludes non-LLM spans", func(t *testing.T) {
		traceID := "agg-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "agg-llm-001",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  100,
					"outputTokens": 50,
					"status":       "success",
				},
				{
					"traceId":    traceID,
					"spanId":     "agg-tool-001",
					"spanType":   "tool",
					"name":       "search",
					"durationMs": 200,
					"status":     "success",
					// No tokens for tool
				},
				{
					"traceId":      traceID,
					"spanId":       "agg-llm-002",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  200,
					"outputTokens": 100,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Expected: (100+50) + (200+100) = 450 (tool has no tokens)
		totalTokens := int(trace["TotalTokens"].(float64))
		if totalTokens != 450 {
			t.Errorf("TotalTokens: expected 450 (excluding tool), got %d", totalTokens)
		}
	})

	t.Run("TotalSpans counts all spans", func(t *testing.T) {
		traceID := "agg-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": "span-1", "spanType": "agent", "name": "agent", "status": "success"},
				{"traceId": traceID, "spanId": "span-2", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "span-3", "spanType": "tool", "name": "tool", "status": "success"},
				{"traceId": traceID, "spanId": "span-4", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "span-5", "spanType": "retrieval", "name": "retrieval", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		totalSpans := int(trace["TotalSpans"].(float64))
		if totalSpans != 5 {
			t.Errorf("TotalSpans: expected 5, got %d", totalSpans)
		}
	})
}

// TestTokensFromList verifies token aggregation in trace list endpoint
func TestTokensFromList(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "listtest@example.com", "password": "SecurePass123", "name": "List User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "List Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}
	jwtHeaders := map[string]string{"Authorization": "Bearer " + auth.Token}

	t.Run("trace list shows correct totals", func(t *testing.T) {
		traceID := "list-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "list-span-001",
					"spanType":     "llm",
					"inputTokens":  500,
					"outputTokens": 250,
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       "list-span-002",
					"spanType":     "llm",
					"inputTokens":  500,
					"outputTokens": 250,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		// Use dashboard endpoint which returns list
		listResp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces", nil, jwtHeaders)
		if listResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", listResp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, listResp, &result)

		if result.Total < 1 {
			t.Fatal("expected at least 1 trace")
		}

		// Find our trace
		var found *TraceData
		for i := range result.Data {
			if result.Data[i].ID == traceID {
				found = &result.Data[i]
				break
			}
		}

		if found == nil {
			t.Fatal("trace not found in list")
		}

		// Expected: (500+250) + (500+250) = 1500
		if found.TotalTokens != 1500 {
			t.Errorf("TotalTokens in list: expected 1500, got %d", found.TotalTokens)
		}
		if found.TotalSpans != 2 {
			t.Errorf("TotalSpans in list: expected 2, got %d", found.TotalSpans)
		}
	})
}
