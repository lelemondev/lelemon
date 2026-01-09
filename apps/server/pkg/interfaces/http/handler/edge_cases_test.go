package handler_test

import (
	"net/http"
	"strings"
	"testing"
)

// =============================================================================
// EDGE CASE TESTS
// Verify system handles unusual inputs correctly
// =============================================================================

// TestEdgeCaseNullFields verifies handling of null/missing fields
func TestEdgeCaseNullFields(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "nullfields@example.com", "password": "SecurePass123", "name": "NullFields User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "NullFields Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("minimal span (only required fields)", func(t *testing.T) {
		traceID := "null-001"
		spanID := "null-span-001"

		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				// No other fields
			}},
		}, apiKeyHeaders)

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 1 {
			t.Error("span should be created with minimal fields")
		}
	})

	t.Run("null input and output", func(t *testing.T) {
		traceID := "null-002"
		spanID := "null-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"input":    nil,
				"output":   nil,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Nil should be preserved, not cause errors
		if span["Input"] != nil {
			t.Log("Input is preserved as non-nil (may be empty object)")
		}
	})

	t.Run("empty string fields", func(t *testing.T) {
		traceID := "null-003"
		spanID := "null-span-003"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"provider": "",
				"model":    "",
				"name":     "",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Error("empty strings should not cause errors")
		}
	})
}

// TestEdgeCaseLargeData verifies handling of large payloads
func TestEdgeCaseLargeData(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "largedata@example.com", "password": "SecurePass123", "name": "LargeData User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "LargeData Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("large input content", func(t *testing.T) {
		traceID := "large-001"
		spanID := "large-span-001"

		// Create a 10KB input
		largeText := strings.Repeat("This is a large text block. ", 500)

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"input":    map[string]any{"content": largeText},
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Input"] == nil {
			t.Error("large input should be preserved")
		}
	})

	t.Run("many spans in single trace", func(t *testing.T) {
		traceID := "large-002"

		// Create 50 spans in one trace
		events := make([]map[string]any, 50)
		for i := 0; i < 50; i++ {
			events[i] = map[string]any{
				"traceId":  traceID,
				"spanId":   "span-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
				"spanType": "llm",
				"status":   "success",
			}
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": events,
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 50 {
			t.Errorf("expected 50 spans, got %d", len(spans))
		}
	})

	t.Run("deep nested metadata", func(t *testing.T) {
		traceID := "large-003"
		spanID := "large-span-003"

		// Create deeply nested metadata
		deepNested := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"level4": map[string]any{
							"value": "deep value",
						},
					},
				},
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"metadata": deepNested,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Metadata"] == nil {
			t.Error("deeply nested metadata should be preserved")
		}
	})
}

// TestEdgeCaseSpecialCharacters verifies handling of special characters
func TestEdgeCaseSpecialCharacters(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "special@example.com", "password": "SecurePass123", "name": "Special User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Special Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("unicode characters in content", func(t *testing.T) {
		traceID := "special-001"
		spanID := "special-span-001"

		unicodeText := "Hello 你好 مرحبا שלום 🎉 emoji test"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"input":    map[string]any{"content": unicodeText},
				"output":   unicodeText,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Output"].(string) != unicodeText {
			t.Errorf("unicode not preserved: expected '%s', got '%s'", unicodeText, span["Output"])
		}
	})

	t.Run("special characters in name", func(t *testing.T) {
		traceID := "special-002"
		spanID := "special-span-002"

		specialName := "test'with\"special\nchars\ttab"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"name":     specialName,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Name"].(string) != specialName {
			t.Errorf("special chars not preserved in name")
		}
	})

	t.Run("JSON-like strings in content", func(t *testing.T) {
		traceID := "special-003"
		spanID := "special-span-003"

		jsonString := `{"key": "value", "array": [1, 2, 3]}`

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"output":   jsonString,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Output"].(string) != jsonString {
			t.Error("JSON-like string should be preserved as-is")
		}
	})
}

