import process from 'node:process';
import { defineConfig, oauthProvider, type AuthorizeDecision } from '@mcify/core';
import { jwtVerify } from 'jose';
import { HttpOAuthStore } from './src/oauth-store.js';
import { createAnalyticsTool } from './src/tools/analytics.js';
import { createGetProjectTool } from './src/tools/get-project.js';
import { createGetTraceTool } from './src/tools/get-trace.js';
import { createListSessionsTool } from './src/tools/list-sessions.js';
import { createListTracesTool } from './src/tools/list-traces.js';

/**
 * Lelemon MCP server. mcify is the OAuth 2.1 authorization server — a user runs
 * `claude mcp add` and authorizes in the browser ("Connect Claude"), no API key copied.
 *
 * - Storage of clients/codes/tokens lives in the Lelemon backend (HttpOAuthStore → Go RPC).
 * - Identity + consent is the dashboard's job: when there is no session, `/authorize` redirects
 *   to the dashboard consent page, which (after the user picks a project) mints a short-lived,
 *   audience-bound consent JWT and sends the browser back here with `?consent=<jwt>`. We verify it
 *   and bind `{ userId, projectId }` to the token — tools then query that project (see context.ts).
 */

const DASHBOARD_URL = (process.env['DASHBOARD_URL'] ?? 'https://lelemon.dev').replace(/\/+$/, '');
const CONSENT_AUDIENCE = 'mcp-consent';

/** Verify the dashboard-issued consent JWT (HS256, MCP_CONSENT_SECRET). Null if missing/invalid. */
async function verifyConsent(token: string): Promise<{ userId: string; projectId: string } | null> {
  const secret = process.env['MCP_CONSENT_SECRET'];
  if (!secret) return null;
  try {
    const { payload } = await jwtVerify(token, new TextEncoder().encode(secret), {
      audience: CONSENT_AUDIENCE,
    });
    const userId = typeof payload.sub === 'string' ? payload.sub : '';
    const projectId = typeof payload['projectId'] === 'string' ? payload['projectId'] : '';
    if (!userId || !projectId) return null;
    return { userId, projectId };
  } catch {
    return null;
  }
}

export default defineConfig({
  name: 'lelemon',
  version: '0.2.0',
  description:
    'Lelemon LLM observability — query traces, spans, cost breakdowns, sessions and analytics ' +
    'for your project. Connect with OAuth (no API key needed). Read-only.',
  auth: oauthProvider({
    // Public URL of this MCP (issuer for the OAuth metadata). Derived from the request if unset.
    issuer: process.env['MCIFY_ISSUER'],
    resourceName: 'Lelemon MCP',
    store: new HttpOAuthStore({ secret: process.env['MCP_STORE_SECRET'] ?? '' }),
    authorize: async (request): Promise<AuthorizeDecision> => {
      // Railway terminates TLS, so request.url arrives as http://. Honor X-Forwarded-Proto
      // (as the Go API already does) so the return URL's origin is https and matches the
      // dashboard's allow-list (safeReturnUrl) instead of being rejected as invalid.
      const reqUrl = new URL(request.url);
      const forwardedProto = request.headers.get('x-forwarded-proto')?.split(',')[0]?.trim();
      if (forwardedProto) reqUrl.protocol = `${forwardedProto}:`;

      const consent = reqUrl.searchParams.get('consent');
      if (consent) {
        const subject = await verifyConsent(consent);
        if (subject) return { status: 'authenticated', subject };
      }
      // No (valid) consent yet → send the user to the dashboard to sign in + pick a project.
      const ret = encodeURIComponent(reqUrl.toString());
      return { status: 'redirect', url: `${DASHBOARD_URL}/dashboard/mcp/authorize?return=${ret}` };
    },
  }),
  tools: [
    createGetProjectTool(),
    createListTracesTool(),
    createGetTraceTool(),
    createListSessionsTool(),
    createAnalyticsTool(),
  ],
});
