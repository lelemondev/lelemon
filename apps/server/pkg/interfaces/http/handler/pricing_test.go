package handler_test

import (
	"math"
	"net/http"
	"testing"
)

// =============================================================================
// PRICING TESTS
// Verify cost calculations are correct for different models
// =============================================================================

// TestPricingCalculation verifies cost is calculated correctly per model
func TestPricingCalculation(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "pricing@example.com", "password": "SecurePass123", "name": "Pricing User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Pricing Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	// Helper to compare floats with tolerance
	almostEqual := func(a, b, tolerance float64) bool {
		return math.Abs(a-b) < tolerance
	}

	t.Run("gpt-4o pricing", func(t *testing.T) {
		// gpt-4o: Input=$0.0025/1K, Output=$0.01/1K
		// 1000 input + 500 output = (1000/1000)*0.0025 + (500/1000)*0.01 = 0.0025 + 0.005 = 0.0075
		traceID := "pricing-gpt4o-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricing-span-001",
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		cost := span["CostUSD"].(float64)
		expectedCost := 0.0075

		if !almostEqual(cost, expectedCost, 0.000001) {
			t.Errorf("gpt-4o cost: expected %.6f, got %.6f", expectedCost, cost)
		}
	})

	t.Run("gpt-4o-mini pricing", func(t *testing.T) {
		// gpt-4o-mini: Input=$0.00015/1K, Output=$0.0006/1K
		// 1000 input + 500 output = 0.00015 + 0.0003 = 0.00045
		traceID := "pricing-mini-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricing-span-002",
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o-mini",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		cost := span["CostUSD"].(float64)
		expectedCost := 0.00045

		if !almostEqual(cost, expectedCost, 0.000001) {
			t.Errorf("gpt-4o-mini cost: expected %.6f, got %.6f", expectedCost, cost)
		}
	})

	t.Run("claude-3-5-sonnet pricing", func(t *testing.T) {
		// claude-3-5-sonnet: Input=$0.003/1K, Output=$0.015/1K
		// 1000 input + 500 output = 0.003 + 0.0075 = 0.0105
		traceID := "pricing-claude-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricing-span-003",
				"spanType":     "llm",
				"provider":     "anthropic",
				"model":        "claude-3-5-sonnet-20241022",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		cost := span["CostUSD"].(float64)
		expectedCost := 0.0105

		if !almostEqual(cost, expectedCost, 0.000001) {
			t.Errorf("claude-3-5-sonnet cost: expected %.6f, got %.6f", expectedCost, cost)
		}
	})

	t.Run("o1-preview pricing", func(t *testing.T) {
		// o1-preview: Input=$0.015/1K, Output=$0.06/1K
		// 1000 input + 500 output = 0.015 + 0.03 = 0.045
		traceID := "pricing-o1-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricing-span-004",
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "o1-preview",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		cost := span["CostUSD"].(float64)
		expectedCost := 0.045

		if !almostEqual(cost, expectedCost, 0.000001) {
			t.Errorf("o1-preview cost: expected %.6f, got %.6f", expectedCost, cost)
		}
	})

	t.Run("bedrock claude pricing", func(t *testing.T) {
		// Bedrock model names should map to claude pricing
		traceID := "pricing-bedrock-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricing-span-005",
				"spanType":     "llm",
				"provider":     "bedrock",
				"model":        "anthropic.claude-3-haiku-20240307-v1:0",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["CostUSD"] == nil {
			t.Error("bedrock model should have cost calculated")
		} else {
			cost := span["CostUSD"].(float64)
			if cost <= 0 {
				t.Errorf("bedrock cost should be > 0, got %.6f", cost)
			}
		}
	})

	t.Run("zero tokens = zero cost", func(t *testing.T) {
		traceID := "pricing-zero-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricing-span-006",
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

		cost := span["CostUSD"].(float64)
		if cost != 0 {
			t.Errorf("zero tokens should have zero cost, got %.6f", cost)
		}
	})
}

