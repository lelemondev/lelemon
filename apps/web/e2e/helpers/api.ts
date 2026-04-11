import { APIRequestContext } from '@playwright/test';

/**
 * API helper for E2E tests.
 * Provides methods to interact with the backend directly.
 */
export class ApiHelper {
  private baseUrl: string;
  private request: APIRequestContext;

  constructor(request: APIRequestContext, baseUrl?: string) {
    this.request = request;
    // Use 127.0.0.1 to avoid DNS resolution issues in browsers
    this.baseUrl = baseUrl || process.env.PLAYWRIGHT_API_URL || 'http://127.0.0.1:8080';
  }

  /**
   * Get the features configuration (OSS vs EE).
   */
  async getFeatures() {
    const response = await this.request.get(`${this.baseUrl}/api/v1/features`);
    return response.json();
  }

  /**
   * Register a new user.
   */
  async register(email: string, password: string, name: string) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/auth/register`, {
      data: { email, password, name },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Login and get JWT token.
   */
  async login(email: string, password: string) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/auth/login`, {
      data: { email, password },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Create a project (requires auth).
   */
  async createProject(token: string, name: string) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/dashboard/projects`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get user's projects (requires auth).
   */
  async getProjects(token: string) {
    const response = await this.request.get(`${this.baseUrl}/api/v1/dashboard/projects`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Create an organization (EE only, requires auth).
   */
  async createOrganization(token: string, name: string, slug?: string) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/organizations`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { name, slug },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Ingest events (requires API key).
   * Note: API expects { events: [...] } not { spans: [...] }
   */
  async ingestSpans(apiKey: string, events: object[]) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/ingest`, {
      headers: { Authorization: `Bearer ${apiKey}` },
      data: { events },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Health check.
   */
  async healthCheck() {
    try {
      const response = await this.request.get(`${this.baseUrl}/health`);
      return response.status() === 200;
    } catch {
      return false;
    }
  }

  /**
   * Register with full response (includes headers).
   */
  async registerRaw(email: string, password: string, name: string) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/auth/register`, {
      data: { email, password, name },
    });
    return {
      status: response.status(),
      headers: response.headers(),
      body: await response.text(),
    };
  }

  /**
   * Login with full response (includes headers).
   */
  async loginRaw(email: string, password: string) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/auth/login`, {
      data: { email, password },
    });
    return {
      status: response.status(),
      headers: response.headers(),
      body: await response.text(),
    };
  }

  /**
   * Get response headers from any endpoint.
   */
  async getHeaders(path: string) {
    const response = await this.request.get(`${this.baseUrl}${path}`);
    return {
      status: response.status(),
      headers: response.headers(),
    };
  }

  /**
   * Make authenticated request and get headers.
   */
  async getAuthenticatedHeaders(path: string, token: string) {
    const response = await this.request.get(`${this.baseUrl}${path}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return {
      status: response.status(),
      headers: response.headers(),
    };
  }

  /**
   * Get the OAuth base URL for constructing test URLs.
   */
  getBaseUrl(): string {
    return this.baseUrl;
  }

  /**
   * Initiate Google OAuth flow (returns redirect info).
   * Note: This initiates the flow but we can't complete it without real Google.
   */
  async initiateGoogleOAuth() {
    const response = await this.request.get(`${this.baseUrl}/api/v1/auth/google`, {
      maxRedirects: 0, // Don't follow redirects
    });
    return {
      status: response.status(),
      headers: response.headers(),
      location: response.headers()['location'],
    };
  }

  /**
   * Call OAuth callback directly with custom parameters.
   * Used for testing error handling in OAuth flow.
   */
  async oauthCallback(params: {
    code?: string;
    state?: string;
    error?: string;
    cookie?: string;
  }) {
    const searchParams = new URLSearchParams();
    if (params.code) searchParams.set('code', params.code);
    if (params.state) searchParams.set('state', params.state);
    if (params.error) searchParams.set('error', params.error);

    const headers: Record<string, string> = {};
    if (params.cookie) {
      headers['Cookie'] = params.cookie;
    }

    const response = await this.request.get(
      `${this.baseUrl}/api/v1/auth/google/callback?${searchParams.toString()}`,
      {
        headers,
        maxRedirects: 0, // Don't follow redirects to capture redirect URL
      }
    );

    return {
      status: response.status(),
      headers: response.headers(),
      location: response.headers()['location'] || '',
    };
  }

  /**
   * Check if OAuth is configured on the server.
   */
  async isOAuthConfigured(): Promise<boolean> {
    try {
      const response = await this.request.get(`${this.baseUrl}/api/v1/auth/google`, {
        maxRedirects: 0,
      });
      // 307 = redirect to Google (configured), 501 = not configured
      return response.status() === 307;
    } catch {
      return false;
    }
  }

  // ==================== Post-Login Security Tests ====================

  /**
   * Get project details including API key hint.
   */
  async getProject(token: string, projectId: string) {
    const response = await this.request.get(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Rotate project API key.
   */
  async rotateApiKey(token: string, projectId: string) {
    const response = await this.request.post(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/rotate-key`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Delete a project.
   */
  async deleteProject(token: string, projectId: string) {
    const response = await this.request.delete(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get traces for a project.
   */
  async getTraces(token: string, projectId: string) {
    const response = await this.request.get(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/traces`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get a specific trace.
   */
  async getTrace(token: string, projectId: string, traceId: string) {
    const response = await this.request.get(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/traces/${traceId}`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get project stats.
   */
  async getProjectStats(token: string, projectId: string) {
    const response = await this.request.get(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/stats`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get current user info.
   */
  async getMe(token: string) {
    const response = await this.request.get(`${this.baseUrl}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Ingest spans and return full response.
   */
  async ingestSpansRaw(apiKey: string, events: object[]) {
    const response = await this.request.post(`${this.baseUrl}/api/v1/ingest`, {
      headers: { Authorization: `Bearer ${apiKey}` },
      data: { events },
    });
    return {
      status: response.status(),
      headers: response.headers(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Try to access a project with wrong user token.
   */
  async getProjectUnauthorized(token: string, projectId: string) {
    const response = await this.request.get(
      `${this.baseUrl}/api/v1/dashboard/projects/${projectId}`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  // ==================== Enterprise Analytics Tests ====================

  /**
   * Get cost breakdown by tags (EE only).
   */
  async getCostBreakdown(
    token: string,
    projectId: string,
    params?: { tagPrefix?: string; from?: string; to?: string; limit?: number }
  ) {
    const searchParams = new URLSearchParams();
    if (params?.tagPrefix) searchParams.set('tagPrefix', params.tagPrefix);
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    if (params?.limit) searchParams.set('limit', params.limit.toString());

    const queryString = searchParams.toString();
    const url = `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/analytics/cost-breakdown${queryString ? `?${queryString}` : ''}`;

    const response = await this.request.get(url, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get error metrics (EE only).
   */
  async getErrorMetrics(
    token: string,
    projectId: string,
    params?: { tagPrefix?: string; from?: string; to?: string; topLimit?: number }
  ) {
    const searchParams = new URLSearchParams();
    if (params?.tagPrefix) searchParams.set('tagPrefix', params.tagPrefix);
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    if (params?.topLimit) searchParams.set('topLimit', params.topLimit.toString());

    const queryString = searchParams.toString();
    const url = `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/analytics/errors${queryString ? `?${queryString}` : ''}`;

    const response = await this.request.get(url, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return {
      status: response.status(),
      data: await response.json().catch(() => null),
    };
  }

  /**
   * Get traces with filters (supports tags, name, dateRange).
   * Note: Backend returns {Data: [], Total: n} in PascalCase.
   * This method normalizes the response to return data as an array.
   */
  async getTracesFiltered(
    token: string,
    projectId: string,
    params?: {
      tags?: string[];
      name?: string;
      sessionId?: string;
      userId?: string;
      status?: string;
      from?: string;
      to?: string;
      limit?: number;
      offset?: number;
    }
  ) {
    const searchParams = new URLSearchParams();
    if (params?.tags) {
      params.tags.forEach(tag => searchParams.append('tags', tag));
    }
    if (params?.name) searchParams.set('name', params.name);
    if (params?.sessionId) searchParams.set('sessionId', params.sessionId);
    if (params?.userId) searchParams.set('userId', params.userId);
    if (params?.status) searchParams.set('status', params.status);
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());

    const queryString = searchParams.toString();
    const url = `${this.baseUrl}/api/v1/dashboard/projects/${projectId}/traces${queryString ? `?${queryString}` : ''}`;

    const response = await this.request.get(url, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const json = await response.json().catch(() => null);

    // Normalize Go PascalCase response: {Data: [], Total: n} -> array with lowercase fields
    const normalizedData = json?.Data?.map((trace: Record<string, unknown>) => ({
      id: trace.ID,
      projectId: trace.ProjectID,
      name: trace.Name,
      status: trace.Status,
      tags: trace.Tags || [],
      sessionId: trace.SessionID,
      userId: trace.UserID,
      totalTokens: trace.TotalTokens,
      totalCostUsd: trace.TotalCostUSD,
      totalDurationMs: trace.TotalDurationMs,
      totalSpans: trace.TotalSpans,
      createdAt: trace.CreatedAt,
    })) || [];

    return {
      status: response.status(),
      data: normalizedData,
      total: json?.Total || 0,
    };
  }

  /**
   * Ingest events with specific tags for testing tag-based features.
   * Each trace is created as an agent span (so tags are properly extracted)
   * with its own unique traceId.
   *
   * Note: Backend uses async processing, so we add a small delay
   * to ensure traces are fully processed before returning.
   */
  async ingestTracesWithTags(
    apiKey: string,
    traces: Array<{
      name: string;
      tags: string[];
      status?: 'success' | 'error';
      errorMessage?: string;
      inputTokens?: number;
      outputTokens?: number;
      model?: string;
      provider?: string;
    }>
  ) {
    // Create each trace as a separate agent span with unique traceId
    const events = traces.map((trace, index) => ({
      traceId: `test-trace-${Date.now()}-${index}`,
      spanId: `test-span-${Date.now()}-${index}`,
      spanType: 'agent', // Agent span ensures tags are extracted properly
      name: trace.name,
      tags: trace.tags,
      status: trace.status || 'success',
      errorMessage: trace.errorMessage,
      durationMs: 100,
      input: { message: 'test' },
      output: 'test response',
    }));

    const result = await this.ingestSpans(apiKey, events);

    // Wait for async processing to complete
    // Backend uses async workers, so we need enough time for all traces to be processed
    // Increase delay based on number of traces to handle larger batches
    const delay = Math.max(1500, traces.length * 150);
    await new Promise(resolve => setTimeout(resolve, delay));

    return result;
  }
}
