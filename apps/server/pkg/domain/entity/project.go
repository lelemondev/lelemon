package entity

import "time"

type Project struct {
	ID         string
	Name       string
	APIKey     string // Public key (le_xxx...)
	APIKeyHash string // SHA-256 hash for lookup
	OwnerEmail string
	Settings   ProjectSettings
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ProjectSettings struct {
	RetentionDays *int              `json:"retentionDays,omitempty"`
	WebhookURL    *string           `json:"webhookUrl,omitempty"`
	ModelAliases  map[string]string `json:"modelAliases,omitempty"`  // e.g. {"us.anthropic.claude-sonnet-4-6": "Claude Sonnet"}
	SpanColors    map[string]string `json:"spanColors,omitempty"`    // e.g. {"sales": "#22c55e", "support": "#3b82f6"}
}

type ProjectUpdate struct {
	Name     *string
	Settings *ProjectSettings
}

type NewProject struct {
	Name       string
	OwnerEmail string
}
