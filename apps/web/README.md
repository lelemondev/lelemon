# Lelemon Dashboard

Next.js frontend for the Lelemon LLM observability platform.

## Quick Start

```bash
pnpm install
pnpm dev
```

Open [http://localhost:3000](http://localhost:3000).

## Environment Variables

Create `.env.local`:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

## Tech Stack

- Next.js 16 (App Router)
- React 19
- Tailwind CSS
- shadcn/ui components

## Structure

```
src/
├── app/
│   ├── (auth)/           # Login, signup pages
│   └── dashboard/        # Main dashboard
├── components/
│   ├── ui/               # shadcn/ui components
│   └── traces/           # Trace visualization
└── lib/
    ├── api.ts            # API client
    ├── auth-context.tsx  # Auth state
    └── project-context.tsx
```

## Notes

- This is a frontend-only app - no API routes
- Auth uses JWT stored in localStorage
- API responses are normalized from Go's PascalCase to camelCase
