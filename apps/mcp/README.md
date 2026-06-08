# @lelemondev/mcp 🍋

Model Context Protocol (MCP) server for [Lelemon](https://lelemon.dev) — query your LLM
traces, **cost breakdowns**, sessions and analytics straight from your agent (Claude Desktop,
Cursor, Claude Code, or any MCP client). Built with [mcify](https://github.com/Lelemon-studio/mcify).

> **Read-only.** v1 exposes query tools only — it never mutates your data.

## Tools

| Tool | What it does |
|------|--------------|
| `lelemon_get_project` | The project your API key belongs to (confirm scope). |
| `lelemon_list_traces` | List traces, filter by status / session / user / time range. Paginated (`hasMore`). |
| `lelemon_get_trace` | One trace + its span tree. Each LLM span carries a **`costBreakdown`** (input / output / cacheRead / cacheWrite / reasoning / total / **cacheSavings**). Concise by default; pass `detail: true` for full prompts/completions. |
| `lelemon_list_sessions` | Sessions (conversations) with rollup metrics. Paginated. |
| `lelemon_analytics` | One tool over every metric via `metric`: `summary`, `usage`, `models` (💸 cost by model), `tags`, `top_users`, `heatmap`, `latency_distribution`, `latency_timeseries`. Filter with `from`/`to`; `granularity` for time series; `limit` for ranked metrics. |

Example asks once connected:
- *"What did each model cost me this week?"* → `lelemon_analytics(metric: "models", from: …)`
- *"Show the last failed trace and where the money went."* → `lelemon_list_traces(status: "error")` → `lelemon_get_trace(id)`
- *"How much is prompt caching saving me?"* → read `cacheSavings` in the trace's cost breakdown.

## Auth model

The **bearer token you present IS your Lelemon project API key** (`le_…`). The server validates it
on every request against `GET /projects/me`, so:

- It's **multi-tenant for free** — each key resolves to exactly one project.
- No server-side secret to manage; the key travels in each request's `Authorization` header.
- Your key never leaves the handler layer (not logged, not echoed).

Get your key from the Lelemon dashboard → Project settings.

## Run locally

```bash
cd lelemon/apps/mcp
pnpm install
pnpm dev
```

- MCP endpoint: `http://localhost:8888/mcp`
- Inspector (try tools in a browser): `http://localhost:3001` — paste your `le_…` key as the bearer.

### Environment

| Var | Default | Purpose |
|-----|---------|---------|
| `LELEMON_ENDPOINT` | `https://api.lelemon.dev` | API base URL. Point it at your self-hosted server, e.g. `http://localhost:8080`. |

Auth needs **no** server-side env var — clients authenticate per request with their own API key.

## Connect a client

In every config below, replace `le_your_key` with your Lelemon API key. For a self-hosted Lelemon,
also run the MCP with `LELEMON_ENDPOINT` pointing at your server.

### Claude Code

```bash
claude mcp add --transport http lelemon http://localhost:8888/mcp \
  --header "Authorization: Bearer le_your_key"
```

### Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or
`%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "lelemon": {
      "url": "http://localhost:8888/mcp",
      "headers": { "authorization": "Bearer le_your_key" }
    }
  }
}
```

### Cursor

`~/.cursor/mcp.json` (or `.cursor/mcp.json` in your project):

```json
{
  "mcpServers": {
    "lelemon": {
      "url": "http://localhost:8888/mcp",
      "headers": { "authorization": "Bearer le_your_key" }
    }
  }
}
```

Restart the client; the five `lelemon_*` tools appear.

## Design notes

Tools follow agent-tooling best practices (Anthropic's *Writing effective tools for agents*,
and Langfuse's MCP): **few, consolidated tools** (one `lelemon_analytics` over 8 endpoints rather
than one per chart), **typed filters instead of a query DSL** (more reliable for agents),
**token-efficient responses** (`get_trace` is concise by default; lists return `hasMore`), and
**actionable errors** (e.g. a 401 tells the agent to check its API key).

## Scripts

```bash
pnpm dev         # run MCP + inspector (watch)
pnpm build       # tsc -> dist
pnpm typecheck   # tsc --noEmit
pnpm test        # vitest
pnpm lint        # eslint
```

## Roadmap

- Write tools (create datasets from traces, annotate scores)
- Hosted/edge deploy (multi-tenant) via mcify
- EE tools (evals, prompts)
