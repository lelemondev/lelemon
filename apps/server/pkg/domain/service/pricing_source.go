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

// StartPricingRefresh launches a background loop that refreshes the pricing table
// from `url` at boot and every `interval`. It never blocks startup and always
// keeps the previous (or local) table on failure, so the server is offline-safe.
// The loop exits when ctx is cancelled. Call once at server startup.
func StartPricingRefresh(ctx context.Context, url string, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultPricingRefreshInterval
	}
	go func() {
		refreshPricingOnce(ctx, url) // best-effort at boot
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refreshPricingOnce(ctx, url)
			}
		}
	}()
}

// refreshPricingOnce fetches the external table once and applies it on success.
// Failures are logged and swallowed (the existing table stays in effect).
func refreshPricingOnce(ctx context.Context, url string) {
	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	external, err := FetchLiteLLMPricing(fetchCtx, url)
	if err != nil {
		slog.Warn("pricing refresh failed; keeping current table", "error", err)
		return
	}

	applyExternalPricing(external)
	slog.Info("pricing table refreshed", "source", "litellm", "models", len(external))
}
