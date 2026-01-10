# LLM Response Fixtures

## Best Practice: One Fixture per API SCENARIO, not per Model

The response FORMAT is determined by the API, not the model. gpt-4o and gpt-4-turbo return identical structures. We only need separate fixtures when there's a STRUCTURAL difference (e.g., O1 models have `reasoning_tokens`).

## Fixture Organization

### Real Fixtures (from live API calls)

| Provider | Scenario | File | Description |
|----------|----------|------|-------------|
| OpenAI | text | `real_openai_text.json` | Standard text response |
| OpenAI | tools | `real_openai_tools.json` | Parallel function calling |
| OpenAI | reasoning | `real_openai_reasoning.json` | O1 model with reasoning_tokens |
| Gemini | text | `real_gemini_text.json` | Standard text response |
| Gemini | functions | `real_gemini_functions.json` | Function calling |

### Synthetic Fixtures (from API documentation)

Used for providers without API keys or for edge cases:

| Provider | Scenario | File |
|----------|----------|------|
| Anthropic | text | `anthropic_text_response.json` |
| Anthropic | tool_use | `anthropic_tool_use.json` |
| Anthropic | cache | `anthropic_with_cache.json` |
| Anthropic | thinking | `anthropic_with_thinking.json` |
| Bedrock | converse | `bedrock_converse.json` |

## Generating Fixtures

```bash
cd apps/server

# Set API keys
export ANTHROPIC_API_KEY=your-key
export OPENAI_API_KEY=your-key
export GOOGLE_API_KEY=your-key

# Generate
go run scripts/generate_fixtures.go
```

## Fixture Structure

```json
{
  "_description": "Provider API - Scenario",
  "_scenario": "text_response | tool_calls | reasoning | ...",
  "_model": "model-id used",
  "_captured": "ISO timestamp",
  "request": { ... },
  "response": { ... }
}
```

## Why This Approach?

1. **Less redundancy** - 7 fixtures instead of 20+
2. **Easier maintenance** - new model ≠ new fixture
3. **Focused testing** - tests what matters (parsing logic)
4. **Model-specific only when needed** - O1 reasoning is structurally different

## References

- [Langfuse Testing Guide](https://langfuse.com/blog/2025-10-21-testing-llm-applications)
- [LLM Testing Best Practices](https://www.confident-ai.com/blog/llm-testing-in-2024-top-methods-and-strategies)
