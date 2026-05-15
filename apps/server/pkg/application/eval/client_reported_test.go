package eval

import (
	"context"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

// Phase 2B: client_reported scorers let the SDK provide pass/fail verdicts for
// scorers the platform cannot or will not run server-side (LLM-as-judge with
// the customer's own provider key, domain-specific assertions, etc). The
// platform still ANDs them with its built-in scorers — a single mode that
// preserves the "platform decides the overall verdict" property.
//
// These tests drive the design RED → GREEN. They MUST fail on first run.

func TestCreateEval_AcceptsClientReportedScorer(t *testing.T) {
	svc, _, _, d, _ := makeServiceWithDataset(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "judge-eval",
		Scorers: []entity.Scorer{
			{ID: "judge", Type: entity.ScorerClientReported, Name: "LLM judge"},
		},
	})
	if err != nil {
		t.Fatalf("create with client_reported scorer should succeed, got %v", err)
	}
}

func TestPostResult_ClientReported_UsesClientScore(t *testing.T) {
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()

	ev, err := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "judge",
		Scorers: []entity.Scorer{{ID: "judge", Type: entity.ScorerClientReported}},
	})
	if err != nil {
		t.Fatalf("create eval: %v", err)
	}
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	// Client says passed=true for the judge — platform must honour it.
	clientPassed := []entity.ScorerResult{
		{ScorerID: "judge", Passed: true, Score: 0.92, Details: "judge approved"},
	}
	res, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID,
		Actual:        "anything",
		ClientScores:  clientPassed,
	})
	if err != nil {
		t.Fatalf("post result: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected overall passed when client_reported scorer passes, got %+v", res)
	}
	if len(res.Scores) != 1 || res.Scores[0].ScorerID != "judge" || !res.Scores[0].Passed {
		t.Errorf("scorer roundtrip lost data: %+v", res.Scores)
	}
	if res.Scores[0].Score != 0.92 {
		t.Errorf("client score not preserved verbatim: got %v", res.Scores[0].Score)
	}
	if res.Scores[0].Details != "judge approved" {
		t.Errorf("details not preserved: %q", res.Scores[0].Details)
	}
}

func TestPostResult_ClientReported_MissingScoreErrors(t *testing.T) {
	// The scorer is declared in the eval, but the client did not provide a
	// matching ClientScore. We surface that as an error result, NOT a silent
	// pass — pretending the case passed when it never ran would be deeply
	// wrong for a CI gate.
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "judge",
		Scorers: []entity.Scorer{{ID: "judge", Type: entity.ScorerClientReported}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	res, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID,
		Actual:        "x",
		// No ClientScores — judge was expected but client said nothing.
	})
	if err != nil {
		t.Fatalf("post result: %v", err)
	}
	if res.Passed {
		t.Errorf("missing client_reported score must NOT pass: %+v", res)
	}
	if len(res.Scores) != 1 || res.Scores[0].Error == "" {
		t.Errorf("expected one scorer with Error set, got %+v", res.Scores)
	}
}

func TestPostResult_BuiltInIgnoresClientOverride(t *testing.T) {
	// Built-in scorers run server-side; a malicious client cannot flip the
	// verdict by sending a fake clientScore for a server-side scorer id.
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "exact",
		Scorers: []entity.Scorer{{ID: "exact", Type: entity.ScorerExactMatch}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	// Item's expected is "honda" (set in makeServiceWithDataset). Client sends
	// the wrong actual but claims it passed.
	res, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID,
		Actual:        "WRONG",
		ClientScores: []entity.ScorerResult{
			{ScorerID: "exact", Passed: true, Score: 1.0, Details: "trust me"},
		},
	})
	if err != nil {
		t.Fatalf("post result: %v", err)
	}
	if res.Passed {
		t.Errorf("client override of a built-in scorer must be ignored — server-side scoring is the source of truth")
	}
	if res.Scores[0].Passed {
		t.Errorf("scorer s1 should be the server's verdict (failed), not the client's")
	}
}

func TestPostResult_MixedScorers_ANDsCorrectly(t *testing.T) {
	// One built-in + one client_reported. Overall passed iff BOTH pass.
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "mixed",
		Scorers: []entity.Scorer{
			{ID: "exact", Type: entity.ScorerExactMatch},
			{ID: "judge", Type: entity.ScorerClientReported},
		},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	// Built-in passes (item 0 has expected="honda", we send "honda") and the
	// client says judge passed → overall pass.
	res, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID,
		Actual:        "honda",
		ClientScores:  []entity.ScorerResult{{ScorerID: "judge", Passed: true, Score: 1}},
	})
	if err != nil {
		t.Fatalf("post result A: %v", err)
	}
	if !res.Passed {
		t.Errorf("both pass → overall must pass, got %+v", res)
	}

	// Built-in fails, client says judge passed → overall fail.
	res2, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[1].ID,
		Actual:        "WRONG",
		ClientScores:  []entity.ScorerResult{{ScorerID: "judge", Passed: true, Score: 1}},
	})
	if err != nil {
		t.Fatalf("post result B: %v", err)
	}
	if res2.Passed {
		t.Errorf("any scorer fail → overall must fail (AND), got %+v", res2)
	}
}

func TestPostResult_ClientScoreForUnknownScorerIgnored(t *testing.T) {
	// Client sends a ClientScore for a scorer id that the eval does NOT
	// declare. The service ignores the stray entry — no error, no leak.
	svc, _, _, d, items := makeServiceWithDataset(t)
	ctx := context.Background()
	ev, _ := svc.Create(ctx, "proj", CreateEvalRequest{
		DatasetID: d.ID, Name: "single",
		Scorers: []entity.Scorer{{ID: "judge", Type: entity.ScorerClientReported}},
	})
	run, _ := svc.StartRun(ctx, "proj", StartEvalRunRequest{EvalID: ev.ID})

	res, err := svc.PostResult(ctx, "proj", run.ID, PostEvalRunResultRequest{
		DatasetItemID: items[0].ID,
		Actual:        "x",
		ClientScores: []entity.ScorerResult{
			{ScorerID: "judge", Passed: true, Score: 1},
			{ScorerID: "ghost-scorer", Passed: true, Score: 1}, // not declared
		},
	})
	if err != nil {
		t.Fatalf("post result: %v", err)
	}
	if len(res.Scores) != 1 {
		t.Errorf("expected exactly the declared scorers in result, got %d", len(res.Scores))
	}
}

func TestValidateScorers_ClientReportedRequiresOnlyID(t *testing.T) {
	// client_reported is configurationless — only the scorer id matters.
	// Verify validation accepts an empty config.
	err := validateScorers([]entity.Scorer{
		{ID: "judge", Type: entity.ScorerClientReported},
	})
	if err != nil {
		t.Errorf("validation should accept client_reported with no config: %v", err)
	}
}

