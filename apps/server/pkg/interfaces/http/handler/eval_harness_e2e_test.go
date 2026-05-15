package handler_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/lelemon/server/pkg/application/analytics"
	appauth "github.com/lelemon/server/pkg/application/auth"
	"github.com/lelemon/server/pkg/application/dataset"
	"github.com/lelemon/server/pkg/application/eval"
	"github.com/lelemon/server/pkg/application/ingest"
	"github.com/lelemon/server/pkg/application/project"
	"github.com/lelemon/server/pkg/application/prompt"
	"github.com/lelemon/server/pkg/application/trace"
	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/service"
	"github.com/lelemon/server/pkg/infrastructure/auth"
	"github.com/lelemon/server/pkg/infrastructure/store/sqlite"
	apphttp "github.com/lelemon/server/pkg/interfaces/http"
)

// TestEvalHarness_EndToEnd walks the full SDK-driven flow that a customer's
// CI would execute against the platform:
//
//  1. Define a prompt + version.
//  2. Curate a dataset with eval cases.
//  3. Create an eval mixing built-in (exact_match) + client_reported scorers.
//  4. Start a run referencing the prompt version.
//  5. Post per-item results — some clean, some with a wrong actual, some
//     with errors, exercising the AND-of-scorers semantics across mixed
//     types.
//  6. Finalize → server computes aggregates from posted results.
//  7. Verify the run lands in the prompt-version payoff list (the loop
//     closing on itself).
//
// Real HTTP via httptest, real SQLite, real router — no mocks below the
// service interface. This is the gate Phase 2A/2B/3A all converge on; if it
// passes, the CI workflow described in the spec actually works end-to-end.
func TestEvalHarness_EndToEnd(t *testing.T) {
	h := newHarness(t)

	// ----- 1. Prompt + version (dashboard scope, JWT user) -----
	promptID := h.createPrompt("agent-system", "WhatsApp sales agent")
	versionID := h.createPromptVersion(promptID, "v1",
		"You are a helpful sales agent.\nUse the tools when relevant.",
		"initial cut")

	// ----- 2. Dataset + items -----
	datasetID := h.createDataset("vehicle-search-regressions")
	itemHonda := h.createDatasetItem(datasetID, "honda civic", "honda civic")
	itemLambo := h.createDatasetItem(datasetID, "lambo", "no-stock")
	itemTypo := h.createDatasetItem(datasetID, "lamborgini", "lamborghini")

	// ----- 3. Eval — built-in exact_match + a client_reported judge -----
	evalID := h.createEval(datasetID, "exact-and-judge",
		[]map[string]any{
			{"id": "exact", "type": "exact_match"},
			{"id": "judge", "type": "client_reported"},
		},
	)

	// ----- 4. Start a run (API-key scope, simulates the SDK harness) -----
	runID := h.startRun(evalID, versionID)

	// ----- 5. Post per-item results -----

	// Item 1: actual matches exact-match expected ("honda civic"); client says
	// judge passed. Both pass → overall pass.
	dur := 120
	r1 := h.postResult(runID, postResultPayload{
		DatasetItemID: itemHonda,
		Actual:        "honda civic",
		DurationMs:    &dur,
		ClientScores:  []scorerResult{{ScorerID: "judge", Passed: true, Score: 1}},
	})
	if !r1.Passed {
		t.Fatalf("item 1: want overall pass, got %+v", r1)
	}

	// Item 2: actual matches expected ("no-stock"); but the client says judge
	// FAILED (the response sounded snarky). Overall must fail thanks to AND.
	r2 := h.postResult(runID, postResultPayload{
		DatasetItemID: itemLambo,
		Actual:        "no-stock",
		ClientScores:  []scorerResult{{ScorerID: "judge", Passed: false, Score: 0, Details: "rude tone"}},
	})
	if r2.Passed {
		t.Fatalf("item 2: client_reported should fail overall, got %+v", r2)
	}

	// Item 3: wrong actual ("WRONG"). Even if the client tries to override the
	// built-in scorer's verdict, the server-side exact_match must reject. We
	// also include a missing client_reported entry — should produce an error
	// on the judge scorer (not silent pass).
	r3 := h.postResult(runID, postResultPayload{
		DatasetItemID: itemTypo,
		Actual:        "WRONG",
		// Client tries to lie about the built-in scorer:
		ClientScores: []scorerResult{
			{ScorerID: "exact", Passed: true, Score: 1, Details: "trust me"},
			// no entry for "judge" — must produce an error
		},
	})
	if r3.Passed {
		t.Fatalf("item 3: server-side scoring must override client lie, got %+v", r3)
	}
	if !containsScorerError(r3, "judge") {
		t.Fatalf("item 3: missing client_reported entry should produce error result, got %+v", r3.Scores)
	}
	// Defensive: the "exact" scorer should reflect the server's verdict (fail),
	// not the client's "trust me".
	exactScore := findScore(r3, "exact")
	if exactScore == nil || exactScore.Passed {
		t.Fatalf("item 3: server-side exact_match must be the source of truth, got %+v", exactScore)
	}

	// ----- 6. Finalize -----
	finalized := h.finalizeRun(runID, "completed")
	if finalized.TotalItems != 3 {
		t.Fatalf("totalItems: want 3, got %d", finalized.TotalItems)
	}
	if finalized.PassedItems != 1 {
		t.Fatalf("passedItems: want 1, got %d", finalized.PassedItems)
	}
	if finalized.FailedItems != 2 {
		t.Fatalf("failedItems: want 2, got %d", finalized.FailedItems)
	}
	if finalized.Status != "completed" {
		t.Fatalf("status: want completed, got %s", finalized.Status)
	}
	if finalized.PromptVersionID == nil || *finalized.PromptVersionID != versionID {
		t.Fatalf("promptVersionId not threaded through: got %v, want %s", finalized.PromptVersionID, versionID)
	}

	// ----- 7. Payoff: the version page can list its runs -----
	runs := h.listRunsByPromptVersion(versionID)
	if len(runs) != 1 || runs[0].ID != runID {
		t.Fatalf("prompt-version payoff: want exactly our run, got %+v", runs)
	}

	// And it's idempotent — finalize again returns the same row.
	finalizedAgain := h.finalizeRun(runID, "completed")
	if finalizedAgain.PassedItems != finalized.PassedItems {
		t.Fatalf("re-finalize must be idempotent: %+v vs %+v", finalized, finalizedAgain)
	}
}

