/**
 * Lelemon SDK - LLM Observability
 *
 * @example
 * ```typescript
 * import { LLMTracer } from '@lelemon/sdk';
 *
 * const tracer = new LLMTracer({
 *   apiKey: process.env.LELEMON_API_KEY
 * });
 *
 * // Manual tracing
 * const trace = await tracer.startTrace({
 *   sessionId: 'conv-123',
 *   userId: 'user-456'
 * });
 *
 * const span = trace.startSpan({
 *   type: 'llm',
 *   name: 'chat-completion',
 *   input: { messages }
 * });
 *
 * const response = await openai.chat.completions.create({ ... });
 *
 * span.end({
 *   output: response,
 *   model: response.model,
 *   inputTokens: response.usage.prompt_tokens,
 *   outputTokens: response.usage.completion_tokens
 * });
 *
 * await trace.end();
 * ```
 */

export { LLMTracer, Trace, Span } from './tracer';

export type {
  LelemonConfig,
  TraceOptions,
  SpanOptions,
  SpanEndOptions,
  TraceStatus,
  SpanType,
  SpanStatus,
} from './types';

// Re-export for convenience
export { Transport } from './transport';
