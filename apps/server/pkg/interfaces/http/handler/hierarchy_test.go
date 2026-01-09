package handler_test

import (
	"testing"
)

// =============================================================================
// HIERARCHY TESTS
// Verify complex span hierarchies are correctly stored and retrieved
// =============================================================================

// TestHierarchyFlat verifies flat traces (no parent-child relationships)
func TestHierarchyFlat(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "flat@example.com", "password": "SecurePass123", "name": "Flat User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Flat Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("single span trace", func(t *testing.T) {
		traceID := "flat-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   "flat-span-001",
				"spanType": "llm",
				"status":   "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 1 {
			t.Errorf("expected 1 span, got %d", len(spans))
		}

		span := spans[0].(map[string]any)
		if span["ParentSpanID"] != nil {
			t.Error("single span should have no parent")
		}
	})

	t.Run("multiple flat spans (no parent)", func(t *testing.T) {
		traceID := "flat-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": "flat-a", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "flat-b", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "flat-c", "spanType": "llm", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 3 {
			t.Errorf("expected 3 spans, got %d", len(spans))
		}

		for _, s := range spans {
			span := s.(map[string]any)
			if span["ParentSpanID"] != nil {
				t.Errorf("span %s should have no parent", span["ID"])
			}
		}
	})
}

// TestHierarchyTwoLevels verifies parent-child relationships
func TestHierarchyTwoLevels(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "twolevel@example.com", "password": "SecurePass123", "name": "TwoLevel User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "TwoLevel Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("agent with one LLM child", func(t *testing.T) {
		traceID := "twolevel-001"
		agentID := "agent-001"
		llmID := "llm-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  traceID,
					"spanId":   agentID,
					"spanType": "agent",
					"name":     "my-agent",
					"status":   "success",
				},
				{
					"traceId":      traceID,
					"spanId":       llmID,
					"parentSpanId": agentID,
					"spanType":     "llm",
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

		spanMap := make(map[string]map[string]any)
		for _, s := range spans {
			span := s.(map[string]any)
			spanMap[span["ID"].(string)] = span
		}

		// Agent should have no parent
		if spanMap[agentID]["ParentSpanID"] != nil {
			t.Error("agent span should have no parent")
		}

		// LLM should have agent as parent
		if spanMap[llmID]["ParentSpanID"] == nil {
			t.Error("LLM span should have parent")
		} else if spanMap[llmID]["ParentSpanID"].(string) != agentID {
			t.Errorf("LLM parent should be %s, got %s", agentID, spanMap[llmID]["ParentSpanID"])
		}
	})

	t.Run("agent with multiple children", func(t *testing.T) {
		traceID := "twolevel-002"
		agentID := "agent-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": agentID, "spanType": "agent", "name": "agent", "status": "success"},
				{"traceId": traceID, "spanId": "child-1", "parentSpanId": agentID, "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "child-2", "parentSpanId": agentID, "spanType": "tool", "name": "tool", "status": "success"},
				{"traceId": traceID, "spanId": "child-3", "parentSpanId": agentID, "spanType": "llm", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 4 {
			t.Fatalf("expected 4 spans, got %d", len(spans))
		}

		childCount := 0
		for _, s := range spans {
			span := s.(map[string]any)
			if span["ParentSpanID"] != nil && span["ParentSpanID"].(string) == agentID {
				childCount++
			}
		}

		if childCount != 3 {
			t.Errorf("expected 3 children of agent, got %d", childCount)
		}
	})
}

// TestHierarchyDeep verifies 3+ level hierarchies
func TestHierarchyDeep(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "deep@example.com", "password": "SecurePass123", "name": "Deep User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Deep Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("three level hierarchy", func(t *testing.T) {
		// Agent -> LLM -> Tool
		traceID := "deep-001"
		agentID := "deep-agent"
		llmID := "deep-llm"
		toolID := "deep-tool"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": agentID, "spanType": "agent", "name": "agent", "status": "success"},
				{"traceId": traceID, "spanId": llmID, "parentSpanId": agentID, "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": toolID, "parentSpanId": llmID, "spanType": "tool", "name": "tool", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		spanMap := make(map[string]map[string]any)
		for _, s := range spans {
			span := s.(map[string]any)
			spanMap[span["ID"].(string)] = span
		}

		// Level 1: Agent (no parent)
		if spanMap[agentID]["ParentSpanID"] != nil {
			t.Error("agent (level 1) should have no parent")
		}

		// Level 2: LLM -> Agent
		if spanMap[llmID]["ParentSpanID"].(string) != agentID {
			t.Errorf("llm (level 2) parent should be agent")
		}

		// Level 3: Tool -> LLM
		if spanMap[toolID]["ParentSpanID"].(string) != llmID {
			t.Errorf("tool (level 3) parent should be llm")
		}
	})

	t.Run("four level hierarchy", func(t *testing.T) {
		// Agent -> LLM1 -> Tool -> LLM2
		traceID := "deep-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": "level1", "spanType": "agent", "name": "agent", "status": "success"},
				{"traceId": traceID, "spanId": "level2", "parentSpanId": "level1", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "level3", "parentSpanId": "level2", "spanType": "tool", "name": "tool", "status": "success"},
				{"traceId": traceID, "spanId": "level4", "parentSpanId": "level3", "spanType": "llm", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		spanMap := make(map[string]map[string]any)
		for _, s := range spans {
			span := s.(map[string]any)
			spanMap[span["ID"].(string)] = span
		}

		// Verify chain: level4 -> level3 -> level2 -> level1 -> nil
		if spanMap["level4"]["ParentSpanID"].(string) != "level3" {
			t.Error("level4 parent should be level3")
		}
		if spanMap["level3"]["ParentSpanID"].(string) != "level2" {
			t.Error("level3 parent should be level2")
		}
		if spanMap["level2"]["ParentSpanID"].(string) != "level1" {
			t.Error("level2 parent should be level1")
		}
		if spanMap["level1"]["ParentSpanID"] != nil {
			t.Error("level1 should have no parent")
		}
	})
}

