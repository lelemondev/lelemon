import { describe, expect, it } from 'vitest';
import { conciseMetadata, conciseSpanTree } from './summarize.js';

describe('conciseSpanTree', () => {
  it('replaces heavy span payloads with a placeholder but keeps cost/tokens/costBreakdown and recurses children', () => {
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

    expect(span.input).toMatch(/^\[omitted ~\d+ chars — fetch with detail:true\]$/);
    expect(span.output).toMatch(/^\[omitted/);
    expect(span.thinking).toMatch(/^\[omitted/);
    // Placeholder is tiny — far smaller than the original payload.
    expect((span.input as string).length).toBeLessThan(60);
    expect(span.id).toBe('sp_1');
    expect(span.inputTokens).toBe(1000);
    expect(span.costBreakdown).toEqual({ total: 0.0123, cacheSavings: 0.004 });

    const child = (concise[0].children as Array<Record<string, unknown>>)[0];
    expect((child.span as Record<string, unknown>).output).toMatch(/^\[omitted/);
    expect((child.span as Record<string, unknown>).name).toBe('search');
  });

  it('strips heavy fields nested inside span.metadata (Lelemon SDK records input there)', () => {
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
    const tree = [{ span: { id: 'sp_1', input: 'keep me' }, children: [] }];
    conciseSpanTree(tree);
    expect(tree[0].span.input).toBe('keep me');
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
    expect((concise.input as string).length).toBeLessThan(60);
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
