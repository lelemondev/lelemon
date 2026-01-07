package project

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/repository"
)

// Service handles project operations
type Service struct {
	store repository.Store
}

// NewService creates a new project service
func NewService(store repository.Store) *Service {
	return &Service{store: store}
}

// GetCurrent returns the current project (from API key auth)
func (s *Service) GetCurrent(ctx context.Context, project *entity.Project) *ProjectResponse {
	return &ProjectResponse{
		ID:        project.ID,
		Name:      project.Name,
		Settings:  project.Settings,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}

// UpdateCurrent updates the current project
func (s *Service) UpdateCurrent(ctx context.Context, projectID string, req *UpdateProjectRequest) error {
	updates := entity.ProjectUpdate{}

	if req.Name != nil {
		updates.Name = req.Name
	}
	if req.Settings != nil {
		updates.Settings = req.Settings
	}

	return s.store.UpdateProject(ctx, projectID, updates)
}

// Create creates a new project
func (s *Service) Create(ctx context.Context, ownerEmail string, req *CreateProjectRequest) (*entity.Project, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(apiKey))
	hashStr := hex.EncodeToString(hash[:])

	project := &entity.Project{
		Name:       req.Name,
		APIKey:     apiKey,
		APIKeyHash: hashStr,
		OwnerEmail: ownerEmail,
		Settings:   entity.ProjectSettings{},
	}

	if err := s.store.CreateProject(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// List returns all projects for an owner
func (s *Service) List(ctx context.Context, ownerEmail string) ([]entity.Project, error) {
	return s.store.ListProjectsByOwner(ctx, ownerEmail)
}

// Update updates a project by ID (for dashboard)
func (s *Service) Update(ctx context.Context, projectID string, ownerEmail string, req *UpdateProjectRequest) error {
	// Verify ownership
	project, err := s.store.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerEmail != ownerEmail {
		return entity.ErrNotFound
	}

	updates := entity.ProjectUpdate{}
	if req.Name != nil {
		updates.Name = req.Name
	}
	if req.Settings != nil {
		updates.Settings = req.Settings
	}

	return s.store.UpdateProject(ctx, projectID, updates)
}

// Delete deletes a project
func (s *Service) Delete(ctx context.Context, projectID string, ownerEmail string) error {
	// Verify ownership
	project, err := s.store.GetProjectByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerEmail != ownerEmail {
		return entity.ErrNotFound
	}

	return s.store.DeleteProject(ctx, projectID)
}

// RotateAPIKey generates a new API key for a project
func (s *Service) RotateAPIKey(ctx context.Context, projectID string) (*RotateAPIKeyResponse, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(apiKey))
	hashStr := hex.EncodeToString(hash[:])

	if err := s.store.RotateAPIKey(ctx, projectID, apiKey, hashStr); err != nil {
		return nil, err
	}

	return &RotateAPIKeyResponse{
		APIKey: apiKey,
	}, nil
}

// generateAPIKey creates a new random API key with le_ prefix
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "le_" + hex.EncodeToString(bytes), nil
}
