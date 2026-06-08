import type { HandlerContext } from '@mcify/core';
import { LelemonClient } from './client.js';

/**
 * Build a LelemonClient from the request's bearer token — which IS the caller's
 * project API key. Throws if the request did not authenticate as bearer (the
 * requireAuth middleware should have already rejected it).
 */
export function clientFromContext(ctx: HandlerContext): LelemonClient {
  if (ctx.auth.type !== 'bearer') {
    throw new Error('Lelemon MCP tools require bearer authentication (your project API key).');
  }
  return new LelemonClient({ apiKey: ctx.auth.token });
}
