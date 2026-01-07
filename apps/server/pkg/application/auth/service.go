package auth

import (
	"context"
	"errors"

	"github.com/lelemon/server/pkg/domain/entity"
	"github.com/lelemon/server/pkg/domain/repository"
	"github.com/lelemon/server/pkg/infrastructure/auth"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already registered")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
)

// Service handles authentication operations
type Service struct {
	store  repository.Store
	jwt    *auth.JWTService
	oauth  *auth.OAuthService
}

// NewService creates a new auth service
func NewService(store repository.Store, jwt *auth.JWTService, oauth *auth.OAuthService) *Service {
	return &Service{
		store: store,
		jwt:   jwt,
		oauth: oauth,
	}
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Validate password
	if len(req.Password) < 8 {
		return nil, ErrWeakPassword
	}

	// Check if email exists
	_, err := s.store.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailExists
	}
	if err != entity.ErrNotFound {
		return nil, err
	}

	// Hash password
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &entity.User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: &hash,
	}

	if err := s.store.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Generate token
	token, err := s.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  userToResponse(user),
	}, nil
}

// Login authenticates a user with email and password
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == entity.ErrNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check password
	if user.PasswordHash == nil || !auth.VerifyPassword(req.Password, *user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Generate token
	token, err := s.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  userToResponse(user),
	}, nil
}

// GetGoogleAuthURL returns the Google OAuth URL
func (s *Service) GetGoogleAuthURL(state string) string {
	return s.oauth.GetAuthURL(state)
}

// HandleGoogleCallback processes the Google OAuth callback
func (s *Service) HandleGoogleCallback(ctx context.Context, code string) (*AuthResponse, error) {
	// Exchange code for user info
	googleUser, err := s.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Find or create user
	user, err := s.store.GetUserByEmail(ctx, googleUser.Email)
	if err == entity.ErrNotFound {
		// Create new user
		user = &entity.User{
			Email:    googleUser.Email,
			Name:     googleUser.Name,
			GoogleID: &googleUser.ID,
		}
		if err := s.store.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		// Update Google ID if not set
		if user.GoogleID == nil {
			googleID := googleUser.ID
			s.store.UpdateUser(ctx, user.ID, entity.UserUpdate{})
			user.GoogleID = &googleID
		}
	}

	// Generate token
	token, err := s.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  userToResponse(user),
	}, nil
}

// GetCurrentUser returns the current user from token
func (s *Service) GetCurrentUser(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userToResponse(user), nil
}

// IsOAuthConfigured returns true if OAuth is configured
func (s *Service) IsOAuthConfigured() bool {
	return s.oauth.IsConfigured()
}

func userToResponse(user *entity.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	}
}
