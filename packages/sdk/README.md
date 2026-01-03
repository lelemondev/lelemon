# @lelemon/sdk

Low-friction LLM observability. **3 lines of code.**

```typescript
import { trace } from '@lelemon/sdk';

const t = trace({ input: userMessage });
try {
  // ... your agent code ...
  await t.success(messages);
} catch (error) {
  await t.error(error, messages);
}
```

## Installation

```bash
npm install @lelemon/sdk
# or
yarn add @lelemon/sdk
# or
pnpm add @lelemon/sdk
```

## Setup

Set your API key:

```bash
export LELEMON_API_KEY=le_your_api_key
```

Or configure programmatically:

```typescript
import { init } from '@lelemon/sdk';

init({ apiKey: 'le_xxx' });
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

    // Optional: log response to capture token usage
    t.log(response);

    messages.push(response.choices[0].message);

    await t.success(messages);
    return response.choices[0].message.content;
  } catch (error) {
    await t.error(error, messages);
    throw error;
  }
}
```

## Supported Providers

| Provider | Message Format | Auto-detected |
|----------|---------------|---------------|
| **OpenAI** | `role: 'user' \| 'assistant'` | Yes |
| **Anthropic** | `role: 'user' \| 'assistant'` | Yes |
| **Gemini** | `role: 'user' \| 'model'` | Yes |
| **AWS Bedrock** | Anthropic format | Yes |

## API Reference

### `trace(options)`

Start a new trace.

```typescript
const t = trace({
  input: userMessage,        // Required
  name: 'my-agent',          // Optional
  sessionId: 'session-123',  // Optional
  userId: 'user-456',        // Optional
  metadata: { ... },         // Optional
  tags: ['prod'],            // Optional
});
```

### `t.success(messages)`

Complete trace successfully.

```typescript
await t.success(messages);
```

### `t.error(error, messages?)`

Complete trace with error.

```typescript
await t.error(error, messages);
```

### `t.log(response)`

Log an LLM response to capture tokens (optional).

```typescript
t.log(response);
```

### `init(config)`

Initialize SDK globally (optional).

```typescript
init({
  apiKey: 'le_xxx',
  endpoint: 'https://custom.endpoint.com',
  debug: true,
  disabled: process.env.NODE_ENV === 'test',
});
```

## What Gets Captured

- System prompt
- User input
- Tool calls and results
- Final output
- Token usage
- Duration
- Errors with stack traces

## License

MIT
