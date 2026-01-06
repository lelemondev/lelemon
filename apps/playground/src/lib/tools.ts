/**
 * Tools for the playground agents
 *
 * Each tool has:
 * - name: unique identifier
 * - description: what the tool does (for the LLM)
 * - parameters: JSON schema for tool inputs
 * - execute: function that runs the tool
 */

import { span } from '@lelemondev/sdk';
import { executeQuery } from './data/database';
import { searchDocuments } from './data/documents';

// ─────────────────────────────────────────────────────────────
// Tool Types
// ─────────────────────────────────────────────────────────────

export interface ToolParameter {
  type: string;
  description: string;
  enum?: string[];
  default?: unknown;
  required?: boolean;
}

export interface Tool {
  name: string;
  description: string;
  parameters: Record<string, ToolParameter>;
  execute: (args: Record<string, unknown>) => Promise<unknown>;
}

// ─────────────────────────────────────────────────────────────
// 1. SQL DATABASE QUERY
// ─────────────────────────────────────────────────────────────

export const queryDatabase: Tool = {
  name: 'query_database',
  description: `Query the e-commerce database. Available tables:
- products (id, name, price, category, stock, description) - Product catalog
- orders (id, product_id, customer_name, quantity, total_price, status, created_at) - Customer orders
- customers (id, name, email, tier, total_spent, created_at) - Customer information

Only SELECT queries are allowed. Example: SELECT * FROM products WHERE category = 'laptops'`,
  parameters: {
    sql: {
      type: 'string',
      description: 'SQL SELECT query to execute',
      required: true,
    },
  },
  execute: async (args) => {
    const startTime = Date.now();
    const sql = args.sql as string;

    try {
      const result = executeQuery(sql);

      span({
        type: 'tool',
        name: 'sqlite-query',
        input: { sql },
        output: { rowCount: result.rows.length, columns: result.columns },
        durationMs: Date.now() - startTime,
        status: 'success',
      });

      return {
        success: true,
        rowCount: result.rows.length,
        columns: result.columns,
        rows: result.rows,
      };
    } catch (error) {
      span({
        type: 'tool',
        name: 'sqlite-query',
        input: { sql },
        output: { error: error instanceof Error ? error.message : 'Unknown error' },
        durationMs: Date.now() - startTime,
        status: 'error',
        errorMessage: error instanceof Error ? error.message : 'Unknown error',
      });

      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  },
};

// ─────────────────────────────────────────────────────────────
// 2. DOCUMENT SEARCH (Semantic)
// ─────────────────────────────────────────────────────────────

export const searchKnowledgeBase: Tool = {
  name: 'search_knowledge_base',
  description: `Search through company knowledge base documents. Covers:
- Policies (refund, shipping, warranty)
- Product information and specifications
- Support articles and troubleshooting guides
- FAQ and payment information

Returns relevant documents with similarity scores.`,
  parameters: {
    query: {
      type: 'string',
      description: 'Natural language search query',
      required: true,
    },
    limit: {
      type: 'number',
      description: 'Maximum number of results to return (1-10)',
      default: 5,
    },
  },
  execute: async (args) => {
    const startTime = Date.now();
    const query = args.query as string;
    const limit = (args.limit as number) || 5;

    // Simulate embedding generation delay
    await new Promise(resolve => setTimeout(resolve, 50));

    span({
      type: 'embedding',
      name: 'query-embedding',
      input: { query, queryLength: query.length },
      output: { dimensions: 1536 }, // Simulated embedding dimensions
      durationMs: 50,
      status: 'success',
    });

    // Perform search
    const results = searchDocuments(query, limit);

    span({
      type: 'retrieval',
      name: 'vector-search',
      input: { query, limit },
      output: {
        resultsCount: results.length,
        topScore: results[0]?.score || 0,
        categories: [...new Set(results.map(r => r.category))],
      },
      durationMs: Date.now() - startTime,
      status: 'success',
    });

    return {
      success: true,
      query,
      resultsCount: results.length,
      results: results.map(r => ({
        content: r.text,
        source: r.source,
        category: r.category,
        relevanceScore: r.score,
      })),
    };
  },
};

// ─────────────────────────────────────────────────────────────
// 3. HTTP REQUEST
// ─────────────────────────────────────────────────────────────

const ALLOWED_DOMAINS = [
  'api.github.com',
  'jsonplaceholder.typicode.com',
  'httpbin.org',
  'api.quotable.io',
];

export const httpRequest: Tool = {
  name: 'http_request',
  description: `Make HTTP requests to external APIs. Allowed domains:
- api.github.com - GitHub API (repos, users, etc.)
- jsonplaceholder.typicode.com - Fake REST API for testing
- httpbin.org - HTTP testing service
- api.quotable.io - Random quotes API

Use for fetching real-time data from external services.`,
  parameters: {
    method: {
      type: 'string',
      description: 'HTTP method (GET or POST)',
      enum: ['GET', 'POST'],
      required: true,
    },
    url: {
      type: 'string',
      description: 'Full URL to request',
      required: true,
    },
    body: {
      type: 'object',
      description: 'Request body for POST requests (optional)',
    },
  },
  execute: async (args) => {
    const startTime = Date.now();
    const method = args.method as string;
    const url = args.url as string;
    const body = args.body as Record<string, unknown> | undefined;

    try {
      const parsedUrl = new URL(url);

      if (!ALLOWED_DOMAINS.includes(parsedUrl.hostname)) {
        span({
          type: 'tool',
          name: 'http-request',
          input: { method, url },
          output: { error: 'Domain not allowed' },
          durationMs: Date.now() - startTime,
          status: 'error',
          errorMessage: `Domain ${parsedUrl.hostname} not in allowlist`,
        });

        return {
          success: false,
          error: `Domain ${parsedUrl.hostname} not allowed. Allowed: ${ALLOWED_DOMAINS.join(', ')}`,
        };
      }

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'User-Agent': 'Lelemon-Playground/1.0',
        },
        body: body ? JSON.stringify(body) : undefined,
      });

      const data = await response.json();

      span({
        type: 'tool',
        name: 'http-request',
        input: { method, url, hasBody: !!body },
        output: { status: response.status, dataSize: JSON.stringify(data).length },
        durationMs: Date.now() - startTime,
        status: response.ok ? 'success' : 'error',
      });

      return {
        success: response.ok,
        status: response.status,
        statusText: response.statusText,
        data,
      };
    } catch (error) {
      span({
        type: 'tool',
        name: 'http-request',
        input: { method, url },
        output: { error: error instanceof Error ? error.message : 'Unknown error' },
        durationMs: Date.now() - startTime,
        status: 'error',
        errorMessage: error instanceof Error ? error.message : 'Unknown error',
      });

      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  },
};

