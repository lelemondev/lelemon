package handler_test

import (
	"net/http"
	"testing"
)

// =============================================================================
// CONTRACT TESTS
// These tests verify that the server correctly preserves all fields from the SDK
// =============================================================================

// TestContractSpanIDsPreserved verifies that SDK-provided IDs are preserved
// This is critical for parent-child relationships to work
func TestContractSpanIDsPreserved(t *testing.T) {
	ts := setupTestServer(t)

	// Setup auth and project
	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "contract@example.com", "password": "SecurePass123", "name": "Contract User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Contract Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("spanId is preserved exactly as sent", func(t *testing.T) {
		traceID := "contract-trace-001"
		spanID := "contract-span-001"

		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"provider": "openai",
				"model":    "gpt-4o",
				"status":   "success",
			}},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		// Query the trace and verify span ID
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 1 {
			t.Fatalf("expected 1 span, got %d", len(spans))
		}

		span := spans[0].(map[string]any)
		actualID := span["ID"].(string)

		if actualID != spanID {
			t.Errorf("CRITICAL: spanId not preserved! sent '%s', got '%s'", spanID, actualID)
		}
	})

	t.Run("traceId is preserved exactly as sent", func(t *testing.T) {
		traceID := "contract-trace-002"
		spanID := "contract-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
			}},
		}, apiKeyHeaders)

		// Verify we can fetch by exact traceId
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Errorf("CRITICAL: traceId not preserved! cannot fetch trace by ID '%s'", traceID)
		}

		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		if trace["ID"].(string) != traceID {
			t.Errorf("CRITICAL: traceId mismatch! sent '%s', got '%s'", traceID, trace["ID"])
		}
	})

	t.Run("parentSpanId is preserved exactly as sent", func(t *testing.T) {
		traceID := "contract-trace-003"
		parentID := "contract-parent-003"
		childID := "contract-child-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  traceID,
					"spanId":   parentID,
					"spanType": "agent",
					"name":     "parent-span",
					"status":   "success",
				},
				{
					"traceId":      traceID,
					"spanId":       childID,
					"parentSpanId": parentID,
					"spanType":     "llm",
					"name":         "child-span",
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 2 {
			t.Fatalf("expected 2 spans, got %d", len(spans))
		}

		var childSpan map[string]any
		for _, s := range spans {
			span := s.(map[string]any)
			if span["ID"].(string) == childID {
				childSpan = span
				break
			}
		}

		if childSpan == nil {
			t.Fatal("child span not found by ID")
		}

		if childSpan["ParentSpanID"] == nil {
			t.Error("CRITICAL: parentSpanId not preserved! got nil")
		} else if childSpan["ParentSpanID"].(string) != parentID {
			t.Errorf("CRITICAL: parentSpanId mismatch! sent '%s', got '%s'",
				parentID, childSpan["ParentSpanID"])
		}
	})
}

// =============================================================================
// RELATIONSHIP INTEGRITY TESTS
// These tests verify parent-child relationships are valid (no orphans)
// =============================================================================

// TestNoOrphanSpans verifies that all parentSpanIds point to existing spans
func TestNoOrphanSpans(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "orphan@example.com", "password": "SecurePass123", "name": "Orphan User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Orphan Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("all parent references resolve to existing spans", func(t *testing.T) {
		traceID := "orphan-trace-001"
		rootID := "orphan-root-001"
		child1ID := "orphan-child-001"
		child2ID := "orphan-child-002"
		grandchildID := "orphan-grandchild-001"

		// Create a multi-level hierarchy
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  traceID,
					"spanId":   rootID,
					"spanType": "agent",
					"name":     "root",
					"status":   "success",
				},
				{
					"traceId":      traceID,
					"spanId":       child1ID,
					"parentSpanId": rootID,
					"spanType":     "llm",
					"name":         "child1",
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       child2ID,
					"parentSpanId": rootID,
					"spanType":     "tool",
					"name":         "child2",
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       grandchildID,
					"parentSpanId": child1ID,
					"spanType":     "llm",
					"name":         "grandchild",
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		// Query and verify all relationships
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 4 {
			t.Fatalf("expected 4 spans, got %d", len(spans))
		}

		// Build map of span IDs
		spanIDs := make(map[string]bool)
		for _, s := range spans {
			span := s.(map[string]any)
			spanIDs[span["ID"].(string)] = true
		}

		// Verify no orphans
		for _, s := range spans {
			span := s.(map[string]any)
			spanID := span["ID"].(string)
			parentID := span["ParentSpanID"]

			if parentID != nil {
				parentIDStr := parentID.(string)
				if !spanIDs[parentIDStr] {
					t.Errorf("ORPHAN DETECTED: span '%s' has parentSpanId '%s' which does not exist!",
						spanID, parentIDStr)
				}
			}
		}
	})
}

