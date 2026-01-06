'use client';

import { Badge } from '@/components/ui/badge';
import type { Trace } from '@/lib/api';

interface TracesListProps {
  traces: Trace[];
  selectedTraceId: string | null;
  onSelectTrace: (traceId: string) => void;
  isLoading?: boolean;
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 1000 / 60);

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
  return `${Math.floor(diffMins / 1440)}d ago`;
}

export function TracesList({ traces, selectedTraceId, onSelectTrace, isLoading }: TracesListProps) {
  if (isLoading) {
    return (
      <div className="space-y-2 p-2">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="h-16 bg-zinc-100 dark:bg-zinc-800 rounded-lg animate-pulse" />
        ))}
      </div>
    );
  }

  if (traces.length === 0) {
    return (
      <div className="flex items-center justify-center h-32 text-sm text-zinc-500 dark:text-zinc-400">
        No traces found
      </div>
    );
  }

  return (
    <div className="space-y-1 p-2">
      {traces.map((trace) => (
        <button
          key={trace.id}
          onClick={() => onSelectTrace(trace.id)}
          className={`w-full text-left p-3 rounded-lg transition-colors ${
            selectedTraceId === trace.id
              ? 'bg-amber-500/10 border border-amber-500/30'
              : 'hover:bg-zinc-100 dark:hover:bg-zinc-800 border border-transparent'
          }`}
        >
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs text-zinc-500 dark:text-zinc-400">
              {formatRelativeTime(trace.createdAt)}
            </span>
            <StatusIndicator status={trace.status} />
          </div>

          <div className="flex items-center gap-2 mb-1">
            <span className="font-mono text-xs text-zinc-700 dark:text-zinc-300 truncate">
              {trace.id.slice(0, 12)}...
            </span>
          </div>

          <div className="flex items-center gap-3 text-xs text-zinc-500 dark:text-zinc-400">
            <span title="Spans">{trace.totalSpans} spans</span>
            <span title="Tokens">{trace.totalTokens.toLocaleString()} tok</span>
            <span className="text-amber-600 dark:text-amber-400" title="Cost">
              ${trace.totalCostUsd.toFixed(4)}
            </span>
          </div>

          {trace.sessionId && (
            <div className="mt-1.5 text-xs text-zinc-400 dark:text-zinc-500 truncate">
              Session: {trace.sessionId.slice(0, 16)}...
            </div>
          )}
        </button>
      ))}
    </div>
  );
}

function StatusIndicator({ status }: { status: string }) {
  if (status === 'active') {
    return (
      <span className="relative flex h-2 w-2">
        <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
        <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
      </span>
    );
  }

  if (status === 'error') {
    return (
      <span className="flex h-2 w-2 rounded-full bg-red-500"></span>
    );
  }

  return (
    <span className="flex h-2 w-2 rounded-full bg-zinc-400"></span>
  );
}