// ─────────────────────────────────────────────────────────────
// 4. CALCULATOR
// ─────────────────────────────────────────────────────────────

export const calculate: Tool = {
  name: 'calculate',
  description: `Perform mathematical calculations. Supports:
- Basic operations: +, -, *, /
- Parentheses for order of operations
- Common functions: Math.sqrt, Math.pow, Math.abs, Math.round, Math.floor, Math.ceil
- Constants: Math.PI, Math.E

Example: "2 * (3 + 4)" or "Math.sqrt(16) + Math.pow(2, 3)"`,
  parameters: {
    expression: {
      type: 'string',
      description: 'Mathematical expression to evaluate',
      required: true,
    },
  },
  execute: async (args) => {
    const startTime = Date.now();
    const expression = args.expression as string;

    try {
      // Safe evaluation using Function constructor with Math object
      // Only allows mathematical operations, not arbitrary code
      const sanitized = expression.replace(/[^0-9+\-*/().Math\s\w]/g, '');

      // Create a safe evaluation function
      const safeEval = new Function('Math', `"use strict"; return (${sanitized})`);
      const result = safeEval(Math);

      span({
        type: 'tool',
        name: 'calculator',
        input: { expression },
        output: { result },
        durationMs: Date.now() - startTime,
        status: 'success',
      });

      return {
        success: true,
        expression,
        result,
      };
    } catch (error) {
      span({
        type: 'tool',
        name: 'calculator',
        input: { expression },
        output: { error: error instanceof Error ? error.message : 'Invalid expression' },
        durationMs: Date.now() - startTime,
        status: 'error',
        errorMessage: error instanceof Error ? error.message : 'Invalid expression',
      });

      return {
        success: false,
        error: error instanceof Error ? error.message : 'Invalid expression',
      };
    }
  },
};

// ─────────────────────────────────────────────────────────────
// 5. GET CURRENT TIME
// ─────────────────────────────────────────────────────────────

export const getCurrentTime: Tool = {
  name: 'get_current_time',
  description: 'Get the current date and time in various formats and timezones.',
  parameters: {
    timezone: {
      type: 'string',
      description: 'Timezone (e.g., "America/New_York", "Europe/London", "Asia/Tokyo"). Defaults to UTC.',
      default: 'UTC',
    },
    format: {
      type: 'string',
      description: 'Output format: "full", "date", "time", "iso"',
      enum: ['full', 'date', 'time', 'iso'],
      default: 'full',
    },
  },
  execute: async (args) => {
    const startTime = Date.now();
    const timezone = (args.timezone as string) || 'UTC';
    const format = (args.format as string) || 'full';

    try {
      const now = new Date();

      const options: Intl.DateTimeFormatOptions = {
        timeZone: timezone,
      };

      let formatted: string;
      switch (format) {
        case 'date':
          options.dateStyle = 'full';
          formatted = now.toLocaleDateString('en-US', options);
          break;
        case 'time':
          options.timeStyle = 'long';
          formatted = now.toLocaleTimeString('en-US', options);
          break;
        case 'iso':
          formatted = now.toISOString();
          break;
        default: // full
          options.dateStyle = 'full';
          options.timeStyle = 'long';
          formatted = now.toLocaleString('en-US', options);
      }

      span({
        type: 'tool',
        name: 'get-time',
        input: { timezone, format },
        output: { formatted },
        durationMs: Date.now() - startTime,
        status: 'success',
      });

      return {
        success: true,
        timestamp: now.getTime(),
        iso: now.toISOString(),
        formatted,
        timezone,
      };
    } catch (error) {
      span({
        type: 'tool',
        name: 'get-time',
        input: { timezone, format },
        output: { error: error instanceof Error ? error.message : 'Unknown error' },
        durationMs: Date.now() - startTime,
        status: 'error',
        errorMessage: error instanceof Error ? error.message : 'Unknown error',
      });

      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  },
};

// ─────────────────────────────────────────────────────────────
// Export all tools
// ─────────────────────────────────────────────────────────────

export const allTools: Tool[] = [
  queryDatabase,
  searchKnowledgeBase,
  httpRequest,
  calculate,
  getCurrentTime,
];

// Execute a tool by name
export async function executeTool(
  toolName: string,
  args: Record<string, unknown>
): Promise<unknown> {
  const tool = allTools.find(t => t.name === toolName);
  if (!tool) {
    throw new Error(`Tool "${toolName}" not found`);
  }
  return tool.execute(args);
}
