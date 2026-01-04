'use client';

import { use, useEffect, useState, useCallback, useRef, useMemo } from 'react';
import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LLMDataViewer } from '@/components/llm-data-viewer';

const POLLING_INTERVAL = 2000; // 2 seconds

// Calculate metrics from spans (more accurate than pre-calculated values)
function calculateMetricsFromSpans(spans: Span[]) {
  return spans.reduce(
    (acc, span) => ({
      totalSpans: acc.totalSpans + 1,
      totalTokens: acc.totalTokens + (span.inputTokens || 0) + (span.outputTokens || 0),
      totalCostUsd: acc.totalCostUsd + (span.costUsd ? parseFloat(span.costUsd) : 0),
      totalDurationMs: acc.totalDurationMs + (span.durationMs || 0),
    }),
    { totalSpans: 0, totalTokens: 0, totalCostUsd: 0, totalDurationMs: 0 }
  );
}

interface Span {
  id: string;
  type: 'llm' | 'tool' | 'retrieval' | 'custom';
  name: string;
  model?: string;
  provider?: string;
  inputTokens?: number;
  outputTokens?: number;
  durationMs: number;
  costUsd?: string;
  status: 'success' | 'error' | 'pending';
  errorMessage?: string;
  input?: unknown;
  output?: unknown;
  startedAt: string;
  endedAt?: string;
}

interface Trace {
  id: string;
  sessionId: string | null;
  userId: string | null;
  status: 'active' | 'completed' | 'error';
  totalSpans: number;
  totalTokens: number;
  totalCostUsd: string;
  totalDurationMs: number;
  tags: string[] | null;
  metadata: Record<string, unknown>;
  createdAt: string;
  spans: Span[];
}

