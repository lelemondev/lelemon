# Contributing to Lelemon

Thanks for your interest in contributing to Lelemon! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

- Node.js 20+
- Go 1.21+ (for backend)
- Docker & Docker Compose (optional, for databases)

### Setup

```bash
# Clone the repo
git clone https://github.com/lelemondev/lelemon.git
cd lelemon

# Install dependencies
yarn install

# Start development
yarn dev
```

### Project Structure

```
lelemon/
├── apps/
│   ├── web/        # Next.js dashboard (frontend)
│   ├── server/     # Go backend (API + ingestion)
│   └── playground/ # SDK testing app
└── docs/           # Documentation
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Changes

- Follow existing code style
- Add tests for new functionality
- Update documentation if needed

### 3. Test Your Changes

```bash
# Frontend (apps/web)
cd apps/web
yarn build
yarn lint

# Backend (apps/server)
cd apps/server
go test ./...
go build ./cmd/server
```

### 4. Commit

Write clear commit messages:

```
feat: add session filtering to traces page
fix: correct token count calculation for Claude
docs: update self-hosting instructions
```

### 5. Submit a Pull Request

- Fill out the PR template
- Link related issues
- Wait for review

## Code Style

### TypeScript (Frontend)

- Use TypeScript strict mode
- Prefer `const` over `let`
- Use named exports
- Follow existing patterns in the codebase

### Go (Backend)

- Follow standard Go conventions
- Use `gofmt` and `golint`
- Keep functions small and focused
- Handle errors explicitly

## Reporting Issues

When reporting bugs, please include:

- Steps to reproduce
- Expected vs actual behavior
- Environment (OS, Node/Go version, browser)
- Relevant logs or screenshots

## Feature Requests

Open an issue with:

- Clear description of the feature
- Use case / motivation
- Possible implementation approach (optional)

## License

By contributing, you agree that your contributions will be licensed under the AGPL-3.0 license.

## Questions?

Open an issue or start a discussion. We're happy to help!
