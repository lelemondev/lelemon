import { bearer, defineConfig } from '@mcify/core';
import { LelemonClient } from './src/client.js';
import { createAnalyticsTool } from './src/tools/analytics.js';
import { createGetProjectTool } from './src/tools/get-project.js';
import { createGetTraceTool } from './src/tools/get-trace.js';
import { createListSessionsTool } from './src/tools/list-sessions.js';
import { createListTracesTool } from './src/tools/list-traces.js';

/**
 * Lelemon MCP server. The bearer token a client presents IS its Lelemon project
 * API key (le_xxx) — so the server is multi-tenant for free: every key resolves
 * to exactly one project. Tools are read-only.
 */
export default defineConfig({
  name: 'lelemon',
  version: '0.1.0',
  description:
    'Lelemon LLM observability — query traces, spans, cost breakdowns, sessions and analytics ' +
    'for the project your API key belongs to. Read-only.',
  auth: bearer({
    env: 'LELEMON_API_KEY',
    verify: async (token) => {
      // Validate the key by hitting /projects/me; a success means it is a real
      // key scoped to a project. The credential never leaves the handler layer.
      try {
        await new LelemonClient({ apiKey: token }).getProject();
        return true;
      } catch {
        return false;
      }
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
