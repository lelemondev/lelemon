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

## Environment Variables

```typescript
// DO: Use environment variables
const apiKey = process.env.API_KEY;

// DON'T: Hardcode secrets
const apiKey = 'sk-abc123...';
```

## Input Validation

Always validate user input before processing:

```typescript
import { z } from 'zod';

const schema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
});

const result = schema.safeParse(body);
if (!result.success) {
  return badRequest(result.error.message);
}
```

## SQL Injection Prevention

Always use parameterized queries (Drizzle handles this):

```typescript
// SAFE: Drizzle ORM parameterizes
await db.query.traces.findMany({
  where: eq(traces.id, userInput),
});

// DANGER: Never interpolate user input
await db.execute(`SELECT * FROM traces WHERE id = '${userInput}'`);
```

## Authentication

- All API routes must call `authenticate()`
- Never trust client-provided user/project IDs
- Use the authenticated context for all operations

## Sensitive Data in Logs

```typescript
// DON'T: Log sensitive data
console.log('User password:', password);
console.log('API Key:', apiKey);

// DO: Log safely
console.log('User authenticated:', userId);
console.log('API Key rotated for project:', projectId);
```
