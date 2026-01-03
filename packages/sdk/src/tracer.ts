/**
 * Lelemon Tracer - Simple, low-friction API
 *
 * Usage:
 *   const t = trace({ input: userMessage });
 *   try {
 *     // your agent code
 *     t.success(messages);
 *   } catch (error) {
 *     t.error(error, messages);
 *   }
 */

import { Transport } from './transport';
import { parseMessages, parseResponse } from './parser';
import type {
  LelemonConfig,
  TraceOptions,
  ParsedLLMCall,
  CompleteTraceRequest,
} from './types';

const DEFAULT_ENDPOINT = 'https://api.lelemon.dev';

// Global config (set via init())
let globalConfig: LelemonConfig = {};
let globalTransport: Transport | null = null;

/**
 * Initialize the SDK globally (optional)
 * If not called, trace() will auto-initialize with env vars
 */
export function init(config: LelemonConfig = {}): void {
  globalConfig = config;
  globalTransport = createTransport(config);
}

/**
 * Create a transport instance
 */
function createTransport(config: LelemonConfig): Transport {
  const apiKey = config.apiKey ?? getEnvVar('LELEMON_API_KEY');

  if (!apiKey && !config.disabled) {
    console.warn(
      '[Lelemon] No API key provided. Set apiKey in config or LELEMON_API_KEY env var. Tracing disabled.'
    );
  }

  return new Transport({
    apiKey: apiKey ?? '',
    endpoint: config.endpoint ?? DEFAULT_ENDPOINT,
    debug: config.debug ?? false,
    disabled: config.disabled ?? !apiKey,
  });
}

/**
 * Get transport (create if needed)
 */
function getTransport(): Transport {
  if (!globalTransport) {
    globalTransport = createTransport(globalConfig);
  }
  return globalTransport;
}

/**
 * Get environment variable (works in Node and edge)
 */
function getEnvVar(name: string): string | undefined {
  if (typeof process !== 'undefined' && process.env) {
    return process.env[name];
  }
  return undefined;
}

/**
 * Active trace handle returned by trace()
 */
export class Trace {
  private id: string | null = null;
  private transport: Transport;
  private options: TraceOptions;
  private startTime: number;
  private completed = false;
  private debug: boolean;
  private disabled: boolean;
  private llmCalls: ParsedLLMCall[] = [];

  constructor(options: TraceOptions, transport: Transport, debug: boolean, disabled: boolean) {
    this.options = options;
    this.transport = transport;
    this.startTime = Date.now();
    this.debug = debug;
    this.disabled = disabled;
  }

  /**
   * Initialize trace on server (called internally)
   */
  async init(): Promise<void> {
    if (this.disabled) return;

    try {
      const result = await this.transport.createTrace({
        name: this.options.name,
        sessionId: this.options.sessionId,
        userId: this.options.userId,
        input: this.options.input,
        metadata: this.options.metadata,
        tags: this.options.tags,
      });
      this.id = result.id;
    } catch (error) {
      if (this.debug) {
        console.error('[Lelemon] Failed to create trace:', error);
      }
    }
  }

  /**
   * Log an LLM response (optional - for tracking individual calls)
   * Use this if you want to track tokens per call, not just at the end
   */
  log(response: unknown): void {
    const parsed = parseResponse(response);
    if (parsed.model || parsed.inputTokens || parsed.outputTokens) {
      this.llmCalls.push(parsed);
    }
  }

  /**
   * Complete trace successfully
   * @param messages - The full message history (OpenAI/Anthropic format)
   */
  async success(messages: unknown): Promise<void> {
    if (this.completed) return;
    this.completed = true;

    if (this.disabled || !this.id) return;

    const durationMs = Date.now() - this.startTime;
    const parsed = parseMessages(messages);

    // Merge logged LLM calls with parsed ones
    const allLLMCalls = [...this.llmCalls, ...parsed.llmCalls];

    // Calculate totals
    let totalInputTokens = 0;
    let totalOutputTokens = 0;
    const models = new Set<string>();

    for (const call of allLLMCalls) {
      if (call.inputTokens) totalInputTokens += call.inputTokens;
      if (call.outputTokens) totalOutputTokens += call.outputTokens;
      if (call.model) models.add(call.model);
    }

    try {
      await this.transport.completeTrace(this.id, {
        status: 'completed',
        output: parsed.output,
        systemPrompt: parsed.systemPrompt,
        llmCalls: allLLMCalls,
        toolCalls: parsed.toolCalls,
        models: Array.from(models),
        totalInputTokens,
        totalOutputTokens,
        durationMs,
      });
    } catch (err) {
      if (this.debug) {
        console.error('[Lelemon] Failed to complete trace:', err);
      }
    }
  }

  /**
   * Complete trace with error
   * @param error - The error that occurred
   * @param messages - The message history up to the failure (optional)
   */
  async error(error: Error | unknown, messages?: unknown): Promise<void> {
    if (this.completed) return;
    this.completed = true;

    if (this.disabled || !this.id) return;

    const durationMs = Date.now() - this.startTime;
    const parsed = messages ? parseMessages(messages) : null;

    const errorObj = error instanceof Error ? error : new Error(String(error));

    // Merge logged LLM calls
    const allLLMCalls = parsed
      ? [...this.llmCalls, ...parsed.llmCalls]
      : this.llmCalls;

    // Calculate totals
    let totalInputTokens = 0;
    let totalOutputTokens = 0;
    const models = new Set<string>();

    for (const call of allLLMCalls) {
      if (call.inputTokens) totalInputTokens += call.inputTokens;
      if (call.outputTokens) totalOutputTokens += call.outputTokens;
      if (call.model) models.add(call.model);
    }

    const request: CompleteTraceRequest = {
      status: 'error',
      errorMessage: errorObj.message,
      errorStack: errorObj.stack,
      durationMs,
      totalInputTokens,
      totalOutputTokens,
      models: Array.from(models),
    };

    if (parsed) {
      request.output = parsed.output;
      request.systemPrompt = parsed.systemPrompt;
      request.llmCalls = allLLMCalls;
      request.toolCalls = parsed.toolCalls;
    }

    try {
      await this.transport.completeTrace(this.id, request);
    } catch (err) {
      if (this.debug) {
        console.error('[Lelemon] Failed to complete trace:', err);
      }
    }
  }
}

/**
 * Start a new trace
 *
 * @example
 * const t = trace({ input: userMessage });
 * try {
 *   const messages = [...];
 *   // ... your agent code ...
 *   await t.success(messages);
 * } catch (error) {
 *   await t.error(error, messages);
 *   throw error;
 * }
 */
export function trace(options: TraceOptions): Trace {
  const transport = getTransport();
  const debug = globalConfig.debug ?? false;
  const disabled = globalConfig.disabled ?? !transport.isEnabled();

  const t = new Trace(options, transport, debug, disabled);

  // Initialize async (fire and forget)
  t.init().catch((err) => {
    if (debug) {
      console.error('[Lelemon] Trace init failed:', err);
    }
  });

  return t;
}

// Re-export for backwards compatibility
export { Trace as LLMTracer };