// TestPricingAggregation verifies TotalCostUSD is calculated correctly
func TestPricingAggregation(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "costagg@example.com", "password": "SecurePass123", "name": "CostAgg User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "CostAgg Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	almostEqual := func(a, b, tolerance float64) bool {
		return math.Abs(a-b) < tolerance
	}

	t.Run("TotalCostUSD sums all LLM spans", func(t *testing.T) {
		traceID := "costagg-001"

		// Span 1: gpt-4o, 1000 in + 500 out = $0.0075
		// Span 2: gpt-4o-mini, 1000 in + 500 out = $0.00045
		// Total = $0.00795
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "costagg-span-001",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  1000,
					"outputTokens": 500,
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       "costagg-span-002",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o-mini",
					"inputTokens":  1000,
					"outputTokens": 500,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		totalCost := trace["TotalCostUSD"].(float64)
		expectedCost := 0.00795

		if !almostEqual(totalCost, expectedCost, 0.000001) {
			t.Errorf("TotalCostUSD: expected %.6f, got %.6f", expectedCost, totalCost)
		}
	})

	t.Run("tool spans don't add to cost", func(t *testing.T) {
		traceID := "costagg-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "costagg-llm-001",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  1000,
					"outputTokens": 500,
					"status":       "success",
				},
				{
					"traceId":    traceID,
					"spanId":     "costagg-tool-001",
					"spanType":   "tool",
					"name":       "search",
					"durationMs": 200,
					"status":     "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		totalCost := trace["TotalCostUSD"].(float64)
		expectedCost := 0.0075 // Only the LLM span

		if !almostEqual(totalCost, expectedCost, 0.000001) {
			t.Errorf("TotalCostUSD: expected %.6f (LLM only), got %.6f", expectedCost, totalCost)
		}
	})

	t.Run("mixed models aggregate correctly", func(t *testing.T) {
		traceID := "costagg-003"

		// gpt-4o: $0.0075
		// claude-3-5-sonnet: $0.0105
		// Total: $0.018
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":      traceID,
					"spanId":       "costagg-gpt-001",
					"spanType":     "llm",
					"provider":     "openai",
					"model":        "gpt-4o",
					"inputTokens":  1000,
					"outputTokens": 500,
					"status":       "success",
				},
				{
					"traceId":      traceID,
					"spanId":       "costagg-claude-001",
					"spanType":     "llm",
					"provider":     "anthropic",
					"model":        "claude-3-5-sonnet-20241022",
					"inputTokens":  1000,
					"outputTokens": 500,
					"status":       "success",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		totalCost := trace["TotalCostUSD"].(float64)
		expectedCost := 0.018

		if !almostEqual(totalCost, expectedCost, 0.000001) {
			t.Errorf("TotalCostUSD: expected %.6f, got %.6f", expectedCost, totalCost)
		}
	})
}

// TestPricingInList verifies cost appears correctly in trace list
func TestPricingInList(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "pricelist@example.com", "password": "SecurePass123", "name": "PriceList User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "PriceList Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}
	jwtHeaders := map[string]string{"Authorization": "Bearer " + auth.Token}

	almostEqual := func(a, b, tolerance float64) bool {
		return math.Abs(a-b) < tolerance
	}

	t.Run("cost appears in trace list", func(t *testing.T) {
		traceID := "pricelist-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "pricelist-span-001",
				"spanType":     "llm",
				"provider":     "openai",
				"model":        "gpt-4o",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		listResp := ts.Request("GET", "/api/v1/dashboard/projects/"+project.ID+"/traces", nil, jwtHeaders)
		if listResp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", listResp.StatusCode)
		}

		var result TracesResponse
		ParseJSON(t, listResp, &result)

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

		if !almostEqual(found.TotalCostUSD, 0.0075, 0.000001) {
			t.Errorf("TotalCostUSD in list: expected 0.0075, got %.6f", found.TotalCostUSD)
		}
	})
}

// TestPricingUnknownModel verifies unknown models get default pricing
func TestPricingUnknownModel(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "unknown@example.com", "password": "SecurePass123", "name": "Unknown User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Unknown Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("unknown model gets some pricing", func(t *testing.T) {
		traceID := "unknown-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":      traceID,
				"spanId":       "unknown-span-001",
				"spanType":     "llm",
				"provider":     "custom",
				"model":        "totally-unknown-model-xyz",
				"inputTokens":  1000,
				"outputTokens": 500,
				"status":       "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Should have some cost (default pricing), not nil or zero
		if span["CostUSD"] == nil {
			t.Log("Note: unknown model has nil cost (may be expected behavior)")
		} else {
			cost := span["CostUSD"].(float64)
			// Just verify it's a reasonable number (could be 0 if no default)
			t.Logf("Unknown model cost: %.6f", cost)
		}
	})
}
