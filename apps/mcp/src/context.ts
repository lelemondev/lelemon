import process from 'node:process';
import type { HandlerContext } from '@mcify/core';
import { LelemonClient } from './client.js';

/**
 * Build a LelemonClient scoped to the caller's project.
 *
 *  - Under OAuth (`oauthProvider`), the validated token carries the opaque subject the host bound:
 *    `{ userId, projectId }`. There is no project API key, so the client uses the trusted service
 *    path — `MCP_STORE_SECRET` + `X-Project-Id` — and the backend's ProjectAuth resolves the project.
 *  - Under the legacy bearer auth, the bearer token IS the project API key.
 */
export function clientFromContext(ctx: HandlerContext): LelemonClient {
  if (ctx.auth.type === 'oauth_provider') {
    const projectId = ctx.auth.subject['projectId'];
    if (!projectId) {
      throw new Error('OAuth token is missing a projectId — re-authorize and pick a project.');
    }
    const serviceSecret = process.env['MCP_STORE_SECRET'];
    if (!serviceSecret) {
      throw new Error('MCP_STORE_SECRET is not configured — the MCP cannot reach the backend.');
    }
    return new LelemonClient({ serviceSecret, projectId });
  }

  if (ctx.auth.type === 'bearer') {
    return new LelemonClient({ apiKey: ctx.auth.token });
  }

  throw new Error('Lelemon MCP tools require an authenticated request.');
}
