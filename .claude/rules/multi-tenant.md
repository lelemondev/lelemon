---
paths: "apps/web/src/app/api/**/*.ts"
---

# Multi-Tenant Isolation Rules

## CRITICAL: Always Filter by projectId

Every database query in API routes MUST filter by the authenticated project:

```typescript
// CORRECT
const traces = await db.query.traces.findMany({
  where: eq(traces.projectId, auth.projectId),
});

// WRONG - Exposes all tenants' data!
const traces = await db.query.traces.findMany();
```

## Authentication First

Every API route handler must authenticate before any database operation:

```typescript
export async function GET(request: NextRequest) {
  // 1. ALWAYS authenticate first
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  // 2. Then use auth.projectId in queries
  const data = await db.query.table.findMany({
    where: eq(table.projectId, auth.projectId),
  });
}
```

## Insert Operations

When creating records, always include projectId:

```typescript
await db.insert(traces).values({
  projectId: auth.projectId,  // REQUIRED
  ...data,
});
```

## Never Trust Client Input for projectId

```typescript
// WRONG - Client could send any projectId
const { projectId, ...data } = await request.json();

// CORRECT - Use authenticated projectId
const data = await request.json();
await db.insert(table).values({
  projectId: auth.projectId,
  ...data,
});
```
