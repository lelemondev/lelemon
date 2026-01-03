/**
 * Transport layer for sending data to Lelemon API
 */

import type {
  CreateTraceRequest,
  CreateSpanRequest,
  UpdateTraceRequest,
} from './types';

const DEFAULT_ENDPOINT = 'https://lelemon.vercel.app';

interface TransportConfig {
  apiKey: string;
  endpoint: string;
  debug: boolean;
}

interface QueuedSpan {
  traceId: string;
  data: CreateSpanRequest;
}

export class Transport {
  private config: TransportConfig;
  private spanQueue: QueuedSpan[] = [];
  private flushTimer: NodeJS.Timeout | null = null;
  private batchSize: number;
  private flushInterval: number;

  constructor(
    config: TransportConfig,
    batchSize = 10,
    flushInterval = 1000
  ) {
    this.config = config;
    this.batchSize = batchSize;
    this.flushInterval = flushInterval;
  }

  /**
   * Create a new trace
   */
  async createTrace(data: CreateTraceRequest): Promise<{ id: string }> {
    const response = await this.request<{ id: string }>('POST', '/api/v1/traces', data);
    return response;
  }

  /**
   * Update a trace (e.g., complete it)
   */
  async updateTrace(traceId: string, data: UpdateTraceRequest): Promise<void> {
    await this.request('PATCH', `/v1/traces/${traceId}`, data);
  }

  /**
   * Queue a span for batch sending
   */
  queueSpan(traceId: string, data: CreateSpanRequest): void {
    this.spanQueue.push({ traceId, data });

    // Flush if batch size reached
    if (this.spanQueue.length >= this.batchSize) {
      this.flush();
    }

    // Start flush timer if not already running
    if (!this.flushTimer) {
      this.flushTimer = setTimeout(() => {
        this.flush();
      }, this.flushInterval);
    }
  }

  /**
   * Send a span immediately (bypassing queue)
   */
  async sendSpanImmediate(traceId: string, data: CreateSpanRequest): Promise<void> {
    await this.request('POST', `/v1/traces/${traceId}/spans`, data);
  }

  /**
   * Flush all queued spans
   */
  async flush(): Promise<void> {
    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
      this.flushTimer = null;
    }

    if (this.spanQueue.length === 0) {
      return;
    }

    const spans = [...this.spanQueue];
    this.spanQueue = [];

    // Group spans by trace for efficiency
    const byTrace = new Map<string, CreateSpanRequest[]>();
    for (const { traceId, data } of spans) {
      const existing = byTrace.get(traceId) || [];
      existing.push(data);
      byTrace.set(traceId, existing);
    }

    // Send spans for each trace
    const promises = Array.from(byTrace.entries()).map(
      async ([traceId, traceSpans]) => {
        for (const span of traceSpans) {
          try {
            await this.request('POST', `/v1/traces/${traceId}/spans`, span);
          } catch (error) {
            if (this.config.debug) {
              console.error('[Lelemon] Failed to send span:', error);
            }
          }
        }
      }
    );

    await Promise.all(promises);
  }

  /**
   * Make HTTP request to API
   */
  private async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.config.endpoint}${path}`;

    if (this.config.debug) {
      console.log(`[Lelemon] ${method} ${url}`, body);
    }

    const response = await fetch(url, {
      method,
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.config.apiKey}`,
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Lelemon API error: ${response.status} ${error}`);
    }

    // Handle empty responses
    const text = await response.text();
    if (!text) {
      return {} as T;
    }

    return JSON.parse(text);
  }
}
