import { defineTool } from '@mcify/core';
import { rateLimit, requireAuth, withTimeout } from '@mcify/core/middleware';
import { z } from 'zod';
import { clientFromContext } from '../context.js';

const sessionSummary = z.object({
  sessionId: z.string(),
  userId: z.string().nullable(),
  traceCount: z.number(),
  totalSpans: z.number(),
  totalTokens: z.number(),
  totalCostUsd: z.number(),
  totalDurationMs: z.number(),
  hasError: z.boolean(),
  hasActive: z.boolean(),
  firstTraceAt: z.string().optional(),
  lastTraceAt: z.string().optional(),
});

/** lelemon_list_sessions — conversations grouped by sessionId, with rollup metrics. */
export const createListSessionsTool = () =>
  defineTool({
    name: 'lelemon_list_sessions',
    description:
      'List sessions (conversations grouped by sessionId) for the current project, with rollup ' +
      'metrics: trace count, tokens, cost, duration, and whether any trace errored or is still active. ' +
      'Filter by user or time range.',
    middlewares: [
      requireAuth({ message: 'lelemon_list_sessions requires your project API key.' }),
      rateLimit({ max: 120, windowMs: 60_000 }),
      withTimeout({ ms: 8_000 }),
    ],
    input: z.object({
      limit: z.number().int().positive().max(100).optional().describe('Page size (max 100, default 50).'),
      offset: z.number().int().nonnegative().optional().describe('Pagination offset.'),
      userId: z.string().optional().describe('Only sessions for this end-user id.'),
      from: z.string().optional().describe('Inclusive lower bound, RFC3339 timestamp.'),
      to: z.string().optional().describe('Inclusive upper bound, RFC3339 timestamp.'),
    }),
    output: z.object({
      sessions: z.array(sessionSummary),
      total: z.number(),
      limit: z.number(),
      offset: z.number(),
      hasMore: z.boolean().describe('True if more rows exist; re-call with offset += sessions.length.'),
    }),
    handler: async (input, ctx) => {
      const client = clientFromContext(ctx);
      const page = await client.listSessions(input);
      return {
        sessions: page.data,
        total: page.total,
        limit: page.limit,
        offset: page.offset,
        hasMore: page.offset + page.data.length < page.total,
      };
    },
  });
