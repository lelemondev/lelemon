package trace

import (
	"math"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

func ptrInt(v int) *int          { return &v }
func ptrStr(v string) *string    { return &v }
func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestComputeSpanCostBreakdown_AnthropicCache verifies the per-token-type
// decomposition and cache savings for an Anthropic LLM span.
func TestComputeSpanCostBreakdown_AnthropicCache(t *testing.T) {
	span := entity.Span{
		Type:             entity.SpanTypeLLM,
		Model:            ptrStr("claude-3-5-sonnet-20241022"),
		Provider:         ptrStr("anthropic"),
		InputTokens:      ptrInt(1000),
		OutputTokens:     ptrInt(500),
		CacheReadTokens:  ptrInt(2000),
		CacheWriteTokens: ptrInt(1000),
	}

	b := computeSpanCostBreakdown(span)
	if b == nil {
		t.Fatal("expected a breakdown, got nil")
	}

	// Input 0.003, Output 0.0075, CacheRead 0.0006, CacheWrite 0.00375, total 0.01485.
	// Savings: 2000/1000 * (0.003 - 0.0003) = 0.0054.
	checks := []struct {
		name string
		got  float64
		want float64
	}{
		{"input", b.Input, 0.003},
		{"output", b.Output, 0.0075},
		{"cacheRead", b.CacheRead, 0.0006},
		{"cacheWrite", b.CacheWrite, 0.00375},
		{"reasoning", b.Reasoning, 0},
		{"total", b.Total, 0.01485},
		{"cacheSavings", b.CacheSavings, 0.0054},
	}
	for _, c := range checks {
		if !approxEq(c.got, c.want) {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// TestComputeSpanCostBreakdown_NonLLM returns nil for non-LLM spans and spans
// without a model.
func TestComputeSpanCostBreakdown_NonLLM(t *testing.T) {
	if b := computeSpanCostBreakdown(entity.Span{Type: entity.SpanTypeTool, Name: "search"}); b != nil {
		t.Errorf("tool span should have no breakdown, got %+v", b)
	}
	if b := computeSpanCostBreakdown(entity.Span{Type: entity.SpanTypeLLM}); b != nil {
		t.Errorf("LLM span without model should have no breakdown, got %+v", b)
	}
}
