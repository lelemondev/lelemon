package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// DefaultPricingRefreshInterval is how often the external pricing table is
// refreshed in the background.
const DefaultPricingRefreshInterval = 24 * time.Hour

// pricingSnapshot holds the effective pricing table: the local hardcoded map
// overlaid with the external source (LiteLLM). A nil table means "no external
// data yet — use the local `pricing` map directly". Swapped atomically on each
// successful refresh so lookups never see a partially-built table.
var pricingSnapshot struct {
	mu    sync.RWMutex
	table map[string]ModelPricing
}

// currentPricingTable returns the table findPricing should resolve against:
// the merged snapshot if a refresh has succeeded, otherwise the local map.
func currentPricingTable() map[string]ModelPricing {
	pricingSnapshot.mu.RLock()
	t := pricingSnapshot.table
	pricingSnapshot.mu.RUnlock()
	if t != nil {
		return t
	}
	return pricing
}

// applyExternalPricing builds a new effective table = local map overlaid with
// `external` (external wins per key) and installs it. An external entry with no
// input AND no output cost is ignored so a malformed upstream row can't wipe a
// good local price.
func applyExternalPricing(external map[string]ModelPricing) {
	merged := make(map[string]ModelPricing, len(pricing)+len(external))
	for k, v := range pricing {
		merged[k] = v
	}
	for k, v := range external {
		if v.Input == 0 && v.Output == 0 {
			continue
		}
		merged[k] = v
	}

	pricingSnapshot.mu.Lock()
	pricingSnapshot.table = merged
	pricingSnapshot.mu.Unlock()
}

// PricingSources configures the external pricing refresh. An empty URL disables
// that source. Precedence is LiteLLM (primary) > OpenRouter (secondary) > the
// built-in local map (fallback).
type PricingSources struct {
	LiteLLMURL    string
	OpenRouterURL string
	Interval      time.Duration
}

// fetchSourceTimeout bounds each individual source fetch.
const fetchSourceTimeout = 30 * time.Second

// StartPricingRefresh launches a background loop that refreshes the pricing table
// from the configured sources at boot and every Interval. It never blocks startup
// and always keeps the previous (or local) table on failure, so the server is
// offline-safe. The loop exits when ctx is cancelled. Call once at server startup.
func StartPricingRefresh(ctx context.Context, sources PricingSources) {
	if sources.Interval <= 0 {
		sources.Interval = DefaultPricingRefreshInterval
	}
	go func() {
		refreshPricingOnce(ctx, sources) // best-effort at boot
		ticker := time.NewTicker(sources.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshPricingOnce(ctx, sources)
			}
		}
	}()
}

// fetchSource runs one source fetch under a timeout.
func fetchSource(
	ctx context.Context,
	fn func(context.Context, string) (map[string]ModelPricing, error),
	url string,
) (map[string]ModelPricing, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, fetchSourceTimeout)
	defer cancel()
	return fn(fetchCtx, url)
}

// refreshPricingOnce fetches the configured sources, merges them (LiteLLM wins
// over OpenRouter), and applies the result. Per-source failures are logged and
// swallowed; if every source fails, the current table is left untouched.
func refreshPricingOnce(ctx context.Context, sources PricingSources) {
	combined := make(map[string]ModelPricing)

	// Secondary first (lower precedence)...
	if sources.OpenRouterURL != "" {
		if m, err := fetchSource(ctx, FetchOpenRouterPricing, sources.OpenRouterURL); err != nil {
			slog.Warn("openrouter pricing refresh failed", "error", err)
		} else {
			for k, v := range m {
				combined[k] = v
			}
			slog.Info("pricing source refreshed", "source", "openrouter", "models", len(m))
		}
	}

	// ...then primary overlaid on top (LiteLLM wins).
	if sources.LiteLLMURL != "" {
		if m, err := fetchSource(ctx, FetchLiteLLMPricing, sources.LiteLLMURL); err != nil {
			slog.Warn("litellm pricing refresh failed", "error", err)
		} else {
			for k, v := range m {
				combined[k] = v
			}
			slog.Info("pricing source refreshed", "source", "litellm", "models", len(m))
		}
	}

	if len(combined) == 0 {
		slog.Warn("pricing refresh produced no data; keeping current table")
		return
	}

	applyExternalPricing(combined)
	slog.Info("pricing table refreshed", "models", len(combined))
}
