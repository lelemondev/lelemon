import { defineTool } from '@mcify/core';
import { rateLimit, requireAuth, withTimeout } from '@mcify/core/middleware';
import { z } from 'zod';
import type { AnalyticsMetric } from '../client.js';
import { clientFromContext } from '../context.js';

const metric = z.enum([
  'summary',
  'usage',
  'models',
  'tags',
  'top_users',
  'heatmap',
  'latency_distribution',
  'latency_timeseries',
]);

/**
 * lelemon_analytics — one consolidated tool over every project-level metric,
 * picked by `metric`. Mirrors how Langfuse exposes a single queryMetrics tool
 * instead of one tool per chart.
 */
export const createAnalyticsTool = () =>
  defineTool({
    name: 'lelemon_analytics',
    description:
      'Aggregate analytics for the current project. Pick a metric:\n' +
      '- summary: totals (traces, spans, tokens, cost, latency).\n' +
      '- usage: time series of volume/cost (needs granularity).\n' +
      '- models: cost & tokens grouped by model (best for "what did each model cost?").\n' +
      '- tags: cost & tokens grouped by tag.\n' +
      '- top_users: highest-cost end users (use limit).\n' +
      '- heatmap: activity by hour/day.\n' +
      '- latency_distribution: latency percentiles/buckets.\n' +
      '- latency_timeseries: latency over time (needs granularity).\n' +
      'Filter every metric by from/to (RFC3339). granularity applies to usage & latency_timeseries; ' +
      'limit applies to models, tags & top_users.',
    middlewares: [
      requireAuth({ message: 'lelemon_analytics requires an authorized connection.' }),
      rateLimit({ max: 120, windowMs: 60_000 }),
      withTimeout({ ms: 10_000 }),
    ],
    input: z.object({
      metric,
      from: z.string().optional().describe('Inclusive lower bound, RFC3339 timestamp.'),
      to: z.string().optional().describe('Inclusive upper bound, RFC3339 timestamp.'),
      granularity: z
        .enum(['hour', 'day', 'week'])
        .optional()
        .describe('Bucket size for usage & latency_timeseries.'),
      limit: z
        .number()
        .int()
        .positive()
        .max(1000)
        .optional()
        .describe('Row cap for models, tags & top_users.'),
    }),
    output: z.object({
      metric,
      data: z.unknown().describe('Metric payload; shape depends on the chosen metric.'),
    }),
    handler: async ({ metric: m, from, to, granularity, limit }, ctx) => {
      const client = clientFromContext(ctx);
      const data = await client.analytics(m as AnalyticsMetric, { from, to, granularity, limit });
      return { metric: m, data };
    },
  });
