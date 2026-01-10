# Plan de Resiliencia para Dependencias Externas

> Documento de planificación para reducir la fragilidad del sistema ante cambios en proveedores LLM, APIs externas y otras dependencias críticas.

**Fecha:** Enero 2026
**Estado:** Borrador
**Autor:** Equipo Lelemon

---

## Tabla de Contenidos

1. [Resumen Ejecutivo](#resumen-ejecutivo)
2. [Análisis de Dependencias Actuales](#análisis-de-dependencias-actuales)
3. [Problema 1: Pricing de Modelos](#problema-1-pricing-de-modelos)
4. [Problema 2: Parsing de Respuestas](#problema-2-parsing-de-respuestas)
5. [Problema 3: Detección de Proveedores](#problema-3-detección-de-proveedores)
6. [Problema 4: Otras Dependencias](#problema-4-otras-dependencias)
7. [Arquitectura Propuesta](#arquitectura-propuesta)
8. [Plan de Implementación](#plan-de-implementación)
9. [Métricas de Éxito](#métricas-de-éxito)
10. [Riesgos y Mitigaciones](#riesgos-y-mitigaciones)

---

## Resumen Ejecutivo

### Problema

El sistema actual tiene dependencias hardcodeadas que requieren intervención manual cuando:

1. **Proveedores lanzan nuevos modelos** → Debemos actualizar `pricing.go` manualmente
2. **Proveedores cambian formatos de respuesta** → El parser falla silenciosamente
3. **Proveedores cambian estructura de SDKs** → La detección automática falla
4. **Precios cambian** → Los costos calculados son incorrectos

### Impacto

| Escenario | Frecuencia | Impacto |
|-----------|------------|---------|
| Nuevo modelo sin pricing | Semanal | Costo = $0 (datos incorrectos) |
| Cambio de formato API | Trimestral | Tokens no extraídos |
| SDK actualizado | Mensual | Provider = "unknown" |
| Cambio de precios | Mensual | Costos incorrectos ±20% |

### Solución Propuesta

Implementar un sistema de **configuración dinámica** con:

1. **Pricing Service**: Sincronización automática desde fuentes externas
2. **Parser Registry**: Parsers versionados con auto-detección mejorada
3. **Provider Detection**: Detección por múltiples señales con fallbacks
4. **Observabilidad**: Alertas proactivas para casos no manejados

---

## Análisis de Dependencias Actuales

### Mapa de Dependencias

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LELEMON PLATFORM                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                   │
│  │   SDK        │    │   Backend    │    │   Dashboard  │                   │
│  │ TypeScript   │───▶│   Go API     │◀───│   Next.js    │                   │
│  └──────────────┘    └──────────────┘    └──────────────┘                   │
│         │                   │                                                │
│         ▼                   ▼                                                │
│  ┌──────────────┐    ┌──────────────┐                                       │
│  │  Provider    │    │   Pricing    │                                       │
│  │  Detection   │    │   Table      │                                       │
│  │  (runtime)   │    │  (hardcoded) │                                       │
│  └──────────────┘    └──────────────┘                                       │
│         │                   │                                                │
└─────────┼───────────────────┼────────────────────────────────────────────────┘
          │                   │
          ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DEPENDENCIAS EXTERNAS                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │
│  │   OpenAI     │ │  Anthropic   │ │   Google     │ │  AWS Bedrock │        │
│  │   - SDK      │ │   - SDK      │ │   - SDK      │ │   - SDK      │        │
│  │   - API fmt  │ │   - API fmt  │ │   - API fmt  │ │   - API fmt  │        │
│  │   - Pricing  │ │   - Pricing  │ │   - Pricing  │ │   - Pricing  │        │
│  │   - Models   │ │   - Models   │ │   - Models   │ │   - Models   │        │
│  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘        │
│                                                                              │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                         │
│  │  OpenRouter  │ │   Mistral    │ │   Cohere     │  ... más proveedores    │
│  └──────────────┘ └──────────────┘ └──────────────┘                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Matriz de Riesgo

| Dependencia | Archivo Actual | Volatilidad | Impacto si Falla | Prioridad |
|-------------|----------------|-------------|------------------|-----------|
| Pricing de modelos | `pricing.go` | Alta (semanal) | Medio (datos incorrectos) | **P1** |
| Parser de respuestas | `parser.go` | Media (trimestral) | Alto (pérdida de datos) | **P1** |
| Detección de provider | SDK `parser.ts` | Media (mensual) | Medio (provider unknown) | **P2** |
| OAuth Google | `auth/google.go` | Baja (anual) | Alto (login roto) | **P3** |
| Lemon Squeezy | `ee/lemonsqueezy/` | Baja (anual) | Alto (billing roto) | **P3** |

---

## Problema 1: Pricing de Modelos

### Estado Actual

```go
// apps/server/pkg/domain/service/pricing.go
var pricing = map[string]ModelPricing{
    "gpt-4o":           {Input: 0.0025, Output: 0.01},
    "claude-3-5-sonnet": {Input: 0.003, Output: 0.015},
    // ... 80+ modelos hardcodeados
}
```

**Problemas:**
1. Actualización manual requerida
2. Sin alertas cuando hay modelos desconocidos
3. Precios desactualizados = métricas incorrectas
4. No escala con nuevos proveedores

### Fuentes de Datos Disponibles

| Fuente | URL/API | Cobertura | Actualización | Licencia |
|--------|---------|-----------|---------------|----------|
| **LiteLLM** | `github.com/BerriAI/litellm/.../model_prices_and_context_window.json` | 300+ modelos | Diaria | MIT |
| **OpenRouter** | `openrouter.ai/api/v1/models` | 200+ modelos | Real-time | API pública |
| **Anthropic** | No tiene API pública de pricing | Solo Claude | N/A | - |
| **OpenAI** | No tiene API pública de pricing | Solo OpenAI | N/A | - |
| **Scraping** | Páginas de pricing | Todos | Manual | Riesgoso |

### Solución Propuesta: Pricing Service

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PRICING SERVICE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         Cache Layer                                  │    │
│  │                    (In-memory + Redis)                               │    │
│  │                      TTL: 24 horas                                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│                    ┌───────────────┼───────────────┐                        │
│                    ▼               ▼               ▼                        │
│  ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐            │
│  │  LiteLLM JSON    │ │  OpenRouter API  │ │  Local Fallback  │            │
│  │  (Primario)      │ │  (Secundario)    │ │  (pricing.go)    │            │
│  │                  │ │                  │ │                  │            │
│  │  - 300+ modelos  │ │  - Real-time     │ │  - Siempre       │            │
│  │  - Gratis        │ │  - Rate limited  │ │    disponible    │            │
│  │  - GitHub raw    │ │  - API key req   │ │  - Fallback      │            │
│  └──────────────────┘ └──────────────────┘ └──────────────────┘            │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Observabilidad                                  │    │
│  │  - Métrica: modelos sin pricing                                      │    │
│  │  - Alerta: >10 requests con modelo desconocido                      │    │
│  │  - Log: cada modelo nuevo detectado                                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Diseño Técnico

#### Nueva Estructura de Archivos

```
apps/server/pkg/
├── domain/
│   └── service/
│       ├── pricing.go          # Interfaz + fallback local
│       └── pricing_test.go
├── infrastructure/
│   └── pricing/
│       ├── provider.go         # Interfaz PricingProvider
│       ├── litellm.go          # Implementación LiteLLM
│       ├── openrouter.go       # Implementación OpenRouter
│       ├── cache.go            # Cache en memoria
│       └── sync.go             # Sincronización periódica
```

#### Interfaz Principal

```go
// pkg/infrastructure/pricing/provider.go
package pricing

import "context"

// ModelPrice representa el pricing de un modelo
type ModelPrice struct {
    Model           string  `json:"model"`
    Provider        string  `json:"provider"`
    InputPerMToken  float64 `json:"input_per_m_token"`   // Precio por millón de tokens
    OutputPerMToken float64 `json:"output_per_m_token"`
    ContextWindow   int     `json:"context_window,omitempty"`
    UpdatedAt       string  `json:"updated_at"`
    Source          string  `json:"source"` // "litellm", "openrouter", "local"
}

// PricingProvider es la interfaz para obtener pricing
type PricingProvider interface {
    GetPrice(ctx context.Context, model string) (*ModelPrice, error)
    GetAllPrices(ctx context.Context) (map[string]ModelPrice, error)
    Refresh(ctx context.Context) error
}

// PricingService orquesta múltiples providers con fallback
type PricingService struct {
    providers []PricingProvider
    cache     *Cache
    metrics   *Metrics
}

func (s *PricingService) GetPrice(ctx context.Context, model string) (*ModelPrice, error) {
    // 1. Buscar en cache
    if price, ok := s.cache.Get(model); ok {
        return price, nil
    }

    // 2. Intentar cada provider en orden
    for _, provider := range s.providers {
        price, err := provider.GetPrice(ctx, model)
        if err == nil && price != nil {
            s.cache.Set(model, price)
            return price, nil
        }
    }

    // 3. Registrar modelo desconocido
    s.metrics.RecordUnknownModel(model)

    // 4. Retornar precio cero (transparente para el usuario)
    return &ModelPrice{
        Model:           model,
        InputPerMToken:  0,
        OutputPerMToken: 0,
        Source:          "unknown",
    }, nil
}
```

#### Implementación LiteLLM

```go
// pkg/infrastructure/pricing/litellm.go
package pricing

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
)

const liteLLMURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

type LiteLLMProvider struct {
    client    *http.Client
    prices    map[string]ModelPrice
    updatedAt time.Time
}

// LiteLLM JSON format
type liteLLMEntry struct {
    InputCostPerToken  float64 `json:"input_cost_per_token"`
    OutputCostPerToken float64 `json:"output_cost_per_token"`
    MaxTokens          int     `json:"max_tokens"`
    MaxInputTokens     int     `json:"max_input_tokens"`
    MaxOutputTokens    int     `json:"max_output_tokens"`
    Mode               string  `json:"mode"`
}

func (p *LiteLLMProvider) Refresh(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", liteLLMURL, nil)
    if err != nil {
        return err
    }

    resp, err := p.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var data map[string]liteLLMEntry
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return err
    }

    prices := make(map[string]ModelPrice, len(data))
    for model, entry := range data {
        prices[model] = ModelPrice{
            Model:           model,
            InputPerMToken:  entry.InputCostPerToken * 1_000_000,
            OutputPerMToken: entry.OutputCostPerToken * 1_000_000,
            ContextWindow:   entry.MaxTokens,
            Source:          "litellm",
            UpdatedAt:       time.Now().Format(time.RFC3339),
        }
    }

    p.prices = prices
    p.updatedAt = time.Now()
    return nil
}
```

#### Sincronización Automática

```go
// pkg/infrastructure/pricing/sync.go
package pricing

import (
    "context"
    "log/slog"
    "time"
)

type SyncService struct {
    service  *PricingService
    interval time.Duration
    stop     chan struct{}
}

func NewSyncService(service *PricingService, interval time.Duration) *SyncService {
    return &SyncService{
        service:  service,
        interval: interval,
        stop:     make(chan struct{}),
    }
}

func (s *SyncService) Start() {
    go func() {
        ticker := time.NewTicker(s.interval)
        defer ticker.Stop()

        // Sync inicial
        s.sync()

        for {
            select {
            case <-ticker.C:
                s.sync()
            case <-s.stop:
                return
            }
        }
    }()
}

func (s *SyncService) sync() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := s.service.Refresh(ctx); err != nil {
        slog.Error("failed to sync pricing", "error", err)
    } else {
        slog.Info("pricing synced successfully", "models", s.service.ModelCount())
    }
}
```

---

## Problema 2: Parsing de Respuestas

### Estado Actual

```go
// apps/server/pkg/domain/service/parser.go
func ParseProviderResponse(provider string, rawResponse any) *ParsedResponse {
    switch strings.ToLower(provider) {
    case "anthropic":
        return parseAnthropicResponse(rawResponse)
    case "bedrock":
        return parseBedrockResponse(rawResponse)
    case "openai", "openrouter":
        return parseOpenAIResponse(rawResponse)
    case "gemini":
        return parseGeminiResponse(rawResponse)
    default:
        return parseAutoDetect(rawResponse)
    }
}
```

**Problemas:**
1. Si un proveedor cambia su formato, el parser falla silenciosamente
2. No hay versionado de parsers
3. Auto-detección puede fallar con formatos nuevos
4. Sin tests con respuestas reales actualizadas

### Solución Propuesta: Parser Registry

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PARSER REGISTRY                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Parser Selection                                │    │
│  │                                                                      │    │
│  │   1. Explicit provider → Use registered parser                       │    │
│  │   2. Unknown provider → Try auto-detection                           │    │
│  │   3. Auto-detect fails → Use generic parser (extract minimum)        │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│         ┌──────────────────────────┼──────────────────────────┐             │
│         ▼                          ▼                          ▼             │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐        │
│  │  Anthropic   │         │   OpenAI     │         │   Generic    │        │
│  │  Parser v2   │         │  Parser v1   │         │   Parser     │        │
│  │              │         │              │         │              │        │
│  │ - Messages   │         │ - Chat       │         │ - Best       │        │
│  │ - Streaming  │         │ - Responses  │         │   effort     │        │
│  │ - Tool use   │         │ - Tools      │         │ - Fallback   │        │
│  └──────────────┘         └──────────────┘         └──────────────┘        │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Validation Layer                                │    │
│  │                                                                      │    │
│  │   - Verificar que se extrajeron tokens                              │    │
│  │   - Verificar que hay output                                         │    │
│  │   - Log warning si faltan campos esperados                          │    │
│  │   - Métricas de parsing success/failure                             │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Diseño Técnico

#### Nueva Estructura

```
apps/server/pkg/domain/service/
├── parser/
│   ├── registry.go       # Registry de parsers
│   ├── types.go          # Interfaces y tipos
│   ├── anthropic.go      # Parser Anthropic
│   ├── openai.go         # Parser OpenAI
│   ├── bedrock.go        # Parser Bedrock
│   ├── gemini.go         # Parser Gemini
│   ├── generic.go        # Parser genérico (fallback)
│   ├── detector.go       # Auto-detección de formato
│   └── validator.go      # Validación de resultados
```

#### Interfaz de Parser

```go
// pkg/domain/service/parser/types.go
package parser

// Parser es la interfaz que todos los parsers deben implementar
type Parser interface {
    // Name retorna el nombre del parser (para logs/métricas)
    Name() string

    // CanParse verifica si este parser puede manejar la respuesta
    CanParse(raw any) bool

    // Parse extrae datos estructurados de la respuesta
    Parse(raw any) (*ParsedResponse, error)

    // Version retorna la versión del parser
    Version() string
}

// ParsedResponse contiene los datos extraídos
type ParsedResponse struct {
    Output           any
    InputTokens      int
    OutputTokens     int
    CacheReadTokens  *int
    CacheWriteTokens *int
    ReasoningTokens  *int
    StopReason       *string
    Thinking         *string
    ToolUses         []ToolUse
    SubType          *string

    // Metadata de parsing
    ParserUsed       string
    ParserVersion    string
    FieldsExtracted  []string  // Para debugging
    FieldsMissing    []string  // Para alertas
}
```

#### Registry con Fallback

```go
// pkg/domain/service/parser/registry.go
package parser

import (
    "log/slog"
)

type Registry struct {
    parsers  map[string]Parser
    detector *Detector
    generic  Parser
    metrics  *Metrics
}

func NewRegistry() *Registry {
    r := &Registry{
        parsers:  make(map[string]Parser),
        detector: NewDetector(),
        generic:  NewGenericParser(),
    }

    // Registrar parsers conocidos
    r.Register("anthropic", NewAnthropicParser())
    r.Register("openai", NewOpenAIParser())
    r.Register("openrouter", NewOpenAIParser())  // Mismo formato
    r.Register("bedrock", NewBedrockParser())
    r.Register("gemini", NewGeminiParser())

    return r
}

func (r *Registry) Parse(provider string, raw any) *ParsedResponse {
    // 1. Intentar parser explícito
    if parser, ok := r.parsers[provider]; ok {
        result, err := parser.Parse(raw)
        if err == nil && result != nil {
            r.metrics.RecordSuccess(provider, parser.Name())
            return result
        }
        slog.Warn("explicit parser failed", "provider", provider, "error", err)
    }

    // 2. Intentar auto-detección
    if detected := r.detector.Detect(raw); detected != nil {
        result, err := detected.Parse(raw)
        if err == nil && result != nil {
            r.metrics.RecordSuccess("auto", detected.Name())
            return result
        }
    }

    // 3. Usar parser genérico
    result, _ := r.generic.Parse(raw)
    if result != nil {
        r.metrics.RecordFallback(provider)
        slog.Warn("using generic parser", "provider", provider, "fieldsExtracted", result.FieldsExtracted)
    } else {
        r.metrics.RecordFailure(provider)
        slog.Error("all parsers failed", "provider", provider)
    }

    return result
}
```

#### Parser Genérico (Fallback)

```go
// pkg/domain/service/parser/generic.go
package parser

// GenericParser intenta extraer campos comunes de cualquier respuesta
type GenericParser struct{}

func (p *GenericParser) Parse(raw any) (*ParsedResponse, error) {
    resp, ok := raw.(map[string]any)
    if !ok {
        return nil, ErrInvalidFormat
    }

    result := &ParsedResponse{
        ParserUsed:    "generic",
        ParserVersion: "1.0",
    }

    // Intentar extraer usage de múltiples formatos conocidos
    result.InputTokens, result.OutputTokens = p.extractTokens(resp)

    // Intentar extraer output
    result.Output = p.extractOutput(resp)

    // Registrar qué se encontró y qué falta
    if result.InputTokens > 0 {
        result.FieldsExtracted = append(result.FieldsExtracted, "inputTokens")
    } else {
        result.FieldsMissing = append(result.FieldsMissing, "inputTokens")
    }

    return result, nil
}

func (p *GenericParser) extractTokens(resp map[string]any) (input, output int) {
    // Intentar formatos conocidos en orden de probabilidad
    tokenPaths := []struct {
        usageKey string
        inputKey string
        outputKey string
    }{
        {"usage", "input_tokens", "output_tokens"},       // Anthropic
        {"usage", "prompt_tokens", "completion_tokens"},  // OpenAI
        {"usageMetadata", "promptTokenCount", "candidatesTokenCount"}, // Gemini
    }

    for _, path := range tokenPaths {
        if usage, ok := resp[path.usageKey].(map[string]any); ok {
            if v, ok := usage[path.inputKey].(float64); ok {
                input = int(v)
            }
            if v, ok := usage[path.outputKey].(float64); ok {
                output = int(v)
            }
            if input > 0 || output > 0 {
                return
            }
        }
    }

    return 0, 0
}
```

### Tests con Fixtures Reales

```go
// pkg/domain/service/parser/anthropic_test.go
package parser

import (
    "encoding/json"
    "os"
    "testing"
)

// fixtures/anthropic_messages_v1.json - Respuesta real capturada
func TestAnthropicParser_RealResponse(t *testing.T) {
    // Cargar fixture
    data, err := os.ReadFile("testdata/fixtures/anthropic_messages_v1.json")
    if err != nil {
        t.Skip("fixture not available")
    }

    var raw map[string]any
    json.Unmarshal(data, &raw)

    parser := NewAnthropicParser()
    result, err := parser.Parse(raw)

    if err != nil {
        t.Fatalf("parse failed: %v", err)
    }

    // Verificar campos críticos
    if result.InputTokens == 0 {
        t.Error("expected input tokens > 0")
    }
    if result.OutputTokens == 0 {
        t.Error("expected output tokens > 0")
    }
    if result.Output == nil {
        t.Error("expected output to be set")
    }
}
```

---

## Problema 3: Detección de Proveedores

### Estado Actual (SDK)

```typescript
// lelemondev-sdk/src/providers/base.ts
function detectProvider(client: any): string {
    const name = client.constructor?.name || '';

    if (name.includes('Anthropic')) return 'anthropic';
    if (name.includes('OpenAI')) return 'openai';
    if (name.includes('Bedrock')) return 'bedrock';
    // ...

    return 'unknown';
}
```

**Problemas:**
1. Depende del nombre del constructor (puede cambiar)
2. Nuevos SDKs no son detectados
3. Wrappers/proxies pueden ocultar el nombre real

### Solución Propuesta: Detección Multi-Señal

```typescript
// Múltiples señales para detectar el provider
interface ProviderSignal {
    check: (client: any) => boolean;
    confidence: number;  // 0-100
    provider: string;
}

const signals: ProviderSignal[] = [
    // Constructor name (alta confianza)
    {
        check: (c) => c.constructor?.name?.includes('Anthropic'),
        confidence: 90,
        provider: 'anthropic'
    },

    // Métodos característicos (media confianza)
    {
        check: (c) => typeof c.messages?.create === 'function',
        confidence: 70,
        provider: 'anthropic'
    },

    // Propiedades características (media confianza)
    {
        check: (c) => c._options?.baseURL?.includes('anthropic'),
        confidence: 80,
        provider: 'anthropic'
    },

    // SDK package name via prototype chain
    {
        check: (c) => c[Symbol.toStringTag] === 'Anthropic',
        confidence: 95,
        provider: 'anthropic'
    },
];

function detectProvider(client: any): { provider: string; confidence: number } {
    const scores: Record<string, number> = {};

    for (const signal of signals) {
        try {
            if (signal.check(client)) {
                scores[signal.provider] = (scores[signal.provider] || 0) + signal.confidence;
            }
        } catch {
            // Ignorar errores de acceso
        }
    }

    const best = Object.entries(scores)
        .sort(([,a], [,b]) => b - a)[0];

    if (best && best[1] > 50) {
        return { provider: best[0], confidence: best[1] };
    }

    return { provider: 'unknown', confidence: 0 };
}
```

---

## Problema 4: Otras Dependencias

### OAuth (Google)

| Aspecto | Estado Actual | Riesgo | Mitigación |
|---------|---------------|--------|------------|
| Client ID/Secret | En env vars | Bajo | Rotación anual |
| Token refresh | Implementado | Bajo | Ya manejado |
| Fallback | Email/password | Bajo | Siempre disponible |

**Acción:** Bajo riesgo, mantener como está.

### Lemon Squeezy (Billing)

| Aspecto | Estado Actual | Riesgo | Mitigación |
|---------|---------------|--------|------------|
| Webhooks | Procesados sync | Medio | Hacer idempotentes |
| API calls | Sin retry | Medio | Agregar retry con backoff |
| Estado local | No sincronizado | Alto | Sync periódico |

**Acciones recomendadas:**

```go
// 1. Webhooks idempotentes
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    event := parseWebhook(r)

    // Verificar si ya procesamos este evento
    if h.store.WebhookProcessed(event.ID) {
        w.WriteHeader(http.StatusOK)
        return
    }

    // Procesar y marcar como procesado
    if err := h.processEvent(event); err != nil {
        // Retornar 500 para que LS reintente
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    h.store.MarkWebhookProcessed(event.ID)
    w.WriteHeader(http.StatusOK)
}

// 2. Sync periódico
func (s *BillingService) SyncSubscriptions(ctx context.Context) error {
    // Obtener subscripciones activas de LS
    subs, err := s.lsClient.ListSubscriptions(ctx)
    if err != nil {
        return err
    }

    // Actualizar estado local
    for _, sub := range subs {
        s.store.UpsertSubscription(ctx, sub)
    }

    return nil
}
```

---

## Arquitectura Propuesta

### Vista General

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LELEMON v2                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         SDK (TypeScript)                             │    │
│  ├─────────────────────────────────────────────────────────────────────┤    │
│  │  Provider Detection                                                  │    │
│  │  ├── Multi-signal detection                                          │    │
│  │  ├── Confidence scoring                                              │    │
│  │  └── Fallback to "unknown"                                           │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                     │                                        │
│                                     ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         Backend (Go)                                 │    │
│  ├─────────────────────────────────────────────────────────────────────┤    │
│  │                                                                      │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │    │
│  │  │   Pricing    │  │    Parser    │  │   Config     │              │    │
│  │  │   Service    │  │   Registry   │  │   Service    │              │    │
│  │  │              │  │              │  │              │              │    │
│  │  │ - LiteLLM    │  │ - Versioned  │  │ - Feature    │              │    │
│  │  │ - OpenRouter │  │ - Auto-detect│  │   flags      │              │    │
│  │  │ - Fallback   │  │ - Fallback   │  │ - Runtime    │              │    │
│  │  │ - Cache      │  │ - Validation │  │   config     │              │    │
│  │  └──────────────┘  └──────────────┘  └──────────────┘              │    │
│  │                                                                      │    │
│  │  ┌─────────────────────────────────────────────────────────────┐    │    │
│  │  │                    Observability                             │    │    │
│  │  │  - Unknown models counter                                    │    │    │
│  │  │  - Parser failure rate                                       │    │    │
│  │  │  - Provider detection confidence                             │    │    │
│  │  │  - Alerting (Slack/email)                                    │    │    │
│  │  └─────────────────────────────────────────────────────────────┘    │    │
│  │                                                                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Endpoints Administrativos

```
POST /admin/pricing/refresh     # Forzar sync de pricing
GET  /admin/pricing/unknown     # Listar modelos sin precio
GET  /admin/pricing/stats       # Estadísticas de pricing
POST /admin/pricing/override    # Override manual de precio

GET  /admin/parser/stats        # Estadísticas de parsing
GET  /admin/parser/failures     # Últimos fallos de parsing

GET  /admin/health/dependencies # Estado de dependencias externas
```

---

## Plan de Implementación

### Fase 1: Observabilidad (1-2 días)

**Objetivo:** Visibilidad del problema actual antes de resolver.

| Tarea | Descripción | Archivos |
|-------|-------------|----------|
| 1.1 | Agregar logging de modelos sin pricing | `pricing.go` |
| 1.2 | Agregar métricas de parsing failures | `parser.go` |
| 1.3 | Crear endpoint `/admin/pricing/unknown` | `handler/admin.go` |
| 1.4 | Dashboard de modelos desconocidos (opcional) | `apps/web` |

**Entregable:** Saber exactamente qué modelos faltan y con qué frecuencia.

### Fase 2: Pricing Service (3-5 días)

**Objetivo:** Automatizar actualización de precios.

| Tarea | Descripción | Archivos |
|-------|-------------|----------|
| 2.1 | Crear interfaz `PricingProvider` | `infrastructure/pricing/` |
| 2.2 | Implementar provider LiteLLM | `litellm.go` |
| 2.3 | Implementar cache en memoria | `cache.go` |
| 2.4 | Implementar sync service | `sync.go` |
| 2.5 | Migrar `pricing.go` a usar nuevo servicio | `domain/service/` |
| 2.6 | Tests unitarios y de integración | `*_test.go` |
| 2.7 | Endpoint refresh manual | `handler/admin.go` |

**Entregable:** Pricing auto-actualizado cada 24h con fallback.

### Fase 3: Parser Mejorado (2-3 días)

**Objetivo:** Parsers más robustos con mejor fallback.

| Tarea | Descripción | Archivos |
|-------|-------------|----------|
| 3.1 | Refactorizar a parser registry | `parser/registry.go` |
| 3.2 | Agregar parser genérico (fallback) | `parser/generic.go` |
| 3.3 | Agregar validación de resultados | `parser/validator.go` |
| 3.4 | Capturar fixtures de respuestas reales | `testdata/fixtures/` |
| 3.5 | Tests con fixtures | `*_test.go` |

**Entregable:** Parsers que nunca fallan completamente.

### Fase 4: SDK Mejoras (2-3 días)

**Objetivo:** Detección de providers más robusta.

| Tarea | Descripción | Archivos |
|-------|-------------|----------|
| 4.1 | Implementar detección multi-señal | `providers/detector.ts` |
| 4.2 | Agregar confidence scoring | `providers/detector.ts` |
| 4.3 | Tests con múltiples versiones de SDKs | `tests/` |
| 4.4 | Documentar cómo agregar nuevos providers | `docs/` |

**Entregable:** Detección confiable incluso con SDKs nuevos.

### Fase 5: Alertas y Monitoreo (1-2 días)

**Objetivo:** Notificaciones proactivas de problemas.

| Tarea | Descripción | Archivos |
|-------|-------------|----------|
| 5.1 | Configurar alertas para modelos desconocidos | `infrastructure/alerts/` |
| 5.2 | Alertas para parsing failures | `infrastructure/alerts/` |
| 5.3 | Dashboard de dependencias (opcional) | `apps/web` |

**Entregable:** Saber inmediatamente cuando algo falla.

---

## Métricas de Éxito

### KPIs

| Métrica | Actual | Objetivo | Cómo Medir |
|---------|--------|----------|------------|
| Modelos con pricing correcto | ~80% | >98% | `unknown_model_count / total_models` |
| Tiempo para agregar nuevo modelo | Manual (días) | Automático (24h) | Tiempo hasta que aparece en pricing |
| Parsing success rate | ~95% | >99% | `parse_success / parse_attempts` |
| Provider detection accuracy | ~90% | >98% | `correct_provider / total_detections` |

### Alertas

| Alerta | Condición | Acción |
|--------|-----------|--------|
| `pricing_unknown_model` | >10 requests con modelo desconocido en 1h | Revisar LiteLLM, agregar manual si falta |
| `pricing_sync_failed` | Sync falla 3 veces consecutivas | Revisar conectividad, usar fallback |
| `parser_failure_spike` | >5% failure rate en 15min | Revisar logs, posible cambio de API |

---

## Riesgos y Mitigaciones

| Riesgo | Probabilidad | Impacto | Mitigación |
|--------|--------------|---------|------------|
| LiteLLM discontinúa el JSON | Baja | Alto | OpenRouter como backup, fallback local |
| Rate limit de GitHub raw | Media | Bajo | Cache 24h, retry con backoff |
| Formato de LiteLLM cambia | Baja | Medio | Tests automatizados, alertas |
| Nuevo proveedor no soportado | Alta | Bajo | Parser genérico extrae mínimo |
| Cache se corrompe | Baja | Medio | Fallback a fuente original |

---

## Apéndice A: Estructura de LiteLLM JSON

```json
{
  "gpt-4o": {
    "max_tokens": 16384,
    "max_input_tokens": 128000,
    "max_output_tokens": 16384,
    "input_cost_per_token": 0.0000025,
    "output_cost_per_token": 0.00001,
    "litellm_provider": "openai",
    "mode": "chat",
    "supports_function_calling": true,
    "supports_parallel_function_calling": true,
    "supports_vision": true
  },
  "claude-3-5-sonnet-20241022": {
    "max_tokens": 8096,
    "max_input_tokens": 200000,
    "max_output_tokens": 8096,
    "input_cost_per_token": 0.000003,
    "output_cost_per_token": 0.000015,
    "cache_creation_input_token_cost": 0.00000375,
    "cache_read_input_token_cost": 0.0000003,
    "litellm_provider": "anthropic",
    "mode": "chat",
    "supports_function_calling": true,
    "supports_vision": true
  }
}
```

---

## Apéndice B: Checklist de Nuevo Proveedor

Cuando se agregue un nuevo proveedor LLM:

- [ ] **SDK**: Agregar señales de detección en `detector.ts`
- [ ] **Backend**: Crear parser en `parser/[provider].go`
- [ ] **Backend**: Registrar en `parser/registry.go`
- [ ] **Tests**: Capturar fixture de respuesta real
- [ ] **Tests**: Agregar tests con fixture
- [ ] **Pricing**: Verificar que LiteLLM lo incluye (sino, agregar manual)
- [ ] **Docs**: Actualizar lista de proveedores soportados

---

## Historial de Cambios

| Fecha | Versión | Cambios |
|-------|---------|---------|
| 2026-01 | 1.0 | Documento inicial |
