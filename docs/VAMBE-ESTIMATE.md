# Vambe.ai - Estimación de Capacidad

> Cliente potencial: [Vambe.ai](https://www.vambe.ai/)
> Fecha: 2026-01-09
> Contacto: Por definir

## Perfil del Cliente

**Vambe** es una plataforma de agentes IA conversacionales para ventas B2C.

| Métrica | Valor |
|---------|-------|
| Empresas usando plataforma | 1,700+ |
| Agentes por empresa | 1-7 (promedio ~3) |
| Canales | WhatsApp, Instagram, Webchat |
| Crecimiento mensual | 17% |
| Financiamiento | Serie A $14M USD |
| Inversores | Monashees, Cathay Latam, Atlantico |
| Board | Simón Borrero (CEO Rappi) |

### Caso de Referencia: Global66
- 60,000 conversaciones/mes
- 4x más contactabilidad
- Tiempo de respuesta ~0

---

## Cálculo de Spans

### Por Conversación

```
Conversación típica (10 mensajes promedio)
├── Usuario escribe → Agente procesa
│   ├── Span: LLM call (clasificación intent)
│   ├── Span: Tool call (consulta CRM)
│   └── Span: LLM call (genera respuesta)
├── ... (más interacciones)
└── Cierre
    ├── Span: Tool call (crear cita/pedido)
    └── Span: LLM call (confirmar)
```

**Estimado: 20-50 spans por conversación**
**Promedio usado: 30 spans/conversación**

### Escenarios de Volumen

| Escenario | Conv/empresa/día | Conv/día total | Spans/día | Spans/mes |
|-----------|------------------|----------------|-----------|-----------|
| Conservador | 10 | 17,000 | 510,000 | **15.3M** |
| Moderado | 30 | 51,000 | 1,530,000 | **45.9M** |
| Activo | 50 | 85,000 | 2,550,000 | **76.5M** |
| Agresivo | 100 | 170,000 | 5,100,000 | **153M** |

### Proyección con Crecimiento (17% mensual)

| Mes | Empresas | Spans/mes (moderado) |
|-----|----------|----------------------|
| 1 | 1,700 | 46M |
| 3 | 2,300 | 62M |
| 6 | 3,600 | 97M |
| 12 | 8,900 | 240M |

---

## Requisitos de Infraestructura

### Opción A: PostgreSQL (Solo MVP/Piloto)

**Capacidad máxima recomendada:** ~5-10M spans/mes

```
Droplet 8GB RAM / 4 vCPU ($48/mes)
├── Go Backend
└── PostgreSQL (optimizado)
    ├── Particionado por mes
    ├── BRIN indexes
    └── Retención 90 días
```

**Limitaciones:**
- Queries analíticas lentas >10M registros
- Sin compresión nativa
- Scaling vertical únicamente

### Opción B: PostgreSQL + ClickHouse (Recomendada)

**Capacidad:** 100M+ spans/mes

```
┌─────────────────────────────────────────────────────┐
│              Arquitectura Híbrida                   │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Droplet 4GB ($24/mes)                             │
│  ├── Go Backend                                    │
│  └── PostgreSQL (users, projects, config)          │
│                                                     │
│  Droplet 16GB ($96/mes) o ClickHouse Cloud         │
│  └── ClickHouse                                    │
│      ├── traces (particionado por día)            │
│      ├── spans (particionado por día)             │
│      └── Compresión ~10x                          │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**Storage estimado (con compresión ClickHouse):**
- 50M spans/mes × 500 bytes promedio = 25GB raw
- Con compresión 10x = **2.5GB/mes**
- 1 año = **30GB**

### Opción C: ClickHouse Cloud (Managed)

| Tier | Incluido | Costo estimado |
|------|----------|----------------|
| Development | 10GB storage | $0-50/mes |
| Scale | Pay-per-use | $100-300/mes |
| Enterprise | SLA, support | Custom |

---

## Costos Estimados

### Escenario Conservador (15M spans/mes)

| Componente | Opción PG | Opción Híbrida |
|------------|-----------|----------------|
| Compute | $48 | $24 + $96 = $120 |
| Storage | Incluido | ~$10/mes |
| Backups | $5 | $10 |
| **Total** | **$53/mes** | **$140/mes** |

### Escenario Activo (75M spans/mes)

| Componente | Opción PG | Opción Híbrida |
|------------|-----------|----------------|
| Compute | ❌ No viable | $48 + $192 = $240 |
| Storage | - | ~$30/mes |
| **Total** | - | **~$280/mes** |

---

## Pricing Sugerido para Vambe

### Modelo por Spans

| Tier | Spans incluidos | Precio/mes | $/1M spans extra |
|------|-----------------|------------|------------------|
| Starter | 10M | $99 | $15 |
| Growth | 50M | $299 | $10 |
| Scale | 200M | $799 | $5 |
| Enterprise | Unlimited | Custom | - |

### Modelo por Empresa (más simple)

| Tier | Empresas | Precio/mes |
|------|----------|------------|
| Growth | Hasta 500 | $199 |
| Scale | Hasta 2,000 | $499 |
| Enterprise | Unlimited | $999+ |

---

## Roadmap Técnico para Vambe

### Fase 1: Piloto (Semana 1-2)
- [ ] Deploy con PostgreSQL optimizado
- [ ] 50-100 empresas de prueba
- [ ] Monitorear volumen real de spans
- [ ] Validar estimaciones

### Fase 2: Producción (Semana 3-4)
- [ ] Migrar a arquitectura híbrida si necesario
- [ ] Implementar ClickHouse para analytics
- [ ] Setup alertas y monitoring
- [ ] Data retention policy

### Fase 3: Escala (Mes 2+)
- [ ] Optimizar según patrones de uso reales
- [ ] Evaluar ClickHouse Cloud vs self-hosted
- [ ] Implementar dashboard personalizado si necesario

---

## Preguntas para Vambe

1. ¿Cuántas conversaciones promedio tienen por empresa activa?
2. ¿Cuántos tool calls/API calls hace un agente típico por conversación?
3. ¿Necesitan retención histórica? ¿Cuántos meses?
4. ¿Tienen requisitos de compliance (SOC2, GDPR)?
5. ¿Prefieren self-hosted o managed?
6. ¿Tienen equipo DevOps interno?

---

## Competencia / Alternativas

| Solución | Pricing | Notas |
|----------|---------|-------|
| Langfuse | $0-500+/mes | Open source, similar |
| Datadog APM | ~$35/host + $0.10/span | Muy caro a escala |
| New Relic | ~$0.30/GB ingest | Costoso |
| Helicone | $0-500/mes | Específico para LLM |
| **Lelemon** | $99-999/mes | Nuestro pricing |

**Ventaja competitiva:**
- Open source (pueden self-host si quieren)
- Pricing predecible
- Especializado en LLM/Agentes
- Soporte en español/LATAM

---

## Contacto Interno

- Owner del deal: [Por definir]
- Technical lead: [Por definir]
- Fecha próximo seguimiento: [Por definir]
