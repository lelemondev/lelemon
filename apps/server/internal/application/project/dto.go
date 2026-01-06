package project

import (
	"time"

	"github.com/lelemon/server/internal/domain/entity"
)

// CreateProjectRequest is the request to create a project
type CreateProjectRequest struct {
	Name string `json:"name"`
}

// UpdateProjectRequest is the request to update a project
type UpdateProjectRequest struct {
	Name     *string                  `json:"name,omitempty"`
	Settings *entity.ProjectSettings `json:"settings,omitempty"`
}

// ProjectResponse is the response for project endpoints
type ProjectResponse struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Settings  entity.ProjectSettings `json:"settings"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

// RotateAPIKeyResponse is the response for API key rotation
type RotateAPIKeyResponse struct {
	APIKey string `json:"apiKey"`
}
