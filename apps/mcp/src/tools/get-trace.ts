import { defineTool } from '@mcify/core';
import { rateLimit, requireAuth, withTimeout } from '@mcify/core/middleware';
import { z } from 'zod';
import { clientFromContext } from '../context.js';
import { conciseMetadata, conciseSpanTree } from '../summarize.js';

/**
 * lelemon_get_trace — full processed trace: the hierarchical span tree plus,
 * for every LLM span, a per-token-type cost breakdown (input / output /
 * cacheRead / cacheWrite / reasoning) and the cache savings.
 *
 * Concise by default: the dialogue (input.messages) and completion are KEPT so
 * the trace is readable, while the static system prompt + tool schemas (and the
 * redundant metadata.input request copy) are replaced with size placeholders.
 * Pass detail:true for the raw request verbatim.
 */
export const createGetTraceTool = () =>
  defineTool({
    name: 'lelemon_get_trace',
    description:
      'Get one trace by id with its full span tree. Each LLM span includes a costBreakdown ' +
      '(input, output, cacheRead, cacheWrite, reasoning, total, cacheSavings) so you can explain ' +
      'exactly where the money went and how much prompt caching saved. By default the view is ' +
      'READABLE-CONCISE: the conversation (input.messages) and the completion are kept, but the ' +
      'huge static parts repeated on every span — the system prompt, the tool schemas, and the ' +
      'redundant full-request copy under metadata.input — are replaced with a size placeholder ' +
      '(a real agent trace drops from ~150k to a few k chars). Pass detail:true for the raw request.',
    middlewares: [
      requireAuth({ message: 'lelemon_get_trace requires an authorized connection.' }),
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
      return {
        ...trace,
        metadata: conciseMetadata(trace['metadata']),
        spanTree: conciseSpanTree(trace.spanTree),
        concise: true,
      };
    },
  });
