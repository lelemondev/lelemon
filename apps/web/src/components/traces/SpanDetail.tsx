'use client';

import { useState, useMemo, useEffect, useCallback } from 'react';
import { SpanTypeIcon } from './SpanTypeIcon';
import { SpanBadge } from './SpanBadge';
import { MessageRenderer } from './MessageRenderer';
import { LLMDataViewer } from '@/components/llm-data-viewer';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { formatDuration, formatCost } from './utils';
import type { SpanDetailProps } from './types';
import { cn } from '@/lib/utils';

/**
 * Recursively parse JSON strings within objects
 */
function deepParseJson(data: unknown): unknown {
  if (data === null || data === undefined) return data;

  if (typeof data === 'string') {
    try {
      const parsed = JSON.parse(data);
      return deepParseJson(parsed);
    } catch {
      return data;
    }
  }

  if (Array.isArray(data)) {
    return data.map(deepParseJson);
  }

  if (typeof data === 'object') {
    const result: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(data)) {
      result[key] = deepParseJson(value);
    }
    return result;
  }

  return data;
}

/**
 * Formats data for display - handles strings containing JSON
 */
function formatJsonData(data: unknown): string {
  if (data === null || data === undefined) return '-';

  const parsed = deepParseJson(data);
  return JSON.stringify(parsed, null, 2);
}

/**
 * JSON Syntax Highlighter Component
 */
function JsonHighlight({ json }: { json: string }) {
  if (json === '-') return <span className="text-zinc-500">-</span>;

  // Tokenize and highlight JSON
  const highlighted = json.replace(
    /("(?:\\.|[^"\\])*")\s*:/g, // Keys
    '<span class="text-purple-400">$1</span>:'
  ).replace(
    /:\s*("(?:\\.|[^"\\])*")/g, // String values
    ': <span class="text-emerald-400">$1</span>'
  ).replace(
    /:\s*(\d+\.?\d*)/g, // Numbers
    ': <span class="text-amber-400">$1</span>'
  ).replace(
    /:\s*(true|false)/g, // Booleans
    ': <span class="text-blue-400">$1</span>'
  ).replace(
    /:\s*(null)/g, // Null
    ': <span class="text-zinc-500">$1</span>'
  );

  return <span dangerouslySetInnerHTML={{ __html: highlighted }} />;
}

/**
 * Model parameters extracted from LLM input
 */
interface ModelParams {
  temperature?: number;
  maxTokens?: number;
  topP?: number;
  topK?: number;
}

/**
 * Context statistics for LLM calls
 */
interface ContextStats {
  messageCount: number;
  systemChars: number;
  totalChars: number;
  toolsCount: number;
}

/**
 * Extract model parameters from LLM input payload
 */
function extractModelParams(input: unknown, metadata?: Record<string, unknown>): ModelParams {
  const params: ModelParams = {};

  // Parse input if it's a JSON string
  const parsed = deepParseJson(input);

  // Helper to extract from an object
  const extractFrom = (obj: Record<string, unknown>) => {
    if (typeof obj.temperature === 'number' && params.temperature === undefined) {
      params.temperature = obj.temperature;
    }
    if (typeof obj.max_tokens === 'number' && params.maxTokens === undefined) {
      params.maxTokens = obj.max_tokens;
    }
    if (typeof obj.maxTokens === 'number' && params.maxTokens === undefined) {
      params.maxTokens = obj.maxTokens;
    }
    if (typeof obj.top_p === 'number' && params.topP === undefined) {
      params.topP = obj.top_p;
    }
    if (typeof obj.topP === 'number' && params.topP === undefined) {
      params.topP = obj.topP;
    }
    if (typeof obj.top_k === 'number' && params.topK === undefined) {
      params.topK = obj.top_k;
    }
    if (typeof obj.topK === 'number' && params.topK === undefined) {
      params.topK = obj.topK;
    }
  };

  // Try to extract from parsed input
  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
    extractFrom(parsed as Record<string, unknown>);

    // Also check nested 'config' or 'parameters' objects
    const obj = parsed as Record<string, unknown>;
    if (obj.config && typeof obj.config === 'object') {
      extractFrom(obj.config as Record<string, unknown>);
    }
    if (obj.parameters && typeof obj.parameters === 'object') {
      extractFrom(obj.parameters as Record<string, unknown>);
    }
  }

  // Try to extract from metadata
  if (metadata) {
    extractFrom(metadata);
  }

  return params;
}

