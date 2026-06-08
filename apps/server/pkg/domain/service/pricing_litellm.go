package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultLiteLLMPricingURL is LiteLLM's community-maintained pricing table
// (MIT-licensed, ~daily updates, 300+ models). Used as the primary pricing
// source; the hardcoded map in pricing.go remains the offline fallback.
const DefaultLiteLLMPricingURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// maxLiteLLMResponseBytes caps the download size defensively (~a few MB today).
const maxLiteLLMResponseBytes = 32 << 20 // 32 MiB

// litellmModel is the subset of a LiteLLM model entry we consume. Every *_cost
// field is USD PER TOKEN (we convert to per-1K to match ModelPricing). Pointers
// distinguish "absent" from "zero".
type litellmModel struct {
	Mode                        string   `json:"mode"`
	InputCostPerToken           *float64 `json:"input_cost_per_token"`
	OutputCostPerToken          *float64 `json:"output_cost_per_token"`
	CacheReadInputTokenCost     *float64 `json:"cache_read_input_token_cost"`
	CacheCreationInputTokenCost *float64 `json:"cache_creation_input_token_cost"`
	OutputCostPerReasoningToken *float64 `json:"output_cost_per_reasoning_token"`
}

// nonTokenModes are LiteLLM modes that don't price by text token; we skip them
// so the table stays focused on LLM token costs.
var nonTokenModes = map[string]bool{
	"image_generation":    true,
	"audio_transcription": true,
	"audio_speech":        true,
	"moderation":          true,
	"moderations":         true,
	"rerank":              true,
}

// per1K converts a per-token cost (LiteLLM's unit) to per-1K (ModelPricing's unit).
func per1K(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v * 1000
}

// ParseLiteLLMPricing converts LiteLLM's pricing JSON into a ModelPricing map.
// It skips the `sample_spec` documentation entry, non-token modes, and any entry
// without an input or output token cost. Parsing is defensive: unknown fields are
// ignored and missing cost fields default to 0.
func ParseLiteLLMPricing(data []byte) (map[string]ModelPricing, error) {
	var raw map[string]litellmModel
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse litellm pricing: %w", err)
	}

	out := make(map[string]ModelPricing, len(raw))
	for name, m := range raw {
		if name == "sample_spec" {
			continue
		}
		if nonTokenModes[m.Mode] {
			continue
		}
		// Require at least one token cost; this drops doc entries and models
		// priced by image/second/etc.
		if m.InputCostPerToken == nil && m.OutputCostPerToken == nil {
			continue
		}

		out[name] = ModelPricing{
			Input:      per1K(m.InputCostPerToken),
			Output:     per1K(m.OutputCostPerToken),
			CacheRead:  per1K(m.CacheReadInputTokenCost),
			CacheWrite: per1K(m.CacheCreationInputTokenCost),
			Reasoning:  per1K(m.OutputCostPerReasoningToken),
		}
	}
	return out, nil
}

// FetchLiteLLMPricing downloads and parses the LiteLLM pricing table. The caller
// supplies the context (for timeout/cancellation) and URL. Network/HTTP failures
// are returned as errors so the caller can fall back to the local table.
func FetchLiteLLMPricing(ctx context.Context, url string) (map[string]ModelPricing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build litellm request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch litellm pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch litellm pricing: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxLiteLLMResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read litellm pricing: %w", err)
	}

	return ParseLiteLLMPricing(data)
}
