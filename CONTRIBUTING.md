# Contributing to Lelemon

Thanks for your interest in contributing! This guide will help you get started.

## Prerequisites

- **Go 1.24+** (backend)
- **Node.js 20+** with **pnpm** (dashboard)
- **Docker & Docker Compose** (optional, for databases)

## Setup

```bash
git clone https://github.com/lelemondev/lelemon.git
cd lelemon

# Backend
cd apps/server
go run ./cmd/server

# Dashboard (separate terminal)
cd apps/web
pnpm install
pnpm dev
```

The API runs on http://localhost:8080 and the dashboard on http://localhost:3000.

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feat/your-feature    # new feature
git checkout -b fix/issue-description # bug fix
```

### 2. Make Changes

- Follow existing code patterns
- Add tests for new functionality
- Update documentation if needed

### 3. Test

```bash
# Backend
cd apps/server
go test ./...
go vet ./...

# Dashboard
cd apps/web
pnpm build       # build check
pnpm lint        # lint check
```

### 4. Commit

Use [conventional commits](https://www.conventionalcommits.org/):

```
feat: add session filtering to traces page
fix: correct token count calculation for Claude
docs: update self-hosting instructions
feat(analytics): add model breakdown endpoint
fix(auth): handle OAuth callback errors
```

### 5. Open a Pull Request

- Fill out the PR template
- Link related issues
- Ensure CI passes

## Code Style

### Go (Backend)

- Standard Go conventions (`gofmt`, `go vet`)
- Clean Architecture: `domain` -> `application` -> `infrastructure` -> `interfaces`
- Always parameterize SQL queries (never string interpolation)
- Wrap errors with context: `fmt.Errorf("failed to get traces: %w", err)`
- Filter by `projectID` in every query (multi-tenant isolation)

### TypeScript (Dashboard)

- TypeScript strict mode
- Use `@/` path aliases for imports
- Use shadcn/ui components (`Card`, `Button`, `Table`, etc.)
- Tailwind CSS with semantic classes (prefer `text-muted-foreground` over hex colors)
- Client components only when needed (`'use client'` for hooks/events)

## Reporting Issues

- **Bugs**: Use the [bug report template](https://github.com/lelemondev/lelemon/issues/new?template=bug_report.yml)
- **Features**: Use the [feature request template](https://github.com/lelemondev/lelemon/issues/new?template=feature_request.yml)
- **Security**: See [SECURITY.md](SECURITY.md) (do not open public issues)

## License

By contributing, you agree that your contributions will be licensed under the [AGPL-3.0](LICENSE) license.
