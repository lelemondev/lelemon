'use client';

import { useEffect, useState, useCallback, useRef } from 'react';
import { useProject } from '@/lib/project-context';
import { dashboardAPI, TraceDetailResponse } from '@/lib/api';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { TraceViewer } from './TraceViewer';
import { formatDuration } from './utils';

const POLLING_INTERVAL = 2000;

interface TraceDetailProps {
  traceId: string;
  onClose?: () => void;
}

export function TraceDetail({ traceId, onClose }: TraceDetailProps) {
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
      const data = await dashboardAPI.getTrace(currentProject.id, traceId);
      setTrace(data);
      setError(null);
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
  }, [currentProject, traceId]);

  useEffect(() => {
    let isMounted = true;

    // Reset state when traceId changes
    setIsLoading(true);
    setError(null);
    setTrace(null);

    // Clear previous polling
    if (pollingRef.current) {
      clearInterval(pollingRef.current);
      pollingRef.current = null;
    }

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
  }, [fetchTrace, traceId]);

  if (isLoading) {
    return (
      <div className="p-6 space-y-6">
        <div className="h-8 w-48 bg-zinc-200 dark:bg-zinc-800 rounded animate-pulse" />
        <div className="grid gap-4 grid-cols-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-20 bg-zinc-200 dark:bg-zinc-800 rounded-xl animate-pulse" />
          ))}
        </div>
        <div className="h-64 bg-zinc-200 dark:bg-zinc-800 rounded-xl animate-pulse" />
      </div>
    );
  }

  if (error || !trace) {
    return (
      <div className="flex flex-col items-center justify-center h-full py-16">
        <div className="w-12 h-12 rounded-full bg-red-100 dark:bg-red-500/10 flex items-center justify-center mb-3">
          <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
        </div>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">{error || 'Trace not found'}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-zinc-200 dark:border-zinc-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {onClose && (
              <button
                onClick={onClose}
                className="p-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded"
              >
                <svg className="w-4 h-4 text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
            <div>
              <h2 className="font-semibold text-zinc-900 dark:text-white">
                Trace {traceId.slice(0, 8)}
              </h2>
              <p className="text-xs text-zinc-500 dark:text-zinc-400">
                {trace.sessionId ? `Session: ${trace.sessionId.slice(0, 16)}...` : 'No session'}
                {trace.userId ? ` | User: ${trace.userId}` : ''}
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
        </div>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-auto p-4 space-y-4">
        {/* Summary Cards - Compact */}
        <div className="grid gap-3 grid-cols-4">
          <Card className="py-2">
            <CardContent className="p-3">
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">Duration</div>
              <div className="text-lg font-bold text-zinc-900 dark:text-white">
                {formatDuration(trace.totalDurationMs ?? 0)}
              </div>
            </CardContent>
          </Card>
          <Card className="py-2">
            <CardContent className="p-3">
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">Spans</div>
              <div className="text-lg font-bold text-zinc-900 dark:text-white">
                {trace.totalSpans ?? 0}
              </div>
            </CardContent>
          </Card>
          <Card className="py-2">
            <CardContent className="p-3">
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">Tokens</div>
              <div className="text-lg font-bold text-zinc-900 dark:text-white">
                {trace.totalTokens && trace.totalTokens > 0 ? trace.totalTokens.toLocaleString() : '-'}
              </div>
            </CardContent>
          </Card>
          <Card className="py-2">
            <CardContent className="p-3">
              <div className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">Cost</div>
              <div className="text-lg font-bold text-amber-600 dark:text-amber-400">
                {trace.totalCostUsd && trace.totalCostUsd > 0 ? `$${trace.totalCostUsd.toFixed(4)}` : '-'}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Trace Viewer (Tree + Detail) */}
        <TraceViewer trace={trace} />
      </div>
    </div>
  );
}

// Empty state component for when no trace is selected
export function TraceDetailEmpty() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center p-8">
      <div className="w-16 h-16 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-4">
        <svg className="w-8 h-8 text-zinc-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59" />
        </svg>
      </div>
      <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
        Select a trace
      </h3>
      <p className="text-sm text-zinc-500 dark:text-zinc-400 max-w-xs">
        Click on a trace from the list to view its details, spans, and execution timeline.
      </p>
    </div>
  );
}
