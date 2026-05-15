package eval

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/lelemon/server/pkg/domain/entity"
)

// Score applies a single Scorer to (expected, actual) and returns a verdict.
//
// "expected" is the dataset_item.expected value — meaningful only for the
// exact_match scorer. The other scorers carry their target inside Scorer.Config.
// Scoring is pure: same inputs → same output. Errors inside a scorer (bad
// config, type mismatch) yield a not-passed result with Error set, NOT a
// thrown error. The run-level error path is reserved for caller-reported
// execution failures (the target itself blew up).
func Score(scorer entity.Scorer, expected, actual any) entity.ScorerResult {
	switch scorer.Type {
	case entity.ScorerExactMatch:
		return scoreExactMatch(scorer.ID, expected, actual)
	case entity.ScorerContains:
		return scoreContains(scorer.ID, scorer.Config, actual)
	case entity.ScorerJSONPath:
		return scoreJSONPath(scorer.ID, scorer.Config, actual)
	case entity.ScorerRegex:
		return scoreRegex(scorer.ID, scorer.Config, actual)
	default:
		return entity.ScorerResult{
			ScorerID: scorer.ID,
			Passed:   false,
			Score:    0,
			Error:    fmt.Sprintf("unknown scorer type %q", scorer.Type),
		}
	}
}

// ----- exact_match -------------------------------------------------------

func scoreExactMatch(scorerID string, expected, actual any) entity.ScorerResult {
	if expected == nil {
		return entity.ScorerResult{
			ScorerID: scorerID,
			Passed:   false,
			Score:    0,
			Error:    "exact_match requires the dataset item to have an 'expected' value",
		}
	}
	passed := jsonDeepEqual(expected, actual)
	return verdict(scorerID, passed, "")
}

// ----- contains ----------------------------------------------------------

func scoreContains(scorerID string, config map[string]any, actual any) entity.ScorerResult {
	target, ok := config["value"]
	if !ok {
		return errResult(scorerID, "contains requires config.value")
	}

	switch a := actual.(type) {
	case string:
		needle, ok := target.(string)
		if !ok {
			return errResult(scorerID, "contains: actual is a string but config.value is not")
		}
		return verdict(scorerID, strings.Contains(a, needle), "")

	case []any:
		for _, el := range a {
			if jsonDeepEqual(el, target) {
				return verdict(scorerID, true, "")
			}
		}
		return verdict(scorerID, false, "")

	case map[string]any:
		key, ok := target.(string)
		if !ok {
			return errResult(scorerID, "contains: actual is a map but config.value is not a key string")
		}
		_, exists := a[key]
		return verdict(scorerID, exists, "")

	case nil:
		return errResult(scorerID, "contains: actual is nil")

	default:
		return errResult(scorerID, fmt.Sprintf("contains: unsupported actual type %T", actual))
	}
}

// ----- json_path ---------------------------------------------------------

func scoreJSONPath(scorerID string, config map[string]any, actual any) entity.ScorerResult {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return errResult(scorerID, "json_path requires config.path (string)")
	}
	opStr, ok := config["op"].(string)
	if !ok || opStr == "" {
		return errResult(scorerID, "json_path requires config.op (eq|ne|gt|gte|lt|lte)")
	}
	op := entity.ScorerOp(opStr)
	want, hasWant := config["value"]
	if !hasWant {
		return errResult(scorerID, "json_path requires config.value")
	}

	got, err := extractPath(actual, path)
	if err != nil {
		return errResult(scorerID, err.Error())
	}

	passed, err := compareOp(op, got, want)
	if err != nil {
		return errResult(scorerID, err.Error())
	}
	details := fmt.Sprintf("at %s: %v %s %v", path, got, op, want)
	return verdict(scorerID, passed, details)
}

// extractPath walks a dotted path through nested JSON. Path segments are map
// keys; a segment that parses as a non-negative integer is treated as an
// array index. Returns (value, nil) on hit, (nil, err) on miss/type-fail.
func extractPath(root any, path string) (any, error) {
	if path == "" {
		return root, nil
	}
	cur := root
	segments := strings.Split(path, ".")
	for i, seg := range segments {
		if cur == nil {
			return nil, fmt.Errorf("path %s: hit nil at segment %d (%q)", path, i, seg)
		}
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[seg]
			if !ok {
				return nil, fmt.Errorf("path %s: key %q not found", path, seg)
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 {
				return nil, fmt.Errorf("path %s: segment %q is not a non-negative integer index for array", path, seg)
			}
			if idx >= len(v) {
				return nil, fmt.Errorf("path %s: index %d out of range (len=%d)", path, idx, len(v))
			}
			cur = v[idx]
		default:
			return nil, fmt.Errorf("path %s: segment %q cannot descend into %T", path, seg, cur)
		}
	}
	return cur, nil
}