// TestHierarchyBranching verifies traces with multiple branches
func TestHierarchyBranching(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "branch@example.com", "password": "SecurePass123", "name": "Branch User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Branch Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("multiple branches from root", func(t *testing.T) {
		/*
			Agent
			├── LLM1
			│   └── Tool1
			├── LLM2
			│   ├── Tool2
			│   └── Tool3
			└── LLM3
		*/
		traceID := "branch-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": "agent", "spanType": "agent", "name": "agent", "status": "success"},
				{"traceId": traceID, "spanId": "llm1", "parentSpanId": "agent", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "tool1", "parentSpanId": "llm1", "spanType": "tool", "name": "tool1", "status": "success"},
				{"traceId": traceID, "spanId": "llm2", "parentSpanId": "agent", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "tool2", "parentSpanId": "llm2", "spanType": "tool", "name": "tool2", "status": "success"},
				{"traceId": traceID, "spanId": "tool3", "parentSpanId": "llm2", "spanType": "tool", "name": "tool3", "status": "success"},
				{"traceId": traceID, "spanId": "llm3", "parentSpanId": "agent", "spanType": "llm", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 7 {
			t.Fatalf("expected 7 spans, got %d", len(spans))
		}

		spanMap := make(map[string]map[string]any)
		for _, s := range spans {
			span := s.(map[string]any)
			spanMap[span["ID"].(string)] = span
		}

		// Verify structure
		tests := []struct {
			spanID   string
			parentID string
		}{
			{"agent", ""},
			{"llm1", "agent"},
			{"tool1", "llm1"},
			{"llm2", "agent"},
			{"tool2", "llm2"},
			{"tool3", "llm2"},
			{"llm3", "agent"},
		}

		for _, tc := range tests {
			span := spanMap[tc.spanID]
			if span == nil {
				t.Errorf("span %s not found", tc.spanID)
				continue
			}

			if tc.parentID == "" {
				if span["ParentSpanID"] != nil {
					t.Errorf("span %s should have no parent", tc.spanID)
				}
			} else {
				if span["ParentSpanID"] == nil {
					t.Errorf("span %s should have parent %s", tc.spanID, tc.parentID)
				} else if span["ParentSpanID"].(string) != tc.parentID {
					t.Errorf("span %s parent: expected %s, got %s", tc.spanID, tc.parentID, span["ParentSpanID"])
				}
			}
		}
	})
}

// TestHierarchyNoOrphans verifies no orphan spans exist
func TestHierarchyNoOrphans(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "noorphan@example.com", "password": "SecurePass123", "name": "NoOrphan User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "NoOrphan Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("all parents exist in trace", func(t *testing.T) {
		traceID := "orphan-check-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{"traceId": traceID, "spanId": "root", "spanType": "agent", "name": "agent", "status": "success"},
				{"traceId": traceID, "spanId": "child1", "parentSpanId": "root", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "child2", "parentSpanId": "root", "spanType": "llm", "status": "success"},
				{"traceId": traceID, "spanId": "grandchild", "parentSpanId": "child1", "spanType": "tool", "name": "tool", "status": "success"},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)

		// Build set of all span IDs
		spanIDs := make(map[string]bool)
		for _, s := range spans {
			span := s.(map[string]any)
			spanIDs[span["ID"].(string)] = true
		}

		// Verify all parent IDs exist
		for _, s := range spans {
			span := s.(map[string]any)
			spanID := span["ID"].(string)
			if span["ParentSpanID"] != nil {
				parentID := span["ParentSpanID"].(string)
				if !spanIDs[parentID] {
					t.Errorf("ORPHAN: span %s has parent %s which doesn't exist", spanID, parentID)
				}
			}
		}
	})
}

