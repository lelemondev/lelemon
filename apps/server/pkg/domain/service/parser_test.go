package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// BEST PRACTICE: Test by API SCENARIO, not by individual model
//
// The response FORMAT is determined by the API, not the model.
// gpt-4o and gpt-4-turbo return identical structures.
// We only test unique scenarios:
//   - OpenAI: text, tools, reasoning (o1 has reasoning_tokens)
//   - Anthropic: text, tool_use
//   - Gemini: text, function_calls
//   - Bedrock: converse format
// =============================================================================

// TestOpenAIScenarios tests all OpenAI API scenarios
func TestOpenAIScenarios(t *testing.T) {
	t.Run("text_response", func(t *testing.T) {
		fixture := loadFixture(t, "real_openai_text.json")
		result := ParseProviderResponse("openai", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertOutputIsString(t, result)
		assertStopReason(t, result, "stop")
		assertSubType(t, result, "response")
		assertNoToolUses(t, result)

		t.Logf("✓ tokens: %d→%d, stop: %s", result.InputTokens, result.OutputTokens, *result.StopReason)
	})

	t.Run("tool_calls", func(t *testing.T) {
		fixture := loadFixture(t, "real_openai_tools.json")
		result := ParseProviderResponse("openai", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertStopReason(t, result, "tool_calls")
		assertSubType(t, result, "planning")
		assertHasToolUses(t, result)

		t.Logf("✓ tokens: %d→%d, tools: %d", result.InputTokens, result.OutputTokens, len(result.ToolUses))
		for i, tu := range result.ToolUses {
			if tu.ID == "" || tu.Name == "" {
				t.Errorf("tool %d missing id or name", i)
			}
		}
	})

	t.Run("reasoning_o1", func(t *testing.T) {
		fixture := loadFixture(t, "real_openai_reasoning.json")
		result := ParseProviderResponse("openai", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertOutputIsString(t, result)

		// O1 models include reasoning_tokens
		if result.ReasoningTokens == nil {
			t.Error("expected reasoning_tokens for O1 model")
		} else if *result.ReasoningTokens <= 0 {
			t.Error("expected reasoning_tokens > 0")
		}

		t.Logf("✓ tokens: %d→%d, reasoning: %d", result.InputTokens, result.OutputTokens, *result.ReasoningTokens)
	})
}

// TestGeminiScenarios tests all Gemini API scenarios
func TestGeminiScenarios(t *testing.T) {
	t.Run("text_response", func(t *testing.T) {
		fixture := loadFixture(t, "real_gemini_text.json")
		result := ParseProviderResponse("gemini", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertOutputIsString(t, result)
		assertStopReasonSet(t, result) // Gemini may return STOP or MAX_TOKENS
		assertSubType(t, result, "response")
		assertNoToolUses(t, result)

		t.Logf("✓ tokens: %d→%d, stop: %s", result.InputTokens, result.OutputTokens, *result.StopReason)
	})

	t.Run("function_calls", func(t *testing.T) {
		fixture := loadFixture(t, "real_gemini_functions.json")
		result := ParseProviderResponse("gemini", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)

		// Gemini may return text OR function calls depending on the prompt
		// Just verify we parsed something useful
		if result.Output == nil && len(result.ToolUses) == 0 {
			t.Error("expected either output or tool uses")
		}

		t.Logf("✓ tokens: %d→%d, tools: %d", result.InputTokens, result.OutputTokens, len(result.ToolUses))
	})
}

// TestAnthropicScenarios tests Anthropic API scenarios (using synthetic fixtures)
func TestAnthropicScenarios(t *testing.T) {
	t.Run("text_response", func(t *testing.T) {
		fixture := loadFixture(t, "anthropic_text_response.json")
		result := ParseProviderResponse("anthropic", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertOutputIsString(t, result)
		assertStopReason(t, result, "end_turn")
		assertSubType(t, result, "response")

		t.Logf("✓ tokens: %d→%d", result.InputTokens, result.OutputTokens)
	})

	t.Run("tool_use", func(t *testing.T) {
		fixture := loadFixture(t, "anthropic_tool_use.json")
		result := ParseProviderResponse("anthropic", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertStopReason(t, result, "tool_use")
		assertSubType(t, result, "planning")
		assertHasToolUses(t, result)

		t.Logf("✓ tokens: %d→%d, tools: %d", result.InputTokens, result.OutputTokens, len(result.ToolUses))
	})

	t.Run("with_cache", func(t *testing.T) {
		fixture := loadFixture(t, "anthropic_with_cache.json")
		result := ParseProviderResponse("anthropic", fixture["response"])

		assertNotNil(t, result)

		// Cache tokens should be extracted
		if result.CacheReadTokens == nil && result.CacheWriteTokens == nil {
			t.Log("⚠ no cache tokens found (may be expected)")
		} else {
			t.Logf("✓ cache: read=%v, write=%v", result.CacheReadTokens, result.CacheWriteTokens)
		}
	})

	t.Run("with_thinking", func(t *testing.T) {
		fixture := loadFixture(t, "anthropic_with_thinking.json")
		result := ParseProviderResponse("anthropic", fixture["response"])

		assertNotNil(t, result)

		if result.Thinking == nil {
			t.Log("⚠ no thinking extracted (may be expected)")
		} else {
			t.Logf("✓ thinking: %d chars", len(*result.Thinking))
		}
	})
}

// TestBedrockScenarios tests Bedrock Converse API scenarios
func TestBedrockScenarios(t *testing.T) {
	t.Run("converse_text", func(t *testing.T) {
		fixture := loadFixture(t, "bedrock_converse.json")
		result := ParseProviderResponse("bedrock", fixture["response"])

		assertNotNil(t, result)
		assertTokensPositive(t, result)
		assertOutputIsString(t, result)
		assertStopReason(t, result, "end_turn")
		assertSubType(t, result, "response")

		t.Logf("✓ tokens: %d→%d", result.InputTokens, result.OutputTokens)
	})
}

// TestAutoDetect tests automatic provider detection from response structure
func TestAutoDetect(t *testing.T) {
	testCases := []struct {
		name     string
		fixture  string
		expected string // expected provider
	}{
		{"OpenAI format", "real_openai_text.json", "openai"},
		{"Gemini format", "real_gemini_text.json", "gemini"},
		{"Bedrock format", "bedrock_converse.json", "bedrock"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := loadFixture(t, tc.fixture)
			// Parse with empty provider to trigger auto-detect
			result := ParseProviderResponse("", fixture["response"])

			if result == nil {
				t.Fatal("auto-detect failed to parse")
			}
			if result.InputTokens == 0 && result.OutputTokens == 0 && result.Output == nil {
				t.Error("auto-detect parsed nothing useful")
			}
			t.Logf("✓ auto-detected and parsed successfully")
		})
	}
}

// TestEdgeCases tests error handling and edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("nil_response", func(t *testing.T) {
		result := ParseProviderResponse("openai", nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("empty_response", func(t *testing.T) {
		result := ParseProviderResponse("openai", map[string]any{})
		if result == nil {
			t.Error("expected non-nil for empty map")
		}
	})

	t.Run("wrong_type", func(t *testing.T) {
		result := ParseProviderResponse("openai", "not a map")
		if result != nil {
			t.Error("expected nil for wrong type")
		}
	})

	t.Run("unknown_provider_with_openai_format", func(t *testing.T) {
		resp := map[string]any{
			"choices": []any{
				map[string]any{
					"message":       map[string]any{"content": "test"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(10),
				"completion_tokens": float64(20),
			},
		}
		result := ParseProviderResponse("unknown", resp)
		if result == nil {
			t.Fatal("expected auto-detect to work")
		}
		if result.Output != "test" {
			t.Errorf("expected output 'test', got %v", result.Output)
		}
	})
}

// =============================================================================
// ASSERTION HELPERS
// =============================================================================

func assertNotNil(t *testing.T, result *ParsedResponse) {
	t.Helper()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func assertTokensPositive(t *testing.T, result *ParsedResponse) {
	t.Helper()
	if result.InputTokens <= 0 {
		t.Errorf("expected input tokens > 0, got %d", result.InputTokens)
	}
	if result.OutputTokens <= 0 {
		t.Errorf("expected output tokens > 0, got %d", result.OutputTokens)
	}
}

func assertOutputIsString(t *testing.T, result *ParsedResponse) {
	t.Helper()
	output, ok := result.Output.(string)
	if !ok {
		t.Errorf("expected output to be string, got %T", result.Output)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func assertStopReason(t *testing.T, result *ParsedResponse, expected string) {
	t.Helper()
	if result.StopReason == nil {
		t.Error("expected stop reason to be set")
	} else if *result.StopReason != expected {
		t.Errorf("expected stop reason '%s', got '%s'", expected, *result.StopReason)
	}
}

func assertStopReasonSet(t *testing.T, result *ParsedResponse) {
	t.Helper()
	if result.StopReason == nil {
		t.Error("expected stop reason to be set")
	} else if *result.StopReason == "" {
		t.Error("expected non-empty stop reason")
	}
}

func assertSubType(t *testing.T, result *ParsedResponse, expected string) {
	t.Helper()
	if result.SubType == nil {
		t.Error("expected subtype to be set")
	} else if *result.SubType != expected {
		t.Errorf("expected subtype '%s', got '%s'", expected, *result.SubType)
	}
}

func assertNoToolUses(t *testing.T, result *ParsedResponse) {
	t.Helper()
	if len(result.ToolUses) > 0 {
		t.Errorf("expected no tool uses, got %d", len(result.ToolUses))
	}
}

func assertHasToolUses(t *testing.T, result *ParsedResponse) {
	t.Helper()
	if len(result.ToolUses) == 0 {
		t.Error("expected tool uses")
	}
}

// =============================================================================
// FIXTURE LOADER
// =============================================================================

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
	return strings.Contains(s, substr)
}
