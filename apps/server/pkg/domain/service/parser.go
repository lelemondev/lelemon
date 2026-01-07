package service

import (
	"fmt"
	"strings"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ParsedResponse contains extracted data from raw LLM responses
type ParsedResponse struct {
	Output           any
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  *int
	CacheWriteTokens *int
	ReasoningTokens  *int
	StopReason       *string
	Thinking         *string
	ToolUses         []entity.ToolUse
	SubType          *string // "planning" | "response"
}

// ParseProviderResponse extracts structured data from a raw LLM response
func ParseProviderResponse(provider string, rawResponse any) *ParsedResponse {
	if rawResponse == nil {
		return nil
	}

	switch strings.ToLower(provider) {
	case "anthropic":
		return parseAnthropicResponse(rawResponse)
	case "bedrock":
		return parseBedrockResponse(rawResponse)
	case "openai", "openrouter":
		return parseOpenAIResponse(rawResponse)
	case "gemini":
		return parseGeminiResponse(rawResponse)
	default:
		// Try to detect format from response structure
		return parseAutoDetect(rawResponse)
	}
}

// parseAnthropicResponse parses Anthropic Messages API response
// Format: { content: [...], usage: { input_tokens, output_tokens, ... }, stop_reason }
func parseAnthropicResponse(raw any) *ParsedResponse {
	resp, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := &ParsedResponse{}

	// Extract usage
	if usage, ok := resp["usage"].(map[string]any); ok {
		if v, ok := usage["input_tokens"].(float64); ok {
			result.InputTokens = int(v)
		}
		if v, ok := usage["output_tokens"].(float64); ok {
			result.OutputTokens = int(v)
		}
		if v, ok := usage["cache_read_input_tokens"].(float64); ok {
			val := int(v)
			result.CacheReadTokens = &val
		}
		if v, ok := usage["cache_creation_input_tokens"].(float64); ok {
			val := int(v)
			result.CacheWriteTokens = &val
		}
	}

	// Extract stop reason
	if v, ok := resp["stop_reason"].(string); ok {
		result.StopReason = &v
	}

	// Extract content
	if content, ok := resp["content"].([]any); ok {
		var textParts []string
		var thinkingParts []string
		hasToolUse := false

		for i, block := range content {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}

			blockType, _ := blockMap["type"].(string)

			switch blockType {
			case "text":
				if text, ok := blockMap["text"].(string); ok {
					textParts = append(textParts, text)
				}
			case "thinking":
				if thinking, ok := blockMap["thinking"].(string); ok {
					thinkingParts = append(thinkingParts, thinking)
				}
			case "tool_use":
				hasToolUse = true
				id, _ := blockMap["id"].(string)
				name, _ := blockMap["name"].(string)
				input := blockMap["input"]

				if name != "" {
					result.ToolUses = append(result.ToolUses, entity.ToolUse{
						ID:     id,
						Name:   name,
						Input:  input,
						Status: "pending",
					})
				}
				_ = i // suppress unused warning
			}
		}

		// Set output (prefer raw content array if has tool use)
		if hasToolUse {
			result.Output = content
			subType := "planning"
			result.SubType = &subType
		} else {
			if len(textParts) > 0 {
				result.Output = strings.Join(textParts, "")
			}
			subType := "response"
			result.SubType = &subType
		}

		// Set thinking
		if len(thinkingParts) > 0 {
			thinking := strings.Join(thinkingParts, "\n\n")
			result.Thinking = &thinking
		}
	}

	return result
}

// parseBedrockResponse parses AWS Bedrock Converse API response
// Format: { output: { message: { content: [...] } }, usage: { inputTokens, ... }, stopReason }
func parseBedrockResponse(raw any) *ParsedResponse {
	resp, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := &ParsedResponse{}

	// Extract usage
	if usage, ok := resp["usage"].(map[string]any); ok {
		if v, ok := usage["inputTokens"].(float64); ok {
			result.InputTokens = int(v)
		}
		if v, ok := usage["outputTokens"].(float64); ok {
			result.OutputTokens = int(v)
		}
		if v, ok := usage["cacheReadInputTokens"].(float64); ok {
			val := int(v)
			result.CacheReadTokens = &val
		}
		if v, ok := usage["cacheWriteInputTokens"].(float64); ok {
			val := int(v)
			result.CacheWriteTokens = &val
		}
	}

	// Extract stop reason
	if v, ok := resp["stopReason"].(string); ok {
		result.StopReason = &v
	}

	// Extract content from output.message.content
	var content []any
	if output, ok := resp["output"].(map[string]any); ok {
		if message, ok := output["message"].(map[string]any); ok {
			content, _ = message["content"].([]any)
		}
	}

	if len(content) > 0 {
		var textParts []string
		hasToolUse := false

		for i, block := range content {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}

			// Text block
			if text, ok := blockMap["text"].(string); ok {
				textParts = append(textParts, text)
			}

			// Tool use block
			if toolUse, ok := blockMap["toolUse"].(map[string]any); ok {
				hasToolUse = true
				id, _ := toolUse["toolUseId"].(string)
				name, _ := toolUse["name"].(string)
				input := toolUse["input"]

				if name != "" {
					if id == "" {
						id = fmt.Sprintf("tool-%d", i)
					}
					result.ToolUses = append(result.ToolUses, entity.ToolUse{
						ID:     id,
						Name:   name,
						Input:  input,
						Status: "pending",
					})
				}
			}
		}

		// Set output
		if hasToolUse {
			result.Output = content
			subType := "planning"
			result.SubType = &subType
		} else {
			if len(textParts) > 0 {
				result.Output = strings.Join(textParts, "")
			}
			subType := "response"
			result.SubType = &subType
		}
	}

	return result
}

