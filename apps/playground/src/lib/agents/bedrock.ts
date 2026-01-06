/**
 * AWS Bedrock Agent
 *
 * Uses Bedrock Converse API with Claude models
 * Supports tool use with automatic execution loop
 */

import {
  BedrockRuntimeClient,
  ConverseCommand,
  type ContentBlock,
  type ToolConfiguration,
  type ToolUseBlock,
  type Message,
} from '@aws-sdk/client-bedrock-runtime';
import { init, observe, trace, span, flush, getTraceContext } from '@lelemondev/sdk/bedrock';
import { allTools, executeTool } from '../tools';
import type { AgentResult, AgentOptions } from './types';

const PROVIDER = 'bedrock';
const MODEL_ID = 'us.anthropic.claude-sonnet-4-5-20250929-v1:0';

// Convert our tool format to Bedrock format
function getBedrockTools(): ToolConfiguration {
  return {
    tools: allTools.map(tool => ({
      toolSpec: {
        name: tool.name,
        description: tool.description,
        inputSchema: {
          json: {
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
      },
    })),
  };
}

// Extract text from content blocks
function extractText(content: ContentBlock[] | undefined): string {
  if (!content) return '';
  return content
    .filter((block): block is ContentBlock & { text: string } => 'text' in block)
    .map(block => block.text)
    .join('\n');
}

// Extract tool use blocks
function extractToolUse(content: ContentBlock[] | undefined) {
  if (!content) return [];
  return content
    .filter((block): block is ContentBlock & { toolUse: NonNullable<ContentBlock['toolUse']> } =>
      'toolUse' in block && block.toolUse !== undefined
    )
    .map(block => block.toolUse);
}

export async function runBedrockAgent(message: string, options?: AgentOptions): Promise<AgentResult> {
  const startTime = Date.now();
  const sessionId = options?.sessionId;

  // Read credentials directly from .env file to avoid global env vars interference
  const fs = await import('fs');
  const path = await import('path');

  const envPath = path.join(process.cwd(), '.env');
  const envContent = fs.readFileSync(envPath, 'utf-8');
  const envVars: Record<string, string> = {};

  for (const line of envContent.split('\n')) {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith('#')) {
      const [key, ...valueParts] = trimmed.split('=');
      if (key && valueParts.length > 0) {
        envVars[key.trim()] = valueParts.join('=').trim().replace(/^["']|["']$/g, '');
      }
    }
  }

  const accessKeyId = envVars.AWS_ACCESS_KEY_ID;
  const secretAccessKey = envVars.AWS_SECRET_ACCESS_KEY;
  const region = envVars.AWS_REGION || 'us-east-1';

  if (!accessKeyId || !secretAccessKey) {
    throw new Error('AWS credentials not configured in .env file. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY');
  }

  console.log('[Bedrock] Using credentials from .env file:', {
    accessKeyId: accessKeyId.slice(0, 8) + '...',
    region,
    sessionId,
  });

  // Initialize SDK with API URL
  init({
    apiKey: process.env.LELEMON_API_KEY,
    endpoint: process.env.NEXT_PUBLIC_LELEMON_API_URL || 'http://localhost:8080',
    debug: true,
  });

  // Create observed Bedrock client with explicit credentials from .env
  // Pass sessionId to group traces together
  const client = observe(
    new BedrockRuntimeClient({
      region,
      credentials: {
        accessKeyId,
        secretAccessKey,
      },
    }),
    { sessionId, userId: options?.userId }
  );

  let traceId: string | undefined;
  let finalResponse = '';
  const toolsUsed: string[] = [];

  try {
    const result = await trace({ name: 'bedrock-playground-agent' }, async () => {
      // Get trace ID from context
      traceId = getTraceContext()?.traceId;

      // Initial messages
      const messages: Message[] = [
        {
          role: 'user',
          content: [{ text: message }],
        },
      ];

      // System prompt
      const systemPrompt = `You are a helpful assistant with access to tools. Use them when needed to answer questions accurately.

Available tools:
- query_database: Query the e-commerce database (products, orders, customers)
- search_knowledge_base: Search company knowledge base for policies, product info, support articles
- http_request: Make HTTP requests to external APIs
- calculate: Perform mathematical calculations
- get_current_time: Get current date/time in various formats

Always use tools when they would help provide accurate information. After using tools, provide a clear and helpful response based on the results.`;

      // Tool use loop (max 5 iterations to prevent infinite loops)
      let iterations = 0;
      const maxIterations = 5;

      while (iterations < maxIterations) {
        iterations++;

        // Call Bedrock
        const response = await client.send(
          new ConverseCommand({
            modelId: MODEL_ID,
            messages,
            system: [{ text: systemPrompt }],
            toolConfig: getBedrockTools(),
          })
        );

        const assistantContent = response.output?.message?.content || [];
        const stopReason = response.stopReason;

        // Add assistant message to history
        messages.push({
          role: 'assistant',
          content: assistantContent,
        });

        // Check if we need to execute tools
        if (stopReason === 'tool_use') {
          const toolUses = extractToolUse(assistantContent);

          // Execute each tool
          const toolResults: ContentBlock[] = [];

          for (const toolUse of toolUses) {
            const toolName = toolUse.name!;
            const toolInput = toolUse.input as Record<string, unknown>;

            toolsUsed.push(toolName);

            try {
              const result = await executeTool(toolName, toolInput);
              toolResults.push({
                toolResult: {
                  toolUseId: toolUse.toolUseId,
                  content: [{ text: JSON.stringify(result) }],
                  status: 'success',
                },
              });
            } catch (error) {
              toolResults.push({
                toolResult: {
                  toolUseId: toolUse.toolUseId,
                  content: [
                    {
                      text: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`,
                    },
                  ],
                  status: 'error',
                },
              });
            }
          }

          // Add tool results to messages
          messages.push({
            role: 'user',
            content: toolResults,
          });
        } else {
          // No more tool use, extract final response
          finalResponse = extractText(assistantContent);
          break;
        }
      }

      return finalResponse;
    });

    // Flush traces to backend
    await flush();

    return {
      response: result || 'No response generated',
      traceId,
      sessionId,
      provider: PROVIDER,
      model: MODEL_ID,
      durationMs: Date.now() - startTime,
      toolsUsed: [...new Set(toolsUsed)], // Dedupe
    };
  } catch (error) {
    // Record error span
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
