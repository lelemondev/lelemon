---
paths: "**/*"
alwaysApply: true
---

# Anti-Patterns to Avoid

## Go Backend

### Request Body Size
Always limit request body size to prevent DoS:
```go
// DO: Limit body size
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

// DON'T: Read unlimited body
json.NewDecoder(r.Body).Decode(&req)
```

### SQL Granularity/Enum in fmt.Sprintf
Never pass user input directly into fmt.Sprintf for SQL. Validate against a whitelist first:
```go
// DO: Validate before interpolating
if !entity.ValidGranularity(granularity) {
    http.Error(w, `{"error":"invalid granularity"}`, 400)
    return
}
query := fmt.Sprintf(`SELECT date_trunc('%s', created_at)...`, granularity)

// DON'T: Pass user input directly
query := fmt.Sprintf(`SELECT date_trunc('%s', created_at)...`, r.URL.Query().Get("granularity"))
```

### rows.Err() After Iteration
Always check for iteration errors after scanning rows:
```go
for rows.Next() {
    // scan...
}
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("iteration error: %w", err)
}
```

### Project Ownership Verification
Every dashboard handler MUST verify the user owns the project before returning data:
```go
// DO: Use verifyProjectOwnership helper
projectID, ok := h.verifyProjectOwnership(w, r)
if !ok {
    return
}

// DON'T: Skip ownership check
projectID := chi.URLParam(r, "id")
result, _ := h.service.Get(ctx, projectID) // any user can access!
```

### Error Response Encoding
Check json.Encode errors — headers are already written at that point:
```go
w.Header().Set("Content-Type", "application/json")
if err := json.NewEncoder(w).Encode(result); err != nil {
    slog.Error("failed to encode response", "error", err)
}
```

## TypeScript Frontend

### Never Store JWT in localStorage
localStorage is accessible to any XSS attack. Use httpOnly cookies instead:
```typescript
// DON'T
localStorage.setItem('token', jwt);

// DO: Set httpOnly cookie from the backend
// The frontend should never see or store the raw JWT
```

### Never Put Secrets in URLs
API keys and tokens must not appear in query parameters (browser history, logs, Referer):
```typescript
// DON'T
router.push(`/config?key=${apiKey}`);

// DO: Pass via state or context
router.push('/config', { state: { key: apiKey } });
```

### Promise.allSettled for Multiple Fetches
When loading multiple independent data sources, one failure should not crash the page:
```typescript
// DON'T
const [a, b, c] = await Promise.all([fetchA(), fetchB(), fetchC()]);

// DO
const results = await Promise.allSettled([fetchA(), fetchB(), fetchC()]);
const a = results[0].status === 'fulfilled' ? results[0].value : fallback;
```

### Avoid `as unknown as T`
Don't bypass TypeScript with double casts. Use runtime validation or specific normalizer functions:
```typescript
// DON'T
const data = response as unknown as ModelStats[];

// DO: Normalize explicitly
const data = response.map(normalizeModelStats);
```

### React Keys in Lists
Every element in a map must have a unique key. Fragments (`<>`) can't have keys:
```tsx
// DON'T
{items.map(item => <>{/* ... */}</>)}

// DO: Use a keyed wrapper
{items.map(item => <div key={item.id}>{/* ... */}</div>)}
```
