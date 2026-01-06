# Lelemon Roadmap

## Completed

### Phase 1: Core MVP
- [x] Go backend structure (Clean Architecture)
- [x] Domain entities (Project, Trace, Span)
- [x] SQLite store implementation
- [x] Ingest service + handler
- [x] API Key auth middleware
- [x] Health endpoint
- [x] Cost calculation (25+ models)

### Phase 2: Full API
- [x] Trace service (CRUD)
- [x] Analytics service (stats, usage time series)
- [x] Project service (CRUD + rotate key)
- [x] Rate limiting (100 req/min)
- [x] All handlers

### Phase 3: Dashboard Auth
- [x] JWT auth (register/login)
- [x] Google OAuth integration
- [x] Session middleware
- [x] Dashboard routes

### Phase 4: PostgreSQL
- [x] PostgreSQL store implementation
- [x] Connection pooling (pgx/v5, 5-25 connections)
- [x] Native JSONB support for metadata

### Phase 5: ClickHouse
- [x] ClickHouse store implementation
- [x] ReplacingMergeTree for Users/Projects/Traces (updates)
- [x] MergeTree for Spans (append-only, high-volume)
- [x] Optimized analytics queries (toStartOfHour, toDate, etc.)
- [x] Batch inserts via PrepareBatch
- [x] Full test coverage

### Phase 6: Production Ready
- [x] Structured logging (slog) with request IDs
- [x] Graceful shutdown with 30s timeout
- [x] Docker compose examples (SQLite, PostgreSQL, ClickHouse)
- [x] Improved health checks (/health, /health/live, /health/ready)

### TypeScript SDK (`@lelemondev/sdk`)
- [x] Core SDK: `init()`, `observe()`, `flush()`, `captureSpan()`
- [x] Batch transport (10 spans or 1 second)
- [x] Auto-detect OpenAI/Anthropic/Gemini/Bedrock/OpenRouter formats
- [x] Zero dependencies, tree-shakeable
- [x] Compatible with Go backend `/api/v1/ingest` endpoint

---

## In Progress

### Phase 7: Hierarchical Tracing & Visualization

#### Problema Actual
Cada llamada LLM crea un trace independiente. No hay agrupación de pasos de un agente bajo un trace padre.

```
ACTUAL:
  Trace 1 (LLM call) → Span 1
  Trace 2 (LLM call) → Span 1
  Trace 3 (LLM call) → Span 1

DESEADO:
  Trace 1 (Agent workflow)
    ├── Span 1 (Retrieval)
    ├── Span 2 (LLM call) → thinking, tool_use
    │   └── Span 3 (Tool execution)
    └── Span 4 (LLM call) → final response
```

#### 7.1 Enriquecer Captura del SDK
**Sin cambiar la API pública.**

- [ ] Extraer `stopReason` / `finishReason`
  - Anthropic: `response.stop_reason` ('end_turn', 'tool_use', 'max_tokens')
  - OpenAI: `choice.finish_reason` ('stop', 'tool_calls', 'length')
- [ ] Extraer tokens de cache (Anthropic)
  - `usage.cache_read_input_tokens`
  - `usage.cache_creation_input_tokens`
- [ ] Extraer reasoning tokens (OpenAI o1)
  - `usage.completion_tokens_details.reasoning_tokens`
- [ ] Extraer `thinking` blocks (Claude extended thinking)
  - `content.filter(b => b.type === 'thinking')`
- [ ] Medir `firstTokenMs` en streaming
- [ ] Actualizar schema del backend con nuevos campos

**Nuevos campos en Span:**
```typescript
{
  stopReason?: string;       // 'end_turn' | 'tool_use' | 'stop' | 'max_tokens'
  cacheReadTokens?: number;
  cacheWriteTokens?: number;
  reasoningTokens?: number;
  firstTokenMs?: number;
  thinking?: string;
}
```

#### 7.2 Contexto de Trace (AsyncLocalStorage)
**Nueva API opcional, la API simple sigue funcionando igual.**

```typescript
import { observe, withTrace } from '@lelemondev/sdk';

// API SIMPLE (sin cambios, sigue funcionando)
const client = observe(new Anthropic());
await client.messages.create({...}); // Trace independiente

// NUEVA API: Agrupar bajo un trace padre
await withTrace({ name: 'sales-agent', input: userMessage }, async () => {
  await client.messages.create({...}); // Span 1 bajo el trace
  await client.messages.create({...}); // Span 2 bajo el trace
  await client.messages.create({...}); // Span 3 bajo el trace
});
```

**Implementación:**
- [ ] `AsyncLocalStorage` para contexto implícito
- [ ] `withTrace(options, fn)` - crea trace padre y ejecuta fn
- [ ] Modificar `captureTrace` para usar contexto
- [ ] Propagar `traceId` y `parentSpanId` automáticamente
- [ ] Crear span raíz tipo `agent` automáticamente

