package service

import (
	"math"
	"testing"
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// TestDeriveRates_Anthropic verifies cache write = 1.25x input, cache read =
// 0.1x input, and reasoning = output for Claude models (incl. Bedrock prefixes).
func TestDeriveRates_Anthropic(t *testing.T) {
	models := []string{
		"claude-opus-4.5",
		"anthropic.claude-sonnet-4-5",
		"us.anthropic.claude-haiku-4-5",
	}
	for _, model := range models {
		mp, ok := findPricing(model)
		if !ok {
			t.Fatalf("%s: expected pricing to be found", model)
		}
		if !approxEqual(mp.CacheWrite, mp.Input*1.25) {
			t.Errorf("%s: cache write = %v, want %v", model, mp.CacheWrite, mp.Input*1.25)
		}
		if !approxEqual(mp.CacheRead, mp.Input*0.10) {
			t.Errorf("%s: cache read = %v, want %v", model, mp.CacheRead, mp.Input*0.10)
		}
		if !approxEqual(mp.Reasoning, mp.Output) {
			t.Errorf("%s: reasoning = %v, want %v (output)", model, mp.Reasoning, mp.Output)
		}
	}
}

// TestDeriveRates_OpenAI verifies cached input = 0.5x input, no write surcharge,
// and reasoning = output.
func TestDeriveRates_OpenAI(t *testing.T) {
	for _, model := range []string{"gpt-5", "gpt-4o", "o3-mini"} {
		mp, ok := findPricing(model)
		if !ok {
			t.Fatalf("%s: expected pricing to be found", model)
		}
		if !approxEqual(mp.CacheRead, mp.Input*0.50) {
			t.Errorf("%s: cache read = %v, want %v", model, mp.CacheRead, mp.Input*0.50)
		}
		if !approxEqual(mp.CacheWrite, mp.Input) {
			t.Errorf("%s: cache write = %v, want %v (no surcharge)", model, mp.CacheWrite, mp.Input)
		}
		if !approxEqual(mp.Reasoning, mp.Output) {
			t.Errorf("%s: reasoning = %v, want %v (output)", model, mp.Reasoning, mp.Output)
		}
	}
}

// TestDeriveRates_Gemini verifies implicit cache read = 0.25x input.
func TestDeriveRates_Gemini(t *testing.T) {
	mp, ok := findPricing("gemini-3-pro")
	if !ok {
		t.Fatal("gemini-3-pro: expected pricing to be found")
	}
	if !approxEqual(mp.CacheRead, mp.Input*0.25) {
		t.Errorf("gemini cache read = %v, want %v", mp.CacheRead, mp.Input*0.25)
	}
}

// TestDeriveRates_UnknownModel keeps everything at 0 (transparent indicator).
func TestDeriveRates_UnknownModel(t *testing.T) {
	mp, ok := findPricing("totally-made-up-model")
	if ok {
		t.Fatal("expected unknown model to be unpriced")
	}
	if mp.CacheRead != 0 || mp.CacheWrite != 0 || mp.Reasoning != 0 {
		t.Errorf("unknown model should have zero derived rates, got %+v", mp)
	}
}

// TestCalculateCostBreakdown_AnthropicCache verifies each token bucket is priced
// at its own rate and the total sums them.
func TestCalculateCostBreakdown_AnthropicCache(t *testing.T) {
	p := NewPricingCalculator()
	// claude-3-5-sonnet: Input=0.003, Output=0.015 per 1K.
	// Derived: cacheRead=0.0003, cacheWrite=0.00375 per 1K.
	b := p.CalculateCostBreakdown("claude-3-5-sonnet-20241022", TokenUsage{
		Input: 1000, Output: 500, CacheRead: 2000, CacheWrite: 1000,
	})

	want := CostBreakdown{
		Input:      0.003,    // 1 * 0.003
		Output:     0.0075,   // 0.5 * 0.015
		CacheRead:  0.0006,   // 2 * 0.0003
		CacheWrite: 0.00375,  // 1 * 0.00375
		Reasoning:  0,
		Total:      0.01485,  // 0.003 + 0.0075 + 0.0006 + 0.00375
	}
	if !approxEqual(b.Input, want.Input) || !approxEqual(b.Output, want.Output) ||
		!approxEqual(b.CacheRead, want.CacheRead) || !approxEqual(b.CacheWrite, want.CacheWrite) ||
		!approxEqual(b.Reasoning, want.Reasoning) || !approxEqual(b.Total, want.Total) {
		t.Errorf("breakdown = %+v, want %+v", b, want)
	}
}

// TestCalculateCostBreakdown_Reasoning verifies reasoning tokens are priced at
// the output rate by default.
func TestCalculateCostBreakdown_Reasoning(t *testing.T) {
	p := NewPricingCalculator()
	// o1-preview: Output=0.06 per 1K, so reasoning (derived = output) = 0.06.
	b := p.CalculateCostBreakdown("o1-preview", TokenUsage{Reasoning: 1000})
	if !approxEqual(b.Reasoning, 0.06) {
		t.Errorf("reasoning cost = %v, want 0.06", b.Reasoning)
	}
	if !approxEqual(b.Total, 0.06) {
		t.Errorf("total = %v, want 0.06", b.Total)
	}
}

// TestNormalizeTokenUsage verifies per-provider disjoint-bucket normalization.
func TestNormalizeTokenUsage(t *testing.T) {
	tests := []struct {
		name                                       string
		provider                                   string
		in, out, cacheRead, cacheWrite, reasoning  int
		want                                       TokenUsage
	}{
		{
			name:     "anthropic keeps input disjoint from cache",
			provider: "anthropic",
			in:       1000, out: 500, cacheRead: 2000, cacheWrite: 1000,
			want: TokenUsage{Input: 1000, Output: 500, CacheRead: 2000, CacheWrite: 1000},
		},
		{
			name:     "bedrock behaves like anthropic",
			provider: "bedrock",
			in:       800, out: 400, cacheRead: 1000,
			want: TokenUsage{Input: 800, Output: 400, CacheRead: 1000},
		},
		{
			name:     "openai subtracts reasoning from output",
			provider: "openai",
			in:       1000, out: 500, reasoning: 200,
			want: TokenUsage{Input: 1000, Output: 300, Reasoning: 200},
		},
		{
			name:     "openai subtracts cached from input",
			provider: "openai",
			in:       1000, out: 500, cacheRead: 400,
			want: TokenUsage{Input: 600, Output: 500, CacheRead: 400},
		},
		{
			name:     "gemini subtracts cached from input but not reasoning from output",
			provider: "gemini",
			in:       1000, out: 500, cacheRead: 300, reasoning: 150,
			want: TokenUsage{Input: 700, Output: 500, CacheRead: 300, Reasoning: 150},
		},
		{
			name:     "unknown provider assumes openai-style overlap",
			provider: "custom",
			in:       1000, out: 500, cacheRead: 100, reasoning: 50,
			want: TokenUsage{Input: 900, Output: 450, CacheRead: 100, Reasoning: 50},
		},
		{
			name:     "never goes negative",
			provider: "openai",
			in:       100, out: 50, cacheRead: 500, reasoning: 500,
			want: TokenUsage{Input: 0, Output: 0, CacheRead: 500, Reasoning: 500},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTokenUsage(tt.provider, tt.in, tt.out, tt.cacheRead, tt.cacheWrite, tt.reasoning)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestCalculateCost_BackwardCompatible verifies the legacy wrapper still returns
// the input+output total unchanged.
func TestCalculateCost_BackwardCompatible(t *testing.T) {
	p := NewPricingCalculator()
	cases := []struct {
		model    string
		in, out  int
		expected float64
	}{
		{"gpt-4o", 1000, 500, 0.0075},
		{"gpt-4o-mini", 1000, 500, 0.00045},
		{"claude-3-5-sonnet-20241022", 1000, 500, 0.0105},
		{"o1-preview", 1000, 500, 0.045},
	}
	for _, c := range cases {
		got := p.CalculateCost(c.model, c.in, c.out)
		if !approxEqual(got, c.expected) {
			t.Errorf("%s: CalculateCost = %v, want %v", c.model, got, c.expected)
		}
		// Wrapper must equal the breakdown total with only input/output set.
		bd := p.CalculateCostBreakdown(c.model, TokenUsage{Input: c.in, Output: c.out}).Total
		if !approxEqual(got, bd) {
			t.Errorf("%s: wrapper %v != breakdown total %v", c.model, got, bd)
		}
	}
}
