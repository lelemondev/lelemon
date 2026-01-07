---
paths: "apps/server/pkg/**/*.go"
---

# Multi-Tenant Isolation Rules

## CRITICAL: Always Filter by ProjectID

Every database query MUST filter by the authenticated project or user:

```go
// CORRECT - Scoped to project
traces, err := s.store.GetTracesByProject(ctx, projectID, filters)

// WRONG - Exposes all tenants' data!
traces, err := s.store.GetAllTraces(ctx)
```

## Authentication First

Every handler must authenticate before any database operation:

```go
func (h *Handler) GetTraces(w http.ResponseWriter, r *http.Request) {
    // 1. ALWAYS authenticate first
    projectID := middleware.GetProjectID(r.Context())
    if projectID == "" {
        http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
        return
    }

    // 2. Then use projectID in queries
    traces, err := h.service.GetTraces(r.Context(), projectID, filters)
}
```

## Insert Operations

When creating records, always include projectID:

```go
trace := &entity.Trace{
    ID:        uuid.New().String(),
    ProjectID: projectID,  // REQUIRED - from auth context
    // ...other fields
}
err := s.store.CreateTrace(ctx, trace)
```

## Never Trust Client Input for ProjectID

```go
// WRONG - Client could send any projectID
var req struct {
    ProjectID string `json:"project_id"`
    Name      string `json:"name"`
}
json.NewDecoder(r.Body).Decode(&req)

// CORRECT - Use authenticated projectID from context
projectID := middleware.GetProjectID(r.Context())
trace := &entity.Trace{
    ProjectID: projectID,  // From auth, not request
    Name:      req.Name,
}
```

## Store Layer Enforcement

All store methods should require projectID or userID:

```go
// Good - Explicit scope
type Store interface {
    GetTracesByProject(ctx context.Context, projectID string, filters Filters) ([]entity.Trace, error)
    GetProjectsByUser(ctx context.Context, userID string) ([]entity.Project, error)
}

// Bad - No scope
type Store interface {
    GetAllTraces(ctx context.Context) ([]entity.Trace, error)
}
```
