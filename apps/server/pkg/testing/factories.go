package testing

import (
	"time"

	"github.com/google/uuid"
	"github.com/lelemon/server/pkg/domain/entity"
)

// Helper to create pointer to value
func ptr[T any](v T) *T {
	return &v
}

// UserFactory creates test users.
type UserFactory struct {
	counter int
}

// NewUserFactory creates a new user factory.
func NewUserFactory() *UserFactory {
	return &UserFactory{}
}

// Create creates a new test user with unique data.
func (f *UserFactory) Create() *entity.User {
	f.counter++
	now := time.Now()
	return &entity.User{
		ID:           uuid.New().String(),
		Email:        f.uniqueEmail("user"),
		Name:         f.uniqueName("Test User"),
		PasswordHash: ptr("$2a$10$abcdefghijklmnopqrstuv"), // bcrypt hash placeholder
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// CreateWithEmail creates a user with a specific email.
func (f *UserFactory) CreateWithEmail(email string) *entity.User {
	user := f.Create()
	user.Email = email
	return user
}

func (f *UserFactory) uniqueEmail(prefix string) string {
	return uuid.New().String()[:8] + "-" + prefix + "@test.lelemon.dev"
}

func (f *UserFactory) uniqueName(prefix string) string {
	return prefix + " " + uuid.New().String()[:4]
}

// ProjectFactory creates test projects.
type ProjectFactory struct {
	counter int
}

// NewProjectFactory creates a new project factory.
func NewProjectFactory() *ProjectFactory {
	return &ProjectFactory{}
}

// Create creates a new test project.
func (f *ProjectFactory) Create(ownerEmail string) *entity.Project {
	f.counter++
	apiKey := "le_test_" + uuid.New().String()
	now := time.Now()
	return &entity.Project{
		ID:         uuid.New().String(),
		OwnerEmail: ownerEmail,
		Name:       f.uniqueName("Test Project"),
		APIKey:     apiKey,
		APIKeyHash: hashAPIKey(apiKey),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (f *ProjectFactory) uniqueName(prefix string) string {
	return prefix + " " + uuid.New().String()[:4]
}

// Simple hash for testing (not secure, just for tests)
func hashAPIKey(key string) string {
	return "hash_" + key
}

// TraceFactory creates test traces.
type TraceFactory struct {
	counter int
}

// NewTraceFactory creates a new trace factory.
func NewTraceFactory() *TraceFactory {
	return &TraceFactory{}
}

// Create creates a new test trace.
func (f *TraceFactory) Create(projectID string) *entity.Trace {
	f.counter++
	now := time.Now()
	return &entity.Trace{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Status:    entity.TraceStatusCompleted,
		Tags:      []string{"test"},
		Metadata:  map[string]any{"source": "factory"},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateWithSession creates a trace with a session ID.
func (f *TraceFactory) CreateWithSession(projectID, sessionID string) *entity.Trace {
	trace := f.Create(projectID)
	trace.SessionID = &sessionID
	return trace
}

// SpanFactory creates test spans.
type SpanFactory struct {
	counter int
}

// NewSpanFactory creates a new span factory.
func NewSpanFactory() *SpanFactory {
	return &SpanFactory{}
}

// Create creates a new test span.
func (f *SpanFactory) Create(traceID string) *entity.Span {
	f.counter++
	now := time.Now()
	return &entity.Span{
		ID:           uuid.New().String(),
		TraceID:      traceID,
		Type:         entity.SpanTypeLLM,
		Name:         f.uniqueName("Test Span"),
		Status:       entity.SpanStatusSuccess,
		Model:        ptr("gpt-4o"),
		Provider:     ptr("openai"),
		InputTokens:  ptr(50),
		OutputTokens: ptr(50),
		CostUSD:      ptr(0.001),
		DurationMs:   ptr(200),
		StartedAt:    now,
	}
}

// CreateTool creates a tool span.
func (f *SpanFactory) CreateTool(traceID string, name string) *entity.Span {
	f.counter++
	now := time.Now()
	return &entity.Span{
		ID:         uuid.New().String(),
		TraceID:    traceID,
		Type:       entity.SpanTypeTool,
		Name:       name,
		Status:     entity.SpanStatusSuccess,
		DurationMs: ptr(100),
		StartedAt:  now,
	}
}

func (f *SpanFactory) uniqueName(prefix string) string {
	return prefix + " " + uuid.New().String()[:4]
}