#### 7.3 Spans Manuales para Tools y Retrieval
**Para operaciones no-LLM.**

```typescript
import { captureSpan } from '@lelemondev/sdk';

// Dentro de withTrace
const t0 = Date.now();
const docs = await pinecone.query({ vector, topK: 5 });
captureSpan({
  type: 'retrieval',
  name: 'pinecone-search',
  input: { query, topK: 5 },
  output: { count: docs.length },
  durationMs: Date.now() - t0,
});
```

**Tipos de span soportados:**
| Tipo | Icono | Uso |
|------|-------|-----|
| `agent` | 🤖 | Trace raíz / workflow |
| `llm` | 🔷 | Llamada a modelo LLM |
| `tool` | 🔧 | Ejecución de herramienta |
| `retrieval` | 🔍 | Búsqueda vectorial / RAG |
| `embedding` | 📊 | Generación de embeddings |
| `rerank` | 🎯 | Reranking de documentos |
| `guardrail` | 🛡️ | Validación de contenido |
| `custom` | 📋 | Cualquier operación |

#### 7.4 UI de Visualización (3 Columnas)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  TRACES                                                                  │
├─────────────────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌───────────────────────────┐ ┌───────────────────────┐ │
│ │ COL 1       │ │ COL 2                     │ │ COL 3                 │ │
│ │ TRACE LIST  │ │ TRACE TIMELINE            │ │ SPAN DETAIL           │ │
│ │             │ │                           │ │                       │ │
│ │ ┌─────────┐ │ │ ▼ 🤖 sales-agent          │ │ 🔷 GENERATION         │ │
│ │ │sales-ag │ │ │   ├─ 🔍 vector-search     │ │ claude-sonnet-4       │ │
│ │ │8.5s $0.02│ │ │   ├─ 🔷 intent-class     │ │ 2.8s | $0.0089        │ │
│ │ └─────────┘ │ │   ├─ 🔷 agent-response ◀──┼─┼───────────────────────│ │
│ │ ┌─────────┐ │ │   │   └─ 🔧 schedule_demo │ │ 💭 THINKING           │ │
│ │ │rag-quer │ │ │   └─ 🔷 final-response    │ │ "El usuario quiere    │ │
│ │ │2.1s $0.01│ │ │                           │ │  agendar para..."     │ │
│ │ └─────────┘ │ │ INPUT: "hola quisiera..." │ │                       │ │
│ │             │ │ OUTPUT: "¡Listo Antonio!" │ │ 🔧 TOOL USE           │ │
│ │             │ │                           │ │ schedule_demo({...})  │ │
│ └─────────────┘ └───────────────────────────┘ └───────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

- [ ] Layout 3 columnas responsive
- [ ] TraceList component (col 1)
- [ ] TraceTimeline component (col 2) - árbol jerárquico
- [ ] SpanDetail component (col 3)
- [ ] Renderizar `thinking` blocks
- [ ] Renderizar `tool_use` con input/output
- [ ] Timeline visual (barras de duración)
- [ ] Indicadores de error/status
- [ ] Mobile: tabs o drawer

---

## Estructura de Datos Objetivo

### Trace
```typescript
interface Trace {
  id: string;
  projectId: string;
  name?: string;
  sessionId?: string;
  userId?: string;

  input?: unknown;   // Input inicial del workflow
  output?: unknown;  // Output final del workflow

  status: 'active' | 'completed' | 'error';
  errorMessage?: string;

  metadata?: Record<string, unknown>;
  tags?: string[];

  // Métricas agregadas (calculadas de spans)
  metrics: {
    totalSpans: number;
    totalLLMCalls: number;
    totalToolCalls: number;
    totalTokens: number;
    totalInputTokens: number;
    totalOutputTokens: number;
    totalCostUsd: number;
    totalDurationMs: number;
  };

  createdAt: Date;
  endedAt?: Date;
}
```

### Span
```typescript
interface Span {
  id: string;
  traceId: string;
  parentSpanId?: string;  // Para jerarquía

  type: 'agent' | 'llm' | 'tool' | 'retrieval' | 'embedding' | 'guardrail' | 'custom';
  name: string;

  // LLM específico
  model?: string;
  provider?: string;

  input?: unknown;
  output?: unknown;

  // Tokens
  inputTokens?: number;
  outputTokens?: number;
  cacheReadTokens?: number;
  cacheWriteTokens?: number;
  reasoningTokens?: number;

  // Timing
  durationMs?: number;
  firstTokenMs?: number;

  // Status
  status: 'pending' | 'success' | 'error';
  stopReason?: string;
  errorMessage?: string;

  // Extras
  thinking?: string;
  toolCallId?: string;

  costUsd?: number;
  metadata?: Record<string, unknown>;

  startedAt: Date;
  endedAt?: Date;
}
```

