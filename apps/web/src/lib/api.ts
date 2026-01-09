/**
 * Lelemon API Client - Connects to Go Backend
 */

// API URL: use env var if set, otherwise relative (proxied by Next.js in dev)
const API_URL = process.env.NEXT_PUBLIC_API_URL || '';

// Helper to get auth token
function getAuthToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('lelemon_token');
}

// Normalized interfaces (camelCase for frontend)
export interface Trace {
  id: string;
  projectId: string;
  sessionId: string | null;
  userId: string | null;
  metadata: Record<string, unknown>;
  tags: string[] | null;
  totalTokens: number;
  totalCostUsd: number;
  totalDurationMs: number;
  totalSpans: number;
  status: 'active' | 'completed' | 'error';
  createdAt: string;
  updatedAt: string;
}

export type SpanType = 'llm' | 'agent' | 'tool' | 'retrieval' | 'embedding' | 'guardrail' | 'rerank' | 'custom';

export interface Span {
  id: string;
  traceId: string;
  parentSpanId: string | null;
  type: SpanType;
  name: string;
  input: unknown;
  output: unknown;
  inputTokens: number | null;
  outputTokens: number | null;
  costUsd: number | null;
  durationMs: number | null;
  status: 'pending' | 'success' | 'error';
  errorMessage: string | null;
  model: string | null;
  provider: string | null;
  metadata: Record<string, unknown>;
  startedAt: string;
  endedAt: string | null;
  // Extended fields (Phase 7.1)
  stopReason: string | null;
  cacheReadTokens: number | null;
  cacheWriteTokens: number | null;
  reasoningTokens: number | null;
  firstTokenMs: number | null;
  thinking: string | null;
}

export interface TraceWithSpans extends Trace {
  spans: Span[];
}

// Optimized trace detail response from backend
export interface ToolUse {
  id: string;
  name: string;
  input: unknown;
  output: unknown;
  status: 'success' | 'error' | 'pending';
  durationMs: number | null;
}

export interface ProcessedSpan extends Span {
  subType?: 'planning' | 'response';
  toolUses?: ToolUse[];
  userInput?: string;
  isToolUse?: boolean;
  toolUseData?: ToolUse;
}

export interface SpanNode {
  span: ProcessedSpan;
  children: SpanNode[];
  depth: number;
  timelineStart: number;
  timelineWidth: number;
}

export interface TimelineContext {
  minTime: number;
  maxTime: number;
  totalDuration: number;
}

export interface TraceDetailResponse {
  id: string;
  projectId: string;
  sessionId: string | null;
  userId: string | null;
  status: 'active' | 'completed' | 'error';
  tags: string[] | null;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: number;
  totalDurationMs: number;
  spanTree: SpanNode[];
  timeline: TimelineContext;
}

export interface Project {
  id: string;
  name: string;
  apiKey?: string;
  apiKeyHint?: string;
  ownerEmail: string;
  settings: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface Stats {
  totalTraces: number;
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: number;
  avgDurationMs: number;
  errorRate: number;
}

export interface UsageDataPoint {
  date: string;
  traces: number;
  spans: number;
  tokens: number;
  costUsd: number;
}

export interface TracesPage {
  data: Trace[];
  total: number;
  limit: number;
  offset: number;
}

export interface Session {
  sessionId: string;
  userId: string | null;
  traceCount: number;
  totalTokens: number;
  totalCostUsd: number;
  totalDurationMs: number;
  totalSpans: number;
  hasError: boolean;
  hasActive: boolean;
  firstTraceAt: string;
  lastTraceAt: string;
}

export interface SessionsPage {
  data: Session[];
  total: number;
  limit: number;
  offset: number;
}

class APIError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
    this.name = 'APIError';
  }
}

// Normalize Go backend response (PascalCase) to frontend (camelCase)
function normalizeTrace(t: Record<string, unknown>): Trace {
  return {
    id: t.ID as string,
    projectId: t.ProjectID as string,
    sessionId: t.SessionID as string | null,
    userId: t.UserID as string | null,
    metadata: (t.Metadata || {}) as Record<string, unknown>,
    tags: t.Tags as string[] | null,
    totalTokens: (t.TotalTokens as number) || 0,
    totalCostUsd: (t.TotalCostUSD as number) || 0,
    totalDurationMs: (t.TotalDurationMs as number) || 0,
    totalSpans: (t.TotalSpans as number) || 0,
    status: t.Status as 'active' | 'completed' | 'error',
    createdAt: t.CreatedAt as string,
    updatedAt: t.UpdatedAt as string,
  };
}

