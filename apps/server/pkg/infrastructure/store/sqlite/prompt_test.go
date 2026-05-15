package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/lelemon/server/pkg/domain/entity"
)

func TestPrompt_CRUD(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	desc := "agent system prompt"
	p := &entity.Prompt{ProjectID: projA.ID, Name: "agent", Description: &desc}
	if err := store.CreatePrompt(ctx, p); err != nil {
		t.Fatalf("create prompt: %v", err)
	}
	if p.ID == "" {
		t.Fatal("expected store to fill ID")
	}

	got, err := store.GetPrompt(ctx, projA.ID, p.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "agent" || got.Description == nil || *got.Description != desc {
		t.Errorf("roundtrip lost data: %+v", got)
	}

	newName := "agent-v2"
	newDesc := "renamed"
	if err := store.UpdatePrompt(ctx, projA.ID, p.ID, entity.PromptUpdate{Name: &newName, Description: &newDesc}); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = store.GetPrompt(ctx, projA.ID, p.ID)
	if got.Name != "agent-v2" {
		t.Errorf("rename did not stick: %q", got.Name)
	}

	if err := store.DeletePrompt(ctx, projA.ID, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := store.GetPrompt(ctx, projA.ID, p.ID); err != entity.ErrNotFound {
		t.Errorf("after delete: want ErrNotFound, got %v", err)
	}
}

func TestPrompt_TenantIsolation(t *testing.T) {
	store, projA, projB := datasetTestStore(t)
	ctx := context.Background()

	pA := &entity.Prompt{ProjectID: projA.ID, Name: "A-prompt"}
	pB := &entity.Prompt{ProjectID: projB.ID, Name: "B-prompt"}
	_ = store.CreatePrompt(ctx, pA)
	_ = store.CreatePrompt(ctx, pB)

	// Cross-tenant reads must miss.
	if _, err := store.GetPrompt(ctx, projB.ID, pA.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant get should miss: %v", err)
	}
	if err := store.UpdatePrompt(ctx, projB.ID, pA.ID, entity.PromptUpdate{Name: ptr("hijack")}); err != entity.ErrNotFound {
		t.Errorf("cross-tenant update should miss: %v", err)
	}
	if err := store.DeletePrompt(ctx, projB.ID, pA.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant delete should miss: %v", err)
	}

	pageA, _ := store.ListPrompts(ctx, projA.ID, entity.PromptFilter{})
	pageB, _ := store.ListPrompts(ctx, projB.ID, entity.PromptFilter{})
	if pageA.Total != 1 || pageA.Data[0].Name != "A-prompt" {
		t.Errorf("project A list leaked or missed: %+v", pageA)
	}
	if pageB.Total != 1 || pageB.Data[0].Name != "B-prompt" {
		t.Errorf("project B list leaked or missed: %+v", pageB)
	}
}

func TestPromptVersion_CreateAndUniqueConstraint(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	p := &entity.Prompt{ProjectID: projA.ID, Name: "agent"}
	if err := store.CreatePrompt(ctx, p); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	by := "alice@test"
	cl := "first cut"
	v1 := &entity.PromptVersion{
		PromptID:  p.ID,
		ProjectID: projA.ID,
		Version:   "v1",
		Content:   "You are a helpful agent.",
		Changelog: &cl,
		CreatedBy: &by,
	}
	if err := store.CreatePromptVersion(ctx, v1); err != nil {
		t.Fatalf("create v1: %v", err)
	}

	// Re-creating the same (prompt_id, version) must conflict, not panic.
	dup := &entity.PromptVersion{
		PromptID: p.ID, ProjectID: projA.ID, Version: "v1", Content: "x",
	}
	if err := store.CreatePromptVersion(ctx, dup); err != entity.ErrConflict {
		t.Errorf("duplicate (prompt, version): want ErrConflict, got %v", err)
	}

	// Different version label on the same prompt: fine.
	v2 := &entity.PromptVersion{
		PromptID: p.ID, ProjectID: projA.ID, Version: "v2", Content: "improved",
	}
	if err := store.CreatePromptVersion(ctx, v2); err != nil {
		t.Errorf("v2 should succeed: %v", err)
	}

	// Same version label on a different prompt: fine.
	p2 := &entity.Prompt{ProjectID: projA.ID, Name: "tools"}
	_ = store.CreatePrompt(ctx, p2)
	v3 := &entity.PromptVersion{
		PromptID: p2.ID, ProjectID: projA.ID, Version: "v1", Content: "tool agent",
	}
	if err := store.CreatePromptVersion(ctx, v3); err != nil {
		t.Errorf("same label on different prompt should succeed: %v", err)
	}
}

func TestPromptVersion_RoundtripAndList(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	p := &entity.Prompt{ProjectID: projA.ID, Name: "p"}
	_ = store.CreatePrompt(ctx, p)

	by := "kmilo@test"
	for _, lbl := range []string{"v1", "v2", "v3"} {
		err := store.CreatePromptVersion(ctx, &entity.PromptVersion{
			PromptID: p.ID, ProjectID: projA.ID, Version: lbl,
			Content: "content of " + lbl, CreatedBy: &by,
		})
		if err != nil {
			t.Fatalf("create %s: %v", lbl, err)
		}
	}

	page, err := store.ListPromptVersions(ctx, projA.ID, p.ID, entity.PromptVersionFilter{})
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if page.Total != 3 {
		t.Errorf("want 3 versions, got %d", page.Total)
	}
	// Newest first.
	if page.Data[0].Version != "v3" {
		t.Errorf("ordering wrong: %s", page.Data[0].Version)
	}

	// Roundtrip a fetch — content + created_by survive.
	got, err := store.GetPromptVersion(ctx, projA.ID, page.Data[0].ID)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if !strings.Contains(got.Content, "v3") {
		t.Errorf("content roundtrip lost: %q", got.Content)
	}
	if got.CreatedBy == nil || *got.CreatedBy != by {
		t.Errorf("created_by lost: %v", got.CreatedBy)
	}
}

func TestPromptVersion_CascadesWithPrompt(t *testing.T) {
	store, projA, _ := datasetTestStore(t)
	ctx := context.Background()

	p := &entity.Prompt{ProjectID: projA.ID, Name: "p"}
	_ = store.CreatePrompt(ctx, p)
	v := &entity.PromptVersion{PromptID: p.ID, ProjectID: projA.ID, Version: "v1", Content: "x"}
	_ = store.CreatePromptVersion(ctx, v)

	if err := store.DeletePrompt(ctx, projA.ID, p.ID); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}
	if _, err := store.GetPromptVersion(ctx, projA.ID, v.ID); err != entity.ErrNotFound {
		t.Errorf("version should cascade-delete with prompt, got %v", err)
	}
}

func TestPromptVersion_TenantIsolation(t *testing.T) {
	store, projA, projB := datasetTestStore(t)
	ctx := context.Background()

	pA := &entity.Prompt{ProjectID: projA.ID, Name: "p"}
	_ = store.CreatePrompt(ctx, pA)
	vA := &entity.PromptVersion{PromptID: pA.ID, ProjectID: projA.ID, Version: "v1", Content: "secret"}
	_ = store.CreatePromptVersion(ctx, vA)

	if _, err := store.GetPromptVersion(ctx, projB.ID, vA.ID); err != entity.ErrNotFound {
		t.Errorf("cross-tenant get version should miss, got %v", err)
	}
	page, _ := store.ListPromptVersions(ctx, projB.ID, pA.ID, entity.PromptVersionFilter{})
	if page.Total != 0 {
		t.Errorf("cross-tenant list should be empty, got %d", page.Total)
	}
}

func TestEvalRunFilter_PromptVersionID(t *testing.T) {
	// Wires together: an eval-run carrying prompt_version_id is filterable
	// via the new filter field. This is the SQL leg of the payoff view.
	store, projA, _, d := evalTestStore(t)
	ctx := context.Background()

	e := &entity.Eval{
		ProjectID: projA.ID, DatasetID: d.ID, Name: "e",
		Scorers: []entity.Scorer{{ID: "s", Type: entity.ScorerExactMatch}},
	}
	if err := store.CreateEval(ctx, e); err != nil {
		t.Fatalf("create eval: %v", err)
	}

	pv := "pv-abc"
	other := "pv-xyz"
	r1 := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID, PromptVersionID: &pv}
	r2 := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID, PromptVersionID: &pv}
	r3 := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID, PromptVersionID: &other}
	r4 := &entity.EvalRun{ProjectID: projA.ID, EvalID: e.ID} // no prompt version
	for _, r := range []*entity.EvalRun{r1, r2, r3, r4} {
		if err := store.CreateEvalRun(ctx, r); err != nil {
			t.Fatalf("create run: %v", err)
		}
	}

	page, err := store.ListEvalRuns(ctx, projA.ID, "", entity.EvalRunFilter{PromptVersionID: &pv})
	if err != nil {
		t.Fatalf("filtered list: %v", err)
	}
	if page.Total != 2 {
		t.Errorf("prompt_version filter: want 2 runs for %s, got %d", pv, page.Total)
	}
	for _, run := range page.Data {
		if run.PromptVersionID == nil || *run.PromptVersionID != pv {
			t.Errorf("filtered list returned wrong row: %+v", run)
		}
	}
}