---

## Ejemplos de Uso del SDK

### Caso 1: API Simple (sin cambios)
```typescript
import { init, observe } from '@lelemondev/sdk';
import Anthropic from '@anthropic-ai/sdk';

init({ apiKey: process.env.LELEMON_API_KEY });

const client = observe(new Anthropic(), {
  sessionId: 'conv_123',
  userId: 'user_456',
});

// Cada llamada = 1 trace + 1 span (igual que ahora)
await client.messages.create({
  model: 'claude-sonnet-4-20250514',
  messages: [{ role: 'user', content: 'Hola' }],
});
```

### Caso 2: Agente con Trace Padre
```typescript
import { init, observe, withTrace } from '@lelemondev/sdk';

init({ apiKey: process.env.LELEMON_API_KEY });
const client = observe(new Anthropic());

async function handleUserMessage(userMessage: string) {
  return withTrace({
    name: 'sales-agent',
    input: { message: userMessage },
    metadata: { channel: 'whatsapp' },
  }, async () => {
    // LLM 1: Clasificar intención
    await client.messages.create({ model: 'claude-3-5-haiku-20241022', ... });

    // LLM 2: Generar respuesta con tools
    const response = await client.messages.create({
      model: 'claude-sonnet-4-20250514',
      tools: [...],
      ...
    });

    // Si hay tool_use, ejecutar y continuar
    if (response.stop_reason === 'tool_use') {
      // Ejecutar tools (captureSpan automático o manual)
      // LLM 3: Respuesta final
      await client.messages.create({ ... });
    }

    return response;
  });
}
```

### Caso 3: RAG con Retrieval Manual
```typescript
async function ragQuery(question: string) {
  return withTrace({ name: 'rag-query', input: question }, async () => {
    // 1. Embedding (manual)
    const t0 = Date.now();
    const embedding = await openai.embeddings.create({ input: question, model: 'text-embedding-3-large' });
    captureSpan({
      type: 'embedding',
      name: 'query-embedding',
      input: { text: question },
      output: { dimensions: 3072 },
      durationMs: Date.now() - t0,
    });

    // 2. Retrieval (manual)
    const t1 = Date.now();
    const docs = await pinecone.query({ vector: embedding, topK: 5 });
    captureSpan({
      type: 'retrieval',
      name: 'pinecone-search',
      input: { topK: 5 },
      output: { count: docs.length },
      durationMs: Date.now() - t1,
    });

    // 3. LLM (automático)
    return client.messages.create({
      system: buildPromptWithDocs(docs),
      messages: [{ role: 'user', content: question }],
    });
  });
}
```

### Caso 4: Agente con Guardrails
```typescript
async function safeChat(userMessage: string) {
  return withTrace({ name: 'safe-chat' }, async () => {
    // 1. Input guardrail
    const t0 = Date.now();
    const inputCheck = await checkContent(userMessage);
    captureSpan({
      type: 'guardrail',
      name: 'input-safety',
      input: { content: userMessage },
      output: { passed: inputCheck.safe, violations: inputCheck.violations },
      status: inputCheck.safe ? 'success' : 'error',
      durationMs: Date.now() - t0,
    });

    if (!inputCheck.safe) throw new Error('Input blocked');

    // 2. LLM
    const response = await client.messages.create({ ... });

    // 3. Output guardrail
    const t1 = Date.now();
    const outputCheck = await checkContent(response.content);
    captureSpan({
      type: 'guardrail',
      name: 'output-safety',
      input: { content: response.content },
      output: { passed: outputCheck.safe },
      durationMs: Date.now() - t1,
    });

    return response;
  });
}
```

---

## Backlog

### Vercel AI SDK Integration
- [ ] `withLelemon(model)` wrapper
- [ ] Automatic token extraction
- [ ] Streaming support

### Framework Integrations
- [ ] LangChain callback handler
- [ ] LlamaIndex callback
- [ ] Haystack integration

### Future Considerations

#### GDPR & Compliance
- Self-hosted deployment covers most needs
- Formal compliance documentation (if needed for enterprise)

#### Pricing Tiers (Cloud)
- Free: Self-hosted
- Pro: ~$50/month (hosted, 1M spans)
- Enterprise: Custom pricing

#### Competitive Position
| Feature | Lelemon | Langfuse | Arize |
|---------|---------|----------|-------|
| Self-hosted | ✅ | ✅ | ❌ |
| RAM usage | ~50MB | ~500MB | N/A |
| Language | Go | TypeScript | Python |
| Price | Free | Free/$59 | $800+ |
| Hierarchical traces | ✅ (Phase 7) | ✅ | ✅ |
| Extended thinking | ✅ (Phase 7) | ❌ | ❌ |