function normalizeSpan(s: Record<string, unknown>): Span {
  return {
    id: s.ID as string,
    traceId: s.TraceID as string,
    parentSpanId: s.ParentSpanID as string | null,
    type: s.Type as SpanType,
    name: s.Name as string,
    input: s.Input,
    output: s.Output,
    inputTokens: s.InputTokens as number | null,
    outputTokens: s.OutputTokens as number | null,
    costUsd: s.CostUSD as number | null,
    durationMs: s.DurationMs as number | null,
    status: s.Status as 'pending' | 'success' | 'error',
    errorMessage: s.ErrorMessage as string | null,
    model: s.Model as string | null,
    provider: s.Provider as string | null,
    metadata: (s.Metadata || {}) as Record<string, unknown>,
    startedAt: s.StartedAt as string,
    endedAt: s.EndedAt as string | null,
    // Extended fields (Phase 7.1)
    stopReason: s.StopReason as string | null,
    cacheReadTokens: s.CacheReadTokens as number | null,
    cacheWriteTokens: s.CacheWriteTokens as number | null,
    reasoningTokens: s.ReasoningTokens as number | null,
    firstTokenMs: s.FirstTokenMs as number | null,
    thinking: s.Thinking as string | null,
  };
}

function normalizeProject(p: Record<string, unknown>): Project {
  // Backend returns truncated key like "le_abc12345..." for listings
  const apiKey = (p.apiKey || p.APIKey) as string | undefined;
  return {
    id: (p.ID as string) || (p.id as string),
    name: (p.Name as string) || (p.name as string),
    apiKey: apiKey,
    apiKeyHint: apiKey ? (apiKey.endsWith('...') ? apiKey : apiKey.slice(0, 12) + '...') : undefined,
    ownerEmail: (p.OwnerEmail as string) || '',
    settings: (p.Settings || p.settings || {}) as Record<string, unknown>,
    createdAt: (p.CreatedAt as string) || (p.createdAt as string),
    updatedAt: (p.UpdatedAt as string) || (p.updatedAt as string),
  };
}

function normalizeStats(s: Record<string, unknown>): Stats {
  return {
    totalTraces: (s.TotalTraces as number) || 0,
    totalSpans: (s.TotalSpans as number) || 0,
    totalTokens: (s.TotalTokens as number) || 0,
    totalCostUsd: (s.TotalCostUSD as number) || 0,
    avgDurationMs: (s.AvgDurationMs as number) || 0,
    errorRate: (s.ErrorRate as number) || 0,
  };
}