// TestEdgeCaseMixedTypes verifies handling of mixed data types
func TestEdgeCaseMixedTypes(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "mixed@example.com", "password": "SecurePass123", "name": "Mixed User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Mixed Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("metadata with mixed types", func(t *testing.T) {
		traceID := "mixed-001"
		spanID := "mixed-span-001"

		mixedMetadata := map[string]any{
			"string":  "text",
			"number":  123,
			"float":   1.5,
			"bool":    true,
			"null":    nil,
			"array":   []any{1, "two", 3.0},
			"nested":  map[string]any{"a": 1, "b": "two"},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"metadata": mixedMetadata,
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["Metadata"] == nil {
			t.Error("mixed type metadata should be preserved")
		} else {
			meta := span["Metadata"].(map[string]any)
			if meta["string"] != "text" {
				t.Error("string type not preserved")
			}
			if meta["number"].(float64) != 123 {
				t.Error("number type not preserved")
			}
			if meta["bool"] != true {
				t.Error("bool type not preserved")
			}
		}
	})

	t.Run("array output vs string output", func(t *testing.T) {
		traceID := "mixed-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{
				{
					"traceId":  traceID,
					"spanId":   "mixed-array",
					"spanType": "llm",
					"status":   "success",
					"output":   []map[string]any{{"type": "text", "text": "hello"}},
				},
				{
					"traceId":  traceID,
					"spanId":   "mixed-string",
					"spanType": "llm",
					"status":   "success",
					"output":   "hello",
				},
			},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 2 {
			t.Error("both spans should be created regardless of output type")
		}
	})
}

// TestEdgeCaseDuplicateIDs verifies handling of duplicate IDs
func TestEdgeCaseDuplicateIDs(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "duplicate@example.com", "password": "SecurePass123", "name": "Duplicate User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Duplicate Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("same traceId in multiple requests adds spans", func(t *testing.T) {
		traceID := "dup-001"

		// First request
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   "dup-span-001",
				"spanType": "llm",
				"status":   "success",
			}},
		}, apiKeyHeaders)

		// Second request with same traceId
		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   "dup-span-002",
				"spanType": "llm",
				"status":   "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		if len(spans) != 2 {
			t.Errorf("both spans should be added to same trace, got %d", len(spans))
		}
	})
}

// TestEdgeCaseEmptyArrays verifies handling of empty arrays
func TestEdgeCaseEmptyArrays(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "emptyarray@example.com", "password": "SecurePass123", "name": "EmptyArray User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "EmptyArray Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("empty events array", func(t *testing.T) {
		resp := ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{},
		}, apiKeyHeaders)

		var result IngestResponse
		ParseJSON(t, resp, &result)

		if !result.Success {
			t.Error("empty events should succeed")
		}
		if result.Processed != 0 {
			t.Errorf("processed should be 0 for empty events, got %d", result.Processed)
		}
	})

	t.Run("empty tags array", func(t *testing.T) {
		traceID := "emptyarray-001"
		spanID := "emptyarray-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"tags":     []string{},
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Error("empty tags should not cause errors")
		}
	})

	t.Run("empty metadata object", func(t *testing.T) {
		traceID := "emptyarray-002"
		spanID := "emptyarray-span-002"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":  traceID,
				"spanId":   spanID,
				"spanType": "llm",
				"status":   "success",
				"metadata": map[string]any{},
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		if traceResp.StatusCode != http.StatusOK {
			t.Error("empty metadata should not cause errors")
		}
	})
}

// TestEdgeCaseTimestamps verifies timestamp handling
func TestEdgeCaseTimestamps(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "timestamp@example.com", "password": "SecurePass123", "name": "Timestamp User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Timestamp Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("ISO 8601 timestamp is accepted", func(t *testing.T) {
		traceID := "timestamp-001"
		spanID := "timestamp-span-001"

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":   traceID,
				"spanId":    spanID,
				"spanType":  "llm",
				"status":    "success",
				"timestamp": "2024-01-15T10:30:00Z",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["StartedAt"] == nil {
			t.Error("StartedAt should be set from timestamp")
		}
	})
}
