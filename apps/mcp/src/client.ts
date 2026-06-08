import process from 'node:process';

/** Default production API base URL. Override with LELEMON_ENDPOINT. */
const DEFAULT_BASE_URL = 'https://api.lelemon.dev';

/**
 * How the client authenticates to the Lelemon API:
 *  - `apiKey` — a project API key (le_xxx), sent as `Authorization: Bearer <apiKey>`. The classic
 *    path: the key itself scopes the request to its project.
 *  - `serviceSecret` + `projectId` — the trusted service path used under OAuth, where the MCP has
 *    resolved a project but has no API key. Sends the shared secret as Bearer plus `X-Project-Id`;
 *    the backend's ProjectAuth loads that project (see server middleware).
 */
export type LelemonClientOptions = (
  | { apiKey: string; serviceSecret?: undefined; projectId?: undefined }
  | { serviceSecret: string; projectId: string; apiKey?: undefined }
) & {
  /** API base URL. Defaults to LELEMON_ENDPOINT env or https://api.lelemon.dev. */
  baseUrl?: string;
};

/** Error thrown when the Lelemon API returns a non-2xx response. */
export class LelemonApiError extends Error {
  readonly status: number;
  readonly body: string | undefined;

  constructor(message: string, status: number, body?: string) {
    super(message);
    this.name = 'LelemonApiError';
    this.status = status;
    this.body = body;
  }
}

/** The project the current API key belongs to (GET /projects/me). */
export interface LelemonProject {
  id: string;
  name: string;
  createdAt?: string;
  updatedAt?: string;
  settings?: unknown;
}

/** Query params accepted by request(); undefined values are dropped. */
export type QueryParams = Record<string, string | number | boolean | undefined>;

/** Pagination envelope returned by list endpoints. */
export interface Page<T> {
  data: T[];
  total: number;
  limit: number;
  offset: number;
}

/** A trace row from GET /traces (normalized to camelCase). */
export interface TraceSummary {
  id: string;
  name: string | null;
  sessionId: string | null;
  userId: string | null;
  status: string;
  tags: string[];
  createdAt: string | undefined;
  updatedAt: string | undefined;
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: number;
  totalDurationMs: number;
}

/** A session row from GET /sessions (normalized to camelCase). */
export interface SessionSummary {
  sessionId: string;
  userId: string | null;
  traceCount: number;
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: number;
  totalDurationMs: number;
  hasError: boolean;
  hasActive: boolean;
  firstTraceAt: string | undefined;
  lastTraceAt: string | undefined;
}

/**
 * Processed trace detail from GET /traces/{id}/detail. Already camelCase; the
 * span tree (with per-span `costBreakdown`) is passed through as-is.
 */
export interface TraceDetail {
  id: string;
  status?: string;
  totalSpans?: number;
  totalTokens?: number;
  totalCostUsd?: number;
  totalDurationMs?: number;
  spanTree: unknown[];
  [key: string]: unknown;
}

export interface ListTracesParams {
  limit?: number;
  offset?: number;
  sessionId?: string;
  userId?: string;
  status?: string;
  from?: string;
  to?: string;
}

export interface ListSessionsParams {
  limit?: number;
  offset?: number;
  userId?: string;
  from?: string;
  to?: string;
}

/** Analytics metrics, each backed by one Lelemon analytics endpoint. */
export type AnalyticsMetric =
  | 'summary'
  | 'usage'
  | 'models'
  | 'tags'
  | 'top_users'
  | 'heatmap'
  | 'latency_distribution'
  | 'latency_timeseries';

export interface AnalyticsParams {
  from?: string;
  to?: string;
  /** Only for time-series metrics (usage, latency_timeseries). */
  granularity?: 'hour' | 'day' | 'week';
  /** Only for ranked metrics (models, tags, top_users). */
  limit?: number;
}

const ANALYTICS_PATHS: Record<AnalyticsMetric, string> = {
  summary: '/analytics/summary',
  usage: '/analytics/usage',
  models: '/analytics/models',
  tags: '/analytics/tags',
  top_users: '/analytics/top-users',
  heatmap: '/analytics/heatmap',
  latency_distribution: '/analytics/latency/distribution',
  latency_timeseries: '/analytics/latency/timeseries',
};

/** Turn an HTTP failure into a message the agent can act on. */
function describeFailure(
  method: string,
  path: string,
  status: number,
  body: string | undefined,
): string {
  const base = `Lelemon API ${method} ${path} failed (${status})`;
  switch (status) {
    case 401:
    case 403:
      return `${base}: authentication failed — check your Lelemon project API key (LELEMON_API_KEY).`;
    case 404:
      return `${base}: not found — the id may be wrong or belong to another project.`;
    case 429:
      return `${base}: rate limited — retry after a short pause.`;
    default:
      return body ? `${base}: ${body.slice(0, 200)}` : base;
  }
}

// --- Raw (PascalCase) shapes returned by the Go API -------------------------

interface RawPage<T> {
  Data?: T[] | null;
  Total?: number;
  Limit?: number;
  Offset?: number;
}

interface RawTraceSummary {
  ID: string;
  Name?: string | null;
  SessionID?: string | null;
  UserID?: string | null;
  Status?: string;
  Tags?: string[] | null;
  CreatedAt?: string;
  UpdatedAt?: string;
  TotalSpans?: number;
  TotalTokens?: number;
  TotalCostUSD?: number;
  TotalDurationMs?: number;
}