// =============================================================================
// FIELD PRESERVATION TESTS
// These tests verify all DTO fields are correctly stored and retrieved
// =============================================================================

// TestAllDTOFieldsPreserved verifies every field in IngestEvent is preserved
func TestAllDTOFieldsPreserved(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "fields@example.com", "password": "SecurePass123", "name": "Fields User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Fields Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("LLM span fields are preserved", func(t *testing.T) {
		traceID := "fields-trace-001"
		spanID := "fields-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "llm",
				"provider":     "anthropic",
				"model":        "claude-3-5-sonnet-20241022",
				"name":         "my-llm-call",
				"input":        map[string]any{"messages": []string{"hello"}},
				"output":       map[string]any{"response": "hi there"},
				"inputTokens":  100,
				"outputTokens": 50,
				"durationMs":   1500,
				"status":       "success",
				"sessionId":    "session-xyz",
				"userId":       "user-abc",
				"metadata":     map[string]any{"custom": "value"},
				"tags":         []string{"test", "e2e"},
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Verify all fields
		assertions := []struct {
			field    string
			expected any
			actual   any
		}{
			{"ID", spanID, span["ID"]},
			{"TraceID", traceID, span["TraceID"]},
			{"Type", "llm", span["Type"]},
			{"Name", "my-llm-call", span["Name"]},
			{"Provider", "anthropic", span["Provider"]},
			{"Model", "claude-3-5-sonnet-20241022", span["Model"]},
			{"Status", "success", span["Status"]},
		}

		for _, a := range assertions {
			if a.actual != a.expected {
				t.Errorf("field %s: expected '%v', got '%v'", a.field, a.expected, a.actual)
			}
		}

		// Check numeric fields (may be float64 from JSON)
		if inputTokens, ok := span["InputTokens"].(float64); ok {
			if int(inputTokens) != 100 {
				t.Errorf("InputTokens: expected 100, got %v", inputTokens)
			}
		}
		if outputTokens, ok := span["OutputTokens"].(float64); ok {
			if int(outputTokens) != 50 {
				t.Errorf("OutputTokens: expected 50, got %v", outputTokens)
			}
		}
		if durationMs, ok := span["DurationMs"].(float64); ok {
			if int(durationMs) != 1500 {
				t.Errorf("DurationMs: expected 1500, got %v", durationMs)
			}
		}

		// Check complex fields exist
		if span["Input"] == nil {
			t.Error("Input should not be nil")
		}
		if span["Output"] == nil {
			t.Error("Output should not be nil")
		}
	})

	t.Run("tool span fields are preserved", func(t *testing.T) {
		traceID := "fields-trace-002"
		spanID := "fields-span-002"
		parentID := "fields-parent-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  traceID,
					"spanId":   parentID,
					"spanType": "agent",
					"name":     "parent",
					"status":   "success",
				},
				{
					"traceId":      traceID,
					"spanId":       spanID,
					"parentSpanId": parentID,
					"spanType":     "tool",
					"name":         "search-tool",
					"input":        map[string]any{"query": "test"},
					"output":       map[string]any{"results": []string{"a", "b"}},
					"durationMs":   200,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		var toolSpan map[string]any
		for _, s := range spans {
			span := s.(map[string]any)
			if span["Type"] == "tool" {
				toolSpan = span
				break
			}
		}

		if toolSpan == nil {
			t.Fatal("tool span not found")
		}

		if toolSpan["ID"].(string) != spanID {
			t.Errorf("tool span ID not preserved: expected '%s', got '%s'", spanID, toolSpan["ID"])
		}
		if toolSpan["Name"].(string) != "search-tool" {
			t.Errorf("tool span name not preserved: expected 'search-tool', got '%s'", toolSpan["Name"])
		}
		if toolSpan["ParentSpanID"].(string) != parentID {
			t.Errorf("tool span parentId not preserved: expected '%s', got '%s'", parentID, toolSpan["ParentSpanID"])
		}
	})

	t.Run("error span fields are preserved", func(t *testing.T) {
		traceID := "fields-trace-003"
		spanID := "fields-span-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       spanID,
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"status":       "error",
				"errorMessage": "Rate limit exceeded",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Status"].(string) != "error" {
			t.Errorf("error status not preserved: got '%s'", span["Status"])
		}
		if span["ErrorMessage"] == nil {
			t.Error("errorMessage should be preserved")
		} else if span["ErrorMessage"].(string) != "Rate limit exceeded" {
			t.Errorf("errorMessage not preserved: got '%s'", span["ErrorMessage"])
		}
	})
}

