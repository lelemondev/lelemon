'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  dashboardAPI,
  type EvalRun,
  type EvalRunResult,
  type ScorerResult,
} from '@/lib/api';
import { useProject } from '@/lib/project-context';

type Filter = 'all' | 'passed' | 'failed';

export default function EvalRunDetailPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string; evalId: string; runId: string }>();
  const datasetId = params?.id ?? '';
  const evalId = params?.evalId ?? '';
  const runId = params?.runId ?? '';

  const [run, setRun] = useState<EvalRun | null>(null);
  const [results, setResults] = useState<EvalRunResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>('all');
  const [isPolling, setIsPolling] = useState(false);

  const fetchAll = useCallback(
    async (signal?: AbortSignal): Promise<boolean> => {
      if (!currentProject || !runId) return false;
      try {
        const [r, page] = await Promise.all([
          dashboardAPI.getEvalRun(currentProject.id, runId),
          dashboardAPI.listEvalRunResults(currentProject.id, runId, { limit: 500 }),
        ]);
        if (signal?.aborted) return false;
        setRun(r);
        setResults(page.data);
        setError(null);
        return r.status === 'pending' || r.status === 'running';
      } catch (err) {
        if (!signal?.aborted) {
          setError(err instanceof Error ? err.message : 'Failed to load run');
        }
        return false;
      }
    },
    [currentProject, runId],
  );

  // Initial fetch + live polling while the run is in-flight.
  useEffect(() => {
    const controller = new AbortController();
    let interval: ReturnType<typeof setInterval> | null = null;

    (async () => {
      const shouldPoll = await fetchAll(controller.signal);
      setLoading(false);
      if (shouldPoll) {
        setIsPolling(true);
        interval = setInterval(async () => {
          const keepPolling = await fetchAll(controller.signal);
          if (!keepPolling && interval) {
            clearInterval(interval);
            interval = null;
            setIsPolling(false);
          }
        }, 3000);
      }
    })();

    return () => {
      controller.abort();
      if (interval) clearInterval(interval);
    };
  }, [fetchAll]);

  const filtered = useMemo(() => {
    if (filter === 'passed') return results.filter((r) => r.passed);
    if (filter === 'failed') return results.filter((r) => !r.passed);
    return results;
  }, [results, filter]);

  if (!currentProject) {
    return <p className="text-sm text-muted-foreground">Select a project first.</p>;
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !run) {
    return (
      <div className="space-y-4">
        <Link href={`/dashboard/datasets/${datasetId}/evals/${evalId}`} className="text-sm text-amber-600 hover:underline">
          ← Eval
        </Link>
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error ?? 'Run not found'}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href={`/dashboard/datasets/${datasetId}/evals/${evalId}`}
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Back to eval
        </Link>
        <div className="mt-1 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              Run {run.id.slice(0, 8)}
            </h1>
            <div className="mt-2 flex items-center gap-2 text-xs text-zinc-500">
              <StatusPill status={run.status} />
              {run.promptVersionId && (
                <Badge variant="outline" className="font-mono text-[10px]">
                  prompt: {run.promptVersionId}
                </Badge>
              )}
              <span>· Started {new Date(run.startedAt).toLocaleString()}</span>
              {run.completedAt && (
                <span>· Finished {new Date(run.completedAt).toLocaleString()}</span>
              )}
            </div>
          </div>
          {isPolling && (
            <div className="flex items-center gap-1.5 px-2 py-1 bg-emerald-500/10 border border-emerald-500/20 rounded-full">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500" />
              </span>
              <span className="text-xs font-medium text-emerald-600 dark:text-emerald-400">LIVE</span>
            </div>
          )}
        </div>
      </div>

      <SummaryCards run={run} />

      <div className="flex items-center gap-2 text-xs">
        <FilterBtn active={filter === 'all'} onClick={() => setFilter('all')}>
          All ({results.length})
        </FilterBtn>
        <FilterBtn active={filter === 'passed'} onClick={() => setFilter('passed')}>
          Passed ({results.filter((r) => r.passed).length})
        </FilterBtn>
        <FilterBtn active={filter === 'failed'} onClick={() => setFilter('failed')}>
          Failed ({results.filter((r) => !r.passed).length})
        </FilterBtn>
      </div>

      <ResultsTable results={filtered} datasetId={datasetId} />
    </div>
  );
}

// ----- summary -----------------------------------------------------------

function SummaryCards({ run }: { run: EvalRun }) {
  return (
    <div className="grid gap-3 grid-cols-2 md:grid-cols-5">
      <SummaryCard label="Pass rate" value={passRateText(run)} accent={passRateAccent(run)} />
      <SummaryCard label="Passed" value={`${run.passedItems}`} accent="emerald" />
      <SummaryCard label="Failed" value={`${run.failedItems}`} accent="red" />
      <SummaryCard label="Errored" value={`${run.erroredItems}`} accent="amber" />
      <SummaryCard
        label="Cost"
        value={run.costUsd != null ? `$${run.costUsd.toFixed(4)}` : '—'}
        accent="default"
      />
    </div>
  );
}

