/**
 * Lelemon Tracer - Main SDK class
 */

import { Transport } from './transport';
import type {
  LelemonConfig,
  TraceOptions,
  SpanOptions,
  SpanEndOptions,
  TraceStatus,
  SpanStatus,
  CreateSpanRequest,
} from './types';

const DEFAULT_ENDPOINT = 'https://api.lelemon.dev';

/**
 * Span instance for recording operations
 */
export class Span {
  readonly id: string;
  private traceId: string;
  private transport: Transport;
  private options: SpanOptions;
  private startTime: number;
  private ended = false;
  private debug: boolean;

  constructor(
    id: string,
    traceId: string,
    transport: Transport,
    options: SpanOptions,
    debug: boolean
  ) {
    this.id = id;
    this.traceId = traceId;
    this.transport = transport;
    this.options = options;
    this.startTime = Date.now();
    this.debug = debug;
  }

  /**
   * Create a child span
   */
  startSpan(options: SpanOptions): Span {
    const id = crypto.randomUUID();
    return new Span(id, this.traceId, this.transport, options, this.debug);
  }

  /**
   * End the span and record results
   */
  end(options: SpanEndOptions = {}): void {
    if (this.ended) {
      if (this.debug) {
        console.warn('[Lelemon] Span already ended');
      }
      return;
    }
    this.ended = true;

    const durationMs = options.durationMs ?? Date.now() - this.startTime;
    const now = new Date().toISOString();

    const spanData: CreateSpanRequest = {
      type: this.options.type,
      name: this.options.name,
      input: this.options.input,
      output: options.output,
      inputTokens: options.inputTokens,
      outputTokens: options.outputTokens,
      durationMs,
      status: options.status ?? 'success',
      errorMessage: options.errorMessage,
      model: options.model,
      provider: options.provider,
      metadata: this.options.metadata,
      startedAt: new Date(this.startTime).toISOString(),
      endedAt: now,
    };

    // Queue the span for batch sending
    this.transport.queueSpan(this.traceId, spanData);
  }

  /**
   * Mark span as error
   */
  setError(error: Error): void {
    this.end({
      status: 'error',
      errorMessage: error.message,
    });
  }
}

/**
 * Trace instance for grouping related spans
 */
export class Trace {
  readonly id: string;
  private transport: Transport;
  private options: TraceOptions;
  private debug: boolean;
  private ended = false;

  constructor(
    id: string,
    transport: Transport,
    options: TraceOptions,
    debug: boolean
  ) {
    this.id = id;
    this.transport = transport;
    this.options = options;
    this.debug = debug;
  }

  /**
   * Start a new span in this trace
   */
  startSpan(options: SpanOptions): Span {
    const id = crypto.randomUUID();
    return new Span(id, this.id, this.transport, options, this.debug);
  }

  /**
   * Update trace metadata
   */
  setMetadata(key: string, value: unknown): void {
    this.options.metadata = {
      ...this.options.metadata,
      [key]: value,
    };
  }

  /**
   * Add a tag to the trace
   */
  addTag(tag: string): void {
    this.options.tags = [...(this.options.tags || []), tag];
  }

  /**
   * End the trace
   */
  async end(options: { status?: TraceStatus } = {}): Promise<void> {
    if (this.ended) {
      if (this.debug) {
        console.warn('[Lelemon] Trace already ended');
      }
      return;
    }
    this.ended = true;

    // Flush pending spans first
    await this.transport.flush();

    // Update trace status
    await this.transport.updateTrace(this.id, {
      status: options.status ?? 'completed',
      metadata: this.options.metadata,
      tags: this.options.tags,
    });
  }
}

/**
 * Main Lelemon tracer class
 */
export class LLMTracer {
  private transport: Transport;
  private config: LelemonConfig;
  private disabled: boolean;

  constructor(config: LelemonConfig = {}) {
    const apiKey = config.apiKey ?? process.env.LELEMON_API_KEY;

    if (!apiKey && !config.disabled) {
      console.warn(
        '[Lelemon] No API key provided. Set apiKey in config or LELEMON_API_KEY env var. Tracing disabled.'
      );
    }

    this.config = config;
    this.disabled = config.disabled ?? !apiKey;

    this.transport = new Transport(
      {
        apiKey: apiKey ?? '',
        endpoint: config.endpoint ?? DEFAULT_ENDPOINT,
        debug: config.debug ?? false,
      },
      config.batchSize,
      config.flushInterval
    );
  }

  /**
   * Start a new trace
   */
  async startTrace(options: TraceOptions = {}): Promise<Trace> {
    if (this.disabled) {
      // Return a no-op trace
      return new Trace('disabled', this.transport, options, false);
    }

    const result = await this.transport.createTrace({
      sessionId: options.sessionId,
      userId: options.userId,
      metadata: options.metadata,
      tags: options.tags,
    });

    return new Trace(
      result.id,
      this.transport,
      options,
      this.config.debug ?? false
    );
  }

  /**
   * Flush all pending spans
   * Call this before shutting down to ensure all data is sent
   */
  async flush(): Promise<void> {
    await this.transport.flush();
  }

  /**
   * Check if tracing is enabled
   */
  isEnabled(): boolean {
    return !this.disabled;
  }
}