func compareOp(op entity.ScorerOp, got, want any) (bool, error) {
	switch op {
	case entity.ScorerOpEq:
		return jsonDeepEqual(got, want), nil
	case entity.ScorerOpNe:
		return !jsonDeepEqual(got, want), nil
	case entity.ScorerOpGt, entity.ScorerOpGte, entity.ScorerOpLt, entity.ScorerOpLte:
		gn, ok1 := toFloat(got)
		wn, ok2 := toFloat(want)
		if !ok1 || !ok2 {
			return false, fmt.Errorf("op %s requires numeric operands, got %T vs %T", op, got, want)
		}
		switch op {
		case entity.ScorerOpGt:
			return gn > wn, nil
		case entity.ScorerOpGte:
			return gn >= wn, nil
		case entity.ScorerOpLt:
			return gn < wn, nil
		case entity.ScorerOpLte:
			return gn <= wn, nil
		}
	}
	return false, fmt.Errorf("unsupported op %q", op)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	}
	return 0, false
}

// ----- regex -------------------------------------------------------------

func scoreRegex(scorerID string, config map[string]any, actual any) entity.ScorerResult {
	pattern, ok := config["pattern"].(string)
	if !ok || pattern == "" {
		return errResult(scorerID, "regex requires config.pattern (string)")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return errResult(scorerID, fmt.Sprintf("regex compile: %v", err))
	}
	s, ok := actual.(string)
	if !ok {
		return errResult(scorerID, fmt.Sprintf("regex: actual must be a string, got %T", actual))
	}
	return verdict(scorerID, re.MatchString(s), "")
}

// ----- shared helpers ----------------------------------------------------

// jsonDeepEqual compares two values produced by json.Unmarshal. reflect.DeepEqual
// works well for that shape (consistent typing for numbers/objects/arrays); we
// only call it through this helper to make intent explicit.
func jsonDeepEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

func verdict(scorerID string, passed bool, details string) entity.ScorerResult {
	score := 0.0
	if passed {
		score = 1.0
	}
	return entity.ScorerResult{
		ScorerID: scorerID,
		Passed:   passed,
		Score:    score,
		Details:  details,
	}
}

func errResult(scorerID, msg string) entity.ScorerResult {
	return entity.ScorerResult{
		ScorerID: scorerID,
		Passed:   false,
		Score:    0,
		Error:    msg,
	}
}

// ScoreAll runs every scorer in `scorers` against (expected, actual) and
// returns the per-scorer verdicts plus a single overall pass flag (AND).
//
// Equivalent to ScoreAllWithClient(scorers, expected, actual, nil) — kept as
// a thin alias so older callers (and tests in scoring_test.go) read cleanly.
func ScoreAll(scorers []entity.Scorer, expected, actual any) ([]entity.ScorerResult, bool) {
	return ScoreAllWithClient(scorers, expected, actual, nil)
}

// ScoreAllWithClient is the canonical scorer driver — runs each scorer to
// produce a verdict, AND-ing them into a single overall pass flag.
//
// For each scorer:
//
//   - ScorerClientReported: the caller is expected to provide a matching entry
//     in `clientScores` (keyed by scorer id). The entry is stored verbatim.
//     Missing entries produce an error verdict (NOT a silent pass — a CI gate
//     that fakes results is a broken CI gate).
//   - Any other type: scored server-side via Score(). The client cannot
//     override built-in verdicts — a malicious clientScore entry for a
//     server-side id is ignored.
//
// Scorer errors fail-but-don't-throw — they bubble up through Error and
// contribute to the overall failure.
func ScoreAllWithClient(scorers []entity.Scorer, expected, actual any, clientScores []entity.ScorerResult) ([]entity.ScorerResult, bool) {
	if len(scorers) == 0 {
		return []entity.ScorerResult{}, false // no scorers ≠ passing
	}

	// Build an index of client-supplied verdicts keyed by scorer id. Stray
	// entries (scorer id not declared on the eval) drop out naturally.
	clientByID := make(map[string]entity.ScorerResult, len(clientScores))
	for _, c := range clientScores {
		clientByID[c.ScorerID] = c
	}

	results := make([]entity.ScorerResult, 0, len(scorers))
	allPassed := true
	for _, sc := range scorers {
		var r entity.ScorerResult
		if sc.Type == entity.ScorerClientReported {
			cs, ok := clientByID[sc.ID]
			if !ok {
				r = errResult(sc.ID, "client_reported scorer requires a matching entry in clientScores")
			} else {
				// Trust the client verbatim — but always tag the id so it
				// matches the scorer (defence in depth against a confused
				// client filling in the wrong id field).
				cs.ScorerID = sc.ID
				r = cs
			}
		} else {
			r = Score(sc, expected, actual)
		}
		if !r.Passed {
			allPassed = false
		}
		results = append(results, r)
	}
	return results, allPassed
}