// =============================================================================
// E2E FLOW TESTS
// These tests simulate real SDK payloads and verify the complete flow
// =============================================================================

// TestE2EAgentConversationFlow simulates a real agent conversation flow
func TestE2EAgentConversationFlow(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "e2e@example.com", "password": "SecurePass123", "name": "E2E User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "E2E Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("complete agent flow with LLM and tool calls", func(t *testing.T) {
		// Simulate: User message -> Agent -> LLM (planning) -> Tool -> LLM (response)
		traceID := "e2e-agent-001"
		agentSpanID := "e2e-agent-span"
		llm1SpanID := "e2e-llm-planning"
		toolSpanID := "e2e-tool-search"
		llm2SpanID := "e2e-llm-response"

		// This mimics what the SDK sends for a real agent conversation
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				// Agent span (root)
				{
					"traceId":    traceID,
					"spanId":     agentSpanID,
					"spanType":   "agent",
					"name":       "sales-agent",
					"input":      map[string]any{"message": "What products do you have?"},
					"durationMs": 5000,
					"status":     "success",
					"sessionId":  "conv_123",
					"userId":     "user_456",
				},
				// First LLM call (planning - decides to use tool)
				{
					"traceId":      traceID,
					"spanId":       llm1SpanID,
					"parentSpanId": agentSpanID,
					"spanType":     "llm",
					"name":         "planning",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"input":        []map[string]any{{"role": "user", "content": "What products?"}},
					"output":       []map[string]any{{"type": "tool_use", "name": "search_products"}},
					"inputTokens":  100,
					"outputTokens": 50,
					"durationMs":   800,
					"status":       "success",
				},
				// Tool call
				{
					"traceId":      traceID,
					"spanId":       toolSpanID,
					"parentSpanId": llm1SpanID,
					"spanType":     "tool",
					"name":         "search_products",
					"input":        map[string]any{"query": "products"},
					"output":       map[string]any{"results": []string{"Widget A", "Widget B"}},
					"durationMs":   200,
					"status":       "success",
				},
				// Second LLM call (response with tool results)
				{
					"traceId":      traceID,
					"spanId":       llm2SpanID,
					"parentSpanId": agentSpanID,
					"spanType":     "llm",
					"name":         "response",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"input":        []map[string]any{{"role": "user", "content": "tool results..."}},
					"output":       []map[string]any{{"type": "text", "text": "We have Widget A and B"}},
					"inputTokens":  200,
					"outputTokens": 100,
					"durationMs":   1200,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		// Query and verify the entire flow
		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Fatalf("failed to get trace: %d", traceResp.StatusCode)
		}

		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Verify trace metadata
		if trace["Name"] == nil || trace["Name"].(string) != "sales-agent" {
			t.Errorf("trace name should be 'sales-agent' from agent span, got '%v'", trace["Name"])
		}
		if trace["SessionID"] == nil || trace["SessionID"].(string) != "conv_123" {
			t.Errorf("sessionId not preserved: got '%v'", trace["SessionID"])
		}

		// Verify all spans
		spans := trace["Spans"].([]any)
		if len(spans) != 4 {
			t.Fatalf("expected 4 spans, got %d", len(spans))
		}

		// Build span map for verification
		spanMap := make(map[string]map[string]any)
		for _, s := range spans {
			span := s.(map[string]any)
			spanMap[span["ID"].(string)] = span
		}

		// Verify each span exists with correct ID
		expectedIDs := []string{agentSpanID, llm1SpanID, toolSpanID, llm2SpanID}
		for _, id := range expectedIDs {
			if spanMap[id] == nil {
				t.Errorf("span '%s' not found - ID not preserved!", id)
			}
		}

		// Verify hierarchy
		if spanMap[agentSpanID] != nil && spanMap[agentSpanID]["ParentSpanID"] != nil {
			t.Error("agent span should have no parent")
		}
		if spanMap[llm1SpanID] != nil {
			if spanMap[llm1SpanID]["ParentSpanID"].(string) != agentSpanID {
				t.Errorf("llm1 parent should be agent, got '%v'", spanMap[llm1SpanID]["ParentSpanID"])
			}
		}
		if spanMap[toolSpanID] != nil {
			if spanMap[toolSpanID]["ParentSpanID"].(string) != llm1SpanID {
				t.Errorf("tool parent should be llm1, got '%v'", spanMap[toolSpanID]["ParentSpanID"])
			}
		}
		if spanMap[llm2SpanID] != nil {
			if spanMap[llm2SpanID]["ParentSpanID"].(string) != agentSpanID {
				t.Errorf("llm2 parent should be agent, got '%v'", spanMap[llm2SpanID]["ParentSpanID"])
			}
		}

		// Verify no orphans (critical check)
		for id, span := range spanMap {
			if span["ParentSpanID"] != nil {
				parentID := span["ParentSpanID"].(string)
				if spanMap[parentID] == nil {
					t.Errorf("ORPHAN DETECTED: span '%s' references non-existent parent '%s'", id, parentID)
				}
			}
		}
	})
}

