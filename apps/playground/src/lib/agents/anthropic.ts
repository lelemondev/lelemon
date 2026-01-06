/**
 * Anthropic Agent
 *
 * Uses Anthropic SDK directly with Claude models
 * Supports tool use with automatic execution loop
 */

import Anthropic from '@anthropic-ai/sdk';
import type {
  MessageParam,
  ContentBlock,
  ToolUseBlock,
  ToolResultBlockParam,
} from '@anthropic-ai/sdk/resources/messages';
import { init, observe, trace, span, flush, getTraceContext } from '@lelemondev/sdk/anthropic';
import { allTools, executeTool } from '../tools';
import type { AgentResult, AgentOptions } from './types';

const PROVIDER = 'anthropic';
const MODEL_ID = 'claude-sonnet-4-5-20250929';

// Convert our tool format to Anthropic format
function getAnthropicTools(): Anthropic.Tool[] {
  return allTools.map(tool => ({
    name: tool.name,
    description: tool.description,
    input_schema: {
      type: 'object' as const,
      properties: Object.fromEntries(
        Object.entries(tool.parameters).map(([key, param]) => [
          key,
          {
            type: param.type,
            description: param.description,
            ...(param.enum ? { enum: param.enum } : {}),
          },
        ])
      ),
      required: Object.entries(tool.parameters)
        .filter(([, param]) => param.required !== false)
        .map(([key]) => key),
    },
  }));
}

// Extract text from content blocks
function extractText(content: ContentBlock[]): string {
  return content
    .filter((block): block is Anthropic.TextBlock => block.type === 'text')
    .map(block => block.text)
    .join('\n');
}

// Extract tool use blocks
function extractToolUse(content: ContentBlock[]): ToolUseBlock[] {
  return content.filter(
    (block): block is ToolUseBlock => block.type === 'tool_use'
  );
}

export async function runAnthropicAgent(message: string, options?: AgentOptions): Promise<AgentResult> {
  const startTime = Date.now();
  const sessionId = options?.sessionId;

  // Initialize SDK
  init({
    apiKey: process.env.LELEMON_API_KEY,
    endpoint: process.env.NEXT_PUBLIC_LELEMON_API_URL || 'http://localhost:8080',
    debug: true,
  });

  // Create observed Anthropic client
  const client = observe(
    new Anthropic({
      apiKey: process.env.ANTHROPIC_API_KEY,
    }),
    { sessionId, userId: options?.userId }
  );

  let traceId: string | undefined;
  let finalResponse = '';
  const toolsUsed: string[] = [];

  try {
    const result = await trace({ name: 'anthropic-playground-agent', input: message }, async () => {
      traceId = getTraceContext()?.traceId;

      const messages: MessageParam[] = [
        {
          role: 'user',
          content: message,
        },
      ];

      const systemPrompt = `You are a helpful assistant with access to tools. Use them when needed to answer questions accurately.

Available tools:
- query_database: Query the e-commerce database (products, orders, customers)
- search_knowledge_base: Search company knowledge base for policies, product info, support articles
- http_request: Make HTTP requests to external APIs
- calculate: Perform mathematical calculations
- get_current_time: Get current date/time in various formats

Always use tools when they would help provide accurate information.`;

      // Tool use loop
      let iterations = 0;
      const maxIterations = 5;

      while (iterations < maxIterations) {
        iterations++;

        const response = await client.messages.create({
          model: MODEL_ID,
          max_tokens: 4096,
          system: systemPrompt,
          messages,
          tools: getAnthropicTools(),
        });

        // Add assistant response to history
        messages.push({
          role: 'assistant',
          content: response.content,
        });

        // Check if we need to execute tools
        if (response.stop_reason === 'tool_use') {
          const toolUses = extractToolUse(response.content);
          const toolResults: ToolResultBlockParam[] = [];

          for (const toolUse of toolUses) {
            toolsUsed.push(toolUse.name);

            try {
              const result = await executeTool(
                toolUse.name,
                toolUse.input as Record<string, unknown>
              );
              toolResults.push({
                type: 'tool_result',
                tool_use_id: toolUse.id,
                content: JSON.stringify(result),
              });
            } catch (error) {
              toolResults.push({
                type: 'tool_result',
                tool_use_id: toolUse.id,
                content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
                is_error: true,
              });
            }
          }

          // Add tool results
          messages.push({
            role: 'user',
            content: toolResults,
          });
        } else {
          // Extract final response
          finalResponse = extractText(response.content);
          break;
        }
      }

      return finalResponse;
    });

    await flush();

    return {
      response: result || 'No response generated',
      traceId,
      sessionId,
      provider: PROVIDER,
      model: MODEL_ID,
      durationMs: Date.now() - startTime,
      toolsUsed: [...new Set(toolsUsed)],
    };
  } catch (error) {
    span({
      type: 'custom',
      name: 'agent-error',
      input: { message },
      output: { error: error instanceof Error ? error.message : 'Unknown error' },
      status: 'error',
      errorMessage: error instanceof Error ? error.message : 'Unknown error',
    });

    await flush();
    throw error;
  }
}
