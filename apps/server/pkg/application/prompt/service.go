// Package prompt is the application-layer service for prompt definitions and
// their immutable versions — Phase 3A of the evals & prompt management spec.
//
// Architecture notes:
//
//   - Versions are append-only: there is no UpdateVersion, by design. To
//     change content, create a new version. Past traces and eval runs that
//     reference older versions stay valid.
//   - UNIQUE(prompt_id, version) is enforced at the DB layer; we surface the
//     resulting violation as entity.ErrConflict so the handler can map to 409.
//   - CreatedBy is filled from the auth context (the dashboard user's email
//     when JWT-authenticated; nil for API-key callers — there's no human).
package prompt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lelemon/server/pkg/domain/entity"
)

// PromptRepo is the persistence surface this service consumes (ISP).
type PromptRepo interface {
	CreatePrompt(ctx context.Context, p *entity.Prompt) error
	GetPrompt(ctx context.Context, projectID, promptID string) (*entity.Prompt, error)
	ListPrompts(ctx context.Context, projectID string, filter entity.PromptFilter) (*entity.Page[entity.Prompt], error)
	UpdatePrompt(ctx context.Context, projectID, promptID string, updates entity.PromptUpdate) error
	DeletePrompt(ctx context.Context, projectID, promptID string) error

	CreatePromptVersion(ctx context.Context, v *entity.PromptVersion) error
	GetPromptVersion(ctx context.Context, projectID, versionID string) (*entity.PromptVersion, error)
	ListPromptVersions(ctx context.Context, projectID, promptID string, filter entity.PromptVersionFilter) (*entity.Page[entity.PromptVersion], error)
}

// Validation limits.
const (
	maxPromptName        = 200
	maxPromptDescription = 2000
	maxVersionLabel      = 100
	maxVersionContent    = 1 << 20 // 1 MiB — generous; the SDK doesn't ship 1MB prompts.
	maxVersionChangelog  = 4000
)

// Service is the prompt use-case orchestrator. Construct via NewService.
type Service struct {
	repo PromptRepo
}

// NewService wires a new prompt service.
func NewService(repo PromptRepo) *Service {
	return &Service{repo: repo}
}

// ============================================
// PROMPTS
// ============================================

// Create validates and persists a new prompt.
func (s *Service) Create(ctx context.Context, projectID string, req CreatePromptRequest) (*PromptView, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", entity.ErrBadRequest)
	}
	if len(name) > maxPromptName {
		return nil, fmt.Errorf("%w: name exceeds %d chars", entity.ErrBadRequest, maxPromptName)
	}
	if req.Description != nil && len(*req.Description) > maxPromptDescription {
		return nil, fmt.Errorf("%w: description exceeds %d chars", entity.ErrBadRequest, maxPromptDescription)
	}

	p := &entity.Prompt{
		ProjectID:   projectID,
		Name:        name,
		Description: req.Description,
	}
	if err := s.repo.CreatePrompt(ctx, p); err != nil {
		return nil, fmt.Errorf("create prompt: %w", err)
	}
	v := toPromptView(p)
	return &v, nil
}

// Get returns one prompt scoped to the project.
func (s *Service) Get(ctx context.Context, projectID, promptID string) (*PromptView, error) {
	p, err := s.repo.GetPrompt(ctx, projectID, promptID)
	if err != nil {
		return nil, err
	}
	v := toPromptView(p)
	return &v, nil
}

// List returns a paginated list of prompts in the project.
func (s *Service) List(ctx context.Context, projectID string, filter entity.PromptFilter) (*PromptListResponse, error) {
	page, err := s.repo.ListPrompts(ctx, projectID, filter)
	if err != nil {
		return nil, fmt.Errorf("list prompts: %w", err)
	}
	resp := toPromptListResponse(page)
	return &resp, nil
}

// Update applies a partial patch. Empty updates are a no-op.
func (s *Service) Update(ctx context.Context, projectID, promptID string, req UpdatePromptRequest) error {
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return fmt.Errorf("%w: name cannot be empty", entity.ErrBadRequest)
		}
		if len(trimmed) > maxPromptName {
			return fmt.Errorf("%w: name exceeds %d chars", entity.ErrBadRequest, maxPromptName)
		}
		req.Name = &trimmed
	}
	if req.Description != nil && len(*req.Description) > maxPromptDescription {
		return fmt.Errorf("%w: description exceeds %d chars", entity.ErrBadRequest, maxPromptDescription)
	}
	return s.repo.UpdatePrompt(ctx, projectID, promptID, entity.PromptUpdate{
		Name:        req.Name,
		Description: req.Description,
	})
}