function normalizeSession(s: Record<string, unknown>): Session {
  return {
    sessionId: (s.SessionID as string) || '',
    userId: s.UserID as string | null,
    traceCount: (s.TraceCount as number) || 0,
    totalTokens: (s.TotalTokens as number) || 0,
    totalCostUsd: (s.TotalCostUSD as number) || 0,
    totalDurationMs: (s.TotalDurationMs as number) || 0,
    totalSpans: (s.TotalSpans as number) || 0,
    hasError: (s.HasError as boolean) || false,
    hasActive: (s.HasActive as boolean) || false,
    firstTraceAt: (s.FirstTraceAt as string) || '',
    lastTraceAt: (s.LastTraceAt as string) || '',
  };
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  authToken?: string
): Promise<T> {
  const token = authToken || getAuthToken();

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}`;
    try {
      const error = await response.json();
      errorMessage = error.error || error.message || errorMessage;
    } catch {
      // ignore JSON parse error
    }
    throw new APIError(errorMessage, response.status);
  }

  const text = await response.text();
  if (!text) return {} as T;

  return JSON.parse(text);
}

// Dashboard API (Session Auth)
export const dashboardAPI = {
  async listProjects(): Promise<Project[]> {
    const data = await request<Record<string, unknown>[]>('GET', '/api/v1/dashboard/projects');
    return data.map(normalizeProject);
  },

  async createProject(name: string): Promise<Project> {
    const data = await request<Record<string, unknown>>('POST', '/api/v1/dashboard/projects', { name });
    return normalizeProject(data);
  },

  async updateProject(id: string, updates: { name?: string; settings?: Record<string, unknown> }): Promise<void> {
    await request('PATCH', `/api/v1/dashboard/projects/${id}`, updates);
  },

  async deleteProject(id: string): Promise<void> {
    await request('DELETE', `/api/v1/dashboard/projects/${id}`);
  },

  async rotateProjectAPIKey(id: string): Promise<{ apiKey: string }> {
    return request('POST', `/api/v1/dashboard/projects/${id}/api-key`);
  },

  async getTraces(projectId: string, params?: {
    sessionId?: string;
    userId?: string;
    status?: string;
    from?: string;
    to?: string;
    limit?: number;
    offset?: number;
  }): Promise<TracesPage> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.set(key, String(value));
        }
      });
    }
    const queryStr = searchParams.toString();
    const url = `/api/v1/dashboard/projects/${projectId}/traces${queryStr ? '?' + queryStr : ''}`;
    const data = await request<Record<string, unknown>>('GET', url);
    return {
      data: ((data.Data || []) as Record<string, unknown>[]).map(normalizeTrace),
      total: (data.Total as number) || 0,
      limit: (data.Limit as number) || 50,
      offset: (data.Offset as number) || 0,
    };
  },

  async getTrace(projectId: string, traceId: string): Promise<TraceDetailResponse> {
    // Backend now returns optimized structure directly in camelCase
    return request<TraceDetailResponse>('GET', `/api/v1/dashboard/projects/${projectId}/traces/${traceId}`);
  },

  async getStats(projectId: string, params?: { from?: string; to?: string }): Promise<Stats> {
    const searchParams = new URLSearchParams();
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    const queryStr = searchParams.toString();
    const url = `/api/v1/dashboard/projects/${projectId}/stats${queryStr ? '?' + queryStr : ''}`;
    const data = await request<Record<string, unknown>>('GET', url);
    return normalizeStats(data);
  },

  async getSessions(projectId: string, params?: {
    limit?: number;
    offset?: number;
  }): Promise<SessionsPage> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.set(key, String(value));
        }
      });
    }
    const queryStr = searchParams.toString();
    const url = `/api/v1/dashboard/projects/${projectId}/sessions${queryStr ? '?' + queryStr : ''}`;
    const data = await request<Record<string, unknown>>('GET', url);
    return {
      data: ((data.Data || []) as Record<string, unknown>[]).map(normalizeSession),
      total: (data.Total as number) || 0,
      limit: (data.Limit as number) || 50,
      offset: (data.Offset as number) || 0,
    };
  },

  async deleteAllTraces(projectId: string): Promise<{ deleted: number }> {
    const data = await request<Record<string, unknown>>('DELETE', `/api/v1/dashboard/projects/${projectId}/traces`);
    return { deleted: (data.Deleted as number) || 0 };
  },
};

// Legacy SDK API (for backward compatibility)
class LelemonAPI {
  private apiKey: string;

  constructor(apiKey: string) {
    this.apiKey = apiKey;
  }

  async getTraces(params?: {
    limit?: number;
    offset?: number;
  }): Promise<{ data: Trace[]; total: number }> {
    const searchParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.set(key, String(value));
        }
      });
    }
    const data = await request<Record<string, unknown>>('GET', `/api/v1/traces?${searchParams.toString()}`, undefined, this.apiKey);
    return {
      data: ((data.Data || []) as Record<string, unknown>[]).map(normalizeTrace),
      total: (data.Total as number) || 0,
    };
  }

  async getTrace(id: string): Promise<TraceWithSpans> {
    const data = await request<Record<string, unknown>>('GET', `/api/v1/traces/${id}`, undefined, this.apiKey);
    const trace = normalizeTrace(data);
    const spans = ((data.Spans || []) as Record<string, unknown>[]).map(normalizeSpan);
    return { ...trace, spans };
  }

  async getSummary(): Promise<Stats> {
    const data = await request<Record<string, unknown>>('GET', '/api/v1/analytics/summary', undefined, this.apiKey);
    return normalizeStats(data);
  }

  async getProject(): Promise<{ id: string; name: string; settings: Record<string, unknown>; createdAt: string }> {
    const data = await request<Record<string, unknown>>('GET', '/api/v1/projects/me', undefined, this.apiKey);
    return {
      id: data.id as string,
      name: data.name as string,
      settings: (data.settings || {}) as Record<string, unknown>,
      createdAt: data.createdAt as string,
    };
  }

  async rotateApiKey(): Promise<{ apiKey: string }> {
    return request('POST', '/api/v1/projects/api-key', undefined, this.apiKey);
  }
}

export function createAPI(apiKey: string): LelemonAPI {
  return new LelemonAPI(apiKey);
}

export { API_URL };
