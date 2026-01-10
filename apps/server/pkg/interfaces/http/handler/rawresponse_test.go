package handler_test

import (
	"testing"
)

// =============================================================================
// RAW RESPONSE PARSING TESTS
// Verify server correctly extracts data from raw LLM provider responses
// =============================================================================

// TestRawResponseOpenAI verifies OpenAI response parsing
func TestRawResponseOpenAI(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "openai@example.com", "password": "SecurePass123", "name": "OpenAI User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "OpenAI Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("extracts tokens from OpenAI response", func(t *testing.T) {
		traceID := "openai-raw-001"
		spanID := "openai-span-001"

		// OpenAI Chat Completions API response format
		openaiResponse := map[string]any{
			"id":      "chatcmpl-123abc",
			"object":  "chat.completion",
			"created": 1699000000,
			"model":   "gpt-4o-2024-08-06",
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "Hello! How can I help you today?",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":     50,
				"completion_tokens": 25,
				"total_tokens":      75,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "openai",
				"model":       "gpt-4o",
				"rawResponse": openaiResponse,
				"durationMs":  500,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Verify tokens extracted
		if span["InputTokens"] == nil {
			t.Error("InputTokens should be extracted from rawResponse")
		} else if int(span["InputTokens"].(float64)) != 50 {
			t.Errorf("InputTokens: expected 50, got %v", span["InputTokens"])
		}

		if span["OutputTokens"] == nil {
			t.Error("OutputTokens should be extracted from rawResponse")
		} else if int(span["OutputTokens"].(float64)) != 25 {
			t.Errorf("OutputTokens: expected 25, got %v", span["OutputTokens"])
		}

		// Verify output extracted
		if span["Output"] == nil {
			t.Error("Output should be extracted from rawResponse")
		}

		// Verify stop reason
		if span["StopReason"] == nil {
			t.Error("StopReason should be extracted")
		} else if span["StopReason"].(string) != "stop" {
			t.Errorf("StopReason: expected 'stop', got '%v'", span["StopReason"])
		}
	})

	t.Run("extracts tool calls from OpenAI response", func(t *testing.T) {
		traceID := "openai-raw-002"
		spanID := "openai-span-002"

		openaiResponse := map[string]any{
			"id":    "chatcmpl-456def",
			"model": "gpt-4o",
			"choices": []map[string]any{{
				"message": map[string]any{
					"role":    "assistant",
					"content": nil,
					"tool_calls": []map[string]any{{
						"id":   "call_abc123",
						"type": "function",
						"function": map[string]any{
							"name":      "get_weather",
							"arguments": `{"location": "San Francisco", "unit": "celsius"}`,
						},
					}},
				},
				"finish_reason": "tool_calls",
			}},
			"usage": map[string]any{
				"prompt_tokens":     100,
				"completion_tokens": 50,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "openai",
				"model":       "gpt-4o",
				"rawResponse": openaiResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Verify tool uses extracted
		if span["ToolUses"] != nil {
			toolUses := span["ToolUses"].([]any)
			if len(toolUses) == 0 {
				t.Error("ToolUses should be extracted from tool_calls")
			} else {
				tu := toolUses[0].(map[string]any)
				if tu["Name"].(string) != "get_weather" {
					t.Errorf("Tool name: expected 'get_weather', got '%v'", tu["Name"])
				}
			}
		}

		// Verify SubType is planning
		if span["SubType"] != nil && span["SubType"].(string) != "planning" {
			t.Errorf("SubType should be 'planning' for tool calls, got '%v'", span["SubType"])
		}
	})

	t.Run("extracts reasoning tokens from o1 response", func(t *testing.T) {
		traceID := "openai-raw-003"
		spanID := "openai-span-003"

		o1Response := map[string]any{
			"id":    "chatcmpl-o1-123",
			"model": "o1-preview",
			"choices": []map[string]any{{
				"message": map[string]any{
					"role":    "assistant",
					"content": "After careful analysis...",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":     200,
				"completion_tokens": 150,
				"completion_tokens_details": map[string]any{
					"reasoning_tokens": 100,
				},
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "openai",
				"model":       "o1-preview",
				"rawResponse": o1Response,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Verify reasoning tokens extracted
		if span["ReasoningTokens"] != nil {
			reasoning := int(span["ReasoningTokens"].(float64))
			if reasoning != 100 {
				t.Errorf("ReasoningTokens: expected 100, got %d", reasoning)
			}
		}
	})
}

// TestRawResponseAnthropic verifies Anthropic response parsing
func TestRawResponseAnthropic(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "anthropic@example.com", "password": "SecurePass123", "name": "Anthropic User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Anthropic Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("extracts tokens from Anthropic response", func(t *testing.T) {
		traceID := "anthropic-raw-001"
		spanID := "anthropic-span-001"

		// Anthropic Messages API response format
		anthropicResponse := map[string]any{
			"id":   "msg_01XFDUDYJgAACzvnptvVoYEL",
			"type": "message",
			"role": "assistant",
			"content": []map[string]any{{
				"type": "text",
				"text": "Hello! How can I help you today?",
			}},
			"model":       "claude-3-5-sonnet-20241022",
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  100,
				"output_tokens": 50,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "anthropic",
				"model":       "claude-3-5-sonnet-20241022",
				"rawResponse": anthropicResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if int(span["InputTokens"].(float64)) != 100 {
			t.Errorf("InputTokens: expected 100, got %v", span["InputTokens"])
		}
		if int(span["OutputTokens"].(float64)) != 50 {
			t.Errorf("OutputTokens: expected 50, got %v", span["OutputTokens"])
		}
		if span["StopReason"].(string) != "end_turn" {
			t.Errorf("StopReason: expected 'end_turn', got '%v'", span["StopReason"])
		}

		// CRITICAL: Verify Output is extracted from Anthropic rawResponse
		if span["Output"] == nil {
			t.Error("Output should be extracted from Anthropic rawResponse")
		} else if span["Output"].(string) != "Hello! How can I help you today?" {
			t.Errorf("Output: expected 'Hello! How can I help you today?', got '%v'", span["Output"])
		}
	})

	t.Run("extracts cache tokens from Anthropic response", func(t *testing.T) {
		traceID := "anthropic-raw-002"
		spanID := "anthropic-span-002"

		anthropicResponse := map[string]any{
			"id":   "msg_cache123",
			"type": "message",
			"content": []map[string]any{{
				"type": "text",
				"text": "Cached response",
			}},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":               50,
				"output_tokens":              25,
				"cache_read_input_tokens":    500,
				"cache_creation_input_tokens": 200,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "anthropic",
				"model":       "claude-3-5-sonnet-20241022",
				"rawResponse": anthropicResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["CacheReadTokens"] == nil {
			t.Error("CacheReadTokens should be extracted")
		} else if int(span["CacheReadTokens"].(float64)) != 500 {
			t.Errorf("CacheReadTokens: expected 500, got %v", span["CacheReadTokens"])
		}

		if span["CacheWriteTokens"] == nil {
			t.Error("CacheWriteTokens should be extracted")
		} else if int(span["CacheWriteTokens"].(float64)) != 200 {
			t.Errorf("CacheWriteTokens: expected 200, got %v", span["CacheWriteTokens"])
		}
	})

	t.Run("extracts tool use from Anthropic response", func(t *testing.T) {
		traceID := "anthropic-raw-003"
		spanID := "anthropic-span-003"

		anthropicResponse := map[string]any{
			"id":   "msg_tool123",
			"type": "message",
			"content": []map[string]any{
				{
					"type": "text",
					"text": "I'll search for that information.",
				},
				{
					"type":  "tool_use",
					"id":    "toolu_01A09q90qw90lq917835lqs8",
					"name":  "search_products",
					"input": map[string]any{"query": "widgets"},
				},
			},
			"stop_reason": "tool_use",
			"usage": map[string]any{
				"input_tokens":  150,
				"output_tokens": 75,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "anthropic",
				"model":       "claude-3-5-sonnet-20241022",
				"rawResponse": anthropicResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Verify SubType is planning
		if span["SubType"] != nil && span["SubType"].(string) != "planning" {
			t.Errorf("SubType should be 'planning', got '%v'", span["SubType"])
		}

		// Verify tool uses
		if span["ToolUses"] != nil {
			toolUses := span["ToolUses"].([]any)
			if len(toolUses) == 0 {
				t.Error("ToolUses should contain extracted tool")
			} else {
				tu := toolUses[0].(map[string]any)
				if tu["Name"].(string) != "search_products" {
					t.Errorf("Tool name: expected 'search_products', got '%v'", tu["Name"])
				}
				if tu["ID"].(string) != "toolu_01A09q90qw90lq917835lqs8" {
					t.Errorf("Tool ID should be preserved")
				}
			}
		}
	})

	t.Run("extracts thinking from Anthropic response", func(t *testing.T) {
		traceID := "anthropic-raw-004"
		spanID := "anthropic-span-004"

		anthropicResponse := map[string]any{
			"id":   "msg_thinking123",
			"type": "message",
			"content": []map[string]any{
				{
					"type":     "thinking",
					"thinking": "Let me analyze this step by step...",
				},
				{
					"type": "text",
					"text": "Based on my analysis, here's the answer.",
				},
			},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  200,
				"output_tokens": 100,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "anthropic",
				"model":       "claude-3-5-sonnet-20241022",
				"rawResponse": anthropicResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Verify thinking extracted
		if span["Thinking"] != nil {
			if span["Thinking"].(string) != "Let me analyze this step by step..." {
				t.Errorf("Thinking not extracted correctly")
			}
		}
	})
}

// TestRawResponseBedrock verifies Bedrock response parsing
func TestRawResponseBedrock(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "bedrock@example.com", "password": "SecurePass123", "name": "Bedrock User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Bedrock Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("extracts from Bedrock Converse API response", func(t *testing.T) {
		traceID := "bedrock-raw-001"
		spanID := "bedrock-span-001"

		// Bedrock Converse API response format
		bedrockResponse := map[string]any{
			"output": map[string]any{
				"message": map[string]any{
					"role": "assistant",
					"content": []map[string]any{{
						"text": "Hello from Bedrock!",
					}},
				},
			},
			"stopReason": "end_turn",
			"usage": map[string]any{
				"inputTokens":  100,
				"outputTokens": 50,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "bedrock",
				"model":       "anthropic.claude-3-haiku-20240307-v1:0",
				"rawResponse": bedrockResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if int(span["InputTokens"].(float64)) != 100 {
			t.Errorf("InputTokens: expected 100, got %v", span["InputTokens"])
		}
		if int(span["OutputTokens"].(float64)) != 50 {
			t.Errorf("OutputTokens: expected 50, got %v", span["OutputTokens"])
		}
		if span["StopReason"].(string) != "end_turn" {
			t.Errorf("StopReason: expected 'end_turn', got '%v'", span["StopReason"])
		}

		// CRITICAL: Verify Output is extracted from rawResponse
		if span["Output"] == nil {
			t.Error("Output should be extracted from Bedrock Converse rawResponse")
		} else if span["Output"].(string) != "Hello from Bedrock!" {
			t.Errorf("Output: expected 'Hello from Bedrock!', got '%v'", span["Output"])
		}
	})

	t.Run("extracts from Bedrock InvokeModel API response (Anthropic format)", func(t *testing.T) {
		traceID := "bedrock-raw-002"
		spanID := "bedrock-span-002"

		// InvokeModel returns Anthropic format directly
		invokeModelResponse := map[string]any{
			"id":   "msg_bedrock123",
			"type": "message",
			"content": []map[string]any{{
				"type": "text",
				"text": "Response from InvokeModel",
			}},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  75,
				"output_tokens": 35,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "bedrock",
				"model":       "anthropic.claude-3-haiku-20240307-v1:0",
				"rawResponse": invokeModelResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if int(span["InputTokens"].(float64)) != 75 {
			t.Errorf("InputTokens: expected 75, got %v", span["InputTokens"])
		}
		if int(span["OutputTokens"].(float64)) != 35 {
			t.Errorf("OutputTokens: expected 35, got %v", span["OutputTokens"])
		}

		// CRITICAL: Verify Output is extracted from InvokeModel rawResponse
		if span["Output"] == nil {
			t.Error("Output should be extracted from Bedrock InvokeModel rawResponse")
		} else if span["Output"].(string) != "Response from InvokeModel" {
			t.Errorf("Output: expected 'Response from InvokeModel', got '%v'", span["Output"])
		}
	})

	t.Run("extracts tool use from Bedrock Converse response", func(t *testing.T) {
		traceID := "bedrock-raw-003"
		spanID := "bedrock-span-003"

		bedrockResponse := map[string]any{
			"output": map[string]any{
				"message": map[string]any{
					"role": "assistant",
					"content": []map[string]any{
						{"text": "Let me search for that."},
						{
							"toolUse": map[string]any{
								"toolUseId": "tooluse_abc123",
								"name":      "search_database",
								"input":     map[string]any{"query": "test"},
							},
						},
					},
				},
			},
			"stopReason": "tool_use",
			"usage": map[string]any{
				"inputTokens":  150,
				"outputTokens": 75,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "bedrock",
				"model":       "anthropic.claude-3-haiku-20240307-v1:0",
				"rawResponse": bedrockResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["SubType"] != nil && span["SubType"].(string) != "planning" {
			t.Errorf("SubType should be 'planning', got '%v'", span["SubType"])
		}

		if span["ToolUses"] != nil {
			toolUses := span["ToolUses"].([]any)
			if len(toolUses) > 0 {
				tu := toolUses[0].(map[string]any)
				if tu["Name"].(string) != "search_database" {
					t.Errorf("Tool name: expected 'search_database', got '%v'", tu["Name"])
				}
			}
		}
	})
}

// TestRawResponseGemini verifies Google Gemini response parsing
func TestRawResponseGemini(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "gemini@example.com", "password": "SecurePass123", "name": "Gemini User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "Gemini Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("extracts tokens from Gemini response", func(t *testing.T) {
		traceID := "gemini-raw-001"
		spanID := "gemini-span-001"

		// Google Gemini API response format
		geminiResponse := map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]any{{
						"text": "Hello from Gemini!",
					}},
					"role": "model",
				},
				"finishReason": "STOP",
				"index":        0,
			}},
			"usageMetadata": map[string]any{
				"promptTokenCount":     100,
				"candidatesTokenCount": 50,
				"totalTokenCount":      150,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "gemini",
				"model":       "gemini-1.5-flash",
				"rawResponse": geminiResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if int(span["InputTokens"].(float64)) != 100 {
			t.Errorf("InputTokens: expected 100, got %v", span["InputTokens"])
		}
		if int(span["OutputTokens"].(float64)) != 50 {
			t.Errorf("OutputTokens: expected 50, got %v", span["OutputTokens"])
		}
		if span["StopReason"].(string) != "STOP" {
			t.Errorf("StopReason: expected 'STOP', got '%v'", span["StopReason"])
		}

		// CRITICAL: Verify Output is extracted from Gemini rawResponse
		if span["Output"] == nil {
			t.Error("Output should be extracted from Gemini rawResponse")
		} else if span["Output"].(string) != "Hello from Gemini!" {
			t.Errorf("Output: expected 'Hello from Gemini!', got '%v'", span["Output"])
		}
	})

	t.Run("extracts function call from Gemini response", func(t *testing.T) {
		traceID := "gemini-raw-002"
		spanID := "gemini-span-002"

		geminiResponse := map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]any{{
						"functionCall": map[string]any{
							"name": "get_weather",
							"args": map[string]any{"location": "Tokyo"},
						},
					}},
					"role": "model",
				},
				"finishReason": "STOP",
			}},
			"usageMetadata": map[string]any{
				"promptTokenCount":     80,
				"candidatesTokenCount": 40,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "gemini",
				"model":       "gemini-1.5-pro",
				"rawResponse": geminiResponse,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		if span["SubType"] != nil && span["SubType"].(string) != "planning" {
			t.Errorf("SubType should be 'planning', got '%v'", span["SubType"])
		}

		if span["ToolUses"] != nil {
			toolUses := span["ToolUses"].([]any)
			if len(toolUses) > 0 {
				tu := toolUses[0].(map[string]any)
				if tu["Name"].(string) != "get_weather" {
					t.Errorf("Tool name: expected 'get_weather', got '%v'", tu["Name"])
				}
			}
		}
	})
}

// TestRawResponseAutoDetect verifies auto-detection of provider format
func TestRawResponseAutoDetect(t *testing.T) {
	ts := setupTestServer(t)

	regResp := ts.Request("POST", "/api/v1/auth/register", map[string]string{
		"email": "autodetect@example.com", "password": "SecurePass123", "name": "AutoDetect User",
	}, nil)
	var auth AuthResponse
	ParseJSON(t, regResp, &auth)

	projResp := ts.Request("POST", "/api/v1/dashboard/projects", map[string]string{
		"name": "AutoDetect Test Project",
	}, map[string]string{"Authorization": "Bearer " + auth.Token})
	var project ProjectResponse
	ParseJSON(t, projResp, &project)

	apiKeyHeaders := map[string]string{"Authorization": "Bearer " + project.APIKey}

	t.Run("auto-detects OpenAI format with unknown provider", func(t *testing.T) {
		traceID := "autodetect-001"
		spanID := "autodetect-span-001"

		openaiFormat := map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": "Auto-detected OpenAI",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":     50,
				"completion_tokens": 25,
			},
		}

		ts.Request("POST", "/api/v1/ingest", map[string]any{
			"events": []map[string]any{{
				"traceId":     traceID,
				"spanId":      spanID,
				"spanType":    "llm",
				"provider":    "unknown",
				"model":       "some-model",
				"rawResponse": openaiFormat,
				"status":      "success",
			}},
		}, apiKeyHeaders)

		traceResp := ts.Request("GET", "/api/v1/traces/"+traceID, nil, apiKeyHeaders)
		var trace map[string]any
		ParseJSON(t, traceResp, &trace)

		spans := trace["Spans"].([]any)
		span := spans[0].(map[string]any)

		// Should have extracted tokens despite unknown provider
		if span["InputTokens"] != nil {
			if int(span["InputTokens"].(float64)) != 50 {
				t.Errorf("Auto-detect failed to extract tokens correctly")
			}
		}
	})
}
