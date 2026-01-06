/**
 * OpenAI Agent
 *
 * Uses OpenAI SDK with GPT models
 * Supports tool use (function calling) with automatic execution loop
 */

import OpenAI from 'openai';
import type {
  ChatCompletionMessageParam,
  ChatCompletionTool,
  ChatCompletionToolMessageParam,
} from 'openai/resources/chat/completions';
import { init, observe, trace, span, flush, getTraceContext } from '@lelemondev/sdk/openai';
import { allTools, executeTool } from '../tools';
import type { AgentResult, AgentOptions } from './types';

const PROVIDER = 'openai';
const MODEL_ID = 'gpt-4o-mini';

// Convert our tool format to OpenAI format
function getOpenAITools(): ChatCompletionTool[] {
  return allTools.map(tool => ({
    type: 'function' as const,
    function: {
      name: tool.name,
      description: tool.description,
      parameters: {
        type: 'object',
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
    },
  }));
}

export async function runOpenAIAgent(message: string, options?: AgentOptions): Promise<AgentResult> {
  const startTime = Date.now();
  const sessionId = options?.sessionId;

  // Initialize SDK
  init({
    apiKey: process.env.LELEMON_API_KEY,
    endpoint: process.env.NEXT_PUBLIC_LELEMON_API_URL || 'http://localhost:8080',
    debug: true,
  });

  // Create observed OpenAI client
  const client = observe(
    new OpenAI({
      apiKey: process.env.OPENAI_API_KEY,
    }),
    { sessionId, userId: options?.userId }
  );

  let traceId: string | undefined;
  let finalResponse = '';
  const toolsUsed: string[] = [];

  try {
    const result = await trace({ name: 'openai-playground-agent', input: message }, async () => {
      traceId = getTraceContext()?.traceId;

      const messages: ChatCompletionMessageParam[] = [
        {
          role: 'system',
          content: `You are a helpful assistant with access to tools. Use them when needed to answer questions accurately.

Available tools:
- query_database: Query the e-commerce database (products, orders, customers)
- search_knowledge_base: Search company knowledge base for policies, product info, support articles
- http_request: Make HTTP requests to external APIs
- calculate: Perform mathematical calculations
- get_current_time: Get current date/time in various formats

Always use tools when they would help provide accurate information.`,
        },
        {
          role: 'user',
          content: message,
        },
      ];

      // Tool use loop
      let iterations = 0;
      const maxIterations = 5;

      while (iterations < maxIterations) {
        iterations++;

        const response = await client.chat.completions.create({
          model: MODEL_ID,
          messages,
          tools: getOpenAITools(),
          tool_choice: 'auto',
        });

        const assistantMessage = response.choices[0].message;

        // Add assistant response to history
        messages.push(assistantMessage);

        // Check if we need to execute tools
        if (assistantMessage.tool_calls && assistantMessage.tool_calls.length > 0) {
          for (const toolCall of assistantMessage.tool_calls) {
            // Skip non-function tool calls
            if (toolCall.type !== 'function') continue;
            const toolName = toolCall.function.name;
            toolsUsed.push(toolName);

            let toolResult: string;

            try {
              const args = JSON.parse(toolCall.function.arguments);
              const result = await executeTool(toolName, args);
              toolResult = JSON.stringify(result);
            } catch (error) {
              toolResult = JSON.stringify({
                error: error instanceof Error ? error.message : 'Unknown error',
              });
            }

            // Add tool result
            const toolMessage: ChatCompletionToolMessageParam = {
              role: 'tool',
              tool_call_id: toolCall.id,
              content: toolResult,
            };
            messages.push(toolMessage);
          }
        } else {
          // No more tool calls, extract final response
          finalResponse = assistantMessage.content || '';
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
