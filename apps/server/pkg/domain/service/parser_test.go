package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestParserWithRealFixtures tests the parser using real API responses
func TestParserWithRealFixtures(t *testing.T) {
	t.Run("OpenAI Multi-turn", func(t *testing.T) {
		fixture := loadFixture(t, "real_openai_multi_turn.json")
		response := fixture["response"]

		result := ParseProviderResponse("openai", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		// Verify tokens
		if result.InputTokens == 0 {
			t.Error("expected input tokens > 0")
		}
		if result.OutputTokens == 0 {
			t.Error("expected output tokens > 0")
		}

		// Verify output is a string (text response)
		output, ok := result.Output.(string)
		if !ok {
			t.Errorf("expected output to be string, got %T", result.Output)
		}
		if output == "" {
			t.Error("expected non-empty output")
		}

		// Verify stop reason
		if result.StopReason == nil {
			t.Error("expected stop reason")
		} else if *result.StopReason != "stop" {
			t.Errorf("expected stop reason 'stop', got '%s'", *result.StopReason)
		}

		// Verify subtype is response (no tool calls)
		if result.SubType == nil {
			t.Error("expected subtype")
		} else if *result.SubType != "response" {
			t.Errorf("expected subtype 'response', got '%s'", *result.SubType)
		}

		// Should not have tool uses
		if len(result.ToolUses) > 0 {
			t.Errorf("expected no tool uses, got %d", len(result.ToolUses))
		}

		t.Logf("Parsed: input=%d, output=%d, stopReason=%s",
			result.InputTokens, result.OutputTokens, *result.StopReason)
	})

	t.Run("OpenAI Parallel Tools", func(t *testing.T) {
		fixture := loadFixture(t, "real_openai_parallel_tools.json")
		response := fixture["response"]

		result := ParseProviderResponse("openai", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		// Verify tokens
		if result.InputTokens == 0 {
			t.Error("expected input tokens > 0")
		}
		if result.OutputTokens == 0 {
			t.Error("expected output tokens > 0")
		}

		// Verify stop reason is tool_calls
		if result.StopReason == nil {
			t.Error("expected stop reason")
		} else if *result.StopReason != "tool_calls" {
			t.Errorf("expected stop reason 'tool_calls', got '%s'", *result.StopReason)
		}

		// Verify tool uses are extracted
		if len(result.ToolUses) == 0 {
			t.Error("expected tool uses")
		} else {
			t.Logf("Found %d tool uses", len(result.ToolUses))
			for i, tu := range result.ToolUses {
				if tu.ID == "" {
					t.Errorf("tool use %d has empty ID", i)
				}
				if tu.Name == "" {
					t.Errorf("tool use %d has empty name", i)
				}
				t.Logf("  Tool %d: id=%s, name=%s", i, tu.ID, tu.Name)
			}
		}

		// Verify subtype is planning (has tool calls)
		if result.SubType == nil {
			t.Error("expected subtype")
		} else if *result.SubType != "planning" {
			t.Errorf("expected subtype 'planning', got '%s'", *result.SubType)
		}
	})

	t.Run("OpenAI Structured Output", func(t *testing.T) {
		fixture := loadFixture(t, "real_openai_structured.json")
		response := fixture["response"]

		result := ParseProviderResponse("openai", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		// Verify tokens
		if result.InputTokens == 0 {
			t.Error("expected input tokens > 0")
		}
		if result.OutputTokens == 0 {
			t.Error("expected output tokens > 0")
		}

		// Output should be a JSON string
		output, ok := result.Output.(string)
		if !ok {
			t.Errorf("expected output to be string, got %T", result.Output)
		} else {
			// Verify it's valid JSON
			var parsed map[string]any
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				t.Errorf("output is not valid JSON: %v", err)
			}
		}

		// Verify stop reason
		if result.StopReason == nil {
			t.Error("expected stop reason")
		} else if *result.StopReason != "stop" {
			t.Errorf("expected stop reason 'stop', got '%s'", *result.StopReason)
		}

		// Verify subtype is response
		if result.SubType == nil {
			t.Error("expected subtype")
		} else if *result.SubType != "response" {
			t.Errorf("expected subtype 'response', got '%s'", *result.SubType)
		}
	})
}

// TestParserWithSyntheticFixtures tests the parser using synthetic fixtures
func TestParserWithSyntheticFixtures(t *testing.T) {
	t.Run("OpenAI Chat Completion", func(t *testing.T) {
		fixture := loadFixture(t, "openai_chat_completion.json")
		response := fixture["response"]

		result := ParseProviderResponse("openai", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		expected := fixture["expected"].(map[string]any)

		// Verify input tokens
		if expectedTokens, ok := expected["input_tokens"].(float64); ok {
			if result.InputTokens != int(expectedTokens) {
				t.Errorf("expected input tokens %d, got %d", int(expectedTokens), result.InputTokens)
			}
		}

		// Verify output tokens
		if expectedTokens, ok := expected["output_tokens"].(float64); ok {
			if result.OutputTokens != int(expectedTokens) {
				t.Errorf("expected output tokens %d, got %d", int(expectedTokens), result.OutputTokens)
			}
		}

		// Verify stop reason
		if expectedStop, ok := expected["stop_reason"].(string); ok {
			if result.StopReason == nil || *result.StopReason != expectedStop {
				t.Errorf("expected stop reason '%s', got '%v'", expectedStop, result.StopReason)
			}
		}

		// Verify subtype
		if expectedSubType, ok := expected["sub_type"].(string); ok {
			if result.SubType == nil || *result.SubType != expectedSubType {
				t.Errorf("expected subtype '%s', got '%v'", expectedSubType, result.SubType)
			}
		}
	})

	t.Run("OpenAI Tool Calls", func(t *testing.T) {
		fixture := loadFixture(t, "openai_tool_calls.json")
		response := fixture["response"]

		result := ParseProviderResponse("openai", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		expected := fixture["expected"].(map[string]any)

		// Verify has tool uses
		if expectedHasTools, ok := expected["has_tool_uses"].(bool); ok && expectedHasTools {
			if len(result.ToolUses) == 0 {
				t.Error("expected tool uses")
			}
		}

		// Verify tool uses count
		if expectedCount, ok := expected["tool_uses_count"].(float64); ok {
			if len(result.ToolUses) != int(expectedCount) {
				t.Errorf("expected %d tool uses, got %d", int(expectedCount), len(result.ToolUses))
			}
		}

		// Verify individual tool uses
		if expectedTools, ok := expected["tool_uses"].([]any); ok {
			for i, expTool := range expectedTools {
				if i >= len(result.ToolUses) {
					break
				}
				expToolMap := expTool.(map[string]any)
				actualTool := result.ToolUses[i]

				if expID, ok := expToolMap["id"].(string); ok {
					if actualTool.ID != expID {
						t.Errorf("tool %d: expected id '%s', got '%s'", i, expID, actualTool.ID)
					}
				}
				if expName, ok := expToolMap["name"].(string); ok {
					if actualTool.Name != expName {
						t.Errorf("tool %d: expected name '%s', got '%s'", i, expName, actualTool.Name)
					}
				}
			}
		}

		// Verify subtype is planning
		if result.SubType == nil || *result.SubType != "planning" {
			t.Errorf("expected subtype 'planning', got '%v'", result.SubType)
		}
	})

	t.Run("OpenAI with Reasoning Tokens", func(t *testing.T) {
		fixture := loadFixture(t, "openai_with_reasoning.json")
		response := fixture["response"]

		result := ParseProviderResponse("openai", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		expected := fixture["expected"].(map[string]any)

		// Verify reasoning tokens
		if expectedReasoning, ok := expected["reasoning_tokens"].(float64); ok {
			if result.ReasoningTokens == nil {
				t.Error("expected reasoning tokens to be set")
			} else if *result.ReasoningTokens != int(expectedReasoning) {
				t.Errorf("expected reasoning tokens %d, got %d", int(expectedReasoning), *result.ReasoningTokens)
			}
		}

		// Verify output contains expected text
		if expectedContains, ok := expected["output_contains"].(string); ok {
			output, ok := result.Output.(string)
			if !ok || output == "" {
				t.Error("expected non-empty string output")
			} else if !containsSubstring(output, expectedContains) {
				t.Errorf("output does not contain '%s'", expectedContains)
			}
		}
	})

	t.Run("Bedrock Converse", func(t *testing.T) {
		fixture := loadFixture(t, "bedrock_converse.json")
		response := fixture["response"]

		result := ParseProviderResponse("bedrock", response)
		if result == nil {
			t.Fatal("expected non-nil result")
		}

		expected := fixture["expected"].(map[string]any)

		// Verify input tokens
		if expectedTokens, ok := expected["input_tokens"].(float64); ok {
			if result.InputTokens != int(expectedTokens) {
				t.Errorf("expected input tokens %d, got %d", int(expectedTokens), result.InputTokens)
			}
		}

		// Verify output tokens
		if expectedTokens, ok := expected["output_tokens"].(float64); ok {
			if result.OutputTokens != int(expectedTokens) {
				t.Errorf("expected output tokens %d, got %d", int(expectedTokens), result.OutputTokens)
			}
		}

		// Verify output contains expected text
		if expectedContains, ok := expected["output_contains"].(string); ok {
			output, ok := result.Output.(string)
			if !ok || output == "" {
				t.Error("expected non-empty string output")
			} else if !containsSubstring(output, expectedContains) {
				t.Errorf("output does not contain '%s'", expectedContains)
			}
		}
	})
}

// TestAutoDetect tests automatic provider detection
func TestAutoDetect(t *testing.T) {
	testCases := []struct {
		name     string
		fixture  string
		provider string
	}{
		{"OpenAI auto-detect", "openai_chat_completion.json", ""},
		{"Bedrock auto-detect", "bedrock_converse.json", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := loadFixture(t, tc.fixture)
			response := fixture["response"]

			// Parse with empty provider (auto-detect)
			result := ParseProviderResponse(tc.provider, response)
			if result == nil {
				t.Fatal("expected auto-detect to work")
			}

			// Should have parsed something meaningful
			if result.InputTokens == 0 && result.OutputTokens == 0 && result.Output == nil {
				t.Error("auto-detect failed to parse any data")
			}
		})
	}
}

// TestParserEdgeCases tests edge cases
func TestParserEdgeCases(t *testing.T) {
	t.Run("nil response", func(t *testing.T) {
		result := ParseProviderResponse("openai", nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("empty response", func(t *testing.T) {
		result := ParseProviderResponse("openai", map[string]any{})
		if result == nil {
			t.Error("expected non-nil for empty map")
		}
	})

	t.Run("wrong type response", func(t *testing.T) {
		result := ParseProviderResponse("openai", "not a map")
		if result != nil {
			t.Error("expected nil for wrong type")
		}
	})

	t.Run("unknown provider with auto-detect", func(t *testing.T) {
		// Should try auto-detect
		openAIResponse := map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{
						"content": "test",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(10),
				"completion_tokens": float64(20),
			},
		}
		result := ParseProviderResponse("unknown", openAIResponse)
		if result == nil {
			t.Error("expected auto-detect to work for unknown provider")
		}
		if result.Output != "test" {
			t.Errorf("expected output 'test', got '%v'", result.Output)
		}
	})
}

// Helper functions

func loadFixture(t *testing.T, filename string) map[string]any {
	t.Helper()
	path := filepath.Join("testdata", "fixtures", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", filename, err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("failed to parse fixture %s: %v", filename, err)
	}

	return fixture
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || containsSubstring(s[1:], substr)))
}
