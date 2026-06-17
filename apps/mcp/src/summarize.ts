/**
 * Fields that can be arbitrarily large (full prompts, completions, chain of
 * thought). They appear both directly on a span and inside `metadata` objects
 * (the Lelemon SDK records the full LLM request under `metadata.input`). Dropped
 * in the default concise view so the agent gets the shape, cost and tokens
 * without blowing its context. Fetch them with detail:true.
 */
const HEAVY_FIELDS = ['input', 'output', 'thinking'] as const;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

/**
 * Replace a heavy value with a compact placeholder that still tells the agent it
 * exists and roughly how big it is, so it can decide to re-fetch with detail:true.
 */
function omitHeavy(value: unknown): string {
  let chars = 0;
  try {
    chars = JSON.stringify(value)?.length ?? 0;
  } catch {
    chars = 0;
  }
  return `[omitted ~${chars} chars — fetch with detail:true]`;
}

/** Strip heavy payload fields (input/output/thinking) from a metadata-like record. */
export function conciseMetadata(metadata: unknown): unknown {
  if (!isRecord(metadata)) return metadata;
  const out: Record<string, unknown> = { ...metadata };
  for (const field of HEAVY_FIELDS) {
    if (field in out) out[field] = omitHeavy(out[field]);
  }
  return out;
}

/** Concise copy of one span-tree node: heavy span I/O removed, metadata sanitized, children recursed. */
function conciseNode(node: unknown): unknown {
  if (!isRecord(node)) return node;

  const out: Record<string, unknown> = { ...node };

  if (isRecord(node['span'])) {
    const span: Record<string, unknown> = { ...node['span'] };
    for (const field of HEAVY_FIELDS) {
      if (field in span) span[field] = omitHeavy(span[field]);
    }
    if ('metadata' in span) span['metadata'] = conciseMetadata(span['metadata']);
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
 * payloads (on the span and inside its metadata) are replaced with a size
 * placeholder. Use detail:true on the tool to keep them.
 */
export function conciseSpanTree(spanTree: unknown[]): unknown[] {
  return spanTree.map(conciseNode);
}
