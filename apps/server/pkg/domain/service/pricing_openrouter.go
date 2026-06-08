package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// DefaultOpenRouterPricingURL is OpenRouter's public models endpoint, used as a
// secondary pricing source (fills gaps LiteLLM doesn't cover).
const DefaultOpenRouterPricingURL = "https://openrouter.ai/api/v1/models"

// openRouterResponse is the `{ "data": [ ... ] }` envelope from /api/v1/models.
type openRouterResponse struct {
	Data []openRouterModel `json:"data"`
}

// openRouterModel is the subset we consume. pricing values are STRINGS in USD
// PER TOKEN (we parse and convert to per-1K).
type openRouterModel struct {
	ID      string `json:"id"` // e.g. "openai/gpt-4o"
	Pricing struct {
		Prompt            string `json:"prompt"`
		Completion        string `json:"completion"`
		InputCacheRead    string `json:"input_cache_read"`
		InputCacheWrite   string `json:"input_cache_write"`
		InternalReasoning string `json:"internal_reasoning"`
	} `json:"pricing"`
}

// strPer1K parses a per-token cost string ("0.000005") and converts to per-1K.
// Empty or unparseable values yield 0 (treated as "absent").
func strPer1K(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v * 1000
}

// ParseOpenRouterPricing converts OpenRouter's models JSON into a ModelPricing
// map. Each model is keyed by both its full id ("openai/gpt-4o") and its bare
// name ("gpt-4o"), so it matches whichever form a span reports. Entries without
// prompt/completion cost are skipped.
func ParseOpenRouterPricing(data []byte) (map[string]ModelPricing, error) {
	var resp openRouterResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse openrouter pricing: %w", err)
	}

	out := make(map[string]ModelPricing, len(resp.Data)*2)
	for _, m := range resp.Data {
		input := strPer1K(m.Pricing.Prompt)
		output := strPer1K(m.Pricing.Completion)
		if input == 0 && output == 0 {
			continue
		}

		mp := ModelPricing{
			Input:      input,
			Output:     output,
			CacheRead:  strPer1K(m.Pricing.InputCacheRead),
			CacheWrite: strPer1K(m.Pricing.InputCacheWrite),
			Reasoning:  strPer1K(m.Pricing.InternalReasoning),
		}

		out[m.ID] = mp
		if i := strings.LastIndex(m.ID, "/"); i >= 0 && i+1 < len(m.ID) {
			if bare := m.ID[i+1:]; out[bare] == (ModelPricing{}) {
				out[bare] = mp
			}
		}
	}
	return out, nil
}

// FetchOpenRouterPricing downloads and parses OpenRouter's models table. Network
// or HTTP failures are returned so the caller can fall back to other sources.
func FetchOpenRouterPricing(ctx context.Context, url string) (map[string]ModelPricing, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build openrouter request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch openrouter pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch openrouter pricing: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxLiteLLMResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read openrouter pricing: %w", err)
	}

	return ParseOpenRouterPricing(data)
}
