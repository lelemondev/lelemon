package service

import (
	"log/slog"
	"sync"
)

// unknownModelAlertThreshold is the request count at which an unpriced model is
// escalated from info to a warning — it's silently being costed at $0, which
// skews analytics, so an operator should add/alias it.
const unknownModelAlertThreshold = 10

// unknownModels counts requests for models we couldn't price, since process
// start. Used for observability (logs + an admin/metrics getter).
var unknownModels struct {
	mu     sync.Mutex
	counts map[string]int64
}

// recordUnknownModel tracks one request for a model with no pricing. It logs
// exactly once on first sighting and once when the model crosses the alert
// threshold, so a busy unpriced model surfaces without flooding the logs.
func recordUnknownModel(model string) {
	if model == "" {
		return
	}

	unknownModels.mu.Lock()
	if unknownModels.counts == nil {
		unknownModels.counts = make(map[string]int64)
	}
	unknownModels.counts[model]++
	n := unknownModels.counts[model]
	unknownModels.mu.Unlock()

	switch n {
	case 1:
		slog.Info("model has no pricing; cost reported as $0", "model", model)
	case unknownModelAlertThreshold:
		slog.Warn("model still unpriced after many requests; costs are under-reported",
			"model", model, "requests", n)
	}
}

// UnknownModels returns a snapshot of models seen without pricing and their
// request counts (since process start). Intended for an admin/metrics view.
func UnknownModels() map[string]int64 {
	unknownModels.mu.Lock()
	defer unknownModels.mu.Unlock()

	out := make(map[string]int64, len(unknownModels.counts))
	for k, v := range unknownModels.counts {
		out[k] = v
	}
	return out
}
