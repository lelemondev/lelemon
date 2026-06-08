import { afterEach, describe, expect, it, vi } from 'vitest';
import { LelemonApiError, LelemonClient } from './client.js';

function mockFetchOnce(body: unknown, init?: { ok?: boolean; status?: number; text?: string }) {
  const ok = init?.ok ?? true;
  const status = init?.status ?? (ok ? 200 : 500);
  const fetchMock = vi.fn(async () => ({
    ok,
    status,
    json: async () => body,
    text: async () => init?.text ?? JSON.stringify(body),
  })) as unknown as typeof fetch;
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('LelemonClient', () => {
  it('getProject hits /api/v1/projects/me with the bearer key and parses JSON', async () => {
    const project = { id: 'p_1', name: 'Demo', createdAt: '2026-06-08T00:00:00Z' };
    const fetchMock = mockFetchOnce(project);

    const client = new LelemonClient({ apiKey: 'le_test', baseUrl: 'https://api.example.test' });
    const result = await client.getProject();

    expect(result).toEqual(project);
    const [url, options] = (fetchMock as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(String(url)).toBe('https://api.example.test/api/v1/projects/me');
    expect((options as RequestInit).headers).toMatchObject({ Authorization: 'Bearer le_test' });
  });

  it('strips trailing slashes from baseUrl', async () => {
    const fetchMock = mockFetchOnce({ id: 'p', name: 'n' });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test/' });
    await client.getProject();
    const [url] = (fetchMock as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(String(url)).toBe('https://api.example.test/api/v1/projects/me');
  });

  it('serializes defined query params and drops undefined ones', async () => {
    const fetchMock = mockFetchOnce({ ok: true });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    await client.request('GET', '/traces', { limit: 10, status: undefined, q: 'hi' });
    const [url] = (fetchMock as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
    const parsed = new URL(String(url));
    expect(parsed.searchParams.get('limit')).toBe('10');
    expect(parsed.searchParams.get('q')).toBe('hi');
    expect(parsed.searchParams.has('status')).toBe(false);
  });

  it('throws LelemonApiError on a non-2xx response', async () => {
    mockFetchOnce({}, { ok: false, status: 401, text: 'unauthorized' });
    const client = new LelemonClient({ apiKey: 'bad', baseUrl: 'https://api.example.test' });

    await expect(client.getProject()).rejects.toMatchObject({
      name: 'LelemonApiError',
      status: 401,
    });
    await expect(client.getProject()).rejects.toBeInstanceOf(LelemonApiError);
  });

  it('listTraces normalizes the PascalCase page to camelCase', async () => {
    mockFetchOnce({
      Data: [
        {
          ID: 't_1',
          Name: 'sales-agent',
          SessionID: 's_1',
          UserID: null,
          Status: 'completed',
          Tags: ['prod'],
          CreatedAt: '2026-06-08T00:00:00Z',
          TotalSpans: 3,
          TotalTokens: 1200,
          TotalCostUSD: 0.0123,
          TotalDurationMs: 540,
        },
      ],
      Total: 1,
      Limit: 50,
      Offset: 0,
    });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    const page = await client.listTraces({ status: 'completed', limit: 50 });

    expect(page.total).toBe(1);
    expect(page.data[0]).toEqual({
      id: 't_1',
      name: 'sales-agent',
      sessionId: 's_1',
      userId: null,
      status: 'completed',
      tags: ['prod'],
      createdAt: '2026-06-08T00:00:00Z',
      updatedAt: undefined,
      totalSpans: 3,
      totalTokens: 1200,
      totalCostUsd: 0.0123,
      totalDurationMs: 540,
    });
  });

  it('listTraces tolerates a null Data array', async () => {
    mockFetchOnce({ Data: null, Total: 0, Limit: 50, Offset: 0 });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    const page = await client.listTraces();
    expect(page.data).toEqual([]);
    expect(page.total).toBe(0);
  });

  it('listSessions normalizes session rows', async () => {
    mockFetchOnce({
      Data: [
        {
          SessionID: 's_1',
          UserID: 'u_1',
          TraceCount: 4,
          TotalSpans: 12,
          TotalTokens: 5000,
          TotalCostUSD: 0.08,
          TotalDurationMs: 2200,
          HasError: true,
          HasActive: false,
          FirstTraceAt: '2026-06-07T00:00:00Z',
          LastTraceAt: '2026-06-08T00:00:00Z',
        },
      ],
      Total: 1,
      Limit: 50,
      Offset: 0,
    });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    const page = await client.listSessions({ userId: 'u_1' });

    expect(page.data[0]).toMatchObject({
      sessionId: 's_1',
      userId: 'u_1',
      traceCount: 4,
      totalCostUsd: 0.08,
      hasError: true,
      hasActive: false,
    });
  });

  it('analytics unwraps the { data } envelope for wrapped metrics', async () => {
    const fetchMock = mockFetchOnce({ data: [{ model: 'gpt-5', costUsd: 1.23 }] });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    const data = await client.analytics('models', { from: '2026-06-01T00:00:00Z', limit: 10 });

    expect(data).toEqual([{ model: 'gpt-5', costUsd: 1.23 }]);
    const [url] = (fetchMock as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
    const parsed = new URL(String(url));
    expect(parsed.pathname).toBe('/api/v1/analytics/models');
    expect(parsed.searchParams.get('limit')).toBe('10');
  });

  it('analytics returns summary raw (no data envelope)', async () => {
    mockFetchOnce({ totalTraces: 5, totalCostUsd: 9.99 });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    const data = await client.analytics('summary');
    expect(data).toEqual({ totalTraces: 5, totalCostUsd: 9.99 });
  });

  it('latency_timeseries maps to the nested path', async () => {
    const fetchMock = mockFetchOnce({ data: [] });
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    await client.analytics('latency_timeseries', { granularity: 'day' });
    const [url] = (fetchMock as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
    const parsed = new URL(String(url));
    expect(parsed.pathname).toBe('/api/v1/analytics/latency/timeseries');
    expect(parsed.searchParams.get('granularity')).toBe('day');
  });

  it('gives an actionable message on 401', async () => {
    mockFetchOnce({}, { ok: false, status: 401, text: 'unauthorized' });
    const client = new LelemonClient({ apiKey: 'bad', baseUrl: 'https://api.example.test' });
    await expect(client.getProject()).rejects.toThrow(/API key/i);
  });

  it('getTraceDetail hits the /detail endpoint and passes through the tree', async () => {
    const detail = {
      id: 't_1',
      status: 'completed',
      totalCostUsd: 0.0123,
      spanTree: [{ span: { id: 'sp_1', costBreakdown: { total: 0.0123, cacheSavings: 0.004 } } }],
    };
    const fetchMock = mockFetchOnce(detail);
    const client = new LelemonClient({ apiKey: 'le_x', baseUrl: 'https://api.example.test' });
    const result = await client.getTraceDetail('t_1');

    const [url] = (fetchMock as unknown as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(String(url)).toBe('https://api.example.test/api/v1/traces/t_1/detail');
    expect(result).toEqual(detail);
  });
});
