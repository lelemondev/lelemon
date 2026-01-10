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
//   ANTHROPIC_API_KEY   - Anthropic API key
//   OPENAI_API_KEY      - OpenAI API key
//   GOOGLE_API_KEY      - Google Gemini API key
//   AWS_ACCESS_KEY_ID   - AWS access key (for Bedrock)
//   AWS_SECRET_ACCESS_KEY - AWS secret key (for Bedrock)
//   AWS_REGION          - AWS region (defaults to us-east-1)

package main

import (
	"bufio"
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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
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

	// Bedrock - 2 scenarios: text, tool_use
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		fmt.Println("\n☁️  BEDROCK (claude-3-5-sonnet)")
		fmt.Println("─────────────────────────────────────────────")
		generateBedrockScenarios()
	} else {
		fmt.Println("\n⏭️  Skipping Bedrock (AWS credentials not set)")
	}

	fmt.Println("\n════════════════════════════════════════════════════════════")
	fmt.Println("✅ Done! Fixtures saved to:", fixturesDir)
	fmt.Println()
	fmt.Println("Scenarios covered:")
	fmt.Println("  • OpenAI: text, tools, reasoning (o1)")
	fmt.Println("  • Anthropic: text, tool_use")
	fmt.Println("  • Gemini: text, function_calls")
	fmt.Println("  • Bedrock: text, tool_use")
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
// GEMINI - 4 scenarios (text, functions, streaming, live)
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

	// Scenario 3: Streaming response (SSE)
	fmt.Println("  📡 Scenario: streaming")
	streamModel := "gemini-2.0-flash" // Use flash for faster streaming
	url3 := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", streamModel, apiKey)
	req3 := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]any{{"text": "Count from 1 to 5, explaining each number briefly."}}},
		},
		"generationConfig": map[string]any{"maxOutputTokens": 500},
	}
	if chunks, aggregated, err := callGeminiStreaming(url3, req3); err == nil {
		saveFixture("real_gemini_streaming.json", map[string]any{
			"_description": "Gemini streamGenerateContent - SSE streaming response",
			"_scenario":    "streaming",
			"_note":        "Contains both individual chunks and aggregated response",
			"_model":       streamModel,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      req3,
			"chunks":       chunks,
			"aggregated":   aggregated,
		})
		fmt.Println("    ✅ real_gemini_streaming.json")
	} else {
		fmt.Printf("    ❌ Failed: %v\n", err)
	}

	// Scenario 4: Gemini Live message format (synthetic - WebSocket based)
	fmt.Println("  🎙️  Scenario: live (synthetic)")
	generateGeminiLiveFixture()
	fmt.Println("    ✅ gemini_live.json")
}

