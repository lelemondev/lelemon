import { describe, expect, it } from 'vitest';
import { conciseMetadata, conciseSpanTree } from './summarize.js';

describe('conciseSpanTree', () => {
  it('keeps the dialogue (input.messages) and completion, drops the static system prompt + tool schemas', () => {
    const tree = [
      {
        span: {
          id: 'sp_1',
          type: 'llm',
          model: 'claude-haiku-4-5',
          input: {
            anthropic_version: 'bedrock-2023-05-31',
            max_tokens: 1024,
            temperature: 0.1,
            system: 'a very long system prompt'.repeat(2000),
            tools: ['a very long tool schema'.repeat(500)],
            messages: [
              { role: 'user', content: 'hola, tienen Toyota Hilux?' },
              { role: 'assistant', content: 'Hola, sí tenemos. ¿Para cuándo buscas?' },
            ],
          },
          output: 'respuesta corta al cliente',
          thinking: 'short reasoning',
          inputTokens: 1000,
          costUsd: 0.0123,
          costBreakdown: { total: 0.0123, cacheSavings: 0.004 },
        },
        children: [
          {
            span: { id: 'sp_2', type: 'tool', name: 'search', output: 'small tool result' },
            children: [],
          },
        ],
      },
    ];

    const concise = conciseSpanTree(tree) as Array<Record<string, unknown>>;
    const span = concise[0].span as Record<string, unknown>;
    const input = span.input as Record<string, unknown>;

    // Dialogue + completion are kept (the whole point: readable).
    expect(input.messages).toEqual([
      { role: 'user', content: 'hola, tienen Toyota Hilux?' },
      { role: 'assistant', content: 'Hola, sí tenemos. ¿Para cuándo buscas?' },
    ]);
    expect(span.output).toBe('respuesta corta al cliente');
    expect(span.thinking).toBe('short reasoning');
    // The huge static parts are omitted.
    expect(input.system).toMatch(/^\[omitted ~\d+ chars — fetch with detail:true\]$/);
    expect(input.tools).toMatch(/^\[omitted/);
    // Scalars kept.
    expect(input.temperature).toBe(0.1);
    expect(input.max_tokens).toBe(1024);
    // Cost/tokens preserved, children recursed.
    expect(span.id).toBe('sp_1');
    expect(span.inputTokens).toBe(1000);
    expect(span.costBreakdown).toEqual({ total: 0.0123, cacheSavings: 0.004 });
    const child = (concise[0].children as Array<Record<string, unknown>>)[0];
    expect((child.span as Record<string, unknown>).name).toBe('search');
  });

  it('truncates oversized completion and message blocks instead of dropping them', () => {
    const tree = [
      {
        span: {
          id: 'sp_1',
          input: {
            messages: [
              {
                role: 'tool',
                content: [{ type: 'tool_result', content: 'huge search result '.repeat(1000) }],
              },
            ],
          },
          output: 'long completion '.repeat(1000),
        },
        children: [],
      },
    ];

    const concise = conciseSpanTree(tree) as Array<Record<string, unknown>>;
    const span = concise[0].span as Record<string, unknown>;
    const block = ((span.input as Record<string, unknown>).messages as Array<Record<string, unknown>>)[0];
    const resultBlock = (block.content as Array<Record<string, unknown>>)[0];

    expect(span.output as string).toMatch(/… \[\+\d+ chars — detail:true\]$/);
    expect((span.output as string).length).toBeLessThan(4100);
    expect(resultBlock.content as string).toMatch(/… \[\+\d+ chars — detail:true\]$/);
  });

  it('truncates a large non-chat string input but keeps a small one verbatim', () => {
    const tree = [
      { span: { id: 'big', input: 'x'.repeat(10000) }, children: [] },
      { span: { id: 'small', input: 'just a short prompt' }, children: [] },
    ];

    const concise = conciseSpanTree(tree) as Array<Record<string, unknown>>;
    const bigInput = (concise[0].span as Record<string, unknown>).input as string;
    expect(bigInput).toMatch(/… \[\+\d+ chars — detail:true\]$/);
    expect(bigInput.length).toBeLessThan(4100);
    expect((concise[1].span as Record<string, unknown>).input).toBe('just a short prompt');
  });

  it('strips the redundant full-request copy nested in span.metadata.input', () => {
    const tree = [
      {
        span: {
          id: 'sp_1',
          metadata: { conversationId: 'conv-1', input: 'huge request payload'.repeat(1000) },
        },
        children: [],
      },
    ];

    const concise = conciseSpanTree(tree) as Array<Record<string, unknown>>;
    const meta = (concise[0].span as Record<string, unknown>).metadata as Record<string, unknown>;

    expect(meta.conversationId).toBe('conv-1');
    expect(meta.input).toMatch(/^\[omitted/);
  });

  it('does not mutate the original tree', () => {
    const tree = [{ span: { id: 'sp_1', input: { messages: [{ role: 'user', content: 'hi' }], system: 'x'.repeat(9000) } }, children: [] }];
    conciseSpanTree(tree);
    expect((tree[0].span.input as { system: string }).system.length).toBe(9000);
  });
});

describe('conciseMetadata', () => {
  it('replaces the heavy trace-level metadata.input with a placeholder', () => {
    const metadata = {
      feature: 'whatsapp-agent',
      _traceName: 'agent-turn',
      input: 'system prompt + history + tools'.repeat(2000),
    };

    const concise = conciseMetadata(metadata) as Record<string, unknown>;

    expect(concise.feature).toBe('whatsapp-agent');
    expect(concise._traceName).toBe('agent-turn');
    expect(concise.input).toMatch(/^\[omitted ~\d+ chars — fetch with detail:true\]$/);
  });

  it('passes through non-record metadata untouched', () => {
    expect(conciseMetadata(undefined)).toBeUndefined();
    expect(conciseMetadata(null)).toBeNull();
    expect(conciseMetadata('a string')).toBe('a string');
  });

  it('does not mutate the original metadata', () => {
    const metadata = { input: 'keep me' };
    conciseMetadata(metadata);
    expect(metadata.input).toBe('keep me');
  });
});
