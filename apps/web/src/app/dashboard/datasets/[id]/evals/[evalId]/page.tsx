'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import {
  dashboardAPI,
  type Eval,
  type EvalRun,
} from '@/lib/api';
import { useProject } from '@/lib/project-context';

export default function EvalDetailPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string; evalId: string }>();
  const router = useRouter();
  const datasetId = params?.id ?? '';
  const evalId = params?.evalId ?? '';

  const [ev, setEval] = useState<Eval | null>(null);
  const [runs, setRuns] = useState<EvalRun[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!currentProject || !evalId) return;
    setLoading(true);
    try {
      const results = await Promise.allSettled([
        dashboardAPI.getEval(currentProject.id, evalId),
        dashboardAPI.listEvalRuns(currentProject.id, { evalId, limit: 50 }),
      ]);
      if (results[0].status === 'fulfilled') setEval(results[0].value);
      else throw results[0].reason;
      setRuns(results[1].status === 'fulfilled' ? results[1].value.data : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load eval');
    } finally {
      setLoading(false);
    }
  }, [currentProject, evalId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleDelete = async () => {
    if (!currentProject || !ev) return;
    if (!window.confirm(`Delete eval "${ev.name}" and all its runs?`)) return;
    try {
      await dashboardAPI.deleteEval(currentProject.id, ev.id);
      router.push(`/dashboard/datasets/${datasetId}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete eval');
    }
  };

  if (!currentProject) {
    return <p className="text-sm text-muted-foreground">Select a project first.</p>;
  }

  if (loading && !ev) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error && !ev) {
    return (
      <div className="space-y-4">
        <Link href={`/dashboard/datasets/${datasetId}`} className="text-sm text-amber-600 hover:underline">
          ← Dataset
        </Link>
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error}
        </div>
      </div>
    );
  }

  if (!ev) return null;

  return (
    <div className="space-y-6">
      <div>
        <Link
          href={`/dashboard/datasets/${datasetId}`}
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Back to dataset
        </Link>
        <div className="mt-1 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{ev.name}</h1>
            {ev.description && (
              <p className="mt-1 text-sm text-muted-foreground">{ev.description}</p>
            )}
            <div className="mt-2 flex flex-wrap items-center gap-1.5">
              {ev.scorers.map((s) => (
                <Badge key={s.id} variant="outline" className="text-[10px] font-mono">
                  {s.type}
                  {s.name ? ` · ${s.name}` : ''}
                </Badge>
              ))}
            </div>
          </div>
          <Button
            variant="ghost"
            className="text-red-600 hover:text-red-700"
            onClick={handleDelete}
          >
            Delete
          </Button>
        </div>
      </div>

      <RunsTable datasetId={datasetId} evalId={ev.id} runs={runs} />

      <HowToRunCard evalId={ev.id} />
    </div>
  );
}

// ----- runs table --------------------------------------------------------

function RunsTable({
  datasetId,
  evalId,
  runs,
}: {
  datasetId: string;
  evalId: string;
  runs: EvalRun[];
}) {
  const router = useRouter();

  // Whole-row click is not keyboard-accessible by default; we wire `Enter` and
  // `Space` to the same router push so a keyboard user can navigate too.
  // `router.push()` keeps Next.js's client-side routing — never use
  // `window.location.href` here, it causes a full page reload.
  const openRun = (runId: string) => {
    router.push(`/dashboard/datasets/${datasetId}/evals/${evalId}/runs/${runId}`);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Runs</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {runs.length === 0 ? (
          <div className="px-6 py-8 text-center text-sm text-muted-foreground">
            No runs yet. Trigger one from your CI / SDK harness — see below.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-y border-zinc-200 dark:border-zinc-700 text-left text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Pass rate</th>
                <th className="px-4 py-2 font-medium">Items</th>
                <th className="px-4 py-2 font-medium">Prompt</th>
                <th className="px-4 py-2 font-medium">Started</th>
                <th className="px-4 py-2 font-medium">Duration</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((r) => (
                <tr
                  key={r.id}
                  role="link"
                  tabIndex={0}
                  className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30 cursor-pointer focus:outline-none focus:bg-zinc-50 dark:focus:bg-zinc-800/30"
                  onClick={() => openRun(r.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      openRun(r.id);
                    }
                  }}
                >
                  <td className="px-4 py-2">
                    <StatusBadge status={r.status} />
                  </td>
                  <td className="px-4 py-2">
                    <PassRate rate={r.passRate} total={r.totalItems} />
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-700 dark:text-zinc-300">
                    <span className="text-emerald-600 dark:text-emerald-400">{r.passedItems} ✓</span>
                    {' · '}
                    <span className="text-red-600 dark:text-red-400">{r.failedItems} ✗</span>
                    {r.erroredItems > 0 && (
                      <>
                        {' · '}
                        <span className="text-amber-600 dark:text-amber-400">{r.erroredItems} !</span>
                      </>
                    )}
                    {' / '}
                    {r.totalItems}
                  </td>
                  <td className="px-4 py-2 text-xs">
                    {r.promptVersionId ? (
                      <code className="font-mono text-zinc-700 dark:text-zinc-300">
                        {r.promptVersionId}
                      </code>
                    ) : (
                      <span className="text-zinc-400">—</span>
                    )}
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-500">
                    {new Date(r.startedAt).toLocaleString()}
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-500">
                    {r.durationMs ? `${(r.durationMs / 1000).toFixed(2)}s` : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </CardContent>
    </Card>
  );
}

function StatusBadge({ status }: { status: EvalRun['status'] }) {
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

function PassRate({ rate, total }: { rate: number | null; total: number }) {
  if (rate === null || total === 0) {
    return <span className="text-xs text-zinc-400">—</span>;
  }
  const pct = Math.round(rate * 100);
  const color =
    pct === 100 ? 'text-emerald-600 dark:text-emerald-400'
    : pct >= 80 ? 'text-amber-600 dark:text-amber-400'
    : 'text-red-600 dark:text-red-400';
  return <span className={`font-semibold tabular-nums ${color}`}>{pct}%</span>;
}

// ----- how-to-run snippet ------------------------------------------------

function HowToRunCard({ evalId }: { evalId: string }) {
  const snippet = `# 1. Start a run
RUN_ID=$(curl -s -X POST $LELEMON_URL/api/v1/eval-runs \\
  -H "Authorization: Bearer $LELEMON_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"evalId":"${evalId}","promptVersionId":"agent@v3"}' | jq -r .id)

# 2. For each dataset item, run your target and post the actual output
curl -X POST $LELEMON_URL/api/v1/eval-runs/$RUN_ID/results \\
  -H "Authorization: Bearer $LELEMON_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"datasetItemId":"<item-id>","actual":"<your output>","durationMs":120}'

# 3. Finalize. The server has already scored each result.
curl -X POST $LELEMON_URL/api/v1/eval-runs/$RUN_ID/finalize \\
  -H "Authorization: Bearer $LELEMON_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"status":"completed"}'`;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <span aria-hidden>🚀</span> Run from CI
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground mb-2">
          The platform scores each result server-side; the client just reports
          what its target produced. Hook it into your CI and gate merges on the
          finalize response.
        </p>
        <pre className="text-[11px] bg-zinc-50 dark:bg-zinc-800 rounded-md p-3 overflow-auto font-mono text-zinc-700 dark:text-zinc-300">
          {snippet}
        </pre>
      </CardContent>
    </Card>
  );
}
