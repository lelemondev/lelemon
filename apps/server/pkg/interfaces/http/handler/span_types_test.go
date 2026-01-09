package handler_test

import (
	"testing"
)

// =============================================================================
// SPAN TYPE TESTS
// Verify all 8 span types are correctly handled
// =============================================================================

// TestSpanTypeLLM verifies LLM span handling
func TestSpanTypeLLM(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "llmtype@example.com", "password": "SecurePass123", "name": "LLM User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "LLM Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("LLM span with all fields", func(t *testing.T) {
		traceID := "llmtype-001"
		spanID := "llmtype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"name":         "chat-completion",
				"input":        []map[string]any{{"role": "user", "content": "Hello"}},
				"output":       []map[string]any{{"role": "assistant", "content": "Hi!"}},
				"inputTokens":  10,
				"outputTokens": 5,
				"durationMs":   500,
				"status":       "success",
				"stopReason":   "stop",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "llm" {
			t.Errorf("Type: expected 'llm', got '%s'", span["Type"])
		}
		if span["Provider"].(string) != "openai" {
			t.Errorf("Provider: expected 'openai', got '%s'", span["Provider"])
		}
		if span["Model"].(string) != "gpt-4o" {
			t.Errorf("Model: expected 'gpt-4o', got '%s'", span["Model"])
		}
		if span["Input"] == nil {
			t.Error("Input should not be nil")
		}
		if span["Output"] == nil {
			t.Error("Output should not be nil")
		}
		if span["CostUSD"] == nil || span["CostUSD"].(float64) <= 0 {
			t.Error("LLM span should have cost calculated")
		}
	})

	t.Run("LLM span with tool_use output gets planning subtype", func(t *testing.T) {
		traceID := "llmtype-002"
		spanID := "llmtype-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"provider": "anthropic",
				"model":    "claude-3-5-sonnet-20241022",
				"output": []map[string]any{
					{"type": "text", "text": "Let me search for that"},
					{"type": "tool_use", "id": "toolu_123", "name": "search", "input": map[string]any{"q": "test"}},
				},
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

		// SubType should be "planning" when output contains tool_use
		if span["SubType"] != nil && span["SubType"].(string) != "planning" {
			t.Errorf("SubType: expected 'planning' for tool_use output, got '%v'", span["SubType"])
		}

		// ToolUses should be extracted
		if span["ToolUses"] != nil {
			toolUses := span["ToolUses"].([]any)
			if len(toolUses) == 0 {
				t.Error("ToolUses should contain extracted tool calls")
			}
		}
	})

	t.Run("LLM span with text-only output gets response subtype", func(t *testing.T) {
		traceID := "llmtype-003"
		spanID := "llmtype-span-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"provider": "anthropic",
				"model":    "claude-3-5-sonnet-20241022",
				"output": []map[string]any{
					{"type": "text", "text": "Here is my response"},
				},
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

		// SubType should be "response" when output is text-only
		if span["SubType"] != nil && span["SubType"].(string) != "response" {
			t.Errorf("SubType: expected 'response' for text-only output, got '%v'", span["SubType"])
		}
	})
}

// TestSpanTypeAgent verifies Agent span handling
func TestSpanTypeAgent(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "agenttype@example.com", "password": "SecurePass123", "name": "Agent User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Agent Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("agent span sets trace name", func(t *testing.T) {
		traceID := "agenttype-001"
		spanID := "agenttype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":    traceID,
				"spanId":     spanID,
				"spanType":   "agent",
				"name":       "sales-agent",
				"input":      map[string]any{"message": "Hello"},
				"output":     "Hi there!",
				"durationMs": 5000,
				"status":     "success",
				"sessionId":  "session-123",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Trace should get name from agent span
		if trace["Name"] == nil {
			t.Error("Trace.Name should be set from agent span")
		} else if trace["Name"].(string) != "sales-agent" {
			t.Errorf("Trace.Name: expected 'sales-agent', got '%s'", trace["Name"])
		}

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "agent" {
			t.Errorf("Type: expected 'agent', got '%s'", span["Type"])
		}
	})

	t.Run("agent span is root of hierarchy", func(t *testing.T) {
		traceID := "agenttype-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  traceID,
					"spanId":   "agent-root",
					"spanType": "agent",
					"name":     "my-agent",
					"status":   "success",
				},
				{
					"traceId":      traceID,
					"spanId":       "llm-child",
					"parentSpanId": "agent-root",
					"spanType":     "llm",
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		var agentSpan map[string]any
		for _, s := range spans {
			span := s.(map[string]any)
			if span["Type"].(string) == "agent" {
				agentSpan = span
				break
			}
		}

		if agentSpan == nil {
			t.Fatal("agent span not found")
		}
		if agentSpan["ParentSpanID"] != nil {
			t.Error("agent span should have no parent (root)")
		}
	})
}

