/**
 * Lelemon API Client
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001';

export interface Trace {
  id: string;
  projectId: string;
  sessionId: string | null;
  userId: string | null;
  metadata: Record<string, unknown>;
  tags: string[] | null;
  totalTokens: number;
  totalCostUsd: string;
  totalDurationMs: number;
  totalSpans: number;
  status: 'active' | 'completed' | 'error';
  createdAt: string;
  updatedAt: string;
}

export interface Span {
  id: string;
  traceId: string;
  parentSpanId: string | null;
  type: 'llm' | 'tool' | 'retrieval' | 'custom';
  name: string;
  input: unknown;
  output: unknown;
  inputTokens: number | null;
  outputTokens: number | null;
  costUsd: string | null;
  durationMs: number | null;
  status: 'pending' | 'success' | 'error';
  errorMessage: string | null;
  model: string | null;
  provider: string | null;
  metadata: Record<string, unknown>;
  startedAt: string;
  endedAt: string | null;
}

export interface TraceWithSpans extends Trace {
  spans: Span[];
}

export interface AnalyticsSummary {
  totalTraces: number;
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: number;
  avgDurationMs: number;
  errorRate: number;
}

export interface UsageByDay {
  date: string;
  traces: number;
  spans: number;
  tokens: number;
  costUsd: number;
}

class LelemonAPI {
  private apiKey: string;

  constructor(apiKey: string) {
    this.apiKey = apiKey;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const response = await fetch(`${API_URL}${path}`, {
      method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${this.apiKey}`,
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`API Error: ${response.status} ${error}`);
    }

    return response.json();
  }

  // Traces
  async getTraces(params?: {
    sessionId?: string;
    userId?: string;
    status?: string;
    tags?: string[];
    from?: string;
    to?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ data: Trace[]; total: number }> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          if (Array.isArray(value)) {
            value.forEach(v => searchParams.append(key, v));
          } else {
            searchParams.set(key, String(value));
          }
        }
      });
    }
    return this.request('GET', `/v1/traces?${searchParams.toString()}`);
  }

  async getTrace(id: string): Promise<TraceWithSpans> {
    return this.request('GET', `/v1/traces/${id}`);
  }

  // Analytics
  async getSummary(params?: { from?: string; to?: string }): Promise<AnalyticsSummary> {
    const searchParams = new URLSearchParams();
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    return this.request('GET', `/v1/analytics/summary?${searchParams.toString()}`);
  }

  async getUsage(params?: {
    from?: string;
    to?: string;
    groupBy?: 'day' | 'hour';
  }): Promise<UsageByDay[]> {
    const searchParams = new URLSearchParams();
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    if (params?.groupBy) searchParams.set('groupBy', params.groupBy);
    return this.request('GET', `/v1/analytics/usage?${searchParams.toString()}`);
  }

  // Project
  async getProject(): Promise<{
    id: string;
    name: string;
    settings: Record<string, unknown>;
    createdAt: string;
  }> {
    return this.request('GET', '/v1/projects/me');
  }

  async updateProject(data: {
    name?: string;
    settings?: Record<string, unknown>;
  }): Promise<void> {
    return this.request('PATCH', '/v1/projects/me', data);
  }

  async rotateApiKey(): Promise<{ apiKey: string }> {
    return this.request('POST', '/v1/projects/api-key');
  }
}

export function createAPI(apiKey: string): LelemonAPI {
  return new LelemonAPI(apiKey);
}
