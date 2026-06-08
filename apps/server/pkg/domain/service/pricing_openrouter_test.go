package service

import (
	"os"
	"path/filepath"
	"testing"
)

func loadOpenRouterFixture(t *testing.T) map[string]ModelPricing {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "openrouter_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	pricing, err := ParseOpenRouterPricing(data)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return pricing
}

// TestParseOpenRouter_Conversion parses string per-token costs to per-1K, incl. cache.
func TestParseOpenRouter_Conversion(t *testing.T) {
	pricing := loadOpenRouterFixture(t)

	claude, ok := pricing["anthropic/claude-3.5-sonnet"]
	if !ok {
		t.Fatal("anthropic/claude-3.5-sonnet missing")
	}
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

// TestParseOpenRouter_BareNameKey keys models by both full id and bare name.
func TestParseOpenRouter_BareNameKey(t *testing.T) {
	pricing := loadOpenRouterFixture(t)

	full, okFull := pricing["openai/gpt-4o"]
	bare, okBare := pricing["gpt-4o"]
	if !okFull || !okBare {
		t.Fatalf("expected both full and bare keys: full=%v bare=%v", okFull, okBare)
	}
	if !approxEqual(full.Input, 0.0025) || !approxEqual(bare.Input, 0.0025) {
		t.Errorf("gpt-4o input mismatch: full=%v bare=%v want 0.0025", full.Input, bare.Input)
	}
}

// TestParseOpenRouter_Reasoning maps internal_reasoning to Reasoning.
func TestParseOpenRouter_Reasoning(t *testing.T) {
	pricing := loadOpenRouterFixture(t)
	o1, ok := pricing["o1-preview"]
	if !ok {
		t.Fatal("o1-preview (bare) missing")
	}
	if !approxEqual(o1.Reasoning, 0.06) {
		t.Errorf("o1-preview reasoning = %v, want 0.06", o1.Reasoning)
	}
}

// TestParseOpenRouter_SkipsZero drops models with no prompt/completion cost.
func TestParseOpenRouter_SkipsZero(t *testing.T) {
	pricing := loadOpenRouterFixture(t)
	if _, ok := pricing["some/free-model"]; ok {
		t.Error("free model (0 cost) should be skipped")
	}
}

// TestParseOpenRouter_Malformed returns an error instead of panicking.
func TestParseOpenRouter_Malformed(t *testing.T) {
	if _, err := ParseOpenRouterPricing([]byte("{ not json")); err == nil {
		t.Error("expected error on malformed JSON")
	}
}