function passRateText(run: EvalRun): string {
  if (run.passRate === null) return '—';
  return `${Math.round(run.passRate * 100)}%`;
}
function passRateAccent(run: EvalRun): 'emerald' | 'amber' | 'red' | 'default' {
  if (run.passRate === null) return 'default';
  const pct = run.passRate * 100;
  if (pct === 100) return 'emerald';
  if (pct >= 80) return 'amber';
  return 'red';
}

function SummaryCard({
  label,
  value,
  accent,
}: {
  label: string;
  value: string;
  accent: 'emerald' | 'red' | 'amber' | 'default';
}) {
  const colors = {
    emerald: 'text-emerald-600 dark:text-emerald-400',
    red: 'text-red-600 dark:text-red-400',
    amber: 'text-amber-600 dark:text-amber-400',
    default: 'text-zinc-900 dark:text-white',
  };
  return (
    <Card className="py-2">
      <CardContent className="p-3">
        <div className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">{label}</div>
        <div className={`text-2xl font-bold ${colors[accent]}`}>{value}</div>
      </CardContent>
    </Card>
  );
}

// ----- results table -----------------------------------------------------

function ResultsTable({ results, datasetId }: { results: EvalRunResult[]; datasetId: string }) {
  if (results.length === 0) {
    return (
      <Card>
        <CardContent className="py-10 text-center text-sm text-muted-foreground">
          No results to show in this view.
        </CardContent>
      </Card>
    );
  }
  return (
    <Card>
      <CardContent className="p-0">
        <table className="w-full text-sm">
          <thead className="border-b border-zinc-200 dark:border-zinc-700 text-left text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-2 font-medium w-0">Pass</th>
              <th className="px-4 py-2 font-medium">Item</th>
              <th className="px-4 py-2 font-medium">Scorers</th>
              <th className="px-4 py-2 font-medium">Actual</th>
              <th className="px-4 py-2 font-medium w-0">Duration</th>
            </tr>
          </thead>
          <tbody>
            {results.map((r) => (
              <ResultRow key={r.id} r={r} datasetId={datasetId} />
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}

function ResultRow({ r, datasetId }: { r: EvalRunResult; datasetId: string }) {
  const actualPreview = useMemo(() => previewJSON(r.actual), [r.actual]);

  return (
    <tr className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30">
      <td className="px-4 py-2">
        {r.error ? (
          <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400" title={r.error}>
            !
          </span>
        ) : r.passed ? (
          <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400">
            ✓
          </span>
        ) : (
          <span className="inline-flex items-center justify-center w-6 h-6 rounded-full bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-400">
            ✗
          </span>
        )}
      </td>
      <td className="px-4 py-2 max-w-xs">
        <Link
          href={`/dashboard/datasets/${datasetId}`}
          className="text-xs text-amber-600 hover:underline dark:text-amber-400 font-mono"
          title={`Open dataset (item ${r.datasetItemId})`}
        >
          {r.datasetItemId.slice(0, 8)}
        </Link>
      </td>
      <td className="px-4 py-2">
        <div className="flex flex-wrap gap-1">
          {r.scores.length === 0 && r.error ? (
            <span className="text-[11px] text-amber-600 dark:text-amber-400 italic">
              skipped — {r.error}
            </span>
          ) : (
            r.scores.map((s, i) => <ScorerPill key={i} s={s} />)
          )}
        </div>
      </td>
      <td className="px-4 py-2 max-w-md">
        <span className="block truncate font-mono text-[11px] text-zinc-700 dark:text-zinc-300" title={actualPreview}>
          {actualPreview || <span className="text-zinc-400">—</span>}
        </span>
      </td>
      <td className="px-4 py-2 text-xs text-zinc-500 tabular-nums">
        {r.durationMs != null ? `${r.durationMs}ms` : '—'}
      </td>
    </tr>
  );
}

function ScorerPill({ s }: { s: ScorerResult }) {
  if (s.error) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-mono bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400"
        title={s.error}
      >
        {s.scorerId} !
      </span>
    );
  }
  const cls = s.passed
    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400'
    : 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-400';
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-1.5 py-0.5 text-[10px] font-mono ${cls}`}
      title={s.details || (s.passed ? 'passed' : 'failed')}
    >
      {s.scorerId} {s.passed ? '✓' : '✗'}
    </span>
  );
}

function previewJSON(v: unknown): string {
  if (v === null || v === undefined) return '';
  try {
    return typeof v === 'string' ? v : JSON.stringify(v);
  } catch {
    return String(v);
  }
}

// ----- misc UI -----------------------------------------------------------

function FilterBtn({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        'rounded-full px-3 py-1 transition-colors ' +
        (active
          ? 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400'
          : 'text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800')
      }
    >
      {children}
    </button>
  );
}

function StatusPill({ status }: { status: EvalRun['status'] }) {
  const classes: Record<EvalRun['status'], string> = {
    pending: 'bg-zinc-200 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-200',
    running: 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300',
    completed: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300',
    failed: 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300',
  };
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${classes[status]}`}>
      {status}
    </span>
  );
}
