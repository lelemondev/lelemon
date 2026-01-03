/**
 * Transport layer for sending data to Lelemon API
 */

import type { CreateTraceRequest, CompleteTraceRequest } from './types';

interface TransportConfig {
  apiKey: string;
  endpoint: string;
  debug: boolean;
  disabled: boolean;
}

export class Transport {
  private config: TransportConfig;

  constructor(config: TransportConfig) {
    this.config = config;
  }

  /**
   * Check if transport is enabled
   */
  isEnabled(): boolean {
    return !this.config.disabled && !!this.config.apiKey;
  }

  /**
   * Create a new trace
   */
  async createTrace(data: CreateTraceRequest): Promise<{ id: string }> {
    return this.request<{ id: string }>('POST', '/api/v1/traces', data);
  }

  /**
   * Complete a trace (success or error)
   */
  async completeTrace(traceId: string, data: CompleteTraceRequest): Promise<void> {
    await this.request('PATCH', `/api/v1/traces/${traceId}`, data);
  }

  /**
   * Make HTTP request to API
   */
  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    if (this.config.disabled) {
      return {} as T;
    }

    const url = `${this.config.endpoint}${path}`;

    if (this.config.debug) {
      console.log(`[Lelemon] ${method} ${url}`, body);
    }

    try {
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
    } catch (error) {
      if (this.config.debug) {
        console.error('[Lelemon] Request failed:', error);
      }
      throw error;
    }
  }
}
