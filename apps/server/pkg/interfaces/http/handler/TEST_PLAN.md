# Plan de Tests Comprensivo para Ingest

Este documento describe todos los tests necesarios para validar el flujo completo de ingesta de datos.

---

## 1. Tests de Preservación de Campos (Field Preservation)

### 1.1 Campos de Identificación
| Campo | Input (SDK) | Output (DB) | Validación |
|-------|-------------|-------------|------------|
| traceId | `"trace-123"` | Trace.ID | Exacto |
| spanId | `"span-456"` | Span.ID | Exacto |
| parentSpanId | `"parent-789"` | Span.ParentSpanID | Exacto |

### 1.2 Campos de Span
| Campo | Input (SDK) | Output (DB) | Validación |
|-------|-------------|-------------|------------|
| spanType | `"llm"` | Span.Type | Mapeo correcto |
| provider | `"anthropic"` | Span.Provider | Exacto |
| model | `"claude-3-5-sonnet"` | Span.Model | Exacto |
| name | `"my-call"` | Span.Name | Exacto |
| input | `{messages: [...]}` | Span.Input | JSON preservado |
| output | `{content: [...]}` | Span.Output | JSON preservado |
| durationMs | `1500` | Span.DurationMs | Exacto |
| status | `"success"` | Span.Status | Mapeo correcto |
| errorMessage | `"Rate limit"` | Span.ErrorMessage | Exacto |

### 1.3 Campos de Contexto
| Campo | Input (SDK) | Output (DB) | Validación |
|-------|-------------|-------------|------------|
| sessionId | `"session-abc"` | Trace.SessionID | Exacto |
| userId | `"user-xyz"` | Trace.UserID | Exacto |
| metadata | `{custom: "data"}` | Span.Metadata | JSON preservado |
| tags | `["prod", "v2"]` | Trace.Tags | Array preservado |

---

## 2. Tests de Tokens

### 2.1 Tokens Básicos
```
Test: inputTokens y outputTokens se preservan
Input: { inputTokens: 100, outputTokens: 50 }
Expected: Span.InputTokens = 100, Span.OutputTokens = 50
```

### 2.2 Tokens de Cache (Anthropic/Bedrock)
```
Test: cacheReadTokens y cacheWriteTokens se preservan
Input: { cacheReadTokens: 500, cacheWriteTokens: 200 }
Expected: Span.CacheReadTokens = 500, Span.CacheWriteTokens = 200
```

### 2.3 Tokens de Razonamiento (o1, Claude thinking)
```
Test: reasoningTokens se preserva
Input: { reasoningTokens: 1000 }
Expected: Span.ReasoningTokens = 1000
```

### 2.4 Agregación de Tokens en Trace
```
Test: TotalTokens = sum(inputTokens + outputTokens) de todos los spans
Input:
  - Span1: inputTokens=100, outputTokens=50
  - Span2: inputTokens=200, outputTokens=100
Expected: Trace.TotalTokens = 450
```

---

## 3. Tests de Pricing

### 3.1 Cálculo de Costo por Modelo
| Modelo | Input/1K | Output/1K | Test |
|--------|----------|-----------|------|
| gpt-4o | $0.0025 | $0.01 | 1000 in + 500 out = $0.0075 |
| gpt-4o-mini | $0.00015 | $0.0006 | 1000 in + 500 out = $0.00045 |
| claude-3-5-sonnet | $0.003 | $0.015 | 1000 in + 500 out = $0.0105 |
| claude-3-haiku | $0.00025 | $0.00125 | 1000 in + 500 out = $0.000875 |

### 3.2 Agregación de Costo en Trace
```
Test: TotalCostUSD = sum(costUsd) de todos los spans
Input:
  - Span1 (gpt-4o): 1000 in, 500 out → $0.0075
  - Span2 (claude-3-haiku): 2000 in, 1000 out → $0.00175
Expected: Trace.TotalCostUSD = $0.00925
```

### 3.3 Modelo Desconocido
```
Test: Modelo no reconocido usa pricing por defecto
Input: { model: "unknown-model-xyz", inputTokens: 1000, outputTokens: 500 }
Expected: costUsd calculado con precios default
```

---

## 4. Tests por Tipo de Span

### 4.1 LLM Span
```go
{
  spanType: "llm",
  provider: "openai",
  model: "gpt-4o",
  input: [{role: "user", content: "Hello"}],
  output: [{role: "assistant", content: "Hi!"}],
  inputTokens: 10,
  outputTokens: 5,
}
```
Validar: Type="llm", Model set, Provider set, tokens, cost calculated

### 4.2 Tool Span
```go
{
  spanType: "tool",
  name: "search_products",
  input: {query: "widgets"},
  output: {results: ["A", "B"]},
  durationMs: 200,
}
```
Validar: Type="tool", Name set, Input/Output preserved, no tokens/cost