// TestE2EMultipleTracesIsolation verifies traces don't leak between each other
func TestE2EMultipleTracesIsolation(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "isolation@example.com", "password": "SecurePass123", "name": "Isolation User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Isolation Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("spans from different traces don't mix", func(t *testing.T) {
		trace1ID := "isolation-trace-001"
		trace2ID := "isolation-trace-002"
		span1ID := "isolation-span-001"
		span2ID := "isolation-span-002"

		// Ingest two separate traces
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  trace1ID,
					"spanId":   span1ID,
					"spanType": "llm",
					"name":     "trace1-span",
					"status":   "success",
				},
				{
					"traceId":  trace2ID,
					"spanId":   span2ID,
					"spanType": "llm",
					"name":     "trace2-span",
					"status":   "success",
				},
			},
		}, apiKeyHeaders)

		// Verify trace1 only has its span
		trace1Resp := ts.Request("GET", "/api/v1/traces/"+trace1ID, nil, apiKeyHeaders)
		var trace1 map[string]any
		ParseJSON(t, trace1Resp, &trace1)

		spans1 := trace1["Spans"].([]any)
		if len(spans1) != 1 {
			t.Errorf("trace1 should have 1 span, got %d", len(spans1))
		}
		if spans1[0].(map[string]any)["ID"].(string) != span1ID {
			t.Error("trace1 has wrong span")
		}

		// Verify trace2 only has its span
		trace2Resp := ts.Request("GET", "/api/v1/traces/"+trace2ID, nil, apiKeyHeaders)
		var trace2 map[string]any
		ParseJSON(t, trace2Resp, &trace2)

		spans2 := trace2["Spans"].([]any)
		if len(spans2) != 1 {
			t.Errorf("trace2 should have 1 span, got %d", len(spans2))
		}
		if spans2[0].(map[string]any)["ID"].(string) != span2ID {
			t.Error("trace2 has wrong span")
		}
	})
}
