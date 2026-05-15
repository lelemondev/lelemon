package prompt

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ----- in-memory fake repo -----------------------------------------------

type fakeRepo struct {
	prompts  map[string]*entity.Prompt
	versions map[string]*entity.PromptVersion
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		prompts:  map[string]*entity.Prompt{},
		versions: map[string]*entity.PromptVersion{},
	}
}

func (f *fakeRepo) CreatePrompt(_ context.Context, p *entity.Prompt) error {
	if p.ID == "" {
		p.ID = "pr-" + p.Name
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = p.CreatedAt
	f.prompts[p.ID] = p
	return nil
}

func (f *fakeRepo) GetPrompt(_ context.Context, projectID, promptID string) (*entity.Prompt, error) {
	p, ok := f.prompts[promptID]
	if !ok || p.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) ListPrompts(_ context.Context, projectID string, _ entity.PromptFilter) (*entity.Page[entity.Prompt], error) {
	out := []entity.Prompt{}
	for _, p := range f.prompts {
		if p.ProjectID == projectID {
			out = append(out, *p)
		}
	}
	return &entity.Page[entity.Prompt]{Data: out, Total: len(out)}, nil
}

func (f *fakeRepo) UpdatePrompt(_ context.Context, projectID, promptID string, u entity.PromptUpdate) error {
	p, ok := f.prompts[promptID]
	if !ok || p.ProjectID != projectID {
		return entity.ErrNotFound
	}
	if u.Name != nil {
		p.Name = *u.Name
	}
	if u.Description != nil {
		p.Description = u.Description
	}
	return nil
}

func (f *fakeRepo) DeletePrompt(_ context.Context, projectID, promptID string) error {
	p, ok := f.prompts[promptID]
	if !ok || p.ProjectID != projectID {
		return entity.ErrNotFound
	}
	delete(f.prompts, promptID)
	return nil
}

func (f *fakeRepo) CreatePromptVersion(_ context.Context, v *entity.PromptVersion) error {
	// Enforce UNIQUE(prompt_id, version) in the fake, matching the DB.
	for _, existing := range f.versions {
		if existing.PromptID == v.PromptID && existing.Version == v.Version {
			return entity.ErrConflict
		}
	}
	if v.ID == "" {
		v.ID = "ver-" + v.Version
	}
	v.CreatedAt = time.Now()
	f.versions[v.ID] = v
	return nil
}

func (f *fakeRepo) GetPromptVersion(_ context.Context, projectID, versionID string) (*entity.PromptVersion, error) {
	v, ok := f.versions[versionID]
	if !ok || v.ProjectID != projectID {
		return nil, entity.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (f *fakeRepo) ListPromptVersions(_ context.Context, projectID, promptID string, _ entity.PromptVersionFilter) (*entity.Page[entity.PromptVersion], error) {
	out := []entity.PromptVersion{}
	for _, v := range f.versions {
		if v.ProjectID == projectID && v.PromptID == promptID {
			out = append(out, *v)
		}
	}
	return &entity.Page[entity.PromptVersion]{Data: out, Total: len(out)}, nil
}

func newServiceForTest() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo), repo
}

// ----- prompt validation -------------------------------------------------

func TestCreate_Validation(t *testing.T) {
	svc, _ := newServiceForTest()
	ctx := context.Background()

	cases := []struct {
		name string
		req  CreatePromptRequest
		want error
	}{
		{"empty name", CreatePromptRequest{}, entity.ErrBadRequest},
		{"whitespace name", CreatePromptRequest{Name: "  "}, entity.ErrBadRequest},
		{"too long name", CreatePromptRequest{Name: strings.Repeat("a", maxPromptName+1)}, entity.ErrBadRequest},
		{"too long description", CreatePromptRequest{
			Name:        "ok",
			Description: ptr(strings.Repeat("d", maxPromptDescription+1)),
		}, entity.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, "proj", tc.req)
			if !errors.Is(err, tc.want) {
				t.Errorf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestCreate_TrimsName(t *testing.T) {
	svc, _ := newServiceForTest()
	view, err := svc.Create(context.Background(), "proj", CreatePromptRequest{Name: "  agent-system  "})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.Name != "agent-system" {
		t.Errorf("name not trimmed: %q", view.Name)
	}
}

// ----- version lifecycle -------------------------------------------------

func TestCreateVersion_HappyPath(t *testing.T) {
	svc, _ := newServiceForTest()
	ctx := context.Background()

	pr, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "agent"})

	by := "alice@test"
	view, err := svc.CreateVersion(ctx, "proj", pr.ID, CreatePromptVersionRequest{
		Version: "v1", Content: "You are an agent.", Changelog: ptr("first"),
	}, &by)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if view.Version != "v1" || view.Content != "You are an agent." {
		t.Errorf("roundtrip lost data: %+v", view)
	}
	if view.CreatedBy == nil || *view.CreatedBy != by {
		t.Errorf("created_by not threaded: %v", view.CreatedBy)
	}
}

func TestCreateVersion_APIKeyHasNoCreatedBy(t *testing.T) {
	svc, _ := newServiceForTest()
	ctx := context.Background()

	pr, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "agent"})
	view, err := svc.CreateVersion(ctx, "proj", pr.ID, CreatePromptVersionRequest{
		Version: "v1", Content: "x",
	}, nil) // API-key call → no human
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if view.CreatedBy != nil {
		t.Errorf("API-key calls must produce nil createdBy, got %v", *view.CreatedBy)
	}
}

