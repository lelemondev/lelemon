# CLAUDE.md

> Configuration file for Claude Code AI assistant. Last updated: 2026-01.

## Project Description

**Lelemon** is an open-source LLM observability platform that helps developers track, debug, and optimize their AI agents. It provides hierarchical tracing for LLM calls, tool usage, token consumption, and cost analytics.

**Key Features:**
- Multi-tenant with API key + JWT authentication
- Real-time trace ingestion via SDK with batching
- Cost calculation for all major LLM providers
- High-performance Go backend with multiple database options
- Session tracking and conversation grouping
- Developer-focused dashboard with dark mode

**Tech Stack:**

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24 (Chi router, Clean Architecture) |
| Dashboard | Next.js 16 (App Router, React 19) |
| Database | SQLite / PostgreSQL / ClickHouse |
| UI | Tailwind CSS + shadcn/ui |
| Auth | JWT + Google OAuth2 |

**Related Projects:**
- **SDK:** [@lelemondev/sdk](https://github.com/lelemondev/sdk) - TypeScript SDK (separate repository)
- **Enterprise:** lelemon-cloud (private) - Multi-tenancy, billing, RBAC

---

## Project Structure

```
lelemon/
├── apps/
│   ├── server/                    # Go backend (API + Auth)
│   │   ├── cmd/server/main.go     # Entry point
│   │   ├── pkg/                   # Exportable packages
│   │   │   ├── domain/
│   │   │   │   ├── entity/        # User, Project, Trace, Span, Session
│   │   │   │   ├── repository/    # Store interfaces
│   │   │   │   └── service/       # Pricing calculator, parser
│   │   │   ├── application/       # Business logic
│   │   │   │   ├── ingest/        # Async batch ingestion
│   │   │   │   ├── trace/         # Trace retrieval
│   │   │   │   ├── analytics/     # Stats & usage
│   │   │   │   ├── auth/          # Login, register, JWT
│   │   │   │   └── project/       # Project CRUD
│   │   │   ├── infrastructure/
│   │   │   │   ├── store/         # SQLite, PostgreSQL, ClickHouse
│   │   │   │   ├── auth/          # JWT service, OAuth
│   │   │   │   ├── config/        # Environment loading
│   │   │   │   └── logger/        # Structured logging (slog)
│   │   │   └── interfaces/http/
│   │   │       ├── handler/       # HTTP handlers
│   │   │       ├── middleware/    # Auth, logging, rate-limit
│   │   │       ├── router.go      # Route definitions
│   │   │       └── server.go      # Server setup
│   │   ├── Dockerfile
│   │   └── docker-compose*.yml
│   │
│   ├── web/                       # Next.js dashboard (frontend only)
│   │   ├── src/
│   │   │   ├── app/
│   │   │   │   ├── (auth)/        # Login, signup pages
│   │   │   │   └── dashboard/     # Dashboard pages
│   │   │   ├── components/
│   │   │   │   ├── ui/            # shadcn/ui components
│   │   │   │   └── traces/        # Trace visualization
│   │   │   └── lib/
│   │   │       ├── api.ts         # API client (normalizes Go responses)
│   │   │       ├── auth-context.tsx  # JWT auth provider
│   │   │       └── project-context.tsx
│   │   └── package.json
│   │
│   └── playground/                # SDK testing app (port 3001)
│       └── src/                   # Multi-provider chat interface
│
└── docs/
    └── ROADMAP.md
```

---

## Quick Commands

### Backend (apps/server)
```bash
cd apps/server

go run ./cmd/server           # Run server (port 8080)
go build -o lelemon ./cmd/server  # Build binary
go test ./...                 # Run tests

# Docker
docker-compose up -d                              # SQLite (default)
docker-compose -f docker-compose.postgres.yml up -d   # PostgreSQL
docker-compose -f docker-compose.clickhouse.yml up -d # ClickHouse
```

### Dashboard (apps/web)
```bash
cd apps/web
yarn install
yarn dev              # Dev server (port 3000)
yarn build            # Production build
```

### Playground (apps/playground)
```bash
cd apps/playground
yarn install
yarn dev              # Dev server (port 3001)
```

### Monorepo Root
```bash
yarn install          # Install all dependencies
yarn dev              # Run dashboard in dev mode
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        apps/web (Next.js)                       │
│            Dashboard - Frontend Only (port 3000)                │
│    ┌──────────────────────────────────────────────────┐         │
│    │  AuthContext → API Client → Go Backend           │         │
│    │  (JWT stored in localStorage)                    │         │
│    └──────────────────────────────────────────────────┘         │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTP (JSON)
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     apps/server (Go)                            │
│                   Backend API (port 8080)                       │
├─────────────────────────────────────────────────────────────────┤
│  interfaces/http    │  application/     │  domain/              │
│  ├── handlers       │  ├── ingest       │  ├── entity           │
│  ├── middleware     │  ├── trace        │  ├── repository       │
│  └── router         │  ├── analytics    │  └── service          │
│                     │  ├── auth         │                       │
│                     │  └── project      │                       │
├─────────────────────────────────────────────────────────────────┤
│                    infrastructure/store                         │
│         SQLite │ PostgreSQL │ ClickHouse (selectable)           │
└─────────────────────────────────────────────────────────────────┘
                             ▲
                             │ SDK Ingestion
┌────────────────────────────┴────────────────────────────────────┐
│                    @lelemondev/sdk                              │
│              Instrumented LLM Applications                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## API Endpoints

Base URL: `http://localhost:8080/api/v1`

### Authentication Methods

| Type | Usage | Header |
|------|-------|--------|
| API Key | SDK ingestion | `Authorization: Bearer le_xxx...` |
| JWT | Dashboard | `Authorization: Bearer <jwt_token>` |

### SDK Endpoints (API Key Auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/ingest` | Batch ingest spans |
| POST | `/traces` | Create trace |
| POST | `/traces/:id/spans` | Add span to trace |
| PATCH | `/traces/:id` | Update trace status |

### Dashboard Endpoints (JWT Auth)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/dashboard/projects` | List user projects |
| POST | `/dashboard/projects` | Create project |
| GET | `/dashboard/projects/:id/stats` | Project statistics |
| GET | `/dashboard/projects/:id/traces` | List traces |
| GET | `/dashboard/projects/:id/traces/:traceId` | Trace with spans |
| GET | `/dashboard/projects/:id/sessions` | List sessions |

### Auth Endpoints (No Auth)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register user |
| POST | `/auth/login` | Login (email/password) |
| GET | `/auth/google` | Google OAuth redirect |
| GET | `/auth/google/callback` | OAuth callback |
| POST | `/auth/refresh` | Refresh JWT token |

---

## Database Schema

### Tables

**users**
```sql
id, email, password_hash, name, google_id, created_at
```

**projects**
```sql
id, user_id, name, api_key, api_key_hash, settings, created_at
```

**traces**
```sql
id, project_id, name, session_id, user_id, input, output, metadata, tags,
total_tokens, total_cost_usd, total_duration_ms, total_spans, span_counts,
status ('active' | 'completed' | 'error'), created_at, updated_at
```

**spans**
```sql
id, trace_id, parent_span_id, type, sub_type, name,
input, output, input_tokens, output_tokens, cost_usd, duration_ms,
status, stop_reason, error_message, model, provider,
cache_read_tokens, cache_write_tokens, reasoning_tokens, thinking,
tool_calls, tool_uses, metadata, created_at
```

### Database Options

| Database | Use Case | URL Format |
|----------|----------|------------|
| SQLite | Development, small scale | `sqlite:///data/lelemon.db` |
| PostgreSQL | Production | `postgres://user:pass@host/db` |
| ClickHouse | High volume analytics | `clickhouse://user:pass@host/db` |

---

## Key Patterns

### Go Handler Pattern
```go
// pkg/interfaces/http/handler/example.go
func (h *Handler) GetExample(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    userID := middleware.GetUserID(ctx)

    result, err := h.service.GetExample(ctx, userID)
    if err != nil {
        http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### Go Service Pattern
```go
// pkg/application/example/service.go
type Service struct {
    store repository.Store
}

func (s *Service) GetExample(ctx context.Context, userID string) (*entity.Example, error) {
    return s.store.FindByUserID(ctx, userID)
}
```

### Dashboard API Call
```typescript
// Uses api.ts client which normalizes Go PascalCase to camelCase
import { dashboardAPI } from '@/lib/api';

const traces = await dashboardAPI.getTraces(projectId);
// Response: { id, projectId, totalTokens, totalCostUsd, ... }
```

### Dashboard Page
```typescript
'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuth } from '@/lib/auth-context';

export default function ExamplePage() {
  const { user } = useAuth();

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

### Backend (apps/server/.env)
```bash
# Required
DATABASE_URL=sqlite:///data/lelemon.db
JWT_SECRET=your-random-secret-key
PORT=8080

# Optional
ANALYTICS_DATABASE_URL=   # Separate DB for traces
JWT_EXPIRATION=24h
LOG_LEVEL=info
LOG_FORMAT=json

# OAuth (optional)
GOOGLE_CLIENT_ID=xxx
GOOGLE_CLIENT_SECRET=xxx
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
FRONTEND_URL=http://localhost:3000
```

### Dashboard (apps/web/.env.local)
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Playground (apps/playground/.env.local)
```bash
NEXT_PUBLIC_LELEMON_API_KEY=le_xxx...
NEXT_PUBLIC_LELEMON_ENDPOINT=http://localhost:8080
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GOOGLE_AI_API_KEY=...
```

---

## Adding New Features

### New API Endpoint
1. Add handler in `pkg/interfaces/http/handler/`
2. Add service in `pkg/application/[feature]/`
3. Add store method in `pkg/infrastructure/store/`
4. Register route in `pkg/interfaces/http/router.go`

### New Dashboard Page
1. Create page in `apps/web/src/app/dashboard/[page]/page.tsx`
2. Add navigation link in `dashboard/layout.tsx`
3. Add API method in `lib/api.ts` if needed
4. Use shadcn/ui components

### New Database Support
1. Create store in `pkg/infrastructure/store/[db]/`
2. Implement `repository.Store` interface
3. Add factory case in `store/factory.go`

---

## Deployment

### Docker (Recommended)
```bash
cd apps/server

# Build image
docker build -t lelemon-server .

# Run with SQLite
docker-compose up -d

# Run with PostgreSQL
docker-compose -f docker-compose.postgres.yml up -d
```

### Railway
```bash
# Backend: Connect to apps/server
# Set DATABASE_URL to PostgreSQL

# Frontend: Connect to apps/web
# Set NEXT_PUBLIC_API_URL to backend URL
```

### Manual
```bash
# Backend
cd apps/server
CGO_ENABLED=0 go build -o lelemon ./cmd/server
./lelemon

# Frontend
cd apps/web
yarn build
yarn start
```

---

## References

- **Go Chi Router:** https://go-chi.io
- **Next.js App Router:** https://nextjs.org/docs/app
- **shadcn/ui:** https://ui.shadcn.com
- **SDK Docs:** https://github.com/lelemondev/sdk

---

## Claude Code Instructions

When working on this codebase:

1. **Use pkg/ not internal/** - Code is in `pkg/` for importability (open-source)
2. **Clean Architecture** - domain → application → infrastructure → interfaces
3. **Test changes** - Run `go test ./...` before committing
4. **Dashboard is frontend-only** - No database or API routes in Next.js
5. **API client normalizes responses** - Go uses PascalCase, JS uses camelCase
6. **Use TodoWrite** - Track multi-step tasks

---

**License:** AGPL-3.0