// generateGeminiLiveFixture creates a synthetic fixture for Gemini Live API
// Based on the LiveServerMessage format used in WebSocket connections
func generateGeminiLiveFixture() {
	// Gemini Live uses WebSocket, so we create synthetic fixtures based on the API spec
	liveModel := "gemini-2.5-flash-native-audio-preview-12-2025"

	// Example: Text turn with transcriptions
	textTurnMessage := map[string]any{
		"serverContent": map[string]any{
			"modelTurn": map[string]any{
				"parts": []map[string]any{
					{"text": "Hello! I'm here to help you. What would you like to know?"},
				},
			},
			"outputTranscription": map[string]any{
				"text": "Hello! I'm here to help you. What would you like to know?",
			},
			"turnComplete": true,
		},
	}

	// Example: Audio turn with transcriptions
	audioTurnMessage := map[string]any{
		"serverContent": map[string]any{
			"modelTurn": map[string]any{
				"parts": []map[string]any{
					{
						"inlineData": map[string]any{
							"mimeType": "audio/pcm;rate=24000",
							"data":     "SGVsbG8gV29ybGQh", // base64 placeholder
						},
					},
				},
			},
			"inputTranscription": map[string]any{
				"text": "What's the weather like today?",
			},
			"outputTranscription": map[string]any{
				"text": "I don't have access to real-time weather data, but I can help you find that information.",
			},
			"turnComplete": true,
		},
	}

	// Example: Tool call in live session
	toolCallMessage := map[string]any{
		"toolCall": map[string]any{
			"functionCalls": []map[string]any{
				{
					"id":   "call_abc123",
					"name": "get_weather",
					"args": map[string]any{
						"location": "Seattle",
						"units":    "celsius",
					},
				},
			},
		},
	}

	// Example: Interruption
	interruptionMessage := map[string]any{
		"serverContent": map[string]any{
			"interrupted": true,
		},
	}

	// Example: Session resumption update
	resumptionMessage := map[string]any{
		"sessionResumptionUpdate": map[string]any{
			"resumable": true,
			"newHandle": "CiQxMjM0NTY3OC1hYmNkLTEyMzQtYWJjZC0xMjM0NTY3ODkwYWI",
		},
	}

	// Example: GoAway (server shutdown warning)
	goAwayMessage := map[string]any{
		"goAway": map[string]any{
			"timeLeft": "30s",
		},
	}

	saveFixture("gemini_live.json", map[string]any{
		"_description": "Gemini Live API - WebSocket message formats",
		"_scenario":    "live_streaming",
		"_note":        "Synthetic fixture based on LiveServerMessage format. Gemini Live uses WebSocket, not HTTP.",
		"_model":       liveModel,
		"_captured":    time.Now().Format(time.RFC3339),
		"messages": map[string]any{
			"text_turn":         textTurnMessage,
			"audio_turn":        audioTurnMessage,
			"tool_call":         toolCallMessage,
			"interruption":      interruptionMessage,
			"session_resumption": resumptionMessage,
			"go_away":           goAwayMessage,
		},
		"session_config": map[string]any{
			"responseModalities": []string{"AUDIO"},
			"speechConfig": map[string]any{
				"voiceConfig": map[string]any{
					"prebuiltVoiceConfig": map[string]any{
						"voiceName": "Kore",
					},
				},
			},
			"realtimeInputConfig": map[string]any{
				"automaticActivityDetection": map[string]any{
					"disabled":                 false,
					"startOfSpeechSensitivity": "START_SENSITIVITY_HIGH",
					"endOfSpeechSensitivity":   "END_SENSITIVITY_HIGH",
					"silenceDurationMs":        500,
				},
			},
			"inputAudioTranscription":  map[string]any{},
			"outputAudioTranscription": map[string]any{},
		},
	})
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

// callGeminiStreaming calls the streaming endpoint and returns individual chunks + aggregated response
func callGeminiStreaming(url string, request map[string]any) ([]map[string]any, map[string]any, error) {
	body, _ := json.Marshal(request)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(string(respBody), 150))
	}

	// Parse SSE stream
	var chunks []map[string]any
	var aggregatedText strings.Builder
	var totalInputTokens, totalOutputTokens int
	var lastFinishReason string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk map[string]any
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		chunks = append(chunks, chunk)

		// Extract text from candidates
		if candidates, ok := chunk["candidates"].([]any); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]any); ok {
				if content, ok := candidate["content"].(map[string]any); ok {
					if parts, ok := content["parts"].([]any); ok {
						for _, part := range parts {
							if p, ok := part.(map[string]any); ok {
								if text, ok := p["text"].(string); ok {
									aggregatedText.WriteString(text)
								}
							}
						}
					}
				}
				if fr, ok := candidate["finishReason"].(string); ok {
					lastFinishReason = fr
				}
			}
		}

		// Extract usage from the last chunk
		if usage, ok := chunk["usageMetadata"].(map[string]any); ok {
			if pt, ok := usage["promptTokenCount"].(float64); ok {
				totalInputTokens = int(pt)
			}
			if ct, ok := usage["candidatesTokenCount"].(float64); ok {
				totalOutputTokens = int(ct)
			}
		}
	}

	// Build aggregated response in standard Gemini format
	aggregated := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"text": aggregatedText.String()},
					},
					"role": "model",
				},
				"finishReason": lastFinishReason,
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     totalInputTokens,
			"candidatesTokenCount": totalOutputTokens,
			"totalTokenCount":      totalInputTokens + totalOutputTokens,
		},
	}

	return chunks, aggregated, nil
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

// ============================================================================
// BEDROCK - 2 scenarios (using AWS SDK)
// ============================================================================