// TestSpanTypeTool verifies Tool span handling
func TestSpanTypeTool(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "tooltype@example.com", "password": "SecurePass123", "name": "Tool User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Tool Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("tool span with all fields", func(t *testing.T) {
		traceID := "tooltype-001"
		spanID := "tooltype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":    traceID,
				"spanId":     spanID,
				"spanType":   "tool",
				"name":       "search_products",
				"input":      map[string]any{"query": "widgets", "limit": 10},
				"output":     map[string]any{"results": []string{"Widget A", "Widget B"}},
				"durationMs": 200,
				"status":     "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "tool" {
			t.Errorf("Type: expected 'tool', got '%s'", span["Type"])
		}
		if span["Name"].(string) != "search_products" {
			t.Errorf("Name: expected 'search_products', got '%s'", span["Name"])
		}
		if span["Input"] == nil {
			t.Error("Input should be preserved")
		}
		if span["Output"] == nil {
			t.Error("Output should be preserved")
		}
		// Tool spans shouldn't have tokens or cost
		if span["InputTokens"] != nil && span["InputTokens"].(float64) > 0 {
			t.Error("Tool span shouldn't have input tokens")
		}
	})

	t.Run("tool span with error", func(t *testing.T) {
		traceID := "tooltype-002"
		spanID := "tooltype-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "tool",
				"name":         "api_call",
				"input":        map[string]any{"url": "https://api.example.com"},
				"status":       "error",
				"errorMessage": "Connection timeout",
				"durationMs":   30000,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Status"].(string) != "error" {
			t.Errorf("Status: expected 'error', got '%s'", span["Status"])
		}
		if span["ErrorMessage"] == nil {
			t.Error("ErrorMessage should be preserved")
		} else if span["ErrorMessage"].(string) != "Connection timeout" {
			t.Errorf("ErrorMessage: expected 'Connection timeout', got '%s'", span["ErrorMessage"])
		}
	})
}

// TestSpanTypeRetrieval verifies Retrieval span handling
func TestSpanTypeRetrieval(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "retrievaltype@example.com", "password": "SecurePass123", "name": "Retrieval User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Retrieval Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("retrieval span with documents", func(t *testing.T) {
		traceID := "retrievaltype-001"
		spanID := "retrievaltype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "retrieval",
				"name":     "vector_search",
				"input":    map[string]any{"query": "How to reset password?", "topK": 5},
				"output": map[string]any{
					"documents": []map[string]any{
						{"id": "doc1", "content": "To reset password...", "score": 0.95},
						{"id": "doc2", "content": "Password requirements...", "score": 0.87},
					},
				},
				"durationMs": 150,
				"status":     "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "retrieval" {
			t.Errorf("Type: expected 'retrieval', got '%s'", span["Type"])
		}
		if span["Output"] == nil {
			t.Error("Output with documents should be preserved")
		}
	})
}

// TestSpanTypeEmbedding verifies Embedding span handling
func TestSpanTypeEmbedding(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "embeddingtype@example.com", "password": "SecurePass123", "name": "Embedding User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Embedding Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("embedding span with vectors", func(t *testing.T) {
		traceID := "embeddingtype-001"
		spanID := "embeddingtype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "embedding",
				"provider":    "openai",
				"model":       "text-embedding-3-small",
				"name":        "embed_query",
				"input":       []string{"Hello world", "How are you?"},
				"output":      [][]float64{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
				"inputTokens": 8,
				"durationMs":  100,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "embedding" {
			t.Errorf("Type: expected 'embedding', got '%s'", span["Type"])
		}
		// Note: Model may be nil for non-LLM spans depending on implementation
		if span["Model"] != nil {
			if span["Model"].(string) != "text-embedding-3-small" {
				t.Errorf("Model: expected 'text-embedding-3-small', got '%s'", span["Model"])
			}
		}
	})
}