### 4.3 Retrieval Span
```go
{
  spanType: "retrieval",
  name: "vector_search",
  input: {query: "user question", topK: 5},
  output: {documents: [{id: "doc1", score: 0.95}]},
  durationMs: 150,
}
```
Validar: Type="retrieval", documents in output

### 4.4 Agent Span
```go
{
  spanType: "agent",
  name: "sales-agent",
  input: {message: "Hello"},
  output: "Hi there!",
  durationMs: 5000,
}
```
Validar: Type="agent", sets Trace.Name

### 4.5 Embedding Span
```go
{
  spanType: "embedding",
  provider: "openai",
  model: "text-embedding-3-small",
  input: ["text1", "text2"],
  output: [[0.1, 0.2, ...], [0.3, 0.4, ...]],
  inputTokens: 50,
}
```
Validar: Type="embedding", inputTokens only

### 4.6 Guardrail Span
```go
{
  spanType: "guardrail",
  name: "content-filter",
  input: {text: "user message"},
  output: {passed: true, flags: []},
  durationMs: 50,
}
```
Validar: Type="guardrail"

### 4.7 Rerank Span
```go
{
  spanType: "rerank",
  name: "cohere-rerank",
  input: {query: "q", documents: ["a", "b", "c"]},
  output: {rankings: [{index: 1, score: 0.9}]},
}
```
Validar: Type="rerank"

### 4.8 Custom Span
```go
{
  spanType: "custom",
  name: "my-custom-operation",
  input: {anything: "here"},
  output: {result: "done"},
}
```
Validar: Type="custom"

---

## 5. Tests de Tool Tracking

### 5.1 Extracción de ToolUses (Formato Anthropic)
```go
Input output:
[
  {type: "text", text: "Let me search..."},
  {type: "tool_use", id: "toolu_123", name: "search", input: {q: "test"}}
]

Expected:
- Span.SubType = "planning"
- Span.ToolUses = [{ID: "toolu_123", Name: "search", Input: {q: "test"}}]
```

### 5.2 Extracción de ToolUses (Formato Bedrock)
```go
Input output:
[
  {text: "Let me search..."},
  {toolUse: {toolUseId: "tool_456", name: "search", input: {q: "test"}}}
]

Expected:
- Span.SubType = "planning"
- Span.ToolUses = [{ID: "tool_456", Name: "search", Input: {q: "test"}}]
```

### 5.3 SubType Detection
```
Sin tool_use → SubType = "response"
Con tool_use → SubType = "planning"
```

---

## 6. Tests de Metadata y Tags

### 6.1 Metadata Preservada
```go
Input: metadata: {
  custom_field: "value",
  nested: {a: 1, b: 2},
  array: [1, 2, 3]
}
Expected: Span.Metadata contiene todos los campos
```

### 6.2 Tags en Trace
```go
Input: tags: ["production", "v2", "experiment-a"]
Expected: Trace.Tags = ["production", "v2", "experiment-a"]
```

### 6.3 _traceName en Metadata
```go
Input: metadata: {_traceName: "my-custom-trace"}
Expected: Trace.Name = "my-custom-trace" (si no hay agent span)
```

### 6.4 Metadata Especial (streaming, toolCallId)
```go
Input: {streaming: true, toolCallId: "call_123"}
Expected: Span.Metadata contiene streaming=true, toolCallId="call_123"
```

---

## 7. Tests de Jerarquía Compleja

### 7.1 Trace con 1 Span (Flat)
```
Trace
└── LLM Span (root)

Validar: TotalSpans=1, span sin parent
```

### 7.2 Trace con Múltiples Spans Planos
```
Trace
├── LLM Span 1
├── LLM Span 2
└── LLM Span 3

Validar: TotalSpans=3, todos sin parent
```

### 7.3 Jerarquía de 2 Niveles
```
Trace
└── Agent Span (root)
    ├── LLM Span 1
    └── LLM Span 2

Validar: Agent sin parent, LLMs con parent=Agent
```

### 7.4 Jerarquía de 3+ Niveles
```
Trace
└── Agent Span
    └── LLM Span (planning)
        └── Tool Span
            └── Nested LLM Span

Validar: Cada nivel tiene parent correcto
```

### 7.5 Múltiples Ramas
```
Trace
└── Agent Span
    ├── LLM Span 1
    │   └── Tool Span A
    ├── LLM Span 2
    │   ├── Tool Span B
    │   └── Tool Span C
    └── LLM Span 3

Validar: Todas las relaciones parent-child correctas
```

---

## 8. Tests de Agregaciones

### 8.1 TotalSpans
```
Trace con 5 spans → TotalSpans = 5
```

