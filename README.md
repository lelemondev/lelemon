# Lelemon

[![CI](https://github.com/lelemondev/lelemon/actions/workflows/ci.yml/badge.svg)](https://github.com/lelemondev/lelemon/actions/workflows/ci.yml)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](https://opensource.org/licenses/AGPL-3.0)

**Open-source LLM Observability Platform**

Track, debug, and optimize your AI agents with minimal setup. Self-hostable alternative to Langfuse, Helicone, and Arize.

---

## Features

- **Hierarchical Tracing** - Agent workflows, LLM calls, tool usage, retrieval operations
- **Cost Tracking** - Automatic cost calculation for OpenAI, Anthropic, Gemini, Bedrock
- **Analytics** - Model breakdown, latency percentiles (p50/p95/p99), usage heatmaps, tag-based segmentation
- **Multi-Provider** - OpenAI, Anthropic, Google Gemini, AWS Bedrock, OpenRouter
- **High Performance** - Go backend with SQLite, PostgreSQL, or ClickHouse
- **Self-Hosted** - Run on your infrastructure, own your data
- **SDKs** - TypeScript and Python with zero-config auto-instrumentation

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Your App  │────>│  Go Server  │────>│  Dashboard  │
│  (SDK)      │     │   (API)     │     │  (Next.js)  │
└─────────────┘     └──────┬──────┘     └─────────────┘
                    ┌──────┴──────┐
                    │  Database   │
                    │ SQLite/PG/  │
                    │ ClickHouse  │
                    └─────────────┘
```

## Quick Start

### 1. Start the Server

```bash
git clone https://github.com/lelemondev/lelemon.git
cd lelemon
docker-compose up -d
```

Open http://localhost:3000, create an account, and grab your API key.

### 2. Install an SDK

**TypeScript:**
```bash
npm install @lelemondev/sdk
```

**Python:**
```bash
pip install lelemondev
```

### 3. Instrument Your Code

**TypeScript:**
```typescript
import { init, observe, trace } from '@lelemondev/sdk/openai';
import OpenAI from 'openai';

init({ apiKey: process.env.LELEMON_API_KEY });
const openai = observe(new OpenAI(), {
  userId: 'user-123',
  sessionId: 'conversation-abc',
});

await trace('my-agent', async () => {
  const response = await openai.chat.completions.create({
    model: 'gpt-4',
    messages: [{ role: 'user', content: 'Hello!' }],
  });
  return response.choices[0].message.content;
});
```

**Python:**
```python
from openai import AsyncOpenAI
from lelemondev import init, observe, trace

init(api_key="your-api-key")
client = observe(AsyncOpenAI(), user_id="user-123", session_id="conv-abc")

async with trace("my-agent") as t:
    response = await client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello!"}]
    )
    t.set_result(response.choices[0].message.content)
```

### 4. View Traces

Open the dashboard to see your traces with full cost, latency, and token analytics.

---

## Self-Hosting

### Docker Compose (Recommended)

```bash
# SQLite (default, simplest)
docker-compose up -d

# PostgreSQL (production, >100k traces/day)
docker-compose -f docker-compose.postgres.yml up -d

# ClickHouse (high volume, >1M traces/day)
docker-compose -f docker-compose.clickhouse.yml up -d
```

### Manual Setup

```bash
# Backend
cd apps/server
go build -o lelemon ./cmd/server
DATABASE_URL=sqlite:///data/lelemon.db JWT_SECRET=your-secret ./lelemon

# Dashboard
cd apps/web
pnpm install && pnpm build && pnpm start
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `sqlite:///data/lelemon.db` | Database connection string |
| `JWT_SECRET` | - | Auth secret (**required** in production) |
| `PORT` | `8080` | API server port |
| `FRONTEND_URL` | `http://localhost:3000` | Dashboard URL (for CORS) |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

---

## SDKs

| SDK | Package | Docs |
|-----|---------|------|
| TypeScript | [`@lelemondev/sdk`](https://www.npmjs.com/package/@lelemondev/sdk) | [GitHub](https://github.com/lelemondev/lelemondev-sdk) |
| Python | [`lelemondev`](https://pypi.org/project/lelemondev/) | [GitHub](https://github.com/lelemondev/lelemondev-sdk-python) |

### Supported Providers

| Provider | TypeScript | Python |
|----------|-----------|--------|
| OpenAI | Yes | Yes |
| Anthropic | Yes | Yes |
| Google Gemini | Yes | Yes |
| AWS Bedrock | Yes | Yes |
| OpenRouter | Yes | Yes |

### Framework Integrations (TypeScript)

```typescript
import { withObserve } from '@lelemondev/sdk/next';     // Next.js
import { createMiddleware } from '@lelemondev/sdk/express'; // Express
import { withObserve } from '@lelemondev/sdk/lambda';    // AWS Lambda
import { createMiddleware } from '@lelemondev/sdk/hono';   // Hono
```

---

## Project Structure

```
lelemon/
├── apps/
│   ├── server/     # Go backend (API + ingestion)
│   ├── web/        # Next.js dashboard
│   └── playground/ # SDK testing app
├── ee/             # Enterprise edition (RBAC, billing, orgs)
├── docker-compose*.yml
└── .env.example
```

## Development

```bash
# Prerequisites: Go 1.24+, Node.js 20+, pnpm

# Backend
cd apps/server && go run ./cmd/server

# Dashboard
cd apps/web && pnpm install && pnpm dev

# Run tests
cd apps/server && go test ./...
cd apps/web && pnpm build
```

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

Found a vulnerability? Please report it responsibly. See [SECURITY.md](SECURITY.md).

## License

[AGPL-3.0](LICENSE) - Self-host freely. If you modify and offer as a service, you must open-source your changes.

The `ee/` directory contains enterprise features under a separate proprietary license.