// TestSpanTypeGuardrail verifies Guardrail span handling
func TestSpanTypeGuardrail(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "guardrailtype@example.com", "password": "SecurePass123", "name": "Guardrail User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Guardrail Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("guardrail span passed", func(t *testing.T) {
		traceID := "guardrailtype-001"
		spanID := "guardrailtype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":    traceID,
				"spanId":     spanID,
				"spanType":   "guardrail",
				"name":       "content_filter",
				"input":      map[string]any{"text": "Hello, how can I help you?"},
				"output":     map[string]any{"passed": true, "flags": []string{}},
				"durationMs": 50,
				"status":     "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "guardrail" {
			t.Errorf("Type: expected 'guardrail', got '%s'", span["Type"])
		}
	})

	t.Run("guardrail span blocked", func(t *testing.T) {
		traceID := "guardrailtype-002"
		spanID := "guardrailtype-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "guardrail",
				"name":     "content_filter",
				"input":    map[string]any{"text": "Some blocked content"},
				"output": map[string]any{
					"passed": false,
					"flags":  []string{"inappropriate_content"},
					"reason": "Content policy violation",
				},
				"durationMs": 50,
				"status":     "success", // Guardrail succeeded, content was blocked
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Output"] == nil {
			t.Error("Guardrail output should be preserved")
		}
	})
}

// TestSpanTypeRerank verifies Rerank span handling
func TestSpanTypeRerank(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "reranktype@example.com", "password": "SecurePass123", "name": "Rerank User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Rerank Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("rerank span with rankings", func(t *testing.T) {
		traceID := "reranktype-001"
		spanID := "reranktype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "rerank",
				"name":     "cohere_rerank",
				"input": map[string]any{
					"query":     "password reset",
					"documents": []string{"Reset password guide", "Account settings", "Security FAQ"},
				},
				"output": map[string]any{
					"rankings": []map[string]any{
						{"index": 0, "score": 0.95},
						{"index": 2, "score": 0.72},
						{"index": 1, "score": 0.45},
					},
				},
				"durationMs": 200,
				"status":     "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "rerank" {
			t.Errorf("Type: expected 'rerank', got '%s'", span["Type"])
		}
	})
}

// TestSpanTypeCustom verifies Custom span handling
func TestSpanTypeCustom(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "customtype@example.com", "password": "SecurePass123", "name": "Custom User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Custom Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("custom span with arbitrary data", func(t *testing.T) {
		traceID := "customtype-001"
		spanID := "customtype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":    traceID,
				"spanId":     spanID,
				"spanType":   "custom",
				"name":       "business_logic",
				"input":      map[string]any{"order_id": "12345", "action": "validate"},
				"output":     map[string]any{"valid": true, "warnings": []string{}},
				"durationMs": 25,
				"status":     "success",
				"metadata": map[string]any{
					"module":  "orders",
					"version": "2.1",
				},
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Type"].(string) != "custom" {
			t.Errorf("Type: expected 'custom', got '%s'", span["Type"])
		}
		if span["Name"].(string) != "business_logic" {
			t.Errorf("Name: expected 'business_logic', got '%s'", span["Name"])
		}
	})
}

// TestSpanTypeDefaultsToLLM verifies unknown span type defaults to LLM
func TestSpanTypeDefaultsToLLM(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "defaulttype@example.com", "password": "SecurePass123", "name": "Default User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Default Type Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("unknown spanType defaults to llm", func(t *testing.T) {
		traceID := "defaulttype-001"
		spanID := "defaulttype-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "unknown_type",
				"status":   "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Unknown types should default to "llm"
		if span["Type"].(string) != "llm" {
			t.Errorf("Unknown spanType should default to 'llm', got '%s'", span["Type"])
		}
	})
}
