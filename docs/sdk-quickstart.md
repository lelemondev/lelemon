# Lelemon SDK

Low-friction LLM observability. 3 lines of code.

## Installation

```bash
npm install @lelemon/sdk
```

## Quick Start

```typescript
import { trace } from '@lelemon/sdk';

async function runAgent(userMessage: string) {
  const t = trace({ input: userMessage });

  try {
    const messages = [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: userMessage },
    ];

    const response = await openai.chat.completions.create({
      model: 'gpt-4',
      messages,
    });

    messages.push(response.choices[0].message);

    await t.success(messages);
    return response.choices[0].message.content;
  } catch (error) {
    await t.error(error, messages);
    throw error;
  }
}
```

That's it. Lelemon auto-detects:
- System prompt
- User input
- Tool calls and results
- Final output
- Token usage (if you log responses)

## Environment Variable

```bash
LELEMON_API_KEY=le_your_api_key
```

Or pass it directly:

```typescript
import { init } from '@lelemon/sdk';

init({ apiKey: 'le_xxx' });
```

## With Tool Calls

Lelemon automatically parses tool calls from the message history:

```typescript
import { trace } from '@lelemon/sdk';

async function agentWithTools(userMessage: string) {
  const t = trace({ input: userMessage });

  const messages = [
    { role: 'system', content: systemPrompt },
    { role: 'user', content: userMessage },
  ];

  try {
    while (true) {
      const response = await openai.chat.completions.create({
        model: 'gpt-4',
        messages,
        tools: myTools,
      });

      // Optional: log response to capture token usage
      t.log(response);

      const message = response.choices[0].message;
      messages.push(message);

      if (message.tool_calls) {
        for (const toolCall of message.tool_calls) {
          const result = await executeTool(toolCall);
          messages.push({
            role: 'tool',
            tool_call_id: toolCall.id,
            content: JSON.stringify(result),
          });
        }
      } else {
        await t.success(messages);
        return message.content;
      }
    }
  } catch (error) {
    await t.error(error, messages);
    throw error;
  }
}
```

## API Reference

### `trace(options)`

Start a new trace.

```typescript
const t = trace({
  input: userMessage,        // Required: initial input
  name: 'my-agent',          // Optional: trace name
  sessionId: 'session-123',  // Optional: group related traces
  userId: 'user-456',        // Optional: end user ID
  metadata: { ... },         // Optional: custom data
  tags: ['prod'],            // Optional: tags for filtering
});
```

### `t.success(messages)`

Complete trace successfully. Pass the full message history.

```typescript
await t.success(messages);
```

### `t.error(error, messages?)`

Complete trace with error. Messages are optional but helpful for debugging.

```typescript
await t.error(error, messages);
```

### `t.log(response)`

Optional: Log an LLM response to capture token usage.

```typescript
const response = await openai.chat.completions.create({ ... });
t.log(response);  // Extracts model, input/output tokens
```

### `init(config)`

Optional: Initialize SDK globally.

```typescript
import { init } from '@lelemon/sdk';

init({
  apiKey: 'le_xxx',
  endpoint: 'https://custom.endpoint.com',  // Optional
  debug: true,                               // Optional: log requests
  disabled: process.env.NODE_ENV === 'test', // Optional: disable in tests
});
```

## Supported Formats

Lelemon auto-detects message formats:

### OpenAI

```typescript
const messages = [
  { role: 'system', content: '...' },
  { role: 'user', content: '...' },
  { role: 'assistant', content: '...', tool_calls: [...] },
  { role: 'tool', tool_call_id: '...', content: '...' },
];
```

### Anthropic

```typescript
const messages = [
  { role: 'user', content: '...' },
  { role: 'assistant', content: [
    { type: 'text', text: '...' },
    { type: 'tool_use', id: '...', name: '...', input: {...} },
  ]},
  { role: 'user', content: [
    { type: 'tool_result', tool_use_id: '...', content: '...' },
  ]},
];
```

## What Gets Captured

| Data | Source |
|------|--------|
| Input | First user message |
| System prompt | System message |
| Tool calls | `tool_calls` in assistant messages |
| Tool results | Tool messages |
| Output | Last assistant message |
| Tokens | From `t.log(response)` calls |
| Duration | Auto-calculated |
| Errors | From `t.error()` |

## TypeScript Types

```typescript
import type {
  LelemonConfig,
  TraceOptions,
  ParsedTrace,
  ParsedLLMCall,
  ParsedToolCall,
} from '@lelemon/sdk';
```
