/**
 * Span fields that can be arbitrarily large (full prompts, completions, chain of
 * thought). Dropped in the default concise view of a trace so the agent gets the
 * shape, cost and tokens without blowing its context. Fetch them with detail:true.
 */
const HEAVY_SPAN_FIELDS = ['input', 'output', 'thinking'] as const;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

/** Concise copy of one span-tree node: heavy span I/O removed, children recursed. */
function conciseNode(node: unknown): unknown {
  if (!isRecord(node)) return node;

  const out: Record<string, unknown> = { ...node };

  if (isRecord(node['span'])) {
    const span: Record<string, unknown> = { ...node['span'] };
    for (const field of HEAVY_SPAN_FIELDS) delete span[field];
    out['span'] = span;
  }

  if (Array.isArray(node['children'])) {
    out['children'] = node['children'].map(conciseNode);
  }

  return out;
}

/**
 * Return a token-efficient copy of a span tree: every span keeps its id, type,
 * name, model, tokens, cost and costBreakdown, but its large input/output/thinking
 * payloads are stripped. Use detail:true on the tool to keep them.
 */
export function conciseSpanTree(spanTree: unknown[]): unknown[] {
  return spanTree.map(conciseNode);
}
