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
	RetentionDays *int    `json:"retentionDays,omitempty"`
	WebhookURL    *string `json:"webhookUrl,omitempty"`
}

type ProjectUpdate struct {
	Name     *string
	Settings *ProjectSettings
}

type NewProject struct {
	Name       string
	OwnerEmail string
}
