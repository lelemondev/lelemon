# 🍋 Lelemon

**LLM Observability for AI Agents**

Track, debug, and optimize your LLM applications with minimal setup.

---

## Features

- **Trace Everything** - LLM calls, tool usage, retrieval operations
- **Cost Tracking** - Automatic cost calculation for all major providers
- **Developer-First** - Clean SDK, dark mode dashboard, zero bloat
- **Multi-Tenant** - Secure API key isolation per project

## Quick Start

### 1. Install the SDK

```bash
npm install @lelemon/sdk
# or
yarn add @lelemon/sdk
```

### 2. Initialize

```typescript
import { LLMTracer } from '@lelemon/sdk';

const tracer = new LLMTracer({
  apiKey: process.env.LELEMON_API_KEY,
});
```

### 3. Trace Your LLM Calls

```typescript
// Start a trace (groups related operations)
const trace = await tracer.startTrace({
  sessionId: 'conversation-123',
  userId: 'user-456',
});

// Record a span (single operation)
const span = trace.startSpan({
  type: 'llm',
  name: 'chat-completion',
  input: { messages },
});

// Make your LLM call
const response = await openai.chat.completions.create({
  model: 'gpt-4',
  messages,
});

// End the span with results
span.end({
  output: response.choices[0].message,
  model: response.model,
  inputTokens: response.usage.prompt_tokens,
  outputTokens: response.usage.completion_tokens,
});

// End the trace
await trace.end();
```

### 4. View in Dashboard

Open your Lelemon dashboard to see traces, spans, costs, and analytics.

---

## SDK Reference

### LLMTracer

```typescript
const tracer = new LLMTracer({
  apiKey: string,          // Required: Your API key (le_xxx)
  endpoint?: string,       // Optional: Custom API endpoint
  debug?: boolean,         // Optional: Enable debug logging
  batchSize?: number,      // Optional: Spans per batch (default: 10)
  flushInterval?: number,  // Optional: Batch interval ms (default: 1000)
});

// Methods
await tracer.startTrace(options): Promise<Trace>
await tracer.flush(): Promise<void>
tracer.isEnabled(): boolean
```

### Trace

```typescript
const trace = await tracer.startTrace({
  sessionId?: string,      // Group related traces
  userId?: string,         // End user identifier
  metadata?: object,       // Any custom data
  tags?: string[],         // Filterable tags
});

// Methods
trace.startSpan(options): Span
trace.setMetadata(key, value): void
trace.addTag(tag): void
await trace.end({ status?: 'completed' | 'error' }): Promise<void>
```

### Span

```typescript
const span = trace.startSpan({
  type: 'llm' | 'tool' | 'retrieval' | 'custom',
  name: string,            // e.g., 'openai.chat', 'search_documents'
  input?: any,             // Request data
  metadata?: object,       // Custom data
});

// Methods
span.startSpan(options): Span  // Create child span
span.end({
  output?: any,
  status?: 'success' | 'error',
  errorMessage?: string,
  model?: string,
  provider?: string,
  inputTokens?: number,
  outputTokens?: number,
  durationMs?: number,     // Auto-calculated if not provided
}): void
span.setError(error: Error): void
```

---

## Supported Providers

Cost calculation is built-in for:

| Provider | Models |
|----------|--------|
| OpenAI | gpt-4, gpt-4-turbo, gpt-4o, gpt-4o-mini, gpt-3.5-turbo, o1-preview, o1-mini |
| Anthropic | claude-3-opus, claude-3-sonnet, claude-3-5-sonnet, claude-3-haiku |
| AWS Bedrock | All Claude models via Bedrock |
| Google | gemini-1.5-pro, gemini-1.5-flash, gemini-2.0-flash |

Unknown models use a default pricing estimate.

---

## Self-Hosting

### Prerequisites

- Node.js 20+
- PostgreSQL database (Neon recommended)

### Setup

```bash
# Clone
git clone https://github.com/your-org/lelemon.git
cd lelemon

# Install
yarn install

# Configure
cp apps/web/.env.example apps/web/.env.local
# Edit .env.local with your DATABASE_URL

# Setup database
cd apps/web
yarn db:push

# Run
yarn dev
```

### Deploy to Vercel

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel

# Set environment variables in Vercel dashboard:
# - DATABASE_URL
```

---

## Project Structure

```
lelemon/
├── apps/web/           # Next.js dashboard + API
│   ├── src/app/api/    # REST API routes
│   ├── src/app/dashboard/  # Dashboard UI
│   └── src/db/         # Drizzle schema
├── packages/sdk/       # @lelemon/sdk npm package
└── package.json        # Turborepo workspace
```

---

## Development

```bash
# Install dependencies
yarn install

# Run all in dev mode
yarn dev

# Build everything
yarn build

# Run specific app
cd apps/web && yarn dev

# Build SDK only
cd packages/sdk && yarn build
```

---

## API Reference

### Authentication

All API requests require a Bearer token:

```
Authorization: Bearer le_your_api_key
```

### Endpoints

#### Create Trace
```http
POST /api/v1/traces
Content-Type: application/json

{
  "sessionId": "session-123",
  "userId": "user-456",
  "metadata": { "source": "api" },
  "tags": ["production"]
}

Response: { "id": "trace-uuid" }
```

#### Add Span
```http
POST /api/v1/traces/:traceId/spans
Content-Type: application/json

{
  "type": "llm",
  "name": "chat-completion",
  "input": { "messages": [...] },
  "output": { "content": "..." },
  "model": "gpt-4",
  "inputTokens": 150,
  "outputTokens": 50,
  "durationMs": 1200
}

Response: { "id": "span-uuid" }
```

#### List Traces
```http
GET /api/v1/traces?sessionId=xxx&limit=50&offset=0

Response: {
  "data": [...],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

#### Get Analytics
```http
GET /api/v1/analytics/summary?from=2024-01-01&to=2024-01-31

Response: {
  "totalTraces": 1000,
  "totalSpans": 5000,
  "totalTokens": 500000,
  "totalCostUsd": 15.50,
  "avgDurationMs": 1500,
  "errorRate": 2.5
}
```

---

## License

MIT

---

## Contributing

Contributions welcome! Please read our contributing guidelines first.

1. Fork the repo
2. Create a feature branch
3. Make your changes
4. Run `yarn build` and `yarn lint`
5. Submit a PR

---

Built with 💛 for developers who ship AI products.
