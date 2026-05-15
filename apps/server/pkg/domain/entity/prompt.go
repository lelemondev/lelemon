package entity

import "time"

// Prompt is a named, project-scoped container for a series of PromptVersions.
//
// The Prompt itself carries no content — the *versions* do. Renaming or
// re-describing a prompt does not invalidate past versions; their `prompt_id`
// foreign key remains, and the version stays tied to the same logical prompt
// across renames.
type Prompt struct {
	ID          string
	ProjectID   string
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PromptVersion is an immutable snapshot of a prompt's content.
//
// "Version" here is a user-chosen label (e.g. "v3", "2026-05-15-experiment").
// Uniqueness is enforced per-prompt at the DB layer — re-creating the same
// label on the same prompt returns entity.ErrConflict.
//
// CreatedBy is the email of the dashboard user who created the version, or
// nil when the version was created via API key (a CI script, not a human).
//
// Content is opaque text. Most prompts are strings — if the customer needs
// structured chat messages, they serialise their own JSON. Keeping the column
// untyped lets Phase 3B diff the text directly without a schema dance.
type PromptVersion struct {
	ID        string
	PromptID  string
	ProjectID string // denormalized for 1-hop tenant scoping
	Version   string
	Content   string
	Changelog *string
	CreatedBy *string
	CreatedAt time.Time
}

// NewPromptVersion is the payload passed into the store for creating a
// version. IDs and timestamps are filled by the store; ProjectID + CreatedBy
// come from the auth context — never from the request body.
type NewPromptVersion struct {
	PromptID  string
	ProjectID string
	Version   string
	Content   string
	Changelog *string
	CreatedBy *string
}

// PromptFilter narrows a ListPrompts query.
type PromptFilter struct {
	Name   *string // substring match
	Limit  int
	Offset int
}

// PromptVersionFilter narrows a ListPromptVersions query.
type PromptVersionFilter struct {
	Limit  int
	Offset int
}

// PromptUpdate carries optional fields to patch on an existing prompt.
// Versions are not mutated through this — they're append-only.
type PromptUpdate struct {
	Name        *string
	Description *string
}
