package eval

import (
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

func TestScore_ExactMatch(t *testing.T) {
	sc := entity.Scorer{ID: "s1", Type: entity.ScorerExactMatch}

	t.Run("string equal", func(t *testing.T) {
		got := Score(sc, "hello", "hello")
		assertPassed(t, got, true)
	})
	t.Run("string not equal", func(t *testing.T) {
		got := Score(sc, "hello", "world")
		assertPassed(t, got, false)
	})
	t.Run("nested map equal", func(t *testing.T) {
		exp := map[string]any{"a": float64(1), "b": []any{"x"}}
		got := Score(sc, exp, map[string]any{"a": float64(1), "b": []any{"x"}})
		assertPassed(t, got, true)
	})
	t.Run("nested map not equal", func(t *testing.T) {
		exp := map[string]any{"a": float64(1)}
		got := Score(sc, exp, map[string]any{"a": float64(2)})
		assertPassed(t, got, false)
	})
	t.Run("nil expected → error result, not pass", func(t *testing.T) {
		got := Score(sc, nil, "anything")
		assertPassed(t, got, false)
		if got.Error == "" {
			t.Errorf("expected non-empty Error when expected is nil")
		}
	})
}

func TestScore_Contains(t *testing.T) {
	t.Run("string substring hit", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerContains, Config: map[string]any{"value": "civic"}}
		assertPassed(t, Score(sc, nil, "honda civic 2020"), true)
	})
	t.Run("string substring miss", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerContains, Config: map[string]any{"value": "lambo"}}
		assertPassed(t, Score(sc, nil, "honda civic 2020"), false)
	})
	t.Run("array contains primitive", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerContains, Config: map[string]any{"value": "tool_use"}}
		assertPassed(t, Score(sc, nil, []any{"text", "tool_use", "thinking"}), true)
	})
	t.Run("array contains object", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerContains, Config: map[string]any{"value": map[string]any{"type": "tool_use"}}}
		assertPassed(t, Score(sc, nil, []any{map[string]any{"type": "tool_use"}, map[string]any{"type": "text"}}), true)
	})
	t.Run("map has key", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerContains, Config: map[string]any{"value": "tool_calls"}}
		assertPassed(t, Score(sc, nil, map[string]any{"role": "assistant", "tool_calls": []any{}}), true)
	})
	t.Run("missing config.value → error", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerContains, Config: map[string]any{}}
		r := Score(sc, nil, "x")
		if r.Error == "" {
			t.Errorf("want Error for missing config.value")
		}
	})
}

func TestScore_JSONPath(t *testing.T) {
	actual := map[string]any{
		"results": []any{
			map[string]any{"id": "v1", "price": float64(15000)},
			map[string]any{"id": "v2", "price": float64(22000)},
		},
		"meta": map[string]any{"count": float64(2)},
	}

	t.Run("eq on string deep in arrays", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "results.0.id", "op": "eq", "value": "v1",
		}}
		assertPassed(t, Score(sc, nil, actual), true)
	})
	t.Run("ne with mismatch passes", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "meta.count", "op": "ne", "value": float64(0),
		}}
		assertPassed(t, Score(sc, nil, actual), true)
	})
	t.Run("gt on numeric path", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "meta.count", "op": "gt", "value": float64(1),
		}}
		assertPassed(t, Score(sc, nil, actual), true)
	})
	t.Run("gte boundary", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "meta.count", "op": "gte", "value": float64(2),
		}}
		assertPassed(t, Score(sc, nil, actual), true)
	})
	t.Run("lt fails when not strictly less", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "meta.count", "op": "lt", "value": float64(2),
		}}
		assertPassed(t, Score(sc, nil, actual), false)
	})
	t.Run("missing key path → error", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "results.99.id", "op": "eq", "value": "x",
		}}
		r := Score(sc, nil, actual)
		if r.Error == "" {
			t.Errorf("expected Error for out-of-range index")
		}
		assertPassed(t, r, false)
	})
	t.Run("numeric op on non-numeric → error", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "results.0.id", "op": "gt", "value": float64(1),
		}}
		r := Score(sc, nil, actual)
		if r.Error == "" {
			t.Errorf("want Error when comparing string with gt")
		}
	})
	t.Run("missing op → error", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerJSONPath, Config: map[string]any{
			"path": "meta.count", "value": float64(1),
		}}
		r := Score(sc, nil, actual)
		if r.Error == "" {
			t.Errorf("want Error for missing op")
		}
	})
}

func TestScore_Regex(t *testing.T) {
	t.Run("pattern matches", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerRegex, Config: map[string]any{"pattern": `^honda \w+$`}}
		assertPassed(t, Score(sc, nil, "honda civic"), true)
	})
	t.Run("pattern misses", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerRegex, Config: map[string]any{"pattern": `^toyota`}}
		assertPassed(t, Score(sc, nil, "honda civic"), false)
	})
	t.Run("non-string actual → error", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerRegex, Config: map[string]any{"pattern": `.`}}
		r := Score(sc, nil, float64(1))
		if r.Error == "" {
			t.Errorf("want Error for non-string actual")
		}
	})
	t.Run("bad pattern → error not panic", func(t *testing.T) {
		sc := entity.Scorer{ID: "s", Type: entity.ScorerRegex, Config: map[string]any{"pattern": `(unclosed`}}
		r := Score(sc, nil, "x")
		if r.Error == "" {
			t.Errorf("want Error for invalid regex")
		}
	})
}

func TestScore_UnknownType(t *testing.T) {
	sc := entity.Scorer{ID: "s", Type: "bogus"}
	r := Score(sc, nil, "x")
	if r.Error == "" {
		t.Errorf("want Error for unknown scorer type")
	}
	if r.Passed {
		t.Errorf("unknown scorer must not pass")
	}
}

func TestScoreAll_AndsResults(t *testing.T) {
	scorers := []entity.Scorer{
		{ID: "a", Type: entity.ScorerExactMatch},
		{ID: "b", Type: entity.ScorerContains, Config: map[string]any{"value": "v"}},
	}

	t.Run("all pass → passed=true", func(t *testing.T) {
		results, passed := ScoreAll(scorers, "vv", "vv")
		if !passed {
			t.Errorf("expected overall passed")
		}
		if len(results) != 2 {
			t.Errorf("want 2 results, got %d", len(results))
		}
		for _, r := range results {
			if !r.Passed {
				t.Errorf("scorer %s should pass: %+v", r.ScorerID, r)
			}
		}
	})
	t.Run("one fails → passed=false", func(t *testing.T) {
		_, passed := ScoreAll(scorers, "vv", "different")
		if passed {
			t.Errorf("expected overall fail")
		}
	})
	t.Run("empty scorer list → not passed", func(t *testing.T) {
		// An eval with zero scorers is meaningless — refuse to call it green.
		_, passed := ScoreAll(nil, "x", "x")
		if passed {
			t.Errorf("no scorers must yield not-passed (defensive)")
		}
	})
}

// --- helpers ---

func assertPassed(t *testing.T, r entity.ScorerResult, want bool) {
	t.Helper()
	if r.Passed != want {
		t.Errorf("passed: want %v, got %v (details=%q error=%q)", want, r.Passed, r.Details, r.Error)
	}
	expectedScore := 0.0
	if want {
		expectedScore = 1.0
	}
	if r.Score != expectedScore {
		t.Errorf("score: want %v, got %v", expectedScore, r.Score)
	}
}
