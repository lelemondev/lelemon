/**
 * Trace summarization for the MCP `get_trace` default (concise) view.
 *
 * The goal is READABILITY at low token cost: an LLM span's `input` is the full
 * provider request — `{ messages, system, tools, ... }` — where `system` (the
 * prompt) and `tools` (the tool schemas) are huge AND identical across every
 * span, while the actual conversation lives in `messages` (tiny). A real agent
 * turn measured ~75k chars of input of which only ~2.6k was the dialogue.
 *
 * So the concise view KEEPS the dialogue (`messages`) and the completion
 * (`output`), and replaces only the heavy static parts (`system`, `tools`, and
 * the redundant full-request copy the SDK records under `metadata.input`) with a
 * size placeholder. Result: a trace you can actually read drops from ~150k to a
 * few k chars. Pass detail:true to get the raw request verbatim.
 */

/** Heavy STATIC fields inside an LLM request `input` — identical across spans, useless for reading the dialogue. */
const HEAVY_STATIC_INPUT_FIELDS = ['system', 'tools'] as const;
/** Heavy payload fields inside a `metadata` record (the SDK copies the full request under metadata.input). */
const HEAVY_METADATA_FIELDS = ['input', 'output', 'thinking'] as const;

/** Caps so a single large completion / chain-of-thought / tool result can't blow up the concise view. */
const OUTPUT_CAP = 4000;
const THINKING_CAP = 2000;
const MESSAGE_BLOCK_CAP = 1500;
/** A non-chat `input` (e.g. a bare string prompt) larger than this is omitted; smaller is kept verbatim. */
const PLAIN_INPUT_KEEP_CAP = 4000;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function sizeOf(value: unknown): number {
  try {
    return JSON.stringify(value)?.length ?? 0;
  } catch {
    return 0;
  }
}

/**
 * Replace a heavy value with a compact placeholder that still tells the agent it
 * exists and roughly how big it is, so it can decide to re-fetch with detail:true.
 */
function omitHeavy(value: unknown): string {
  return `[omitted ~${sizeOf(value)} chars — fetch with detail:true]`;
}

/** Truncate a string to `cap`, noting how much was dropped. */
function truncateString(text: string, cap: number): string {
  if (text.length <= cap) return text;
  return `${text.slice(0, cap)}… [+${text.length - cap} chars — detail:true]`;
}

/** Keep a value if small; if it's a long string truncate it; if it's a big object/array, omit it. */
function capValue(value: unknown, cap: number): unknown {
  if (typeof value === 'string') return truncateString(value, cap);
  if (value == null) return value;
  return sizeOf(value) > cap ? omitHeavy(value) : value;
}

/** Truncate a single content block (text / tool_result) so a big payload can't bloat the dialogue. */
function slimContentBlock(block: unknown): unknown {
  if (!isRecord(block)) return block;
  if (typeof block['text'] === 'string') {
    return { ...block, text: truncateString(block['text'], MESSAGE_BLOCK_CAP) };
  }
  // tool_result / tool_use payloads live under `content` or `input`
  if ('content' in block) {
    const c = block['content'];
    const asText = typeof c === 'string' ? c : sizeOf(c) > MESSAGE_BLOCK_CAP ? JSON.stringify(c) : null;
    if (asText != null && asText.length > MESSAGE_BLOCK_CAP) {
      return { ...block, content: truncateString(asText, MESSAGE_BLOCK_CAP) };
    }
  }
  return block;
}

/** Slim one chat message: keep role, truncate oversized content blocks. */
function slimMessage(message: unknown): unknown {
  if (!isRecord(message)) return message;
  const content = message['content'];
  if (typeof content === 'string') return { ...message, content: truncateString(content, MESSAGE_BLOCK_CAP) };
  if (Array.isArray(content)) return { ...message, content: content.map(slimContentBlock) };
  return message;
}

/**
 * Concise an LLM span `input`. When it's a chat request (`{ messages, system,
 * tools, ... }`) keep the dialogue and drop the static system prompt + tool
 * schemas. Otherwise keep it if small, omit it if large.
 */
function conciseInput(value: unknown): unknown {
  if (isRecord(value) && Array.isArray(value['messages'])) {
    const out: Record<string, unknown> = { ...value };
    for (const field of HEAVY_STATIC_INPUT_FIELDS) {
      if (field in out) out[field] = omitHeavy(out[field]);
    }
    out['messages'] = (out['messages'] as unknown[]).map(slimMessage);
    return out;
  }
  return capValue(value, PLAIN_INPUT_KEEP_CAP);
}

/** Strip heavy payload fields (input/output/thinking) from a metadata-like record. */
export function conciseMetadata(metadata: unknown): unknown {
  if (!isRecord(metadata)) return metadata;
  const out: Record<string, unknown> = { ...metadata };
  for (const field of HEAVY_METADATA_FIELDS) {
    if (field in out) out[field] = omitHeavy(out[field]);
  }
  return out;
}

/**
 * Concise copy of one span-tree node: the dialogue (input.messages) and the
 * completion (output) are kept (capped); the static system prompt + tool schemas
 * and the redundant metadata.input copy are replaced with placeholders; children
 * are recursed.
 */
function conciseNode(node: unknown): unknown {
  if (!isRecord(node)) return node;

  const out: Record<string, unknown> = { ...node };

  if (isRecord(node['span'])) {
    const span: Record<string, unknown> = { ...node['span'] };
    if ('input' in span) span['input'] = conciseInput(span['input']);
    if ('output' in span) span['output'] = capValue(span['output'], OUTPUT_CAP);
    if ('thinking' in span) span['thinking'] = capValue(span['thinking'], THINKING_CAP);
    if ('metadata' in span) span['metadata'] = conciseMetadata(span['metadata']);
    out['span'] = span;
  }

  if (Array.isArray(node['children'])) {
    out['children'] = node['children'].map(conciseNode);
  }

  return out;
}

/**
 * Return a token-efficient, READABLE copy of a span tree: every span keeps its
 * id, type, name, model, tokens, cost, costBreakdown, the conversation messages
 * and the completion — but the static system prompt + tool schemas (and the
 * redundant metadata.input request copy) are replaced with size placeholders.
 * Use detail:true on the tool to keep the raw request verbatim.
 */
export function conciseSpanTree(spanTree: unknown[]): unknown[] {
  return spanTree.map(conciseNode);
}
