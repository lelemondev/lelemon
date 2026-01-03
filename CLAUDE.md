# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Description

**Lelemon** is an LLM observability micro-SaaS that helps developers track, debug, and optimize their AI agents. It provides tracing for LLM calls, tool usage, token consumption, and cost analytics.

**Key Features:**
- Multi-tenant with API key authentication
- Real-time trace ingestion via SDK
- Cost calculation for all major LLM providers
- Developer-focused dashboard with dark mode

**Tech Stack:**
- **Framework:** Next.js 16 (App Router)
- **Database:** PostgreSQL via Neon Serverless + Drizzle ORM
- **UI:** Tailwind CSS + shadcn/ui
- **SDK:** TypeScript, zero dependencies, tree-shakeable

---

## Project Structure

```
lelemon/
├── apps/
│   └── web/                    # Next.js app (Dashboard + API)
│       ├── src/
│       │   ├── app/
│       │   │   ├── api/v1/     # REST API routes
│       │   │   │   ├── traces/
│       │   │   │   ├── analytics/
│       │   │   │   └── projects/
│       │   │   ├── dashboard/  # Dashboard pages
│       │   │   └── page.tsx    # Redirect to dashboard
│       │   ├── db/
│       │   │   ├── schema.ts   # Drizzle schema
│       │   │   └── client.ts   # Lazy-loaded DB client
│       │   ├── lib/
│       │   │   ├── auth.ts     # API key authentication
│       │   │   ├── pricing.ts  # LLM cost calculator
│       │   │   ├── api.ts      # Frontend API client
│       │   │   └── utils.ts    # Utilities (cn, etc.)
│       │   └── components/
│       │       ├── ui/         # shadcn/ui components
│       │       ├── theme-provider.tsx
│       │       └── theme-toggle.tsx
│       ├── drizzle.config.ts
│       └── package.json
│
├── packages/
│   └── sdk/                    # @lelemon/sdk (npm package)
│       ├── src/
│       │   ├── index.ts        # Exports
│       │   ├── tracer.ts       # LLMTracer, Trace, Span classes
│       │   ├── transport.ts    # HTTP client with batching
│       │   └── types.ts        # TypeScript types
│       └── package.json
│
├── package.json                # Workspace root
├── turbo.json                  # Turborepo config
└── tsconfig.base.json          # Shared TS config
```

---

## Development Commands

### Monorepo (root)
```bash
yarn install          # Install all dependencies
yarn dev              # Run all apps in dev mode
yarn build            # Build everything
yarn lint             # Lint all packages
```

### Web App (apps/web)
```bash
cd apps/web
yarn dev              # Dev server with Turbopack (port 3000)
yarn build            # Production build
yarn start            # Start production server

# Database
yarn db:generate      # Generate Drizzle migrations
yarn db:push          # Push schema to database
yarn db:studio        # Open Drizzle Studio GUI
```

### SDK (packages/sdk)
```bash
cd packages/sdk
yarn build            # Build CJS + ESM + types
yarn dev              # Watch mode
```

---

## API Routes

All API routes require authentication via Bearer token:
```
Authorization: Bearer le_xxx...
```

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/traces` | Create a new trace |
| GET | `/api/v1/traces` | List traces (with filters) |
| GET | `/api/v1/traces/:id` | Get trace with spans |
| PATCH | `/api/v1/traces/:id` | Update trace status |
| POST | `/api/v1/traces/:id/spans` | Add span to trace |
| GET | `/api/v1/analytics/summary` | Get aggregate metrics |
| GET | `/api/v1/analytics/usage` | Get usage over time |
| GET | `/api/v1/projects/me` | Get current project |
| PATCH | `/api/v1/projects/me` | Update project settings |
| POST | `/api/v1/projects/api-key` | Rotate API key |

---

## Database Schema

### Tables

**projects** - Multi-tenant projects
- `id`, `name`, `apiKey`, `apiKeyHash`, `ownerEmail`, `settings`

**traces** - Conversation/session traces
- `id`, `projectId`, `sessionId`, `userId`, `metadata`, `tags`
- Aggregates: `totalTokens`, `totalCostUsd`, `totalDurationMs`, `totalSpans`
- `status`: 'active' | 'completed' | 'error'

**spans** - Individual operations within a trace
- `id`, `traceId`, `parentSpanId`, `type`, `name`
- `input`, `output`, `inputTokens`, `outputTokens`, `costUsd`, `durationMs`
- `status`, `errorMessage`, `model`, `provider`, `metadata`

### Key Patterns

```typescript
// Always filter by projectId (multi-tenant)
const traces = await db.query.traces.findMany({
  where: eq(traces.projectId, auth.projectId),
});

// Use lazy-loaded db client (avoids build-time connection)
import { db } from '@/db/client';
```

---

## SDK Usage

```typescript
import { LLMTracer } from '@lelemon/sdk';

const tracer = new LLMTracer({
  apiKey: process.env.LELEMON_API_KEY,
  // endpoint: 'https://your-app.vercel.app' // optional
});

