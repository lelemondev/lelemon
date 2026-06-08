package service

import (
	"os"
	"path/filepath"
	"testing"
)

func loadLiteLLMFixture(t *testing.T) map[string]ModelPricing {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "litellm_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	pricing, err := ParseLiteLLMPricing(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return pricing
}

// TestParseLiteLLM_Conversion verifies per-token costs are converted to per-1K
// and cache/reasoning fields are mapped.
func TestParseLiteLLM_Conversion(t *testing.T) {
	pricing := loadLiteLLMFixture(t)

	claude, ok := pricing["claude-3-5-sonnet-20241022"]
	if !ok {
		t.Fatal("claude-3-5-sonnet-20241022 missing")
	}
	// 3e-06/token * 1000 = 0.003 per 1K, etc. — matches our hardcoded table.
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"input", claude.Input, 0.003},
		{"output", claude.Output, 0.015},
		{"cacheRead", claude.CacheRead, 0.0003},
		{"cacheWrite", claude.CacheWrite, 0.00375},
	} {
		if !approxEqual(c.got, c.want) {
			t.Errorf("claude %s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestParseLiteLLM_Reasoning verifies output_cost_per_reasoning_token maps to Reasoning.
func TestParseLiteLLM_Reasoning(t *testing.T) {
	pricing := loadLiteLLMFixture(t)

	o1, ok := pricing["o1-preview"]
	if !ok {
		t.Fatal("o1-preview missing")
	}
	if !approxEqual(o1.Input, 0.015) || !approxEqual(o1.Output, 0.06) {
		t.Errorf("o1-preview in/out = %v/%v, want 0.015/0.06", o1.Input, o1.Output)
	}
	if !approxEqual(o1.Reasoning, 0.06) {
		t.Errorf("o1-preview reasoning = %v, want 0.06", o1.Reasoning)
	}
}

// TestParseLiteLLM_NoCacheFields leaves cache rates at 0 when absent (deriveRates
// fills them later as a fallback).
func TestParseLiteLLM_NoCacheFields(t *testing.T) {
	pricing := loadLiteLLMFixture(t)

	gpt, ok := pricing["gpt-4o"]
	if !ok {
		t.Fatal("gpt-4o missing")
	}
	if !approxEqual(gpt.Input, 0.0025) || !approxEqual(gpt.Output, 0.01) {
		t.Errorf("gpt-4o in/out = %v/%v, want 0.0025/0.01", gpt.Input, gpt.Output)
	}
	if gpt.CacheRead != 0 || gpt.CacheWrite != 0 || gpt.Reasoning != 0 {
		t.Errorf("gpt-4o should have no cache/reasoning from LiteLLM, got %+v", gpt)
	}
}

// TestParseLiteLLM_Skips drops sample_spec and non-token modes, keeps embeddings.
func TestParseLiteLLM_Skips(t *testing.T) {
	pricing := loadLiteLLMFixture(t)

	if _, ok := pricing["sample_spec"]; ok {
		t.Error("sample_spec should be skipped")
	}
	if _, ok := pricing["dall-e-3"]; ok {
		t.Error("dall-e-3 (image_generation) should be skipped")
	}
	// Embeddings carry a real per-token input cost, so they're kept.
	emb, ok := pricing["text-embedding-3-small"]
	if !ok {
		t.Fatal("text-embedding-3-small should be kept")
	}
	if !approxEqual(emb.Input, 2e-05) {
		t.Errorf("embedding input = %v, want 2e-05", emb.Input)
	}
}

// TestParseLiteLLM_Malformed returns an error instead of panicking.
func TestParseLiteLLM_Malformed(t *testing.T) {
	if _, err := ParseLiteLLMPricing([]byte("not json")); err == nil {
		t.Error("expected error on malformed JSON")
	}
}
