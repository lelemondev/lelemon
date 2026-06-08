package service

import (
	"context"
	"log/slog"
	"sync"
	"testing"
)

// resetUnknownModels clears the global unknown-model counter after a test.
func resetUnknownModels(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		unknownModels.mu.Lock()
		unknownModels.counts = nil
		unknownModels.mu.Unlock()
	})
}

// captureHandler records emitted slog records by level for assertions.
type captureHandler struct {
	mu    *sync.Mutex
	byLvl map[slog.Level]int
}

func (h captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	h.byLvl[r.Level]++
	h.mu.Unlock()
	return nil
}
func (h captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h captureHandler) WithGroup(string) slog.Handler       { return h }

func captureLogs(t *testing.T) map[slog.Level]int {
	t.Helper()
	byLvl := make(map[slog.Level]int)
	prev := slog.Default()
	slog.SetDefault(slog.New(captureHandler{&sync.Mutex{}, byLvl}))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return byLvl
}

// TestRecordUnknown_CountsAndGetter tracks counts and exposes them.
func TestRecordUnknown_CountsAndGetter(t *testing.T) {
	resetUnknownModels(t)
	captureLogs(t)

	recordUnknownModel("ghost-a")
	recordUnknownModel("ghost-a")
	recordUnknownModel("ghost-b")

	snap := UnknownModels()
	if snap["ghost-a"] != 2 || snap["ghost-b"] != 1 {
		t.Errorf("counts = %v, want ghost-a:2 ghost-b:1", snap)
	}
}

// TestRecordUnknown_EmptyIgnored does not track empty model names.
func TestRecordUnknown_EmptyIgnored(t *testing.T) {
	resetUnknownModels(t)
	captureLogs(t)

	recordUnknownModel("")
	if len(UnknownModels()) != 0 {
		t.Errorf("empty model should not be recorded, got %v", UnknownModels())
	}
}

// TestRecordUnknown_LogsOnceAndAtThreshold logs Info on first sighting and Warn
// when crossing the threshold — and nothing in between.
func TestRecordUnknown_LogsOnceAndAtThreshold(t *testing.T) {
	resetUnknownModels(t)
	byLvl := captureLogs(t)

	for i := 0; i < unknownModelAlertThreshold; i++ {
		recordUnknownModel("ghost-threshold")
	}

	if got := UnknownModels()["ghost-threshold"]; got != int64(unknownModelAlertThreshold) {
		t.Fatalf("count = %d, want %d", got, unknownModelAlertThreshold)
	}
	if byLvl[slog.LevelInfo] != 1 {
		t.Errorf("info logs = %d, want 1 (first sighting only)", byLvl[slog.LevelInfo])
	}
	if byLvl[slog.LevelWarn] != 1 {
		t.Errorf("warn logs = %d, want 1 (threshold only)", byLvl[slog.LevelWarn])
	}
}

// TestFindPricing_RecordsUnknown wires findPricing → recordUnknownModel.
func TestFindPricing_RecordsUnknown(t *testing.T) {
	resetUnknownModels(t)
	resetSnapshot(t)
	captureLogs(t)

	if _, ok := findPricing("definitely-not-a-real-model-xyz"); ok {
		t.Fatal("expected unknown model")
	}
	if UnknownModels()["definitely-not-a-real-model-xyz"] != 1 {
		t.Errorf("findPricing did not record the unknown model: %v", UnknownModels())
	}

	// A known model must NOT be recorded as unknown.
	if _, ok := findPricing("gpt-4o"); !ok {
		t.Fatal("gpt-4o should be known")
	}
	if _, recorded := UnknownModels()["gpt-4o"]; recorded {
		t.Error("known model gpt-4o was wrongly recorded as unknown")
	}
}
