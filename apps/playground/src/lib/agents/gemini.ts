/**
 * Google Gemini Agent
 *
 * Uses Google Generative AI SDK with Gemini models
 * Supports function calling with automatic execution loop
 */

import { GoogleGenerativeAI, SchemaType } from '@google/generative-ai';
import type {
  Content,
  FunctionDeclaration,
  Part,
  FunctionCall,
} from '@google/generative-ai';
import { init, observe, trace, span, flush, getTraceContext } from '@lelemondev/sdk/gemini';
import { allTools, executeTool } from '../tools';
import type { AgentResult, AgentOptions } from './types';

const PROVIDER = 'gemini';
const MODEL_ID = 'gemini-2.0-flash';

// Map our type to Gemini schema type
function mapToGeminiType(type: string): SchemaType {
  switch (type) {
    case 'string':
      return SchemaType.STRING;
    case 'number':
      return SchemaType.NUMBER;
    case 'boolean':
      return SchemaType.BOOLEAN;
    case 'object':
      return SchemaType.OBJECT;
    case 'array':
      return SchemaType.ARRAY;
    default:
      return SchemaType.STRING;
  }
}

// Convert our tool format to Gemini format
function getGeminiFunctionDeclarations(): FunctionDeclaration[] {
  return allTools.map(tool => ({
    name: tool.name,
    description: tool.description,
    parameters: {
      type: SchemaType.OBJECT,
      properties: Object.fromEntries(
        Object.entries(tool.parameters).map(([key, param]) => [
          key,
          {
            type: mapToGeminiType(param.type),
            description: param.description,
            ...(param.enum ? { enum: param.enum } : {}),
          },
        ])
      ),
      required: Object.entries(tool.parameters)
        .filter(([, param]) => param.required !== false)
        .map(([key]) => key),
    },
  })) as FunctionDeclaration[];
}

// Extract text from parts
function extractText(parts: Part[]): string {
  return parts
    .filter((part): part is Part & { text: string } => 'text' in part)
    .map(part => part.text)
    .join('\n');
}

// Extract function calls
function extractFunctionCalls(parts: Part[]): FunctionCall[] {
  return parts
    .filter((part): part is Part & { functionCall: FunctionCall } => 'functionCall' in part)
    .map(part => part.functionCall);
}

export async function runGeminiAgent(message: string, options?: AgentOptions): Promise<AgentResult> {
  const startTime = Date.now();
  const sessionId = options?.sessionId;

  // Initialize SDK
  init({
    apiKey: process.env.LELEMON_API_KEY,
    endpoint: process.env.NEXT_PUBLIC_LELEMON_API_URL || 'http://localhost:8080',
    debug: true,
  });

  // Create Gemini client and observe the model
  const genAI = new GoogleGenerativeAI(process.env.GEMINI_API_KEY!);
  const model = observe(
    genAI.getGenerativeModel({
      model: MODEL_ID,
      systemInstruction: `You are a helpful assistant with access to tools. Use them when needed to answer questions accurately.

Available tools:
- query_database: Query the e-commerce database (products, orders, customers)
- search_knowledge_base: Search company knowledge base for policies, product info, support articles
- http_request: Make HTTP requests to external APIs
- calculate: Perform mathematical calculations
- get_current_time: Get current date/time in various formats

Always use tools when they would help provide accurate information.`,
    }),
    { sessionId, userId: options?.userId }
  );

  let traceId: string | undefined;
  let finalResponse = '';
  const toolsUsed: string[] = [];

  try {
    const result = await trace({ name: 'gemini-playground-agent' }, async () => {
      traceId = getTraceContext()?.traceId;

      // Start a chat session with function declarations
      const chat = model.startChat({
        tools: [{ functionDeclarations: getGeminiFunctionDeclarations() }],
      });

      // Send initial message
      let response = await chat.sendMessage(message);
      let parts = response.response.candidates?.[0]?.content?.parts || [];

      // Tool use loop
      let iterations = 0;
      const maxIterations = 5;

      while (iterations < maxIterations) {
        iterations++;

        const functionCalls = extractFunctionCalls(parts);

        if (functionCalls.length === 0) {
          // No function calls, extract final response
          finalResponse = extractText(parts);
          break;
        }

        // Execute function calls
        const functionResponses: Part[] = [];

        for (const functionCall of functionCalls) {
          const toolName = functionCall.name;
          toolsUsed.push(toolName);

          try {
            const result = await executeTool(toolName, functionCall.args as Record<string, unknown>);
            functionResponses.push({
              functionResponse: {
                name: toolName,
                response: result as object,
              },
            });
          } catch (error) {
            functionResponses.push({
              functionResponse: {
                name: toolName,
                response: {
                  error: error instanceof Error ? error.message : 'Unknown error',
                },
              },
            });
          }
        }

        // Send function responses back
        response = await chat.sendMessage(functionResponses);
        parts = response.response.candidates?.[0]?.content?.parts || [];
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