// Start a trace
const trace = await tracer.startTrace({
  sessionId: 'conversation-123',
  userId: 'user-456',
  metadata: { source: 'api' },
  tags: ['production'],
});

// Record a span
const span = trace.startSpan({
  type: 'llm',
  name: 'chat-completion',
  input: { messages },
});

// ... make LLM call ...

span.end({
  output: response,
  model: 'gpt-4',
  inputTokens: usage.prompt_tokens,
  outputTokens: usage.completion_tokens,
});

// End the trace
await trace.end();

// Flush before shutdown (serverless)
await tracer.flush();
```

---

## Code Patterns

### API Route Handler
```typescript
// src/app/api/v1/example/route.ts
import { NextRequest } from 'next/server';
import { z } from 'zod';
import { db } from '@/db/client';
import { authenticate, unauthorized, badRequest } from '@/lib/auth';

const schema = z.object({
  name: z.string().max(100),
});

export async function POST(request: NextRequest) {
  // 1. Authenticate
  const auth = await authenticate(request);
  if (!auth) return unauthorized();

  // 2. Validate body
  const body = await request.json();
  const result = schema.safeParse(body);
  if (!result.success) return badRequest(result.error.message);

  // 3. Database operation (always filter by projectId)
  const data = await db.insert(table).values({
    projectId: auth.projectId,
    ...result.data,
  }).returning();

  // 4. Return response
  return Response.json(data, { status: 201 });
}
```

### Dashboard Page (Client Component)
```typescript
'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function ExamplePage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Title</h1>
        <p className="text-muted-foreground">Description</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Section</CardTitle>
        </CardHeader>
        <CardContent>
          {/* Content */}
        </CardContent>
      </Card>
    </div>
  );
}
```

---

## Environment Variables

```bash
# .env.local (apps/web)
DATABASE_URL=postgresql://...   # Neon/Supabase connection string

# For SDK users
LELEMON_API_KEY=le_xxx...       # Project API key
```

---

## Key Design Decisions

### 1. Single Deployment
Dashboard and API in the same Next.js app for simplicity and cost (free Vercel tier).

### 2. Lazy Database Connection
The `db` client uses a Proxy to defer connection until first use, avoiding build-time errors.

### 3. Span Batching in SDK
Spans are queued and sent in batches (default: 10 spans or 1 second) to reduce HTTP overhead.

### 4. Cost Calculation
Costs are calculated server-side using the `pricing.ts` module when spans are created.

### 5. Dark Mode Default
Developer-focused UI defaults to dark theme via `next-themes`.

---

## Mandatory Patterns

### Multi-tenant Isolation
```typescript
// ALWAYS filter by projectId
where: eq(table.projectId, auth.projectId)
```

### Zod Validation
```typescript
// ALWAYS validate request bodies
const result = schema.safeParse(body);
if (!result.success) return badRequest(result.error.message);
```

### Error Responses
```typescript
// Use consistent error helpers
return unauthorized();           // 401
return badRequest('message');    // 400
return notFound('message');      // 404
```

### TypeScript Strict
```typescript
// No `any`, explicit return types on exports
export async function handler(): Promise<Response> { ... }
```

---

## Adding New Features

### New API Endpoint
1. Create route file: `src/app/api/v1/[feature]/route.ts`
2. Add Zod schema for validation
3. Use `authenticate()` for auth
4. Filter queries by `projectId`
5. Update this CLAUDE.md if significant

### New Dashboard Page
1. Create page: `src/app/dashboard/[page]/page.tsx`
2. Add navigation link in `dashboard/layout.tsx`
3. Use existing Card/Table components from shadcn/ui

### New SDK Feature
1. Add types to `packages/sdk/src/types.ts`
2. Implement in appropriate file
3. Export from `packages/sdk/src/index.ts`
4. Run `yarn build` to verify

---

## Deployment

**Platform:** Vercel (recommended)

```bash
# Deploy
vercel

# Environment variables to set:
# - DATABASE_URL (Neon connection string)
```

**Database:** Neon Serverless PostgreSQL
- Create project at neon.tech
- Copy connection string to DATABASE_URL
- Run `yarn db:push` to create tables

---

## Testing (Future)

Tests are not yet implemented. When adding:
- Use Vitest for unit tests
- Use Playwright for E2E tests
- Mock database with in-memory SQLite

---

## Common Issues

### Build fails with "DATABASE_URL not set"
The db client is lazy-loaded, so this shouldn't happen. If it does, check that no file imports `db` at module level outside of functions.

### Zod v4 syntax errors
This project uses Zod v3. Zod v4 has different syntax for some methods.

### Type errors with Drizzle
Run `yarn db:generate` to update types after schema changes.

---

## References

- **Next.js App Router:** https://nextjs.org/docs/app
- **Drizzle ORM:** https://orm.drizzle.team/docs/overview
- **shadcn/ui:** https://ui.shadcn.com
- **Neon:** https://neon.tech/docs

---

**Last updated:** 2026-01-03
