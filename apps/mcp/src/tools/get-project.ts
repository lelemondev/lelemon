import { defineTool } from '@mcify/core';
import { rateLimit, requireAuth, withTimeout } from '@mcify/core/middleware';
import { z } from 'zod';
import { clientFromContext } from '../context.js';

/**
 * lelemon_get_project — identify the project this connection is scoped to.
 * A good first call to confirm scope before listing traces or analytics.
 */
export const createGetProjectTool = () =>
  defineTool({
    name: 'lelemon_get_project',
    description:
      'Get the Lelemon project this connection is scoped to (id, name, settings, timestamps). ' +
      'Use this to confirm which project you are querying before listing traces or analytics.',
    middlewares: [
      requireAuth({ message: 'lelemon_get_project requires an authorized connection.' }),
      rateLimit({ max: 120, windowMs: 60_000 }),
      withTimeout({ ms: 8_000 }),
    ],
    input: z.object({}),
    output: z.object({
      id: z.string(),
      name: z.string(),
      createdAt: z.string().optional(),
      updatedAt: z.string().optional(),
      settings: z.unknown().optional(),
    }),
    handler: async (_input, ctx) => {
      const client = clientFromContext(ctx);
      return client.getProject();
    },
  });