// Delete removes the prompt; the DB cascades to its versions.
func (s *Service) Delete(ctx context.Context, projectID, promptID string) error {
	return s.repo.DeletePrompt(ctx, projectID, promptID)
}

// ============================================
// PROMPT VERSIONS
// ============================================

// CreateVersion appends a new immutable version to a prompt.
//
// `createdBy` is the email of the dashboard user creating the version, or nil
// when called via API key (no human in the loop). The handlers are responsible
// for filling it from the right auth context.
func (s *Service) CreateVersion(ctx context.Context, projectID, promptID string, req CreatePromptVersionRequest, createdBy *string) (*PromptVersionView, error) {
	label := strings.TrimSpace(req.Version)
	if label == "" {
		return nil, fmt.Errorf("%w: version label is required", entity.ErrBadRequest)
	}
	if len(label) > maxVersionLabel {
		return nil, fmt.Errorf("%w: version label exceeds %d chars", entity.ErrBadRequest, maxVersionLabel)
	}
	if req.Content == "" {
		return nil, fmt.Errorf("%w: content is required", entity.ErrBadRequest)
	}
	if len(req.Content) > maxVersionContent {
		return nil, fmt.Errorf("%w: content exceeds %d bytes", entity.ErrBadRequest, maxVersionContent)
	}
	if req.Changelog != nil && len(*req.Changelog) > maxVersionChangelog {
		return nil, fmt.Errorf("%w: changelog exceeds %d chars", entity.ErrBadRequest, maxVersionChangelog)
	}

	// Verify the prompt exists under this project (404 propagates if not).
	if _, err := s.repo.GetPrompt(ctx, projectID, promptID); err != nil {
		return nil, err
	}

	v := &entity.PromptVersion{
		PromptID:  promptID,
		ProjectID: projectID,
		Version:   label,
		Content:   req.Content,
		Changelog: req.Changelog,
		CreatedBy: createdBy,
	}
	if err := s.repo.CreatePromptVersion(ctx, v); err != nil {
		// ErrConflict on (prompt_id, version) propagates unchanged — handler
		// maps to 409.
		if errors.Is(err, entity.ErrConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("create prompt version: %w", err)
	}
	view := toPromptVersionView(v)
	return &view, nil
}

// GetVersion returns one version, scoped to the project.
//
// Anti-leak: the URL carries promptID and versionID. If the version exists in
// this project but under a different prompt, we return ErrNotFound — don't
// reveal that it belongs elsewhere.
func (s *Service) GetVersion(ctx context.Context, projectID, promptID, versionID string) (*PromptVersionView, error) {
	v, err := s.repo.GetPromptVersion(ctx, projectID, versionID)
	if err != nil {
		return nil, err
	}
	if v.PromptID != promptID {
		return nil, entity.ErrNotFound
	}
	view := toPromptVersionView(v)
	return &view, nil
}

// ListVersions returns versions of a prompt, paginated.
func (s *Service) ListVersions(ctx context.Context, projectID, promptID string, filter entity.PromptVersionFilter) (*PromptVersionListResponse, error) {
	// Verify the prompt is in this project before listing — surface a clean
	// 404 instead of an empty list for a non-existent prompt.
	if _, err := s.repo.GetPrompt(ctx, projectID, promptID); err != nil {
		return nil, err
	}
	page, err := s.repo.ListPromptVersions(ctx, projectID, promptID, filter)
	if err != nil {
		return nil, fmt.Errorf("list prompt versions: %w", err)
	}
	resp := toPromptVersionListResponse(page)
	return &resp, nil
}

// IsUnsupported mirrors the dataset/eval services — surfaces
// ClickHouse-as-primary errors so handlers can map to a clear 501.
func IsUnsupported(err error) bool {
	return errors.Is(err, entity.ErrUnsupported)
}
