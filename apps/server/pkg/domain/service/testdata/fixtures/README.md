# API Response Fixtures

This directory contains fixtures for testing the LLM response parser.

## Fixture Types

### Real Fixtures (`real_*.json`)
Generated from actual API calls to LLM providers. These are the most reliable for testing since they contain real response structures with all fields populated.

- `real_openai_multi_turn.json` - Multi-turn customer support conversation (gpt-4o)
- `real_openai_parallel_tools.json` - Parallel function calling with 3 tools
- `real_openai_structured.json` - JSON response format with structured output

### Synthetic Fixtures
Created based on API documentation for edge cases and providers we don't have keys for.

- `openai_chat_completion.json` - Standard text response
- `openai_tool_calls.json` - Function calling response
- `openai_with_reasoning.json` - o1/o3 model with reasoning tokens
- `anthropic_*.json` - Anthropic Messages API responses
- `bedrock_converse.json` - AWS Bedrock Converse API response

## Fixture Structure

Each fixture contains:

```json
{
  "_description": "Human-readable description",
  "_source": "API documentation URL or 'Live API call'",
  "_captured": "ISO timestamp when captured",
  "_model": "Model used",
  "_scenario": "What production scenario this tests",
  "request": { ... },  // The API request sent
  "response": { ... }, // The raw API response
  "expected": { ... }  // Expected parser output (synthetic only)
}
```

## Generating New Fixtures

Run the fixture generator script with API keys set:

```bash
export ANTHROPIC_API_KEY=your-key
export OPENAI_API_KEY=your-key
export GOOGLE_API_KEY=your-key

cd apps/server
go run scripts/generate_fixtures.go
```

## Usage in Tests

```go
func TestParser(t *testing.T) {
    fixture := loadFixture(t, "real_openai_multi_turn.json")
    response := fixture["response"]

    result := ParseProviderResponse("openai", response)
    // Assert on result
}
```

## Provider Format Reference

### OpenAI
- `choices[0].message.content` → output (string)
- `choices[0].message.tool_calls` → tool_uses
- `usage.prompt_tokens` → input_tokens
- `usage.completion_tokens` → output_tokens
- `choices[0].finish_reason` → stop_reason

### Anthropic
- `content[*].text` → output (joined string or content array if tool_use)
- `content[*].type == "tool_use"` → tool_uses
- `usage.input_tokens` → input_tokens
- `usage.output_tokens` → output_tokens
- `stop_reason` → stop_reason

### Bedrock (Converse API)
- `output.message.content[*].text` → output
- `output.message.content[*].toolUse` → tool_uses
- `usage.inputTokens` → input_tokens
- `usage.outputTokens` → output_tokens
- `stopReason` → stop_reason

### Gemini
- `candidates[0].content.parts[*].text` → output
- `candidates[0].content.parts[*].functionCall` → tool_uses
- `usageMetadata.promptTokenCount` → input_tokens
- `usageMetadata.candidatesTokenCount` → output_tokens
- `candidates[0].finishReason` → stop_reason
