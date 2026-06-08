import { describe, expect, it } from 'vitest';
import { conciseSpanTree } from './summarize.js';

describe('conciseSpanTree', () => {
  it('strips heavy span payloads but keeps cost/tokens/costBreakdown and recurses children', () => {
    const tree = [
      {
        span: {
          id: 'sp_1',
          type: 'llm',
          model: 'claude-opus-4-5',
          input: 'a very long prompt'.repeat(1000),
          output: 'a very long completion'.repeat(1000),
          thinking: 'lots of reasoning text',
          inputTokens: 1000,
          costUsd: 0.0123,
          costBreakdown: { total: 0.0123, cacheSavings: 0.004 },
        },
        children: [
          {
            span: { id: 'sp_2', type: 'tool', name: 'search', output: 'big tool result'.repeat(500) },
            children: [],
          },
        ],
      },
    ];

    const concise = conciseSpanTree(tree) as Array<Record<string, unknown>>;
    const span = concise[0].span as Record<string, unknown>;

    expect(span.input).toBeUndefined();
    expect(span.output).toBeUndefined();
    expect(span.thinking).toBeUndefined();
    expect(span.id).toBe('sp_1');
    expect(span.inputTokens).toBe(1000);
    expect(span.costBreakdown).toEqual({ total: 0.0123, cacheSavings: 0.004 });

    const child = (concise[0].children as Array<Record<string, unknown>>)[0];
    expect((child.span as Record<string, unknown>).output).toBeUndefined();
    expect((child.span as Record<string, unknown>).name).toBe('search');
  });

  it('does not mutate the original tree', () => {
    const tree = [{ span: { id: 'sp_1', input: 'keep me' }, children: [] }];
    conciseSpanTree(tree);
    expect(tree[0].span.input).toBe('keep me');
  });
});
