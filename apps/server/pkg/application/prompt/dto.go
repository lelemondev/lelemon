package prompt

// CreatePromptRequest is the body of POST .../prompts.
// Name is required; description is optional.
type CreatePromptRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// UpdatePromptRequest is the body of PATCH .../prompts/{id}.
// Nil pointers mean "leave unchanged" (matches entity.PromptUpdate semantics).
// Versions are not mutated through this — they are append-only.
type UpdatePromptRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreatePromptVersionRequest is the body of POST .../prompts/{id}/versions.
//
// Version is the user-chosen label (e.g. "v3", "2026-05-15-tweak"). It must
// be unique within the prompt — the DB enforces this and the service maps the
// resulting conflict to entity.ErrConflict.
//
// Content is opaque text. Most prompts are strings or templates; structured
// chat-message content should be JSON-serialised by the caller. Keeping the
// column untyped lets Phase 3B diff the text directly.
type CreatePromptVersionRequest struct {
	Version   string  `json:"version"`
	Content   string  `json:"content"`
	Changelog *string `json:"changelog,omitempty"`
}