// TestHierarchyRealWorldScenario simulates a real agent conversation
func TestHierarchyRealWorldScenario(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "realworld@example.com", "password": "SecurePass123", "name": "RealWorld User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "RealWorld Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("sales agent multi-turn conversation", func(t *testing.T) {
		/*
			Real scenario: User asks about products, agent searches and responds

			sales-agent (agent)
			├── turn-1-llm (llm) - Planning: decides to search
			│   └── search-products (tool) - Executes search
			├── turn-1-response (llm) - Responds with results
			├── turn-2-llm (llm) - Planning: more questions
			│   ├── get-inventory (tool)
			│   └── check-pricing (tool)
			└── turn-2-response (llm) - Final response
		*/
		traceID := "realworld-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				// Agent root
				{
					"traceId":    traceID,
					"spanId":     "agent",
					"spanType":   "agent",
					"name":       "sales-agent",
					"input":      map[string]any{"message": "What products do you have?"},
					"durationMs": 8000,
					"status":     "success",
					"sessionId":  "conv_abc123",
				},
				// Turn 1: Planning
				{
					"traceId":      traceID,
					"spanId":       "turn1-planning",
					"parentSpanId": "agent",
					"spanType":     "llm",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"name":         "planning",
					"inputTokens":  150,
					"outputTokens": 80,
					"durationMs":   800,
					"status":       "success",
				},
				// Turn 1: Tool call
				{
					"traceId":      traceID,
					"spanId":       "search-products",
					"parentSpanId": "turn1-planning",
					"spanType":     "tool",
					"name":         "search_products",
					"input":        map[string]any{"query": "all products"},
					"output":       map[string]any{"products": []string{"A", "B", "C"}},
					"durationMs":   200,
					"status":       "success",
				},
				// Turn 1: Response
				{
					"traceId":      traceID,
					"spanId":       "turn1-response",
					"parentSpanId": "agent",
					"spanType":     "llm",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"name":         "response",
					"inputTokens":  250,
					"outputTokens": 120,
					"durationMs":   1200,
					"status":       "success",
				},
				// Turn 2: Planning
				{
					"traceId":      traceID,
					"spanId":       "turn2-planning",
					"parentSpanId": "agent",
					"spanType":     "llm",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"name":         "planning",
					"inputTokens":  300,
					"outputTokens": 100,
					"durationMs":   900,
					"status":       "success",
				},
				// Turn 2: Tool 1
				{
					"traceId":      traceID,
					"spanId":       "get-inventory",
					"parentSpanId": "turn2-planning",
					"spanType":     "tool",
					"name":         "get_inventory",
					"durationMs":   150,
					"status":       "success",
				},
				// Turn 2: Tool 2
				{
					"traceId":      traceID,
					"spanId":       "check-pricing",
					"parentSpanId": "turn2-planning",
					"spanType":     "tool",
					"name":         "check_pricing",
					"durationMs":   100,
					"status":       "success",
				},
				// Turn 2: Response
				{
					"traceId":      traceID,
					"spanId":       "turn2-response",
					"parentSpanId": "agent",
					"spanType":     "llm",
					"provider":     "bedrock",
					"model":        "anthropic.claude-3-haiku-20240307-v1:0",
					"name":         "response",
					"inputTokens":  400,
					"outputTokens": 200,
					"durationMs":   1500,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		// Verify trace metadata
		if trace["Name"] == nil || trace["Name"].(string) != "sales-agent" {
			t.Errorf("trace name should be 'sales-agent', got %v", trace["Name"])
		}
		if trace["SessionID"] == nil || trace["SessionID"].(string) != "conv_abc123" {
			t.Errorf("sessionId should be 'conv_abc123', got %v", trace["SessionID"])
		}

		// Verify span count
		spans := trace["Spans"].([]any)
		if len(spans) != 8 {
			t.Errorf("expected 8 spans, got %d", len(spans))
		}

		// Verify aggregations
		totalSpans := int(trace["TotalSpans"].(float64))
		if totalSpans != 8 {
			t.Errorf("TotalSpans: expected 8, got %d", totalSpans)
		}

		// Expected tokens: (150+80) + (250+120) + (300+100) + (400+200) = 1600
		totalTokens := int(trace["TotalTokens"].(float64))
		if totalTokens != 1600 {
			t.Errorf("TotalTokens: expected 1600, got %d", totalTokens)
		}

		// Build span map and verify hierarchy
		spanMap := make(map[string]map[string]any)
		for _, s := range spans {
			span := s.(map[string]any)
			spanMap[span["ID"].(string)] = span
		}

		// Verify all parent references are valid
		for _, s := range spans {
			span := s.(map[string]any)
			if span["ParentSpanID"] != nil {
				parentID := span["ParentSpanID"].(string)
				if spanMap[parentID] == nil {
					t.Errorf("ORPHAN: span %s references non-existent parent %s", span["ID"], parentID)
				}
			}
		}
	})
}
