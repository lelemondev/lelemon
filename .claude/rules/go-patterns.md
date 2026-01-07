---
paths: "apps/server/**/*.go"
---

# Go Patterns for Lelemon Server

## Error Handling

Always wrap errors with context:

```go
// Good
if err != nil {
    return fmt.Errorf("failed to get traces: %w", err)
}

// Bad
if err != nil {
    return err
}
```

## Context Usage

Always pass context as first parameter:

```go
func (s *Service) GetTraces(ctx context.Context, projectID string) ([]entity.Trace, error)
```

## Database Operations

Use parameterized queries (never string concatenation):

```go
// Good
s.db.QueryRowContext(ctx, "SELECT * FROM traces WHERE id = ?", id)

// Bad - SQL injection risk
s.db.QueryRowContext(ctx, "SELECT * FROM traces WHERE id = '" + id + "'")
```

## HTTP Handlers

Standard pattern:

```go
func (h *Handler) GetExample(w http.ResponseWriter, r *http.Request) {
    // 1. Extract auth from context
    userID := middleware.GetUserID(r.Context())
    if userID == "" {
        http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
        return
    }

    // 2. Parse request (if needed)
    var req RequestType
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
        return
    }

    // 3. Call service
    result, err := h.service.GetExample(r.Context(), userID)
    if err != nil {
        // Map domain errors to HTTP status
        http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
        return
    }

    // 4. Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

## Null Handling in SQL

Use sql.Null types for nullable columns:

```go
var endedAt sql.NullTime
err := row.Scan(&t.ID, &endedAt)
if endedAt.Valid {
    t.EndedAt = &endedAt.Time
}
```

## UUID Generation

Use google/uuid:

```go
import "github.com/google/uuid"

id := uuid.New().String()
```

## Clean Architecture Layers

```
pkg/
├── domain/           # No external dependencies
│   ├── entity/       # Data structures
│   ├── repository/   # Interfaces only
│   └── service/      # Domain services
├── application/      # Use cases, orchestration
├── infrastructure/   # External systems (DB, APIs)
└── interfaces/       # HTTP, CLI, etc.
```

Dependencies flow inward: interfaces → application → domain ← infrastructure

## JSON Response Convention

Go uses PascalCase in JSON (default). Frontend normalizes to camelCase:

```go
type TraceResponse struct {
    ID           string `json:"ID"`          // Will be normalized
    ProjectID    string `json:"ProjectID"`
    TotalTokens  int    `json:"TotalTokens"`
}
```