### 8.2 TotalTokens
```
Span1: input=100, output=50
Span2: input=200, output=100
Span3: (tool, no tokens)
TotalTokens = 100+50+200+100 = 450
```

### 8.3 TotalCostUSD
```
Span1 (gpt-4o): $0.01
Span2 (claude): $0.02
Span3 (tool): $0
TotalCostUSD = $0.03
```

### 8.4 TotalDurationMs
```
¿Es max o sum? Verificar implementación y testear
```

---

## 9. Tests de Estados

### 9.1 Span Success
```go
Input: status: "success"
Expected: Span.Status = "success"
```

### 9.2 Span Error
```go
Input: status: "error", errorMessage: "Rate limit exceeded"
Expected: Span.Status = "error", Span.ErrorMessage = "Rate limit exceeded"
```

### 9.3 Propagación a Trace
```
Todos spans success → Trace.Status = "completed"
Algún span error → Trace.Status = "error"
```

---

## 10. Tests de RawResponse Parsing

### 10.1 OpenAI Response
```json
{
  "id": "chatcmpl-123",
  "choices": [{
    "message": {"role": "assistant", "content": "Hello!"},
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 5,
    "total_tokens": 15
  }
}
```
Expected: output extraído, tokens extraídos, stopReason="stop"

### 10.2 Anthropic Response
```json
{
  "id": "msg_123",
  "content": [{"type": "text", "text": "Hello!"}],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 5,
    "cache_read_input_tokens": 100,
    "cache_creation_input_tokens": 50
  }
}
```
Expected: content extraído, tokens extraídos, cache tokens extraídos

### 10.3 Bedrock Converse Response
```json
{
  "output": {
    "message": {
      "role": "assistant",
      "content": [{"text": "Hello!"}]
    }
  },
  "stopReason": "end_turn",
  "usage": {
    "inputTokens": 10,
    "outputTokens": 5
  }
}
```
Expected: message.content extraído, tokens extraídos

### 10.4 Bedrock InvokeModel Response
```json
{
  "content": [{"type": "text", "text": "Hello!"}],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 5
  }
}
```
Expected: content extraído (formato Anthropic dentro de Bedrock)

### 10.5 Google Gemini Response
```json
{
  "candidates": [{
    "content": {"parts": [{"text": "Hello!"}]},
    "finishReason": "STOP"
  }],
  "usageMetadata": {
    "promptTokenCount": 10,
    "candidatesTokenCount": 5
  }
}
```
Expected: parts[0].text extraído, tokens extraídos

---

## 11. Tests de Extended Thinking (Claude)

### 11.1 Thinking Content
```go
Input: thinking: "Let me analyze this step by step..."
Expected: Span.Thinking = "Let me analyze..."
```

### 11.2 Thinking desde RawResponse
```json
{
  "content": [
    {"type": "thinking", "thinking": "Analysis..."},
    {"type": "text", "text": "Here's my answer"}
  ]
}
```
Expected: Thinking extraído, texto principal en output

---

## 12. Tests de Timing

### 12.1 FirstTokenMs (TTFT)
```go
Input: firstTokenMs: 250
Expected: Span.FirstTokenMs = 250
```

### 12.2 Timestamp Personalizado
```go
Input: timestamp: "2024-01-15T10:30:00Z"
Expected: Span.StartedAt = timestamp proporcionado
```

---

## 13. Tests de Casos Edge

### 13.1 Campos Null/Undefined
```go
Input: { spanType: "llm", status: "success" }
// Sin model, provider, tokens, etc.
Expected: Campos opcionales son null, no error
```

### 13.2 Input/Output Muy Grande
```go
Input con 100KB de datos
Expected: Se almacena correctamente, no truncado
```

### 13.3 Caracteres Especiales en Strings
```go
Input: { name: "test'with\"special\nchars" }
Expected: Se preserva exactamente
```

### 13.4 Metadata con Tipos Mixtos
```go
Input: metadata: {
  string: "text",
  number: 123,
  float: 1.5,
  bool: true,
  null: null,
  array: [1, "two", 3],
  nested: {a: {b: {c: 1}}}
}
Expected: Todos los tipos preservados correctamente
```

---

## Implementación

### Archivos de Test a Crear:
1. `contract_test.go` - Tests de contrato (IDs preservados) ✅ DONE
2. `tokens_test.go` - Tests de tokens y agregaciones
3. `pricing_test.go` - Tests de cálculo de costos
4. `span_types_test.go` - Tests de cada tipo de span
5. `hierarchy_test.go` - Tests de jerarquías complejas
6. `rawresponse_test.go` - Tests de parsing de providers
7. `edge_cases_test.go` - Tests de casos edge

### Prioridad:
1. **Alta**: Tokens, Pricing, Hierarchy (más propensos a bugs)
2. **Media**: Span Types, RawResponse parsing
3. **Normal**: Edge cases, Extended fields