// parseOpenAIResponse parses OpenAI Chat Completions API response
// Format: { choices: [{ message: { content, tool_calls }, finish_reason }], usage: { prompt_tokens, completion_tokens, ... } }
func parseOpenAIResponse(raw any) *ParsedResponse {
	resp, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := &ParsedResponse{}

	// Extract usage
	if usage, ok := resp["usage"].(map[string]any); ok {
		if v, ok := usage["prompt_tokens"].(float64); ok {
			result.InputTokens = int(v)
		}
		if v, ok := usage["completion_tokens"].(float64); ok {
			result.OutputTokens = int(v)
		}
		// Reasoning tokens (o1/o3 models)
		if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
			if v, ok := details["reasoning_tokens"].(float64); ok {
				val := int(v)
				result.ReasoningTokens = &val
			}
		}
	}

	// Extract from first choice
	if choices, ok := resp["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			// Finish reason
			if v, ok := choice["finish_reason"].(string); ok {
				result.StopReason = &v
			}

			// Message content
			if message, ok := choice["message"].(map[string]any); ok {
				if content, ok := message["content"].(string); ok {
					result.Output = content
				}

				// Tool calls
				if toolCalls, ok := message["tool_calls"].([]any); ok && len(toolCalls) > 0 {
					for i, tc := range toolCalls {
						tcMap, ok := tc.(map[string]any)
						if !ok {
							continue
						}

						id, _ := tcMap["id"].(string)
						if fn, ok := tcMap["function"].(map[string]any); ok {
							name, _ := fn["name"].(string)
							args := fn["arguments"]

							// Try to parse arguments JSON
							var input any = args
							if argsStr, ok := args.(string); ok {
								// Keep as string, server can parse if needed
								input = argsStr
							}

							if name != "" {
								if id == "" {
									id = fmt.Sprintf("call-%d", i)
								}
								result.ToolUses = append(result.ToolUses, entity.ToolUse{
									ID:     id,
									Name:   name,
									Input:  input,
									Status: "pending",
								})
							}
						}
					}

					subType := "planning"
					result.SubType = &subType
				} else {
					subType := "response"
					result.SubType = &subType
				}
			}
		}
	}

	return result
}

// parseGeminiResponse parses Google Gemini API response
// Format: { candidates: [{ content: { parts: [...] }, finishReason }], usageMetadata: { promptTokenCount, ... } }
func parseGeminiResponse(raw any) *ParsedResponse {
	resp, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := &ParsedResponse{}

	// Extract usage
	if usage, ok := resp["usageMetadata"].(map[string]any); ok {
		if v, ok := usage["promptTokenCount"].(float64); ok {
			result.InputTokens = int(v)
		}
		if v, ok := usage["candidatesTokenCount"].(float64); ok {
			result.OutputTokens = int(v)
		}
		if v, ok := usage["cachedContentTokenCount"].(float64); ok {
			val := int(v)
			result.CacheReadTokens = &val
		}
		if v, ok := usage["thoughtsTokenCount"].(float64); ok {
			val := int(v)
			result.ReasoningTokens = &val
		}
	}

	// Extract from first candidate
	if candidates, ok := resp["candidates"].([]any); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]any); ok {
			// Finish reason
			if v, ok := candidate["finishReason"].(string); ok {
				result.StopReason = &v
			}

			// Content parts
			if content, ok := candidate["content"].(map[string]any); ok {
				if parts, ok := content["parts"].([]any); ok {
					var textParts []string
					hasFunction := false

					for i, part := range parts {
						partMap, ok := part.(map[string]any)
						if !ok {
							continue
						}

						// Text part
						if text, ok := partMap["text"].(string); ok {
							textParts = append(textParts, text)
						}

						// Function call part
						if fc, ok := partMap["functionCall"].(map[string]any); ok {
							hasFunction = true
							name, _ := fc["name"].(string)
							args := fc["args"]

							if name != "" {
								result.ToolUses = append(result.ToolUses, entity.ToolUse{
									ID:     fmt.Sprintf("gemini-fc-%s-%d", name, i),
									Name:   name,
									Input:  args,
									Status: "pending",
								})
							}
						}
					}

					if len(textParts) > 0 {
						result.Output = strings.Join(textParts, "")
					}

					if hasFunction {
						subType := "planning"
						result.SubType = &subType
					} else {
						subType := "response"
						result.SubType = &subType
					}
				}
			}
		}
	}

	return result
}

// parseAutoDetect tries to detect the provider format from response structure
func parseAutoDetect(raw any) *ParsedResponse {
	resp, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	// Anthropic: has "content" array and "usage.input_tokens"
	if _, hasContent := resp["content"]; hasContent {
		if usage, ok := resp["usage"].(map[string]any); ok {
			if _, ok := usage["input_tokens"]; ok {
				return parseAnthropicResponse(raw)
			}
		}
	}

	// Bedrock: has "output.message.content"
	if output, ok := resp["output"].(map[string]any); ok {
		if _, ok := output["message"]; ok {
			return parseBedrockResponse(raw)
		}
	}

	// OpenAI: has "choices" array
	if _, hasChoices := resp["choices"]; hasChoices {
		return parseOpenAIResponse(raw)
	}

	// Gemini: has "candidates" array
	if _, hasCandidates := resp["candidates"]; hasCandidates {
		return parseGeminiResponse(raw)
	}

	return nil
}