function formatDuration(ms: number | null): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function SpanTypeIcon({ type }: { type: Span['type'] }) {
  const iconConfig = {
    llm: { icon: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09zM18.259 8.715L18 9.75l-.259-1.035a3.375 3.375 0 00-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 002.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 002.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 00-2.456 2.456zM16.894 20.567L16.5 21.75l-.394-1.183a2.25 2.25 0 00-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 001.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 001.423 1.423l1.183.394-1.183.394a2.25 2.25 0 00-1.423 1.423z', color: 'text-purple-500' },
    tool: { icon: 'M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z', color: 'text-blue-500' },
    retrieval: { icon: 'M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z', color: 'text-green-500' },
    custom: { icon: 'M4.5 12a7.5 7.5 0 0015 0m-15 0a7.5 7.5 0 1115 0m-15 0H3m16.5 0H21m-1.5 0H12m-8.457 3.077l1.41-.513m14.095-5.13l1.41-.513M5.106 17.785l1.15-.964m11.49-9.642l1.149-.964M7.501 19.795l.75-1.3m7.5-12.99l.75-1.3m-6.063 16.658l.26-1.477m2.605-14.772l.26-1.477m0 17.726l-.26-1.477M10.698 4.614l-.26-1.477M16.5 19.794l-.75-1.299M7.5 4.205L12 12m6.894 5.785l-1.149-.964M6.256 7.178l-1.15-.964m15.352 8.864l-1.41-.513M4.954 9.435l-1.41-.514M12.002 12l-3.75 6.495', color: 'text-zinc-500' },
  };
  const { icon, color } = iconConfig[type];
  return (
    <svg className={`w-5 h-5 ${color}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d={icon} />
    </svg>
  );
}

export default function TraceDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [trace, setTrace] = useState<Trace | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedSpans, setExpandedSpans] = useState<Set<string>>(new Set());
  const [isPolling, setIsPolling] = useState(false);
  const pollingRef = useRef<NodeJS.Timeout | null>(null);

  const fetchTrace = useCallback(async (isInitial = false) => {
    try {
      const response = await fetch(`/api/v1/dashboard/traces/${id}`);
      if (response.ok) {
        const data = await response.json();
        setTrace(data);
        // Return whether we should continue polling
        return data.status === 'active';
      } else if (response.status === 404) {
        setError('Trace not found');
        return false;
      } else {
        if (isInitial) setError('Failed to load trace');
        return false;
      }
    } catch (err) {
      if (isInitial) setError('Failed to load trace');
      return false;
    } finally {
      if (isInitial) setIsLoading(false);
    }
  }, [id]);

  // Initial fetch and polling setup
  useEffect(() => {
    let isMounted = true;

    const startPolling = async () => {
      const shouldPoll = await fetchTrace(true);

      if (shouldPoll && isMounted) {
        setIsPolling(true);
        pollingRef.current = setInterval(async () => {
          if (!isMounted) return;
          const continuePolling = await fetchTrace(false);
          if (!continuePolling && isMounted) {
            setIsPolling(false);
            if (pollingRef.current) {
              clearInterval(pollingRef.current);
              pollingRef.current = null;
            }
          }
        }, POLLING_INTERVAL);
      }
    };

    startPolling();

    return () => {
      isMounted = false;
      if (pollingRef.current) {
        clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [fetchTrace]);

  const toggleSpan = (spanId: string) => {
    setExpandedSpans((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) {
        next.delete(spanId);
      } else {
        next.add(spanId);
      }
      return next;
    });
  };

  // Calculate metrics from actual spans (always accurate)
  const metrics = useMemo(() => {
    if (!trace) return { totalSpans: 0, totalTokens: 0, totalCostUsd: 0, totalDurationMs: 0 };
    return calculateMetricsFromSpans(trace.spans);
  }, [trace]);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 bg-zinc-200 dark:bg-zinc-800 rounded animate-pulse" />
        <div className="grid gap-4 md:grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-24 bg-zinc-200 dark:bg-zinc-800 rounded-xl animate-pulse" />
          ))}
        </div>
        <div className="h-64 bg-zinc-200 dark:bg-zinc-800 rounded-xl animate-pulse" />
      </div>
    );
  }

  if (error || !trace) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <div className="w-16 h-16 rounded-full bg-red-100 dark:bg-red-500/10 flex items-center justify-center mb-4">
          <svg className="w-8 h-8 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">{error || 'Trace not found'}</h3>
        <Link href="/dashboard/traces">
          <Button variant="outline" className="mt-4">
            Back to Traces
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/dashboard/traces">
            <Button variant="ghost" size="sm">
              <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
              </svg>
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-zinc-900 dark:text-white">
              Trace {id.slice(0, 8)}
            </h1>
            <p className="text-sm text-zinc-500 dark:text-zinc-400">
              Session: {trace.sessionId || '-'} | User: {trace.userId || 'Anonymous'}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isPolling && (
            <div className="flex items-center gap-1.5 px-2 py-1 bg-emerald-500/10 border border-emerald-500/20 rounded-full">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
              </span>
              <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400">LIVE</span>
            </div>
          )}
          {trace.tags?.map((tag) => (
            <Badge key={tag} variant="secondary">
              {tag}
            </Badge>
          ))}
          <Badge
            variant={
              trace.status === 'completed'
                ? 'default'
                : trace.status === 'error'
                ? 'destructive'
                : 'secondary'
            }
          >
            {trace.status}
          </Badge>
        </div>
      </div>

      {/* Summary - calculated from actual spans */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
              Duration
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-zinc-900 dark:text-white">
              {formatDuration(metrics.totalDurationMs)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
              Spans
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-zinc-900 dark:text-white">
              {metrics.totalSpans}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
              Tokens
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-zinc-900 dark:text-white">
              {metrics.totalTokens.toLocaleString()}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-500 dark:text-zinc-400">
              Cost
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-amber-600 dark:text-amber-400">
              ${metrics.totalCostUsd.toFixed(4)}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Spans Timeline */}
      <Card>
        <CardHeader>
          <CardTitle>Spans Timeline</CardTitle>
        </CardHeader>
        <CardContent>
          {trace.spans.length === 0 ? (
            <p className="text-center text-zinc-500 dark:text-zinc-400 py-8">
              No spans recorded for this trace.
            </p>
          ) : (
            <div className="space-y-3">
              {trace.spans.map((span, index) => (
                <div
                  key={span.id}
                  className="border border-zinc-200 dark:border-zinc-700 rounded-lg overflow-hidden"
                >
                  <button
                    onClick={() => toggleSpan(span.id)}
                    className="w-full p-4 flex items-center justify-between hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <span className="text-zinc-400 dark:text-zinc-500 text-sm font-mono w-6">
                        {index + 1}
                      </span>
                      <SpanTypeIcon type={span.type} />
                      <div className="text-left">
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="uppercase text-xs">
                            {span.type}
                          </Badge>
                          <span className="font-medium text-zinc-900 dark:text-white">
                            {span.name}
                          </span>
                        </div>
                        {span.model && (
                          <p className="text-sm text-zinc-500 dark:text-zinc-400">
                            {span.model} ({span.provider})
                          </p>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-4 text-sm">
                      {span.inputTokens && (
                        <span className="text-zinc-500 dark:text-zinc-400">
                          <span className="text-emerald-500">↑</span> {span.inputTokens}
                        </span>
                      )}
                      {span.outputTokens && (
                        <span className="text-zinc-500 dark:text-zinc-400">
                          <span className="text-blue-500">↓</span> {span.outputTokens}
                        </span>
                      )}
                      <span className="text-zinc-500 dark:text-zinc-400 w-16 text-right">
                        {formatDuration(span.durationMs)}
                      </span>
                      {span.costUsd && (
                        <span className="font-mono text-amber-600 dark:text-amber-400 w-16 text-right">
                          ${parseFloat(span.costUsd).toFixed(4)}
                        </span>
                      )}
                      <Badge
                        variant={
                          span.status === 'success'
                            ? 'default'
                            : span.status === 'error'
                            ? 'destructive'
                            : 'secondary'
                        }
                        className="w-16 justify-center"
                      >
                        {span.status}
                      </Badge>
                      <svg
                        className={`w-4 h-4 text-zinc-400 transition-transform ${
                          expandedSpans.has(span.id) ? 'rotate-180' : ''
                        }`}
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        strokeWidth={2}
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 8.25l-7.5 7.5-7.5-7.5" />
                      </svg>
                    </div>
                  </button>

                  {/* Expanded Content */}
                  {expandedSpans.has(span.id) && (
                    <div className="border-t border-zinc-200 dark:border-zinc-700 p-4 bg-zinc-50 dark:bg-zinc-800/50">
                      {span.errorMessage && (
                        <div className="mb-4 p-3 bg-red-50 dark:bg-red-500/10 border border-red-200 dark:border-red-500/20 rounded-lg">
                          <p className="text-sm font-medium text-red-600 dark:text-red-400 mb-1">Error</p>
                          <p className="text-sm text-red-600 dark:text-red-300">{span.errorMessage}</p>
                        </div>
                      )}
                      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                        <LLMDataViewer data={span.input} label="Input" />
                        <LLMDataViewer data={span.output} label="Output" />
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Metadata */}
      {trace.metadata && Object.keys(trace.metadata).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Metadata</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="text-sm bg-zinc-50 dark:bg-zinc-800 p-4 rounded-lg overflow-auto">
              {JSON.stringify(trace.metadata, null, 2)}
            </pre>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
