'use client';

import { use, useEffect, useState, useCallback, useRef } from 'react';
import Link from 'next/link';
import { useProject } from '@/lib/project-context';
import { dashboardAPI, TraceDetailResponse } from '@/lib/api';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { TraceViewer, formatDuration } from '@/components/traces';

const POLLING_INTERVAL = 2000;

export default function TraceDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { currentProject } = useProject();
  const [trace, setTrace] = useState<TraceDetailResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isPolling, setIsPolling] = useState(false);
  const pollingRef = useRef<NodeJS.Timeout | null>(null);

  const fetchTrace = useCallback(async (isInitial = false) => {
    if (!currentProject) {
      if (isInitial) setError('No project selected');
      return false;
    }

    try {
      const data = await dashboardAPI.getTrace(currentProject.id, id);
      setTrace(data);
      return data.status === 'active';
    } catch (err) {
      if (isInitial) {
        const message = err instanceof Error ? err.message : 'Failed to load trace';
        setError(message.includes('404') ? 'Trace not found' : message);
      }
      return false;
    } finally {
      if (isInitial) setIsLoading(false);
    }
  }, [currentProject, id]);

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
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header - Compact */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 flex-shrink-0">
        <div className="flex items-center gap-3">
          <Link href="/dashboard/traces">
            <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18" />
              </svg>
            </Button>
          </Link>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-lg font-bold text-zinc-900 dark:text-white">
                Trace {id.slice(0, 8)}
              </h1>
              {isPolling && (
                <div className="flex items-center gap-1.5 px-2 py-0.5 bg-emerald-500/10 border border-emerald-500/20 rounded-full">
                  <span className="relative flex h-2 w-2">
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                  </span>
                  <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400">LIVE</span>
                </div>
              )}
              <Badge
                variant={
                  trace.status === 'completed'
                    ? 'default'
                    : trace.status === 'error'
                    ? 'destructive'
                    : 'secondary'
                }
                className="text-xs"
              >
                {trace.status}
              </Badge>
            </div>
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              Session: {trace.sessionId || '-'}
            </p>
          </div>
        </div>

        {/* Stats inline in header */}
        <div className="hidden md:flex items-center gap-4 text-sm">
          <div className="text-right">
            <span className="text-zinc-500 dark:text-zinc-400">Duration</span>
            <span className="ml-2 font-semibold text-zinc-900 dark:text-white">{formatDuration(trace.totalDurationMs ?? 0)}</span>
          </div>
          <div className="w-px h-4 bg-zinc-300 dark:bg-zinc-700" />
          <div className="text-right">
            <span className="text-zinc-500 dark:text-zinc-400">Spans</span>
            <span className="ml-2 font-semibold text-zinc-900 dark:text-white">{trace.totalSpans ?? 0}</span>
          </div>
          <div className="w-px h-4 bg-zinc-300 dark:bg-zinc-700" />
          <div className="text-right">
            <span className="text-zinc-500 dark:text-zinc-400">Tokens</span>
            <span className="ml-2 font-semibold text-zinc-900 dark:text-white">
              {trace.totalTokens && trace.totalTokens > 0 ? trace.totalTokens.toLocaleString() : '-'}
            </span>
          </div>
          <div className="w-px h-4 bg-zinc-300 dark:bg-zinc-700" />
          <div className="text-right">
            <span className="text-zinc-500 dark:text-zinc-400">Cost</span>
            <span className="ml-2 font-semibold text-amber-600 dark:text-amber-400">
              {trace.totalCostUsd && trace.totalCostUsd > 0 ? `$${trace.totalCostUsd.toFixed(4)}` : '-'}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {trace.tags?.map((tag) => (
            <Badge key={tag} variant="secondary" className="text-xs">
              {tag}
            </Badge>
          ))}
        </div>
      </div>

      {/* Trace Viewer (Tree + Detail) - Fill remaining space, no gap */}
      <div className="flex-1 min-h-0">
        <TraceViewer trace={trace} />
      </div>
    </div>
  );
}
