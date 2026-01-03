# SDK Quick Start Guide

## Installation

```bash
npm install @lelemon/sdk
# or
yarn add @lelemon/sdk
# or
pnpm add @lelemon/sdk
```

## Basic Usage

```typescript
import { LLMTracer } from '@lelemon/sdk';

// Initialize (reads LELEMON_API_KEY from env if not provided)
const tracer = new LLMTracer({
  apiKey: 'le_your_api_key',
});

// Create a trace for a conversation
const trace = await tracer.startTrace({
  sessionId: 'conv-123',
  userId: 'user-456',
});

// Record an LLM call
const span = trace.startSpan({
  type: 'llm',
  name: 'openai.chat',
  input: { messages: [{ role: 'user', content: 'Hello' }] },
});

// ... your LLM call here ...

span.end({
  output: { content: 'Hi there!' },
  model: 'gpt-4',
  inputTokens: 10,
  outputTokens: 5,
});

// End the trace
await trace.end();
```

## Integration Examples

### OpenAI

```typescript
import OpenAI from 'openai';
import { LLMTracer } from '@lelemon/sdk';

const openai = new OpenAI();
const tracer = new LLMTracer();

async function chat(messages: OpenAI.ChatCompletionMessageParam[]) {
  const trace = await tracer.startTrace({ sessionId: 'chat-session' });

  const span = trace.startSpan({
    type: 'llm',
    name: 'openai.chat',
    input: { messages },
  });

  try {
    const response = await openai.chat.completions.create({
      model: 'gpt-4',
      messages,
    });

    span.end({
      output: response.choices[0].message,
      model: response.model,
      inputTokens: response.usage?.prompt_tokens,
      outputTokens: response.usage?.completion_tokens,
    });

    await trace.end();
    return response.choices[0].message;
  } catch (error) {
    span.setError(error as Error);
    await trace.end({ status: 'error' });
    throw error;
  }
}
```

### Anthropic

```typescript
import Anthropic from '@anthropic-ai/sdk';
import { LLMTracer } from '@lelemon/sdk';

const anthropic = new Anthropic();
const tracer = new LLMTracer();

async function chat(prompt: string) {
  const trace = await tracer.startTrace();

  const span = trace.startSpan({
    type: 'llm',
    name: 'anthropic.messages',
    input: { prompt },
  });

  const response = await anthropic.messages.create({
    model: 'claude-3-5-sonnet-20241022',
    max_tokens: 1024,
    messages: [{ role: 'user', content: prompt }],
  });

  span.end({
    output: response.content,
    model: response.model,
    inputTokens: response.usage.input_tokens,
    outputTokens: response.usage.output_tokens,
  });

  await trace.end();
  return response;
}
```

### AWS Bedrock

```typescript
import { BedrockRuntimeClient, InvokeModelCommand } from '@aws-sdk/client-bedrock-runtime';
import { LLMTracer } from '@lelemon/sdk';

const bedrock = new BedrockRuntimeClient({ region: 'us-east-1' });
const tracer = new LLMTracer();

async function invokeClaude(prompt: string) {
  const trace = await tracer.startTrace();

  const span = trace.startSpan({
    type: 'llm',
    name: 'bedrock.invoke',
    input: { prompt },
  });

  const response = await bedrock.send(new InvokeModelCommand({
    modelId: 'anthropic.claude-3-sonnet-20240229-v1:0',
    body: JSON.stringify({
      anthropic_version: 'bedrock-2023-05-31',
      max_tokens: 1024,
      messages: [{ role: 'user', content: prompt }],
    }),
  }));

  const result = JSON.parse(new TextDecoder().decode(response.body));

  span.end({
    output: result.content,
    model: 'anthropic.claude-3-sonnet-20240229-v1:0',
    provider: 'bedrock',
    inputTokens: result.usage.input_tokens,
    outputTokens: result.usage.output_tokens,
  });

  await trace.end();
  return result;
}
```

## Tool Calls

```typescript
const span = trace.startSpan({
  type: 'tool',
  name: 'search_documents',
  input: { query: 'pricing information' },
});

const results = await searchDocuments(query);

span.end({
  output: { results },
  status: 'success',
});
```

## Nested Spans

```typescript
const parentSpan = trace.startSpan({
  type: 'custom',
  name: 'process-request',
});

// Child span
const childSpan = parentSpan.startSpan({
  type: 'llm',
  name: 'generate-response',
  input: { messages },
});

// ... LLM call ...

childSpan.end({ output: response });
parentSpan.end();
```

## Error Handling

```typescript
const span = trace.startSpan({ type: 'llm', name: 'chat' });

try {
  const response = await llm.call();
  span.end({ output: response, status: 'success' });
} catch (error) {
  span.setError(error as Error);
  // or manually:
  // span.end({ status: 'error', errorMessage: error.message });
}
```

## Serverless / Edge

Always flush before the function ends:

```typescript
export async function handler(event) {
  const tracer = new LLMTracer();
  const trace = await tracer.startTrace();

  // ... your logic ...

  await trace.end();
  await tracer.flush(); // Important for serverless!

  return response;
}
```

## Configuration Options

```typescript
const tracer = new LLMTracer({
  // Required
  apiKey: 'le_xxx',

  // Optional
  endpoint: 'https://your-instance.vercel.app', // Custom endpoint
  debug: true,                                   // Enable console logging
  batchSize: 10,                                 // Spans per batch
  flushInterval: 1000,                           // Batch interval (ms)
  disabled: process.env.NODE_ENV === 'test',    // Disable in tests
});
```

## Metadata & Tags

```typescript
const trace = await tracer.startTrace({
  sessionId: 'session-123',
  userId: 'user-456',
  metadata: {
    environment: 'production',
    version: '1.2.3',
    customField: 'any value',
  },
  tags: ['production', 'high-priority', 'experiment-a'],
});

// Add metadata later
trace.setMetadata('responseTime', 1234);
trace.addTag('cached');
```

## TypeScript Types

```typescript
import type {
  LelemonConfig,
  TraceOptions,
  SpanOptions,
  SpanEndOptions,
  TraceStatus,
  SpanType,
  SpanStatus,
} from '@lelemon/sdk';
```