func generateBedrockScenarios() {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	// Use inference profile for cross-region inference
	model := "us.anthropic.claude-3-5-sonnet-20241022-v2:0"

	// Create Bedrock client with explicit credentials
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		fmt.Printf("  ❌ Failed to load AWS config: %v\n", err)
		return
	}
	client := bedrockruntime.NewFromConfig(cfg)

	// Scenario 1: Text response
	fmt.Println("  📝 Scenario: text_response")
	input1 := &bedrockruntime.ConverseInput{
		ModelId: aws.String(model),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "Explain containerization vs virtualization. Be concise."},
				},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens:   aws.Int32(1024),
			Temperature: aws.Float32(0.7),
		},
	}

	ctx := context.Background()
	resp1, err := client.Converse(ctx, input1)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
	} else {
		// Convert to map for fixture
		respMap := converseOutputToMap(resp1)
		reqMap := map[string]any{
			"messages": []map[string]any{
				{"role": "user", "content": []map[string]any{{"text": "Explain containerization vs virtualization. Be concise."}}},
			},
			"inferenceConfig": map[string]any{"maxTokens": 1024, "temperature": 0.7},
		}
		saveFixture("real_bedrock_text.json", map[string]any{
			"_description": "Bedrock Converse API - Text response",
			"_scenario":    "text_response",
			"_model":       model,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      reqMap,
			"response":     respMap,
		})
		fmt.Println("    ✅ real_bedrock_text.json")
	}

	// Scenario 2: Tool use
	fmt.Println("  🔧 Scenario: tool_use")

	// Create tool input schemas using document
	weatherSchema := document.NewLazyDocument(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{"type": "string", "description": "City name"},
		},
		"required": []string{"location"},
	})
	placesSchema := document.NewLazyDocument(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{"type": "string"},
			"type":     map[string]any{"type": "string"},
		},
		"required": []string{"location", "type"},
	})

	input2 := &bedrockruntime.ConverseInput{
		ModelId: aws.String(model),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "Check the weather in Seattle and find nearby coffee shops."},
				},
			},
		},
		ToolConfig: &types.ToolConfiguration{
			Tools: []types.Tool{
				&types.ToolMemberToolSpec{
					Value: types.ToolSpecification{
						Name:        aws.String("get_weather"),
						Description: aws.String("Get weather for a location"),
						InputSchema: &types.ToolInputSchemaMemberJson{Value: weatherSchema},
					},
				},
				&types.ToolMemberToolSpec{
					Value: types.ToolSpecification{
						Name:        aws.String("search_places"),
						Description: aws.String("Search for places near a location"),
						InputSchema: &types.ToolInputSchemaMemberJson{Value: placesSchema},
					},
				},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(1024),
		},
	}

	resp2, err := client.Converse(ctx, input2)
	if err != nil {
		fmt.Printf("    ❌ Failed: %v\n", err)
	} else {
		respMap := converseOutputToMap(resp2)
		reqMap := map[string]any{
			"messages": []map[string]any{
				{"role": "user", "content": []map[string]any{{"text": "Check the weather in Seattle and find nearby coffee shops."}}},
			},
			"toolConfig": map[string]any{
				"tools": []map[string]any{
					{"toolSpec": map[string]any{"name": "get_weather", "description": "Get weather for a location"}},
					{"toolSpec": map[string]any{"name": "search_places", "description": "Search for places near a location"}},
				},
			},
			"inferenceConfig": map[string]any{"maxTokens": 1024},
		}
		saveFixture("real_bedrock_tools.json", map[string]any{
			"_description": "Bedrock Converse API - Tool use",
			"_scenario":    "tool_use",
			"_model":       model,
			"_captured":    time.Now().Format(time.RFC3339),
			"request":      reqMap,
			"response":     respMap,
		})
		fmt.Println("    ✅ real_bedrock_tools.json")
	}
}

// converseOutputToMap converts the SDK response to a map matching the raw API response format
func converseOutputToMap(resp *bedrockruntime.ConverseOutput) map[string]any {
	result := map[string]any{}

	// Output message
	if resp.Output != nil {
		switch v := resp.Output.(type) {
		case *types.ConverseOutputMemberMessage:
			content := []map[string]any{}
			for _, block := range v.Value.Content {
				switch b := block.(type) {
				case *types.ContentBlockMemberText:
					content = append(content, map[string]any{"text": b.Value})
				case *types.ContentBlockMemberToolUse:
					content = append(content, map[string]any{
						"toolUse": map[string]any{
							"toolUseId": b.Value.ToolUseId,
							"name":      b.Value.Name,
							"input":     b.Value.Input,
						},
					})
				}
			}
			result["output"] = map[string]any{
				"message": map[string]any{
					"role":    string(v.Value.Role),
					"content": content,
				},
			}
		}
	}

	// Stop reason
	result["stopReason"] = string(resp.StopReason)

	// Usage
	if resp.Usage != nil {
		result["usage"] = map[string]any{
			"inputTokens":  resp.Usage.InputTokens,
			"outputTokens": resp.Usage.OutputTokens,
			"totalTokens":  aws.ToInt32(resp.Usage.TotalTokens),
		}
	}

	// Metrics
	if resp.Metrics != nil {
		result["metrics"] = map[string]any{
			"latencyMs": resp.Metrics.LatencyMs,
		}
	}

	return result
}