func TestCreateVersion_Validation(t *testing.T) {
	svc, _ := newServiceForTest()
	ctx := context.Background()
	pr, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "agent"})

	cases := []struct {
		name string
		req  CreatePromptVersionRequest
		want error
	}{
		{"empty version label", CreatePromptVersionRequest{Content: "x"}, entity.ErrBadRequest},
		{"whitespace version", CreatePromptVersionRequest{Version: "  ", Content: "x"}, entity.ErrBadRequest},
		{"too long version label", CreatePromptVersionRequest{Version: strings.Repeat("v", maxVersionLabel+1), Content: "x"}, entity.ErrBadRequest},
		{"empty content", CreatePromptVersionRequest{Version: "v1"}, entity.ErrBadRequest},
		{"too long changelog", CreatePromptVersionRequest{
			Version: "v1", Content: "x",
			Changelog: ptr(strings.Repeat("c", maxVersionChangelog+1)),
		}, entity.ErrBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateVersion(ctx, "proj", pr.ID, tc.req, nil)
			if !errors.Is(err, tc.want) {
				t.Errorf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestCreateVersion_DuplicateLabelConflict(t *testing.T) {
	svc, _ := newServiceForTest()
	ctx := context.Background()
	pr, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "agent"})

	_, err := svc.CreateVersion(ctx, "proj", pr.ID, CreatePromptVersionRequest{Version: "v1", Content: "a"}, nil)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Duplicate label — must surface ErrConflict, not a 500.
	_, err = svc.CreateVersion(ctx, "proj", pr.ID, CreatePromptVersionRequest{Version: "v1", Content: "b"}, nil)
	if !errors.Is(err, entity.ErrConflict) {
		t.Errorf("duplicate (prompt, version): want ErrConflict, got %v", err)
	}
}

func TestCreateVersion_CrossTenantPrompt(t *testing.T) {
	svc, _ := newServiceForTest()
	ctx := context.Background()
	pr, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "agent"})

	// Project B tries to add a version to project A's prompt.
	_, err := svc.CreateVersion(ctx, "other", pr.ID, CreatePromptVersionRequest{Version: "v1", Content: "x"}, nil)
	if !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("cross-tenant version create: want ErrNotFound, got %v", err)
	}
}

func TestGetVersion_AntiLeakAcrossPrompts(t *testing.T) {
	// A version belongs to prompt A; fetching it via prompt B's URL must miss,
	// even though both prompts are in the same project. Mirrors the dataset
	// anti-leak rule (see application/dataset/service_test.go).
	svc, _ := newServiceForTest()
	ctx := context.Background()

	prA, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "A"})
	prB, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "B"})

	verA, err := svc.CreateVersion(ctx, "proj", prA.ID, CreatePromptVersionRequest{Version: "v1", Content: "x"}, nil)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}

	// Reading verA through prompt B's URL must miss.
	if _, err := svc.GetVersion(ctx, "proj", prB.ID, verA.ID); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("anti-leak Get: want ErrNotFound, got %v", err)
	}
	// Sanity: through prompt A's URL it works.
	if _, err := svc.GetVersion(ctx, "proj", prA.ID, verA.ID); err != nil {
		t.Errorf("same-prompt Get should work: %v", err)
	}
}

func TestListVersions_ChecksPromptOwnership(t *testing.T) {
	// Listing versions for a prompt in another tenant must return ErrNotFound,
	// not an empty list — that would leak the existence question.
	svc, _ := newServiceForTest()
	ctx := context.Background()
	pr, _ := svc.Create(ctx, "proj", CreatePromptRequest{Name: "agent"})

	if _, err := svc.ListVersions(ctx, "other", pr.ID, entity.PromptVersionFilter{}); !errors.Is(err, entity.ErrNotFound) {
		t.Errorf("cross-tenant list: want ErrNotFound, got %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
