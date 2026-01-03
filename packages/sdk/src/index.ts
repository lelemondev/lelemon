/**
 * Lelemon SDK
 * Low-friction LLM observability
 *
 * @example
 * import { trace } from '@lelemon/sdk';
 *
 * const t = trace({ input: userMessage });
 * try {
 *   const messages = [...];
 *   // ... your agent code ...
 *   await t.success(messages);
 * } catch (error) {
 *   await t.error(error, messages);
 * }
 */

// Main API
export { trace, init, Trace } from './tracer';

// Types
export type {
  LelemonConfig,
  TraceOptions,
  Message,
  OpenAIMessage,
  AnthropicMessage,
  ParsedTrace,
  ParsedLLMCall,
  ParsedToolCall,
} from './types';

// Parser (for advanced usage)
export { parseMessages, parseResponse, parseBedrockResponse } from './parser';