// TestEvalHarness_PostAfterFinalize_Rejects covers the CI safety property: a
// late-arriving result must not silently mutate a closed run.
func TestEvalHarness_PostAfterFinalize_Rejects(t *testing.T) {
	h := newHarness(t)

	datasetID := h.createDataset("ds")
	itemID := h.createDatasetItem(datasetID, "x", "x")
	evalID := h.createEval(datasetID, "e",
		[]map[string]any{{"id": "exact", "type": "exact_match"}},
	)
	runID := h.startRun(evalID, "")
	h.postResult(runID, postResultPayload{DatasetItemID: itemID, Actual: "x"})
	h.finalizeRun(runID, "completed")

	// Now try to post another result — must 409.
	body, _ := json.Marshal(postResultPayload{DatasetItemID: itemID, Actual: "x"})
	resp, err := h.do("POST",
		fmt.Sprintf("/api/v1/eval-runs/%s/results", runID),
		body,
		map[string]string{"Authorization": "Bearer " + h.apiKey},
	)
	if err != nil {
		t.Fatalf("post after finalize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("post after finalize: want 409 Conflict, got %d (%s)", resp.StatusCode, raw)
	}
}

// TestEvalHarness_DatasetItemMustBelongToEvalDataset covers the anti-leak
// property at the HTTP layer.
func TestEvalHarness_DatasetItemMustBelongToEvalDataset(t *testing.T) {
	h := newHarness(t)

	// Two datasets in the same project. The eval targets dataset A, but we'll
	// try to post a result for an item from dataset B.
	dsA := h.createDataset("A")
	dsB := h.createDataset("B")
	itemB := h.createDatasetItem(dsB, "from-b", "from-b")

	evalID := h.createEval(dsA, "e",
		[]map[string]any{{"id": "exact", "type": "exact_match"}},
	)
	runID := h.startRun(evalID, "")

	body, _ := json.Marshal(postResultPayload{DatasetItemID: itemB, Actual: "from-b"})
	resp, err := h.do("POST",
		fmt.Sprintf("/api/v1/eval-runs/%s/results", runID),
		body,
		map[string]string{"Authorization": "Bearer " + h.apiKey},
	)
	if err != nil {
		t.Fatalf("cross-dataset post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("cross-dataset item: want 404, got %d (%s)", resp.StatusCode, raw)
	}
}

// ============================================
// HARNESS
// ============================================

type evalHarness struct {
	t       *testing.T
	server  *httptest.Server
	apiKey  string
	jwt     string
	project string
}

func newHarness(t *testing.T) *evalHarness {
	t.Helper()

	// In-memory SQLite (file under TempDir auto-cleaned by t.Cleanup).
	tmpDB := t.TempDir() + "/e2e.db"
	store, err := sqlite.New(tmpDB)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	jwtService := auth.NewJWTService("test-secret", 24*time.Hour)
	oauthService := auth.NewOAuthService("", "", "")
	pricing := service.NewPricingCalculator()

	ingestSvc := ingest.NewService(store, pricing)
	traceSvc := trace.NewService(store, pricing)
	analyticsSvc := analytics.NewService(store)
	projectSvc := project.NewService(store)
	authSvc := appauth.NewService(store, jwtService, oauthService)
	datasetSvc := dataset.NewService(store, store)
	evalSvc := eval.NewService(store, store)
	promptSvc := prompt.NewService(store)

	router := apphttp.NewRouter(apphttp.RouterConfig{
		PrimaryStore:   store,
		AnalyticsStore: store,
		IngestSvc:      ingestSvc,
		TraceSvc:       traceSvc,
		AnalyticsSvc:   analyticsSvc,
		ProjectSvc:     projectSvc,
		AuthSvc:        authSvc,
		DatasetSvc:     datasetSvc,
		EvalSvc:        evalSvc,
		PromptSvc:      promptSvc,
		JWTService:     jwtService,
		FrontendURL:    "http://localhost:3000",
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	// Seed: user + project + JWT + API key. We bypass the HTTP signup flow and
	// drop straight into the DB — the auth flow is exercised by other tests.
	userEmail := "e2e@test.local"
	user := &entity.User{Email: userEmail, Name: "E2E"}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	apiKey := "le_e2e_" + uuid.New().String()
	apiKeyHash := sha256hex(apiKey)
	proj := &entity.Project{
		Name: "E2E", APIKey: apiKey, APIKeyHash: apiKeyHash, OwnerEmail: userEmail,
	}
	if err := store.CreateProject(ctx, proj); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	token, err := jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		t.Fatalf("seed token: %v", err)
	}

	return &evalHarness{
		t: t, server: srv,
		apiKey: apiKey, jwt: token, project: proj.ID,
	}
}

// sha256hex matches whatever the auth layer uses for API-key hashing in the
// test store. The middleware itself looks up by hash, so we just need the
// store to find a row whose api_key_hash matches.
func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ----- request helpers ---------------------------------------------------

func (h *evalHarness) do(method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, h.server.URL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return http.DefaultClient.Do(req)
}

func (h *evalHarness) dashboardJSON(method, path string, body any, into any) {
	h.t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := h.do(method, path, raw, map[string]string{"Authorization": "Bearer " + h.jwt})
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		h.t.Fatalf("%s %s: status %d body %s", method, path, resp.StatusCode, respBody)
	}
	if into != nil {
		if err := json.NewDecoder(resp.Body).Decode(into); err != nil && err != io.EOF {
			h.t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
}

func (h *evalHarness) apiKeyJSON(method, path string, body any, into any) {
	h.t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := h.do(method, path, raw, map[string]string{"Authorization": "Bearer " + h.apiKey})
	if err != nil {
		h.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		h.t.Fatalf("%s %s: status %d body %s", method, path, resp.StatusCode, respBody)
	}
	if into != nil {
		if err := json.NewDecoder(resp.Body).Decode(into); err != nil && err != io.EOF {
			h.t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
}

// ----- domain helpers ----------------------------------------------------

type idCarrier struct {
	ID string `json:"id"`
}

func (h *evalHarness) createPrompt(name, desc string) string {
	var out idCarrier
	h.dashboardJSON("POST",
		fmt.Sprintf("/api/v1/dashboard/projects/%s/prompts", h.project),
		map[string]any{"name": name, "description": desc},
		&out,
	)
	return out.ID
}

func (h *evalHarness) createPromptVersion(promptID, label, content, changelog string) string {
	var out idCarrier
	h.dashboardJSON("POST",
		fmt.Sprintf("/api/v1/dashboard/projects/%s/prompts/%s/versions", h.project, promptID),
		map[string]any{"version": label, "content": content, "changelog": changelog},
		&out,
	)
	return out.ID
}

func (h *evalHarness) createDataset(name string) string {
	var out idCarrier
	h.dashboardJSON("POST",
		fmt.Sprintf("/api/v1/dashboard/projects/%s/datasets", h.project),
		map[string]any{"name": name},
		&out,
	)
	return out.ID
}

func (h *evalHarness) createDatasetItem(datasetID, input, expected string) string {
	var out idCarrier
	h.dashboardJSON("POST",
		fmt.Sprintf("/api/v1/dashboard/projects/%s/datasets/%s/items", h.project, datasetID),
		map[string]any{"input": input, "expected": expected},
		&out,
	)
	return out.ID
}

func (h *evalHarness) createEval(datasetID, name string, scorers []map[string]any) string {
	var out idCarrier
	h.dashboardJSON("POST",
		fmt.Sprintf("/api/v1/dashboard/projects/%s/evals", h.project),
		map[string]any{"datasetId": datasetID, "name": name, "scorers": scorers},
		&out,
	)
	return out.ID
}

func (h *evalHarness) startRun(evalID, promptVersionID string) string {
	body := map[string]any{"evalId": evalID}
	if promptVersionID != "" {
		body["promptVersionId"] = promptVersionID
	}
	var out idCarrier
	h.apiKeyJSON("POST", "/api/v1/eval-runs", body, &out)
	return out.ID
}

// scorerResult / postResultPayload mirror the wire shapes — kept local to
// the test to insulate from refactors in the real DTOs.
type scorerResult struct {
	ScorerID string  `json:"scorerId"`
	Passed   bool    `json:"passed"`
	Score    float64 `json:"score"`
	Details  string  `json:"details,omitempty"`
	Error    string  `json:"error,omitempty"`
}

type postResultPayload struct {
	DatasetItemID string         `json:"datasetItemId"`
	Actual        any            `json:"actual,omitempty"`
	DurationMs    *int           `json:"durationMs,omitempty"`
	CostUSD       *float64       `json:"costUsd,omitempty"`
	ClientScores  []scorerResult `json:"clientScores,omitempty"`
	Error         *string        `json:"error,omitempty"`
}

type evalRunResultView struct {
	ID            string         `json:"id"`
	DatasetItemID string         `json:"datasetItemId"`
	Scores        []scorerResult `json:"scores"`
	Passed        bool           `json:"passed"`
}

func (h *evalHarness) postResult(runID string, body postResultPayload) evalRunResultView {
	var out evalRunResultView
	h.apiKeyJSON("POST",
		fmt.Sprintf("/api/v1/eval-runs/%s/results", runID),
		body,
		&out,
	)
	return out
}

type evalRunView struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	TotalItems      int     `json:"totalItems"`
	PassedItems     int     `json:"passedItems"`
	FailedItems     int     `json:"failedItems"`
	ErroredItems    int     `json:"erroredItems"`
	PromptVersionID *string `json:"promptVersionId"`
}

func (h *evalHarness) finalizeRun(runID, status string) evalRunView {
	var out evalRunView
	h.apiKeyJSON("POST",
		fmt.Sprintf("/api/v1/eval-runs/%s/finalize", runID),
		map[string]any{"status": status},
		&out,
	)
	return out
}

type evalRunListResponse struct {
	Data []evalRunView `json:"data"`
}

func (h *evalHarness) listRunsByPromptVersion(versionID string) []evalRunView {
	var out evalRunListResponse
	h.apiKeyJSON("GET",
		fmt.Sprintf("/api/v1/eval-runs?promptVersionId=%s", versionID),
		nil,
		&out,
	)
	return out.Data
}

// ----- assertion helpers -------------------------------------------------

func containsScorerError(r evalRunResultView, scorerID string) bool {
	for _, s := range r.Scores {
		if s.ScorerID == scorerID && s.Error != "" {
			return true
		}
	}
	return false
}

func findScore(r evalRunResultView, scorerID string) *scorerResult {
	for i := range r.Scores {
		if r.Scores[i].ScorerID == scorerID {
			return &r.Scores[i]
		}
	}
	return nil
}