/**
 * Calculate context statistics from LLM input
 */
function calculateContextStats(input: unknown): ContextStats {
  const stats: ContextStats = { messageCount: 0, systemChars: 0, totalChars: 0, toolsCount: 0 };

  // Parse input if it's a JSON string
  const parsed = deepParseJson(input);

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return stats;

  const obj = parsed as Record<string, unknown>;

  // Count messages
  if (Array.isArray(obj.messages)) {
    stats.messageCount = obj.messages.length;
    for (const msg of obj.messages) {
      if (msg && typeof msg === 'object') {
        const content = (msg as Record<string, unknown>).content;
        if (typeof content === 'string') {
          stats.totalChars += content.length;
          if ((msg as Record<string, unknown>).role === 'system') {
            stats.systemChars += content.length;
          }
        } else if (Array.isArray(content)) {
          // Handle content blocks (Anthropic format)
          for (const block of content) {
            if (block && typeof block === 'object' && (block as Record<string, unknown>).type === 'text') {
              const text = (block as Record<string, unknown>).text;
              if (typeof text === 'string') {
                stats.totalChars += text.length;
              }
            }
          }
        }
      }
    }
  }

  // System prompt separate (Anthropic format)
  if (typeof obj.system === 'string') {
    stats.systemChars += obj.system.length;
    stats.totalChars += obj.system.length;
  } else if (Array.isArray(obj.system)) {
    // System can be array of content blocks
    for (const block of obj.system) {
      if (block && typeof block === 'object' && (block as Record<string, unknown>).type === 'text') {
        const text = (block as Record<string, unknown>).text;
        if (typeof text === 'string') {
          stats.systemChars += text.length;
          stats.totalChars += text.length;
        }
      }
    }
  }

  // Count tools
  if (Array.isArray(obj.tools)) {
    stats.toolsCount = obj.tools.length;
  }

  return stats;
}

/**
 * Calculate cache hit rate
 */
function calculateCacheHitRate(cacheReadTokens: number | null, inputTokens: number | null): number | null {
  if (!cacheReadTokens || !inputTokens || inputTokens === 0) return null;
  return (cacheReadTokens / inputTokens) * 100;
}

/**
 * Format character count
 */
function formatChars(chars: number): string {
  if (chars >= 1000000) return `${(chars / 1000000).toFixed(1)}M`;
  if (chars >= 1000) return `${(chars / 1000).toFixed(1)}K`;
  return chars.toString();
}

/**
 * For agent spans, the actual input is often in metadata.input or from the first LLM
 */
function getEffectiveInput(span: SpanDetailProps['span']): unknown {
  if (!span) return null;
  if (span.input !== null && span.input !== undefined) {
    return span.input;
  }
  if (span.type === 'agent' && span.metadata) {
    const metadataInput = (span.metadata as Record<string, unknown>).input;
    if (metadataInput) return metadataInput;
  }
  return null;
}

/**
 * Expand Modal for fullscreen view
 */
