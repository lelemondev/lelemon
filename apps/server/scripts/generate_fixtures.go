//go:build ignore

// Fixture Generator - Best Practice: One fixture per API SCENARIO, not per model
//
// The response FORMAT is determined by the API, not the model.
// gpt-4o and gpt-4-turbo return identical structures.
// We only need separate fixtures when there's a STRUCTURAL difference.
//
// Run: go run scripts/generate_fixtures.go
//
// Required environment variables:
//   ANTHROPIC_API_KEY - Anthropic API key
//   OPENAI_API_KEY    - OpenAI API key
//   GOOGLE_API_KEY    - Google Gemini API key

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const fixturesDir = "pkg/domain/service/testdata/fixtures"

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║     FIXTURE GENERATOR - One per Scenario (Best Practice)   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Strategy: Test API FORMAT, not individual models")
	fmt.Println()

	if err := os.MkdirAll(fixturesDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	// Anthropic - 2 scenarios: text, tool_use
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		fmt.Println("🔮 ANTHROPIC (claude-sonnet-4)")
		fmt.Println("─────────────────────────────────────────────")
		generateAnthropicScenarios(key)
	} else {
		fmt.Println("⏭️  Skipping Anthropic (ANTHROPIC_API_KEY not set)")
	}

	// OpenAI - 3 scenarios: text, tools, reasoning
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		fmt.Println("\n🤖 OPENAI")
		fmt.Println("─────────────────────────────────────────────")
		generateOpenAIScenarios(key)
	} else {
		fmt.Println("\n⏭️  Skipping OpenAI (OPENAI_API_KEY not set)")
	}

	// Gemini - 2 scenarios: text, functions
	if key := os.Getenv("GOOGLE_API_KEY"); key != "" {
		fmt.Println("\n💎 GEMINI (gemini-2.5-pro)")
		fmt.Println("─────────────────────────────────────────────")
		generateGeminiScenarios(key)
	} else {
		fmt.Println("\n⏭️  Skipping Gemini (GOOGLE_API_KEY not set)")
	}

	fmt.Println("\n════════════════════════════════════════════════════════════")
	fmt.Println("✅ Done! Fixtures saved to:", fixturesDir)
	fmt.Println()
	fmt.Println("Scenarios covered:")
	fmt.Println("  • OpenAI: text, tools, reasoning (o1)")
	fmt.Println("  • Anthropic: text, tool_use")
	fmt.Println("  • Gemini: text, function_calls")
	fmt.Println("  • Bedrock: synthetic fixture (no AWS access needed)")
}

// ============================================================================
// ANTHROPIC - 2 scenarios
// ============================================================================

