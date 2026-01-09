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
}