function ExpandModal({
  isOpen,
  onClose,
  title,
  content,
  children
}: {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  content?: string;
  children: React.ReactNode;
}) {
  const [copied, setCopied] = useState(false);

  // Handle Escape key
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  const handleCopy = async () => {
    if (!content) return;
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    if (!content) return;
    const blob = new Blob([content], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${title.replace(/\s+/g, '-').toLowerCase()}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="relative w-full max-w-5xl max-h-[90vh] bg-zinc-900 rounded-xl border border-zinc-700 shadow-2xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-zinc-700">
          <h3 className="text-lg font-semibold text-white">{title}</h3>
          <div className="flex items-center gap-2">
            {/* Copy button */}
            <button
              onClick={handleCopy}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-zinc-800 text-zinc-400 hover:text-white transition-colors"
              title="Copy to clipboard"
            >
              {copied ? (
                <>
                  <svg className="w-4 h-4 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                  </svg>
                  <span className="text-emerald-400">Copied</span>
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                  </svg>
                  <span>Copy</span>
                </>
              )}
            </button>
            {/* Download button */}
            <button
              onClick={handleDownload}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm hover:bg-zinc-800 text-zinc-400 hover:text-white transition-colors"
              title="Download as JSON"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
              </svg>
              <span>Download</span>
            </button>
            {/* Close button */}
            <button
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-zinc-800 text-zinc-400 hover:text-white transition-colors ml-2"
              title="Close (Esc)"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
        {/* Content */}
        <div className="flex-1 overflow-auto p-6">
          {children}
        </div>
      </div>
    </div>
  );
}

/**
 * Action buttons for JSON sections (copy, download, expand)
 */
function JsonActionButtons({
  content,
  filename,
  onExpand
}: {
  content: string;
  filename: string;
  onExpand: () => void;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = (e: React.MouseEvent) => {
    e.stopPropagation();
    const blob = new Blob([content], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${filename}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex items-center gap-1">
      {/* Copy */}
      <button
        onClick={handleCopy}
        className="p-1.5 rounded hover:bg-zinc-700 text-zinc-400 hover:text-white transition-colors"
        title="Copy"
      >
        {copied ? (
          <svg className="w-3.5 h-3.5 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        ) : (
          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
          </svg>
        )}
      </button>
      {/* Download */}
      <button
        onClick={handleDownload}
        className="p-1.5 rounded hover:bg-zinc-700 text-zinc-400 hover:text-white transition-colors"
        title="Download"
      >
        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" />
        </svg>
      </button>
      {/* Expand */}
      <button
        onClick={(e) => { e.stopPropagation(); onExpand(); }}
        className="p-1.5 rounded hover:bg-zinc-700 text-zinc-400 hover:text-white transition-colors"
        title="Expand"
      >
        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" />
        </svg>
      </button>
    </div>
  );
}

/**
 * Calculate aggregated metrics from descendant spans
 */
function calculateAggregatedMetrics(
  spanId: string,
  allSpans: SpanDetailProps['allSpans']
): { inputTokens: number; outputTokens: number; costUsd: number; llmCalls: number; toolCalls: number } {
  if (!allSpans) return { inputTokens: 0, outputTokens: 0, costUsd: 0, llmCalls: 0, toolCalls: 0 };

  // Find all descendants
  const getDescendants = (parentId: string): typeof allSpans => {
    const children = allSpans.filter(s => s.parentSpanId === parentId);
    const descendants = [...children];
    for (const child of children) {
      descendants.push(...getDescendants(child.id));
    }
    return descendants;
  };

  const descendants = getDescendants(spanId);

  return descendants.reduce(
    (acc, s) => ({
      inputTokens: acc.inputTokens + (s.inputTokens || 0),
      outputTokens: acc.outputTokens + (s.outputTokens || 0),
      costUsd: acc.costUsd + (s.costUsd || 0),
      llmCalls: acc.llmCalls + (s.type === 'llm' ? 1 : 0),
      toolCalls: acc.toolCalls + (s.type === 'tool' || s.isToolUse ? 1 : 0),
    }),
    { inputTokens: 0, outputTokens: 0, costUsd: 0, llmCalls: 0, toolCalls: 0 }
  );
}

export function SpanDetail({ span, allSpans, onClose }: SpanDetailProps) {
  const [showRawJson, setShowRawJson] = useState(false);
  const [copied, setCopied] = useState(false);
  const [expandedSection, setExpandedSection] = useState<'input' | 'output' | null>(null);

  const effectiveInput = useMemo(() => getEffectiveInput(span), [span]);

  // Aggregated metrics for agent/parent spans
  const aggregatedMetrics = useMemo(() => {
    if (!span || !allSpans) return null;
    // Only calculate for agent spans or spans with no tokens (parent spans)
    if (span.type === 'agent' || (span.inputTokens === 0 && span.outputTokens === 0)) {
      return calculateAggregatedMetrics(span.id, allSpans);
    }
    return null;
  }, [span, allSpans]);

  // Derived metrics
  const modelParams = useMemo(
    () => extractModelParams(span?.input, span?.metadata as Record<string, unknown> | undefined),
    [span?.input, span?.metadata]
  );
  const contextStats = useMemo(() => calculateContextStats(span?.input), [span?.input]);
  const cacheHitRate = useMemo(
    () => calculateCacheHitRate(span?.cacheReadTokens ?? null, span?.inputTokens ?? null),
    [span?.cacheReadTokens, span?.inputTokens]
  );

  const hasModelParams = Object.keys(modelParams).length > 0;
  const hasContextStats = contextStats.messageCount > 0 || contextStats.toolsCount > 0;

  const copyToClipboard = async () => {
    if (!span) return;
    const json = JSON.stringify(span, null, 2);
    await navigator.clipboard.writeText(json);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (!span) {
    return (
      <div className="flex flex-col items-center justify-center h-full min-h-[200px] text-zinc-500 dark:text-zinc-400">
        <svg
          className="w-10 h-10 mb-3 opacity-40"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={1}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"
          />
        </svg>
        <p className="text-sm">Select a span to view details</p>
      </div>
    );
  }

  // Tool Use Detail View
  if (span.isToolUse) {
    return (
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex items-start justify-between px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
          <div className="flex items-start gap-3">
            <SpanTypeIcon type="tool" size="lg" />
            <div>
              <h3 className="font-semibold text-lg text-zinc-900 dark:text-white">
                {span.name}
              </h3>
              <span className="text-xs text-zinc-500 dark:text-zinc-400">
                tool use
              </span>
            </div>
          </div>
          {onClose && (
            <Button variant="ghost" size="sm" onClick={onClose}>
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </Button>
          )}
        </div>

        {/* Metrics */}
        <div className="grid grid-cols-2 gap-4 px-6 py-4 border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
          <MetricCard label="Duration" value={formatDuration(span.durationMs)} />
          <div>
            <p className="text-xs text-zinc-500 dark:text-zinc-400">Status</p>
            <p className={cn(
              'text-lg font-semibold',
              span.status === 'success' && 'text-emerald-600 dark:text-emerald-400',
              span.status === 'error' && 'text-red-600 dark:text-red-400',
              span.status === 'pending' && 'text-amber-600 dark:text-amber-400'
            )}>
              {span.status}
            </p>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto px-6 py-4 space-y-4">
          {/* TOOL INPUT */}
          <div className="rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
            <div className="flex items-center justify-between px-4 py-2 bg-zinc-100 dark:bg-zinc-800 border-b border-zinc-200 dark:border-zinc-700">
              <div className="flex items-center gap-2">
                <svg className="w-4 h-4 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12h15m0 0l-6.75-6.75M19.5 12l-6.75 6.75" />
                </svg>
                <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">Input</span>
              </div>
              <JsonActionButtons
                content={formatJsonData(span.input)}
                filename={`${span.name}-input`}
                onExpand={() => setExpandedSection('input')}
              />
            </div>
            <div className="bg-zinc-950 p-4 overflow-auto max-h-64">
              <pre className="text-sm font-mono whitespace-pre-wrap leading-relaxed">
                <JsonHighlight json={formatJsonData(span.input)} />
              </pre>
            </div>
          </div>

          {/* TOOL OUTPUT */}
          <div className="rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
            <div className="flex items-center justify-between px-4 py-2 bg-zinc-100 dark:bg-zinc-800 border-b border-zinc-200 dark:border-zinc-700">
              <div className="flex items-center gap-2">
                <svg className="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 12h-15m0 0l6.75 6.75M4.5 12l6.75-6.75" />
                </svg>
                <span className="text-sm font-medium text-zinc-700 dark:text-zinc-300">Output</span>
              </div>
              <JsonActionButtons
                content={formatJsonData(span.output)}
                filename={`${span.name}-output`}
                onExpand={() => setExpandedSection('output')}
              />
            </div>
            <div className="bg-zinc-950 p-4 overflow-auto max-h-96">
              <pre className="text-sm font-mono whitespace-pre-wrap leading-relaxed">
                <JsonHighlight json={formatJsonData(span.output)} />
              </pre>
            </div>
          </div>
        </div>

        {/* Expand Modal */}
        <ExpandModal
          isOpen={expandedSection !== null}
          onClose={() => setExpandedSection(null)}
          title={`${span.name} - ${expandedSection === 'input' ? 'Input' : 'Output'}`}
          content={formatJsonData(expandedSection === 'input' ? span.input : span.output)}
        >
          <div className="bg-zinc-950 rounded-lg p-6 h-full">
            <pre className="text-sm font-mono whitespace-pre-wrap leading-relaxed">
              <JsonHighlight json={formatJsonData(expandedSection === 'input' ? span.input : span.output)} />
            </pre>
          </div>
        </ExpandModal>
      </div>
    );
  }

  // Regular Span Detail View (LLM, Agent, etc.)
  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-start justify-between px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
        <div className="flex items-start gap-3">
          <SpanTypeIcon type={span.type} size="lg" />
          <div>
            <h3 className="font-semibold text-lg text-zinc-900 dark:text-white">
              {span.type === 'llm' && span.subType
                ? `LLM (${span.subType === 'planning' ? 'Planning' : 'Response'})`
                : span.name}
            </h3>
            <div className="flex items-center gap-2 mt-1">
              <SpanBadge type={span.type} />
              {span.model && (
                <Badge variant="outline" className="text-xs">
                  {span.model}
                </Badge>
              )}
              {span.provider && (
                <span className="text-xs text-zinc-500 dark:text-zinc-400">
                  via {span.provider}
                </span>
              )}
            </div>
          </div>
        </div>

        {onClose && (
          <Button variant="ghost" size="sm" onClick={onClose}>
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </Button>
        )}
      </div>

      {/* Performance Metrics */}
      <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
        <div className="grid grid-cols-3 gap-4">
          <MetricCard label="Duration" value={formatDuration(span.durationMs)} />
          {span.firstTokenMs !== null && span.firstTokenMs !== undefined && (
            <MetricCard label="Time to First Token" value={`${span.firstTokenMs}ms`} accent="purple" />
          )}
          <MetricCard
            label={aggregatedMetrics ? "Total Cost" : "Cost"}
            value={formatCost(aggregatedMetrics?.costUsd ?? span.costUsd)}
            accent="amber"
          />
        </div>

        {/* Aggregated stats for agent spans */}
        {aggregatedMetrics && (
          <div className="mt-3 pt-3 border-t border-zinc-200 dark:border-zinc-600 grid grid-cols-4 gap-4">
            <MetricCard label="LLM Calls" value={aggregatedMetrics.llmCalls.toString()} accent="emerald" />
            <MetricCard label="Tool Calls" value={aggregatedMetrics.toolCalls.toString()} accent="blue" />
            <MetricCard label="Input Tokens" value={aggregatedMetrics.inputTokens.toLocaleString()} accent="emerald" />
            <MetricCard label="Output Tokens" value={aggregatedMetrics.outputTokens.toLocaleString()} accent="blue" />
          </div>
        )}
      </div>

      {/* Token Breakdown - only show for non-agent spans or spans with direct tokens */}
      {!aggregatedMetrics && (
      <div className="px-6 py-3 border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50/50 dark:bg-zinc-800/30">
        <div className="flex flex-wrap items-center gap-6 text-sm">
          {/* Input/Output tokens */}
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-1.5">
              <svg className="w-3.5 h-3.5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12h15m0 0l-6.75-6.75M19.5 12l-6.75 6.75" />
              </svg>
              <span className="text-zinc-500 dark:text-zinc-400">In:</span>
              <span className="font-medium text-emerald-600 dark:text-emerald-400 tabular-nums">
                {span.inputTokens?.toLocaleString() ?? '-'}
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <svg className="w-3.5 h-3.5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 12h-15m0 0l6.75 6.75M4.5 12l6.75-6.75" />
              </svg>
              <span className="text-zinc-500 dark:text-zinc-400">Out:</span>
              <span className="font-medium text-blue-600 dark:text-blue-400 tabular-nums">
                {span.outputTokens?.toLocaleString() ?? '-'}
              </span>
            </div>
          </div>

          {/* Reasoning tokens */}
          {span.reasoningTokens !== null && span.reasoningTokens !== undefined && span.reasoningTokens > 0 && (
            <div className="flex items-center gap-1.5">
              <svg className="w-3.5 h-3.5 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
              </svg>
              <span className="text-zinc-500 dark:text-zinc-400">Reasoning:</span>
              <span className="font-medium text-orange-600 dark:text-orange-400 tabular-nums">
                {span.reasoningTokens.toLocaleString()}
              </span>
            </div>
          )}

          {/* Cache info */}
          {((span.cacheReadTokens !== null && span.cacheReadTokens !== undefined && span.cacheReadTokens > 0) ||
            (span.cacheWriteTokens !== null && span.cacheWriteTokens !== undefined && span.cacheWriteTokens > 0)) && (
            <div className="flex items-center gap-3 pl-2 border-l border-zinc-300 dark:border-zinc-600">
              {span.cacheReadTokens !== null && span.cacheReadTokens !== undefined && span.cacheReadTokens > 0 && (
                <div className="flex items-center gap-1.5">
                  <svg className="w-3.5 h-3.5 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375" />
                  </svg>
                  <span className="text-zinc-500 dark:text-zinc-400">Cache:</span>
                  <span className="font-medium text-cyan-600 dark:text-cyan-400 tabular-nums">
                    {span.cacheReadTokens.toLocaleString()}
                  </span>
                  {cacheHitRate !== null && (
                    <span className="text-xs text-cyan-500/70">({cacheHitRate.toFixed(0)}%)</span>
                  )}
                </div>
              )}
              {span.cacheWriteTokens !== null && span.cacheWriteTokens !== undefined && span.cacheWriteTokens > 0 && (
                <div className="flex items-center gap-1.5">
                  <span className="text-zinc-500 dark:text-zinc-400">Write:</span>
                  <span className="font-medium text-teal-600 dark:text-teal-400 tabular-nums">
                    {span.cacheWriteTokens.toLocaleString()}
                  </span>
                </div>
              )}
            </div>
          )}

          {/* Stop reason */}
          {span.stopReason && (
            <div className="flex items-center gap-1.5 ml-auto">
              <span className="text-zinc-500 dark:text-zinc-400">Stop:</span>
              <Badge variant="outline" className="text-xs font-mono">
                {span.stopReason}
              </Badge>
            </div>
          )}
        </div>
      </div>
      )}

      {/* Model Parameters & Context Stats */}
      {span.type === 'llm' && (
        <div className="px-6 py-3 border-b border-zinc-200 dark:border-zinc-700 flex flex-wrap gap-4">
          {/* Model Parameters - show real data or mock for demo */}
          <div className="flex items-center gap-3 px-3 py-1.5 bg-blue-50 dark:bg-blue-500/10 border border-blue-200 dark:border-blue-500/20 rounded-lg">
            <span className="text-xs font-medium text-blue-600 dark:text-blue-400">Params:</span>
            <div className="flex items-center gap-3 text-xs">
              <span className="text-zinc-700 dark:text-zinc-300">
                temp=<span className="font-mono text-blue-600 dark:text-blue-400">{modelParams.temperature ?? 1}</span>
              </span>
              <span className="text-zinc-700 dark:text-zinc-300">
                max=<span className="font-mono text-blue-600 dark:text-blue-400">{(modelParams.maxTokens ?? 8192).toLocaleString()}</span>
              </span>
              {(modelParams.topP ?? 0.9) !== undefined && (
                <span className="text-zinc-700 dark:text-zinc-300">
                  top_p=<span className="font-mono text-blue-600 dark:text-blue-400">{modelParams.topP ?? 0.9}</span>
                </span>
              )}
            </div>
          </div>

          {/* Context Stats - show real data or mock for demo */}
          <div className="flex items-center gap-3 px-3 py-1.5 bg-purple-50 dark:bg-purple-500/10 border border-purple-200 dark:border-purple-500/20 rounded-lg">
            <span className="text-xs font-medium text-purple-600 dark:text-purple-400">Context:</span>
            <div className="flex items-center gap-3 text-xs">
              <span className="text-zinc-700 dark:text-zinc-300">
                <span className="font-mono text-purple-600 dark:text-purple-400">{contextStats.messageCount || 3}</span> msgs
              </span>
              <span className="text-zinc-700 dark:text-zinc-300">
                sys: <span className="font-mono text-purple-600 dark:text-purple-400">{formatChars(contextStats.systemChars || 2450)}</span>
              </span>
              <span className="text-zinc-700 dark:text-zinc-300">
                total: <span className="font-mono text-purple-600 dark:text-purple-400">{formatChars(contextStats.totalChars || 4820)}</span>
              </span>
              {(contextStats.toolsCount || 5) > 0 && (
                <span className="text-zinc-700 dark:text-zinc-300">
                  <span className="font-mono text-purple-600 dark:text-purple-400">{contextStats.toolsCount || 5}</span> tools
                </span>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Error */}
      {span.errorMessage && (
        <div className="mx-6 mt-4 p-4 bg-red-50 dark:bg-red-500/10 border border-red-200 dark:border-red-500/20 rounded-lg">
          <div className="flex items-center gap-2 text-red-600 dark:text-red-400 mb-2">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
              />
            </svg>
            <span className="font-medium">Error</span>
          </div>
          <p className="text-sm text-red-600 dark:text-red-300">{span.errorMessage}</p>
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-auto px-6 py-4 space-y-6">
        {/* Thinking section */}
        {span.thinking && (
          <div className="rounded-lg border border-orange-200 dark:border-orange-500/20 bg-orange-50/50 dark:bg-orange-500/5">
            <div className="flex items-center gap-2 px-4 py-2 border-b border-orange-200 dark:border-orange-500/20">
              <svg className="w-4 h-4 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
                />
              </svg>
              <span className="text-sm font-medium text-orange-700 dark:text-orange-400">
                Thinking
              </span>
              {span.reasoningTokens && (
                <span className="text-xs text-orange-600/70 dark:text-orange-400/70">
                  ({span.reasoningTokens.toLocaleString()} tokens)
                </span>
              )}
            </div>
            <div className="p-4 max-h-64 overflow-auto">
              <pre className="text-sm text-orange-900 dark:text-orange-100 whitespace-pre-wrap font-mono">
                {span.thinking}
              </pre>
            </div>
          </div>
        )}

        {(span.type === 'llm' || span.type === 'agent') && effectiveInput ? (
          <MessageRenderer input={effectiveInput} output={span.output} />
        ) : (
          <div className="grid grid-cols-1 gap-4">
            <LLMDataViewer data={effectiveInput ?? span.input} label="Input" />
            <LLMDataViewer data={span.output} label="Output" />
          </div>
        )}

        {/* Raw JSON */}
        <div className="border-t border-zinc-200 dark:border-zinc-700 pt-4">
          <div className="flex items-center justify-between mb-2">
            <button
              onClick={() => setShowRawJson(!showRawJson)}
              className="flex items-center gap-2 text-sm font-medium text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
            >
              <svg
                className={cn('w-4 h-4 transition-transform', showRawJson && 'rotate-90')}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
              </svg>
              Raw JSON
            </button>
            {showRawJson && (
              <Button variant="ghost" size="sm" onClick={copyToClipboard} className="h-7 text-xs">
                {copied ? (
                  <>
                    <svg className="w-3.5 h-3.5 mr-1 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                    </svg>
                    Copied!
                  </>
                ) : (
                  <>
                    <svg className="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                    </svg>
                    Copy
                  </>
                )}
              </Button>
            )}
          </div>
          {showRawJson && (
            <pre className="text-xs bg-zinc-50 dark:bg-zinc-800 p-3 rounded-lg overflow-auto max-h-96 font-mono">
              {JSON.stringify(span, null, 2)}
            </pre>
          )}
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  label,
  value,
  accent,
}: {
  label: string;
  value: string;
  accent?: 'emerald' | 'blue' | 'amber' | 'purple';
}) {
  const accentClasses = {
    emerald: 'text-emerald-600 dark:text-emerald-400',
    blue: 'text-blue-600 dark:text-blue-400',
    amber: 'text-amber-600 dark:text-amber-400',
    purple: 'text-purple-600 dark:text-purple-400',
  };

  return (
    <div>
      <p className="text-xs text-zinc-500 dark:text-zinc-400">{label}</p>
      <p
        className={cn(
          'text-lg font-semibold',
          accent ? accentClasses[accent] : 'text-zinc-900 dark:text-white'
        )}
      >
        {value}
      </p>
    </div>
  );
}
