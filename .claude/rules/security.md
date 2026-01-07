---
paths: "**/*"
alwaysApply: true
---

# Security Rules

## Never Commit Secrets

These patterns should NEVER be in version control:
- `.env` files (use `.env.example` as template)
- API keys, tokens, passwords
- Private keys, certificates
- OAuth client secrets
- JWT secrets

## Environment Variables

```go
// Go - DO: Use environment variables
apiKey := os.Getenv("API_KEY")

// Go - DON'T: Hardcode secrets
apiKey := "sk-abc123..."
```

```typescript
// TypeScript - DO: Use environment variables
const apiKey = process.env.API_KEY;

// TypeScript - DON'T: Hardcode secrets
const apiKey = 'sk-abc123...';
```

## Input Validation

### Go Backend
```go
// Validate required fields
if req.Name == "" || len(req.Name) > 100 {
    http.Error(w, `{"error":"invalid name"}`, http.StatusBadRequest)
    return
}

// Validate email format
if !isValidEmail(req.Email) {
    http.Error(w, `{"error":"invalid email"}`, http.StatusBadRequest)
    return
}
```

### TypeScript Frontend
```typescript
import { z } from 'zod';

const schema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
});

const result = schema.safeParse(body);
if (!result.success) {
  throw new Error(result.error.message);
}
```

## SQL Injection Prevention

Always use parameterized queries:

```go
// Go - SAFE: Parameterized
db.QueryRowContext(ctx, "SELECT * FROM traces WHERE id = ?", userInput)

// Go - DANGER: Never interpolate
db.QueryRowContext(ctx, "SELECT * FROM traces WHERE id = '" + userInput + "'")
```

## Authentication

- All protected endpoints must verify JWT or API key
- Never trust client-provided user/project IDs
- Use the authenticated context for all operations

```go
// Always check auth first
userID := middleware.GetUserID(r.Context())
if userID == "" {
    http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
    return
}
```

## Sensitive Data in Logs

```go
// DON'T: Log sensitive data
log.Info("user login", "password", password)
log.Info("webhook received", "body", string(body))

// DO: Log safely
log.Info("user authenticated", "userID", userID)
log.Info("trace created", "traceID", traceID)
```

## JWT Security

```go
// Always verify token signature
claims, err := jwtService.ValidateToken(tokenString)
if err != nil {
    http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
    return
}

// Use secure JWT secrets (min 32 chars)
// Never expose JWT secret to frontend
```

## API Key Storage

```go
// Store hashed API keys, not plaintext
apiKeyHash := hashAPIKey(apiKey)
project.APIKeyHash = apiKeyHash

// Compare with constant-time comparison
if !subtle.ConstantTimeCompare([]byte(hash), []byte(computed)) {
    return ErrInvalidAPIKey
}
```
