# Lelemon

**Open-source LLM Observability Platform**

Track, debug, and optimize your AI agents with minimal setup. Self-hostable alternative to Langfuse/Arize.

---

## Features

- **Hierarchical Tracing** - Agent workflows, LLM calls, tool usage, retrieval operations
- **Cost Tracking** - Automatic cost calculation for OpenAI, Anthropic, Gemini, Bedrock
- **High Performance** - Go backend with SQLite/PostgreSQL/ClickHouse support
- **Developer-First** - Clean SDK, dark mode dashboard, zero bloat
- **Self-Hosted** - Run on your infrastructure, own your data

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Your App  │────▶│  Go Server  │────▶│  Dashboard  │
│ (@lelemondev│     │   (API)     │     │  (Next.js)  │
│    /sdk)    │     └──────┬──────┘     └─────────────┘
└─────────────┘            │
                    ┌──────┴──────┐
                    │  Database   │
                    │ SQLite/PG/  │
                    │ ClickHouse  │
                    └─────────────┘
```

## Quick Start

### 1. Install the SDK

```bash
npm install @lelemondev/sdk
```

### 2. Instrument Your Code

```typescript
import { init, observe } from '@lelemondev/sdk';
import Anthropic from '@anthropic-ai/sdk';

init({
  apiKey: process.env.LELEMON_API_KEY,
  endpoint: 'http://localhost:8080', // your server
});

// Wrap your LLM client - all calls are traced automatically
const client = observe(new Anthropic());

const response = await client.messages.create({
  model: 'claude-sonnet-4-20250514',
  messages: [{ role: 'user', content: 'Hello!' }],
});
```

### 3. View Traces

Open the dashboard to see your traces, costs, and analytics.

---

## Self-Hosting

### Option 1: Docker Compose (Recommended)

```bash
git clone https://github.com/lelemondev/lelemon.git
cd lelemon
docker-compose up -d
```

That's it! Open http://localhost:3000 to access the dashboard.

**Alternative databases:**

```bash
# PostgreSQL (production, >100k traces/day)
docker-compose -f docker-compose.postgres.yml up -d

# ClickHouse (high volume, >1M traces/day)
docker-compose -f docker-compose.clickhouse.yml up -d
```

### Option 2: Manual Setup

```bash
# Backend
cd apps/server
go build -o lelemon ./cmd/server
./lelemon

# Dashboard (separate terminal)
cd apps/web
yarn install && yarn build && yarn start
```

### Environment Variables

Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | - | Auth secret (required in production) |
| `LOG_LEVEL` | `info` | debug, info, warn, error |
| `POSTGRES_PASSWORD` | `lelemon` | PostgreSQL password |
| `CLICKHOUSE_PASSWORD` | - | ClickHouse password |

---

## SDK

The TypeScript SDK is maintained in a separate repository:

**[@lelemondev/sdk](https://github.com/lelemondev/sdk)** - Zero-dependency, tree-shakeable SDK

### Supported Providers

| Provider | Auto-detected |
|----------|---------------|
| OpenAI | Yes |
| Anthropic | Yes |
| Google Gemini | Yes |
| AWS Bedrock | Yes |
| OpenRouter | Yes |

### Framework Integrations

```typescript
// Next.js
import { withLelemon } from '@lelemondev/sdk/next';

// Express
import { lelemonMiddleware } from '@lelemondev/sdk/express';

// AWS Lambda
import { withLelemon } from '@lelemondev/sdk/lambda';
```

---

## Project Structure

```
lelemon/
├── docker-compose.yml            # SQLite (default)
├── docker-compose.postgres.yml   # PostgreSQL
├── docker-compose.clickhouse.yml # ClickHouse
├── .env.example
├── apps/
│   ├── server/     # Go backend
│   └── web/        # Next.js dashboard
└── docs/
```

---

## Development

```bash
# Install frontend dependencies
yarn install

# Run dashboard in dev mode
yarn dev

# Run server
cd apps/server && go run ./cmd/server
```

---

## Roadmap

See [docs/ROADMAP.md](docs/ROADMAP.md) for the full roadmap including:

- Hierarchical tracing with `withTrace()` API
- Extended thinking & tool use visualization
- Vercel AI SDK integration
- LangChain/LlamaIndex integrations

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

[AGPL-3.0](LICENSE) - You can self-host freely. If you modify and offer as a service, you must open-source your changes.

---

## Links

- **SDK**: [@lelemondev/sdk](https://github.com/lelemondev/sdk)
- **Documentation**: Coming soon
- **Discord**: Coming soon
