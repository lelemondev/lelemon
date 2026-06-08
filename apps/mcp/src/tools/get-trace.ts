import { defineTool } from '@mcify/core';
import { rateLimit, requireAuth, withTimeout } from '@mcify/core/middleware';
import { z } from 'zod';
import { clientFromContext } from '../context.js';
import { conciseSpanTree } from '../summarize.js';

/**
 * lelemon_get_trace — full processed trace: the hierarchical span tree plus,
 * for every LLM span, a per-token-type cost breakdown (input / output /
 * cacheRead / cacheWrite / reasoning) and the cache savings.
 *
 * Concise by default (span input/output/thinking stripped) so the response stays
 * small; pass detail:true to include the full prompts and completions.
 */
export const createGetTraceTool = () =>
  defineTool({
    name: 'lelemon_get_trace',
    description:
      'Get one trace by id with its full span tree. Each LLM span includes a costBreakdown ' +
      '(input, output, cacheRead, cacheWrite, reasoning, total, cacheSavings) so you can explain ' +
      'exactly where the money went and how much prompt caching saved. By default the large span ' +
      'input/output/thinking payloads are omitted to save tokens — pass detail:true to include them.',
    middlewares: [
      requireAuth({ message: 'lelemon_get_trace requires your project API key.' }),
      rateLimit({ max: 120, windowMs: 60_000 }),
      withTimeout({ ms: 10_000 }),
    ],
    input: z.object({
      id: z.string().min(1).describe('Trace id (from lelemon_list_traces).'),
      detail: z
        .boolean()
        .optional()
        .describe('Include full span input/output/thinking payloads. Default false (concise).'),
    }),
    output: z
      .object({
        id: z.string(),
        status: z.string().optional(),
        totalSpans: z.number().optional(),
        totalTokens: z.number().optional(),
        totalCostUsd: z.number().optional(),
        totalDurationMs: z.number().optional(),
        spanTree: z.array(z.unknown()),
        concise: z.boolean(),
      })
      .passthrough(),
    handler: async ({ id, detail }, ctx) => {
      const client = clientFromContext(ctx);
      const trace = await client.getTraceDetail(id);
      if (detail) {
        return { ...trace, concise: false };
      }
      return { ...trace, spanTree: conciseSpanTree(trace.spanTree), concise: true };
    },
  });