func generateAnthropicScenarios(apiKey string) {
	baseURL := "https://api.anthropic.com/v1/messages"
	model := "claude-sonnet-4-20250514" // Latest model

	// Scenario 1: Text response
	fmt.Println("  📝 Scenario: text_response")
	req1 := map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"system":     "You are a senior software architect. Be concise but thorough.",
		"messages": []map[string]any{
			{"role": "user", "content": "Explain microservices vs monolithic architecture. Include pros/cons."},
		},
	}
	if resp, err := callAnthropic(apiKey, baseURL, req1); err == nil {
		saveFixture("real_anthropic_text.json", map[string]any{
			"_description": "Anthropic Messages API - Text response",
			"_scenario":    "text_response",
			"_model":       model,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req1,
			"response":     resp,
		})
		fmt.Println("    ✅ real_anthropic_text.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}

	// Scenario 2: Tool use
	fmt.Println("  🔧 Scenario: tool_use")
	req2 := map[string]any{
		"model":      model,
		"max_tokens": 1024,
		"system":     "You are a helpful assistant. Use tools when needed.",
		"tools": []map[string]any{
			{
				"name":        "get_weather",
				"description": "Get current weather for a location",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string", "description": "City name"},
						"units":    map[string]any{"type": "string", "enum": []string{"celsius", "fahrenheit"}},
					},
					"required": []string{"location"},
				},
			},
			{
				"name":        "search_restaurants",
				"description": "Search for restaurants in a location",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string"},
						"cuisine":  map[string]any{"type": "string"},
					},
					"required": []string{"location"},
				},
			},
		},
		"messages": []map[string]any{
			{"role": "user", "content": "What's the weather in Tokyo and find me some good sushi restaurants there."},
		},
	}
	if resp, err := callAnthropic(apiKey, baseURL, req2); err == nil {
		saveFixture("real_anthropic_tools.json", map[string]any{
			"_description": "Anthropic Messages API - Tool use",
			"_scenario":    "tool_use",
			"_model":       model,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req2,
			"response":     resp,
		})
		fmt.Println("    ✅ real_anthropic_tools.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}
}

// ============================================================================
// OPENAI - 3 scenarios
// ============================================================================

func generateOpenAIScenarios(apiKey string) {
	baseURL := "https://api.openai.com/v1/chat/completions"

	// Scenario 1: Text response (gpt-4o)
	fmt.Println("  📝 Scenario: text_response (gpt-4o)")
	req1 := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{"role": "system", "content": "You are a senior software engineer. Be practical and concise."},
			{"role": "user", "content": "What are 5 best practices for error handling in production APIs?"},
		},
		"max_tokens":  800,
		"temperature": 0.7,
	}
	if resp, err := callOpenAI(apiKey, baseURL, req1); err == nil {
		saveFixture("real_openai_text.json", map[string]any{
			"_description": "OpenAI Chat Completions - Text response",
			"_scenario":    "text_response",
			"_model":       "gpt-4o",
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req1,
			"response":     resp,
		})
		fmt.Println("    ✅ real_openai_text.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}

	// Scenario 2: Tool calls (gpt-4o)
	fmt.Println("  🔧 Scenario: tool_calls (gpt-4o)")
	req2 := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{"role": "system", "content": "You help with data analysis. Use tools when appropriate."},
			{"role": "user", "content": "Get sales reports for October, November, and December 2024."},
		},
		"tools": []map[string]any{
			{
				"type": "function",
				"function": map[string]any{
					"name":        "get_sales_report",
					"description": "Get sales report for a specific month",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"month": map[string]any{"type": "string"},
							"year":  map[string]any{"type": "integer"},
						},
						"required": []string{"month", "year"},
					},
				},
			},
		},
		"tool_choice": "auto",
		"max_tokens":  1000,
	}
	if resp, err := callOpenAI(apiKey, baseURL, req2); err == nil {
		saveFixture("real_openai_tools.json", map[string]any{
			"_description": "OpenAI Chat Completions - Tool calls (parallel)",
			"_scenario":    "tool_calls",
			"_model":       "gpt-4o",
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req2,
			"response":     resp,
		})
		fmt.Println("    ✅ real_openai_tools.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}

	// Scenario 3: Reasoning with o1 (different structure: reasoning_tokens)
	fmt.Println("  🧠 Scenario: reasoning (o1)")
	req3 := map[string]any{
		"model": "o1",
		"messages": []map[string]any{
			{"role": "user", "content": "A farmer has 17 sheep. All but 9 run away. How many sheep remain? Think step by step."},
		},
		"max_completion_tokens": 2000,
	}
	if resp, err := callOpenAI(apiKey, baseURL, req3); err == nil {
		saveFixture("real_openai_reasoning.json", map[string]any{
			"_description": "OpenAI Chat Completions - O1 reasoning model",
			"_scenario":    "reasoning",
			"_note":        "O1 models include reasoning_tokens in completion_tokens_details",
			"_model":       "o1",
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req3,
			"response":     resp,
		})
		fmt.Println("    ✅ real_openai_reasoning.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}
}

// ============================================================================
// GEMINI - 2 scenarios
// ============================================================================

func generateGeminiScenarios(apiKey string) {
	model := "gemini-2.5-pro" // Latest model

	// Scenario 1: Text response
	fmt.Println("  📝 Scenario: text_response")
	url1 := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req1 := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": "Explain database indexing: types, when to use each, and trade-offs."}}},
		},
		"generationConfig": map[string]any{"maxOutputTokens": 1500, "temperature": 0.7},
	}
	if resp, err := callGemini(url1, req1); err == nil {
		saveFixture("real_gemini_text.json", map[string]any{
			"_description": "Gemini GenerateContent - Text response",
			"_scenario":    "text_response",
			"_model":       model,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req1,
			"response":     resp,
		})
		fmt.Println("    ✅ real_gemini_text.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}

	// Scenario 2: Function calling
	fmt.Println("  🔧 Scenario: function_calls")
	url2 := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req2 := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": "Order a large pepperoni pizza and a medium hawaiian to 123 Main St. Also check for promotions."}}},
		},
		"tools": []map[string]any{
			{
				"functionDeclarations": []map[string]any{
					{
						"name":        "place_order",
						"description": "Place a pizza order",
						"parameters": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"items":            map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
								"delivery_address": map[string]any{"type": "string"},
							},
							"required": []string{"items", "delivery_address"},
						},
					},
					{
						"name":        "get_promotions",
						"description": "Get current promotions",
						"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
					},
				},
			},
		},
		"generationConfig": map[string]any{"maxOutputTokens": 1024},
	}
	if resp, err := callGemini(url2, req2); err == nil {
		saveFixture("real_gemini_functions.json", map[string]any{
			"_description": "Gemini GenerateContent - Function calling",
			"_scenario":    "function_calls",
			"_model":       model,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req2,
			"response":     resp,
		})
		fmt.Println("    ✅ real_gemini_functions.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}
}

// ============================================================================
// HTTP CLIENTS
// ============================================================================

func callAnthropic(apiKey, url string, request map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(respBody), 150))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}

func callOpenAI(apiKey, url string, request map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(respBody), 150))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}

func callGemini(url string, request map[string]any) (map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(respBody), 150))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}

// ============================================================================
// HELPERS
// ============================================================================

func saveFixture(filename string, data map[string]any) {
	path := filepath.Join(fixturesDir, filename)
	content, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(path, content, 0644)
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
