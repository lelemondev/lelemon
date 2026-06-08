package service

import "testing"

// resetSnapshot restores the "no external data" state after a test so the global
// pricing snapshot can't leak across tests.
func resetSnapshot(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		pricingSnapshot.mu.Lock()
		pricingSnapshot.table = nil
		pricingSnapshot.mu.Unlock()
	})
}

// TestCurrentTable_DefaultsToLocal returns the local map when no external data.
func TestCurrentTable_DefaultsToLocal(t *testing.T) {
	resetSnapshot(t)
	if got := currentPricingTable(); len(got) != len(pricing) {
		t.Errorf("default table size = %d, want local %d", len(got), len(pricing))
	}
}

// TestApplyExternal_AddsNewModel makes a model unknown to the local table priceable.
func TestApplyExternal_AddsNewModel(t *testing.T) {
	resetSnapshot(t)
	const model = "brand-new-model-9000"

	if _, ok := findPricing(model); ok {
		t.Fatal("model should be unknown before external load")
	}

	applyExternalPricing(map[string]ModelPricing{
		model: {Input: 0.001, Output: 0.004},
	})

	mp, ok := findPricing(model)
	if !ok {
		t.Fatal("model should be priced after external load")
	}
	if !approxEqual(mp.Input, 0.001) || !approxEqual(mp.Output, 0.004) {
		t.Errorf("got %+v, want input 0.001 output 0.004", mp)
	}
}

// TestApplyExternal_OverridesLocal lets external pricing win for a shared key,
// and exact LiteLLM cache rates survive deriveRates.
func TestApplyExternal_OverridesLocal(t *testing.T) {
	resetSnapshot(t)

	applyExternalPricing(map[string]ModelPricing{
		"gpt-4o": {Input: 0.003, Output: 0.012, CacheRead: 0.0015},
	})

	mp, ok := findPricing("gpt-4o")
	if !ok {
		t.Fatal("gpt-4o missing")
	}
	if !approxEqual(mp.Input, 0.003) || !approxEqual(mp.Output, 0.012) {
		t.Errorf("override failed: got in/out %v/%v, want 0.003/0.012", mp.Input, mp.Output)
	}
	// Explicit external cache rate must be preserved (not replaced by deriveRates' 0.5x).
	if !approxEqual(mp.CacheRead, 0.0015) {
		t.Errorf("explicit cacheRead lost: got %v, want 0.0015", mp.CacheRead)
	}
}

// TestApplyExternal_ZeroDoesNotWipeLocal: a malformed external row (no cost)
// must not override a good local price.
func TestApplyExternal_ZeroDoesNotWipeLocal(t *testing.T) {
	resetSnapshot(t)

	local, ok := findPricing("gpt-4o")
	if !ok {
		t.Fatal("gpt-4o missing from local")
	}

	applyExternalPricing(map[string]ModelPricing{
		"gpt-4o": {Input: 0, Output: 0},
	})

	after, _ := findPricing("gpt-4o")
	if !approxEqual(after.Input, local.Input) || !approxEqual(after.Output, local.Output) {
		t.Errorf("zero external row wiped local price: before %v/%v after %v/%v",
			local.Input, local.Output, after.Input, after.Output)
	}
}

// TestApplyExternal_PrefixStillWorks: prefix matching applies to the merged table.
func TestApplyExternal_PrefixStillWorks(t *testing.T) {
	resetSnapshot(t)
	applyExternalPricing(map[string]ModelPricing{
		"my-llm": {Input: 0.002, Output: 0.006},
	})
	if mp, ok := findPricing("my-llm-v2-20260101"); !ok || !approxEqual(mp.Input, 0.002) {
		t.Errorf("prefix match on external failed: ok=%v mp=%+v", ok, mp)
	}
}
