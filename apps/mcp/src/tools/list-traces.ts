import { defineTool } from '@mcify/core';
import { rateLimit, requireAuth, withTimeout } from '@mcify/core/middleware';
import { z } from 'zod';
import { clientFromContext } from '../context.js';

const traceSummary = z.object({
  id: z.string(),
  name: z.string().nullable(),
  sessionId: z.string().nullable(),
  userId: z.string().nullable(),
  status: z.string(),
  tags: z.array(z.string()),
  createdAt: z.string().optional(),
  updatedAt: z.string().optional(),
  totalSpans: z.number(),
  totalTokens: z.number(),
  totalCostUsd: z.number(),
  totalDurationMs: z.number(),
});

/** lelemon_list_traces — paginated, filterable list of traces for the project. */
export const createListTracesTool = () =>
  defineTool({
    name: 'lelemon_list_traces',
    description:
      'List traces for the current project, newest first. Filter by status (active|completed|error), ' +
      'session, user, or time range. Returns per-trace totals (spans, tokens, cost, duration). ' +
      'Use lelemon_get_trace with an id to drill into one trace and its cost breakdown.',
    middlewares: [
      requireAuth({ message: 'lelemon_list_traces requires your project API key.' }),
      rateLimit({ max: 120, windowMs: 60_000 }),
      withTimeout({ ms: 8_000 }),
    ],
    input: z.object({
      limit: z.number().int().positive().max(100).optional().describe('Page size (max 100, default 50).'),
      offset: z.number().int().nonnegative().optional().describe('Pagination offset.'),
      status: z.enum(['active', 'completed', 'error']).optional().describe('Filter by trace status.'),
      sessionId: z.string().optional().describe('Only traces in this session.'),
      userId: z.string().optional().describe('Only traces for this end-user id.'),
      from: z.string().optional().describe('Inclusive lower bound, RFC3339 timestamp.'),
      to: z.string().optional().describe('Inclusive upper bound, RFC3339 timestamp.'),
    }),
    output: z.object({
      traces: z.array(traceSummary),
      total: z.number(),
      limit: z.number(),
      offset: z.number(),
      hasMore: z.boolean().describe('True if more rows exist; re-call with offset += traces.length.'),
    }),
    handler: async (input, ctx) => {
      const client = clientFromContext(ctx);
      const page = await client.listTraces(input);
      return {
        traces: page.data,
        total: page.total,
        limit: page.limit,
        offset: page.offset,
        hasMore: page.offset + page.data.length < page.total,
      };
    },
  });
