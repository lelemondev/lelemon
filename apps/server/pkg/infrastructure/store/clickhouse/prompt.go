package clickhouse

import (
	"context"

	"github.com/lelemon/server/pkg/domain/entity"
)

// ============================================
// PROMPT OPERATIONS — UNSUPPORTED
// ============================================
//
// Prompts and versions are small relational data with strict uniqueness
// constraints. ClickHouse's columnar engine has no good story for either, so
// the stubs return entity.ErrUnsupported just like datasets and evals.

func (s *Store) CreatePrompt(ctx context.Context, p *entity.Prompt) error {
	return entity.ErrUnsupported
}

func (s *Store) GetPrompt(ctx context.Context, projectID, promptID string) (*entity.Prompt, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) ListPrompts(ctx context.Context, projectID string, filter entity.PromptFilter) (*entity.Page[entity.Prompt], error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) UpdatePrompt(ctx context.Context, projectID, promptID string, updates entity.PromptUpdate) error {
	return entity.ErrUnsupported
}

func (s *Store) DeletePrompt(ctx context.Context, projectID, promptID string) error {
	return entity.ErrUnsupported
}

func (s *Store) CreatePromptVersion(ctx context.Context, v *entity.PromptVersion) error {
	return entity.ErrUnsupported
}

func (s *Store) GetPromptVersion(ctx context.Context, projectID, versionID string) (*entity.PromptVersion, error) {
	return nil, entity.ErrUnsupported
}

func (s *Store) ListPromptVersions(ctx context.Context, projectID, promptID string, filter entity.PromptVersionFilter) (*entity.Page[entity.PromptVersion], error) {
	return nil, entity.ErrUnsupported
}
