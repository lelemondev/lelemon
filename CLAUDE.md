# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Description

**Lelemon** is an open-source LLM observability platform that helps developers track, debug, and optimize their AI agents. It provides hierarchical tracing for LLM calls, tool usage, token consumption, and cost analytics.

**Key Features:**
- Multi-tenant with API key + JWT authentication
- Real-time trace ingestion via SDK
- Cost calculation for all major LLM providers
- High-performance Go backend with multiple database options
- Developer-focused dashboard with dark mode

**Tech Stack:**
- **Backend:** Go 1.24 (Chi router, Clean Architecture)
- **Dashboard:** Next.js 15 (App Router)
- **Database:** SQLite / PostgreSQL / ClickHouse
- **UI:** Tailwind CSS + shadcn/ui

**Related Projects:**
- **SDK:** [@lelemondev/sdk](https://github.com/lelemondev/sdk) - TypeScript SDK (separate repository)

---

## Project Structure

```
lelemon/
├── apps/
│   ├── server/              # Go backend (API + Auth)
│   │   ├── cmd/server/      # Main entry point
│   │   ├── internal/
│   │   │   ├── domain/      # Entities (Project, Trace, Span, User)
│   │   │   ├── application/ # Services (Ingest, Trace, Analytics, Auth)
│   │   │   ├── infrastructure/  # Stores (SQLite, PostgreSQL, ClickHouse)
│   │   │   └── interfaces/  # HTTP handlers, middleware
│   │   ├── migrations/      # SQL migrations
│   │   └── docker-compose*.yml
│   │
│   ├── web/                 # Next.js dashboard
│   │   ├── src/app/
│   │   │   ├── (auth)/      # Login, signup pages
│   │   │   └── dashboard/   # Dashboard pages
│   │   └── src/components/  # UI components
│   │
│   └── playground/          # SDK testing app
│
└── docs/
    └── ROADMAP.md           # Development roadmap
```

---

## Development Commands

### Backend (apps/server)
```bash
cd apps/server

# Run
go run ./cmd/server

# Build
go build -o lelemon ./cmd/server

# Test
go test ./...

# With Docker (SQLite)
docker-compose up -d

# With PostgreSQL
docker-compose -f docker-compose.postgres.yml up -d

# With ClickHouse
docker-compose -f docker-compose.clickhouse.yml up -d
```

### Dashboard (apps/web)
```bash
cd apps/web
yarn install
yarn dev              # Dev server (port 3000)
yarn build            # Production build
```

### Monorepo Root
```bash
yarn install          # Install dashboard dependencies
yarn dev              # Run dashboard in dev mode
```

---

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

### Authentication

SDK requests use API key:
```
Authorization: Bearer le_xxx...
```

Dashboard uses JWT (cookie-based sessions).

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/ingest` | API Key | Batch ingest spans |
| GET | `/traces` | JWT | List traces |
| GET | `/traces/:id` | JWT | Get trace with spans |
| GET | `/analytics/summary` | JWT | Aggregate metrics |
| GET | `/analytics/usage` | JWT | Usage over time |
| POST | `/auth/register` | None | Register user |
| POST | `/auth/login` | None | Login |
| POST | `/auth/google` | None | Google OAuth |
| GET | `/projects` | JWT | List projects |
| POST | `/projects` | JWT | Create project |
| POST | `/projects/:id/rotate-key` | JWT | Rotate API key |

---

## Database Schema

### Tables

**users** - Registered users
- `id`, `email`, `passwordHash`, `name`, `googleId`

**projects** - Multi-tenant projects
- `id`, `userId`, `name`, `apiKey`, `apiKeyHash`, `settings`

**traces** - Agent workflow traces
- `id`, `projectId`, `name`, `sessionId`, `userId`, `input`, `output`
- Metrics: `totalTokens`, `totalCostUsd`, `totalDurationMs`, `totalSpans`
- `status`: 'active' | 'completed' | 'error'

**spans** - Individual operations within a trace
- `id`, `traceId`, `parentSpanId`, `type`, `name`
- `input`, `output`, `inputTokens`, `outputTokens`, `costUsd`, `durationMs`
- `status`, `stopReason`, `errorMessage`, `model`, `provider`

### Database Options

| Database | Use Case | Config |
|----------|----------|--------|
| SQLite | Development, small scale | `DATABASE_URL=sqlite://./data/lelemon.db` |
| PostgreSQL | Production, moderate scale | `DATABASE_URL=postgres://...` |
| ClickHouse | High volume, analytics | `DATABASE_URL=clickhouse://...` |

---

## SDK Integration

The SDK is in a separate repository. Dashboard connects to the Go backend.

```typescript
// SDK usage (in user's app)
import { init, observe } from '@lelemondev/sdk';

init({
  apiKey: process.env.LELEMON_API_KEY,
  endpoint: 'http://localhost:8080',
});

const client = observe(new Anthropic());
await client.messages.create({ ... });
```

---

## Code Patterns

### Go Handler
```go
// internal/interfaces/http/handlers/example.go
func (h *Handler) GetExample(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.GetUserID(ctx)
    
    result, err := h.service.GetExample(ctx, userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(result)
}
```

### Go Service
```go
// internal/application/services/example.go
func (s *ExampleService) GetExample(ctx context.Context, userID string) (*Example, error) {
    return s.store.FindByUserID(ctx, userID)
}
```

### Dashboard Page
```typescript
'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function ExamplePage() {
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">Title</h1>
      <Card>
        <CardHeader>
          <CardTitle>Section</CardTitle>
        </CardHeader>
        <CardContent>{/* ... */}</CardContent>
      </Card>
    </div>
  );
}
```

---

## Environment Variables

### Backend (apps/server)
```bash
DATABASE_URL=sqlite://./data/lelemon.db
JWT_SECRET=your-secret-key
PORT=8080
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=xxx
```

### Dashboard (apps/web)
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

## Key Design Decisions

### 1. Separate Backend
Go backend for performance and multi-database support. Dashboard is purely frontend.

### 2. Clean Architecture
Domain entities are independent of infrastructure. Easy to swap databases.

### 3. Multiple Database Support
SQLite for dev, PostgreSQL for production, ClickHouse for high-volume analytics.

### 4. SDK in Separate Repo
SDK has its own release cycle and dependencies.

---

## Adding New Features

### New API Endpoint
1. Add handler in `internal/interfaces/http/handlers/`
2. Add service method in `internal/application/services/`
3. Add store method in `internal/infrastructure/store/`
4. Register route in `cmd/server/main.go`

### New Dashboard Page
1. Create page in `apps/web/src/app/dashboard/[page]/page.tsx`
2. Add navigation link in `dashboard/layout.tsx`
3. Use shadcn/ui components

---

## References

- **Go Chi Router:** https://go-chi.io
- **Next.js App Router:** https://nextjs.org/docs/app
- **shadcn/ui:** https://ui.shadcn.com
- **SDK Docs:** https://github.com/lelemondev/sdk

---

**License:** AGPL-3.0