interface RawSession {
  SessionID: string;
  UserID?: string | null;
  TraceCount?: number;
  TotalSpans?: number;
  TotalTokens?: number;
  TotalCostUSD?: number;
  TotalDurationMs?: number;
  HasError?: boolean;
  HasActive?: boolean;
  FirstTraceAt?: string;
  LastTraceAt?: string;
}

function normalizePage<R, T>(raw: RawPage<R>, mapItem: (item: R) => T): Page<T> {
  return {
    data: (raw.Data ?? []).map(mapItem),
    total: raw.Total ?? 0,
    limit: raw.Limit ?? 0,
    offset: raw.Offset ?? 0,
  };
}

function normalizeTraceSummary(r: RawTraceSummary): TraceSummary {
  return {
    id: r.ID,
    name: r.Name ?? null,
    sessionId: r.SessionID ?? null,
    userId: r.UserID ?? null,
    status: r.Status ?? 'unknown',
    tags: r.Tags ?? [],
    createdAt: r.CreatedAt,
    updatedAt: r.UpdatedAt,
    totalSpans: r.TotalSpans ?? 0,
    totalTokens: r.TotalTokens ?? 0,
    totalCostUsd: r.TotalCostUSD ?? 0,
    totalDurationMs: r.TotalDurationMs ?? 0,
  };
}

function normalizeSession(r: RawSession): SessionSummary {
  return {
    sessionId: r.SessionID,
    userId: r.UserID ?? null,
    traceCount: r.TraceCount ?? 0,
    totalSpans: r.TotalSpans ?? 0,
    totalTokens: r.TotalTokens ?? 0,
    totalCostUsd: r.TotalCostUSD ?? 0,
    totalDurationMs: r.TotalDurationMs ?? 0,
    hasError: r.HasError ?? false,
    hasActive: r.HasActive ?? false,
    firstTraceAt: r.FirstTraceAt,
    lastTraceAt: r.LastTraceAt,
  };
}

/**
 * Thin fetch wrapper over the Lelemon HTTP API, scoped to a single project via
 * its API key. All paths are relative to `${baseUrl}/api/v1`.
 */
export class LelemonClient {
  private readonly authHeaders: Record<string, string>;
  private readonly baseUrl: string;

  constructor(options: LelemonClientOptions) {
    if (options.apiKey !== undefined) {
      this.authHeaders = { Authorization: `Bearer ${options.apiKey}` };
    } else {
      this.authHeaders = {
        Authorization: `Bearer ${options.serviceSecret}`,
        'X-Project-Id': options.projectId,
      };
    }
    const base = options.baseUrl ?? process.env['LELEMON_ENDPOINT'] ?? DEFAULT_BASE_URL;
    this.baseUrl = base.replace(/\/+$/, '');
  }

  /** GET /projects/me — the project this API key is scoped to. */
  async getProject(): Promise<LelemonProject> {
    return this.request<LelemonProject>('GET', '/projects/me');
  }

  /** GET /traces — paginated list of trace summaries (newest first). */
  async listTraces(params: ListTracesParams = {}): Promise<Page<TraceSummary>> {
    const raw = await this.request<RawPage<RawTraceSummary>>('GET', '/traces', {
      limit: params.limit,
      offset: params.offset,
      sessionId: params.sessionId,
      userId: params.userId,
      status: params.status,
      from: params.from,
      to: params.to,
    });
    return normalizePage(raw, normalizeTraceSummary);
  }

  /** GET /traces/{id}/detail — processed span tree with per-span costBreakdown. */
  async getTraceDetail(traceId: string): Promise<TraceDetail> {
    return this.request<TraceDetail>('GET', `/traces/${encodeURIComponent(traceId)}/detail`);
  }

  /** GET /sessions — paginated list of session summaries. */
  async listSessions(params: ListSessionsParams = {}): Promise<Page<SessionSummary>> {
    const raw = await this.request<RawPage<RawSession>>('GET', '/sessions', {
      limit: params.limit,
      offset: params.offset,
      userId: params.userId,
      from: params.from,
      to: params.to,
    });
    return normalizePage(raw, normalizeSession);
  }

  /**
   * GET /analytics/{metric} — aggregate metrics (cost, usage, latency...).
   * Unwraps the `{ data }` envelope most endpoints use; `summary` returns raw.
   */
  async analytics(metric: AnalyticsMetric, params: AnalyticsParams = {}): Promise<unknown> {
    const raw = await this.request<unknown>('GET', ANALYTICS_PATHS[metric], {
      from: params.from,
      to: params.to,
      granularity: params.granularity,
      limit: params.limit,
    });
    if (raw && typeof raw === 'object' && 'data' in (raw as Record<string, unknown>)) {
      return (raw as Record<string, unknown>)['data'];
    }
    return raw;
  }

  /** Perform an authenticated JSON request against the Lelemon API. */
  async request<T>(method: string, path: string, query?: QueryParams): Promise<T> {
    const url = new URL(`${this.baseUrl}/api/v1${path}`);
    if (query) {
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined) url.searchParams.set(key, String(value));
      }
    }

    const res = await fetch(url, {
      method,
      headers: {
        ...this.authHeaders,
        Accept: 'application/json',
      },
    });

    if (!res.ok) {
      const body = await res.text().catch(() => undefined);
      throw new LelemonApiError(describeFailure(method, path, res.status, body), res.status, body);
    }

    return (await res.json()) as T;
  }
}
