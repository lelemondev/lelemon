'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  dashboardAPI,
  type EvalRun,
  type PromptVersion,
} from '@/lib/api';
import { diffStats, lineDiff, type DiffHunk } from '@/lib/diff';
import { useProject } from '@/lib/project-context';

export default function PromptVersionDetailPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string; versionId: string }>();
  const router = useRouter();
  const promptId = params?.id ?? '';
  const versionId = params?.versionId ?? '';

  const [version, setVersion] = useState<PromptVersion | null>(null);
  const [allVersions, setAllVersions] = useState<PromptVersion[]>([]);
  const [runs, setRuns] = useState<EvalRun[]>([]);
  const [tracesTotal, setTracesTotal] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!currentProject || !versionId) return;
    setLoading(true);
    try {
      // Promise.allSettled — a slow trace count must not gate the rest
      // (anti-patterns rule on multi-fetch). The first call is mandatory;
      // the others fall back to safe defaults.
      const results = await Promise.allSettled([
        dashboardAPI.getPromptVersion(currentProject.id, promptId, versionId),
        dashboardAPI.listEvalRuns(currentProject.id, { promptVersionId: versionId, limit: 100 }),
        dashboardAPI.listPromptVersions(currentProject.id, promptId, { limit: 200 }),
        dashboardAPI.getTraces(currentProject.id, { promptVersionId: versionId, limit: 1 }),
      ]);
      if (results[0].status === 'fulfilled') setVersion(results[0].value);
      else throw results[0].reason;
      setRuns(results[1].status === 'fulfilled' ? results[1].value.data : []);
      setAllVersions(results[2].status === 'fulfilled' ? results[2].value.data : []);
      setTracesTotal(results[3].status === 'fulfilled' ? results[3].value.total : null);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load version');
    } finally {
      setLoading(false);
    }
  }, [currentProject, promptId, versionId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const payoff = useMemo(() => {
    const completed = runs.filter((r) => r.status === 'completed' && r.passRate !== null);
    if (completed.length === 0) {
      return { runsCount: runs.length, avgPassRate: null as number | null };
    }
    const avg =
      completed.reduce((acc, r) => acc + (r.passRate ?? 0), 0) / completed.length;
    return { runsCount: runs.length, avgPassRate: avg };
  }, [runs]);

  const otherVersions = useMemo(
    () => allVersions.filter((v) => v.id !== versionId),
    [allVersions, versionId],
  );

  if (!currentProject) {
    return <p className="text-sm text-muted-foreground">Select a project first.</p>;
  }

  if (loading && !version) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !version) {
    return (
      <div className="space-y-4">
        <Link href={`/dashboard/prompts/${promptId}`} className="text-sm text-amber-600 hover:underline">
          ← Prompt
        </Link>
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error ?? 'Version not found'}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <Link
          href={`/dashboard/prompts/${promptId}`}
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Back to prompt
        </Link>
        <div className="mt-1 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
              <span className="font-mono text-amber-600 dark:text-amber-400">{version.version}</span>
            </h1>
            {version.changelog && (
              <p className="mt-1 text-sm text-muted-foreground">{version.changelog}</p>
            )}
            <div className="mt-2 flex items-center gap-2 text-xs text-zinc-500">
              <Badge variant="outline" className="font-mono text-[10px]" title="Use this id to attach the version to traces and eval runs">
                id: {version.id.slice(0, 8)}…
              </Badge>
              <span>
                ·{' '}
                {version.createdBy ? <>by {version.createdBy} · </> : <>via automation · </>}
                {new Date(version.createdAt).toLocaleString()}
              </span>
            </div>
          </div>
          <CopyIdButton id={version.id} />
        </div>
      </div>

      <PayoffStats
        runsCount={payoff.runsCount}
        avgPassRate={payoff.avgPassRate}
        tracesTotal={tracesTotal}
        onOpenTraces={() => router.push(`/dashboard/traces?promptVersionId=${versionId}`)}
      />

      <ContentCard content={version.content} />

      <DiffCard
        version={version}
        otherVersions={otherVersions}
        currentProjectId={currentProject.id}
        promptId={promptId}
      />

      <RunsCard runs={runs} />
    </div>
  );
}

// ----- copy id button ----------------------------------------------------

function CopyIdButton({ id }: { id: string }) {
  const [copied, setCopied] = useState(false);
  const onClick = async () => {
    try {
      await navigator.clipboard.writeText(id);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard write can fail in non-secure contexts; the user can still
      // see + select the id from the UI badge.
    }
  };
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1.5 rounded-md border border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-800 px-3 py-1.5 text-xs font-medium transition-colors"
    >
      {copied ? (
        <>
          <span aria-hidden>✓</span> Copied
        </>
      ) : (
        <>
          <span aria-hidden>🧷</span> Copy version id
        </>
      )}
    </button>
  );
}

// ----- payoff stats ------------------------------------------------------

function PayoffStats({
  runsCount,
  avgPassRate,
  tracesTotal,
  onOpenTraces,
}: {
  runsCount: number;
  avgPassRate: number | null;
  tracesTotal: number | null;
  onOpenTraces: () => void;
}) {
  return (
    <div className="grid gap-3 grid-cols-2 md:grid-cols-3">
      <PayoffCard
        label="Eval runs"
        value={runsCount.toString()}
        accent="default"
        hint="Runs that referenced this version via promptVersionId."
      />
      <PayoffCard
        label="Avg pass rate"
        value={avgPassRate === null ? '—' : `${Math.round(avgPassRate * 100)}%`}
        accent={avgPassRate === null ? 'default' : passRateAccent(avgPassRate)}
        hint="Mean across completed runs."
      />
      <PayoffCard
        label="Traces"
        value={tracesTotal === null ? '—' : tracesTotal.toLocaleString()}
        accent="default"
        hint={tracesTotal === null ? 'Unavailable.' : 'Production traces tagged with this version.'}
        onClick={tracesTotal && tracesTotal > 0 ? onOpenTraces : undefined}
      />
    </div>
  );
}

function passRateAccent(rate: number): 'emerald' | 'amber' | 'red' {
  const pct = rate * 100;
  if (pct === 100) return 'emerald';
  if (pct >= 80) return 'amber';
  return 'red';
}

function PayoffCard({
  label,
  value,
  accent,
  hint,
  onClick,
}: {
  label: string;
  value: string;
  accent: 'emerald' | 'amber' | 'red' | 'default';
  hint?: string;
  onClick?: () => void;
}) {
  const colors = {
    emerald: 'text-emerald-600 dark:text-emerald-400',
    amber: 'text-amber-600 dark:text-amber-400',
    red: 'text-red-600 dark:text-red-400',
    default: 'text-zinc-900 dark:text-white',
  };
  const Component: 'div' | 'button' = onClick ? 'button' : 'div';
  return (
    <Card className="py-2">
      <CardContent className="p-3">
        <Component
          type={onClick ? 'button' : undefined}
          onClick={onClick}
          className={
            'w-full text-left ' +
            (onClick
              ? 'cursor-pointer hover:opacity-80 transition-opacity focus:outline-none focus:ring-2 focus:ring-amber-500/40 rounded'
              : '')
          }
        >
          <div className="text-xs text-zinc-500 dark:text-zinc-400 mb-1">
            {label}
            {onClick && <span className="ml-1 text-amber-600 dark:text-amber-400" aria-hidden>→</span>}
          </div>
          <div className={`text-2xl font-bold ${colors[accent]}`}>{value}</div>
          {hint && <div className="text-[11px] text-zinc-400 dark:text-zinc-500 mt-1">{hint}</div>}
        </Component>
      </CardContent>
    </Card>
  );
}

// ----- content -----------------------------------------------------------

function ContentCard({ content }: { content: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Content</CardTitle>
      </CardHeader>
      <CardContent>
        <pre className="text-xs bg-zinc-50 dark:bg-zinc-800 rounded-md p-3 overflow-auto font-mono text-zinc-700 dark:text-zinc-300 whitespace-pre-wrap">
          {content}
        </pre>
      </CardContent>
    </Card>
  );
}

// ----- diff -------------------------------------------------------------

function DiffCard({
  version,
  otherVersions,
  currentProjectId,
  promptId,
}: {
  version: PromptVersion;
  otherVersions: PromptVersion[];
  currentProjectId: string;
  promptId: string;
}) {
  const [compareToId, setCompareToId] = useState<string>('');
  // We only persist the *result* of a fetch — success or failure — both keyed
  // by the id the fetch was for. Loading and error states are then *derived*
  // at render time by comparing those ids against the current selection.
  //
  // This is the "don't setState synchronously in an effect" pattern from
  // react-hooks/set-state-in-effect: the effect only triggers async work, the
  // state writes happen inside Promise callbacks (after the effect returns).
  const [fetched, setFetched] = useState<{ id: string; content: string } | null>(null);
  const [fetchError, setFetchError] = useState<{ id: string; message: string } | null>(null);

  useEffect(() => {
    if (!compareToId) return; // nothing to fetch; render guards downstream
    let cancelled = false;
    dashboardAPI
      .getPromptVersion(currentProjectId, promptId, compareToId)
      .then((v) => {
        if (!cancelled) setFetched({ id: compareToId, content: v.content });
      })
      .catch((err) => {
        if (!cancelled) {
          setFetchError({
            id: compareToId,
            message: err instanceof Error ? err.message : 'Failed to load diff',
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [compareToId, currentProjectId, promptId]);

  // Derived states — `compareToId` is the source of truth, the persisted
  // `fetched` / `fetchError` are only honoured when their id still matches.
  const diffError = fetchError?.id === compareToId ? fetchError.message : null;
  const loadingDiff =
    !!compareToId && !diffError && (fetched?.id !== compareToId);

  // The diff is only meaningful when we have content for the currently-selected
  // comparison version. Stale state from a previous selection is filtered out
  // by the id match.
  const hunks = useMemo<DiffHunk[] | null>(() => {
    if (!compareToId || !fetched || fetched.id !== compareToId) return null;
    const other = otherVersions.find((v) => v.id === compareToId);
    if (!other) return null;
    // Convention: `before` = the OLDER version, `after` = current. We sort by
    // createdAt so adds/removes line up with how the prompt evolved.
    const beforeIsOther = new Date(other.createdAt) < new Date(version.createdAt);
    return beforeIsOther
      ? lineDiff(other.content, version.content)
      : lineDiff(version.content, other.content);
  }, [compareToId, fetched, otherVersions, version]);

  const stats = hunks ? diffStats(hunks) : null;
  const compareLabel = otherVersions.find((v) => v.id === compareToId);
  const directionLabel = compareLabel
    ? new Date(compareLabel.createdAt) < new Date(version.createdAt)
      ? `${compareLabel.version} → ${version.version}`
      : `${version.version} → ${compareLabel.version}`
    : '';

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-2 space-y-0">
        <CardTitle className="text-base flex items-center gap-2">
          <span aria-hidden>🔀</span> Diff
          {stats && (
            <span className="ml-2 text-xs font-normal">
              <span className="text-emerald-600 dark:text-emerald-400">+{stats.added}</span>
              {' '}
              <span className="text-red-600 dark:text-red-400">-{stats.removed}</span>
              <span className="ml-2 font-mono text-zinc-500">{directionLabel}</span>
            </span>
          )}
        </CardTitle>
        <Select value={compareToId} onValueChange={setCompareToId} disabled={otherVersions.length === 0}>
          <SelectTrigger className="w-56 text-xs h-8">
            <SelectValue
              placeholder={
                otherVersions.length === 0 ? 'No other versions yet' : 'Compare against…'
              }
            />
          </SelectTrigger>
          <SelectContent>
            {otherVersions.map((v) => (
              <SelectItem key={v.id} value={v.id}>
                {v.version} — {new Date(v.createdAt).toLocaleDateString()}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent>
        {diffError && (
          <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
            {diffError}
          </div>
        )}
        {!compareToId ? (
          <p className="text-xs text-muted-foreground">
            Pick another version to see what changed.
          </p>
        ) : loadingDiff ? (
          <Skeleton className="h-32 w-full" />
        ) : hunks ? (
          <DiffPre hunks={hunks} />
        ) : null}
      </CardContent>
    </Card>
  );
}

function DiffPre({ hunks }: { hunks: DiffHunk[] }) {
  return (
    <pre className="text-[11px] bg-zinc-50 dark:bg-zinc-800 rounded-md overflow-auto font-mono">
      {hunks.map((h, i) => (
        <DiffLine key={i} hunk={h} />
      ))}
    </pre>
  );
}

function DiffLine({ hunk }: { hunk: DiffHunk }) {
  const tone =
    hunk.type === 'added'
      ? 'bg-emerald-100/60 text-emerald-900 dark:bg-emerald-500/15 dark:text-emerald-200'
      : hunk.type === 'removed'
      ? 'bg-red-100/60 text-red-900 dark:bg-red-500/15 dark:text-red-200'
      : 'text-zinc-700 dark:text-zinc-300';
  const sigil = hunk.type === 'added' ? '+' : hunk.type === 'removed' ? '−' : ' ';
  return (
    <div className={`flex gap-2 px-3 py-0.5 ${tone}`}>
      <span className="select-none w-3 text-right opacity-60">{sigil}</span>
      <span className="whitespace-pre-wrap">{hunk.text === '' ? ' ' : hunk.text}</span>
    </div>
  );
}

// ----- runs --------------------------------------------------------------

function RunsCard({ runs }: { runs: EvalRun[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Eval runs of this version</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {runs.length === 0 ? (
          <div className="px-6 py-8 text-center text-sm text-muted-foreground">
            No eval runs reference this version yet. Pass <code className="font-mono text-xs">promptVersionId</code> when
            starting a run from your CI / SDK to wire it in.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-y border-zinc-200 dark:border-zinc-700 text-left text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Pass rate</th>
                <th className="px-4 py-2 font-medium">Items</th>
                <th className="px-4 py-2 font-medium">Started</th>
              </tr>
            </thead>
            <tbody>
              {runs.map((r) => (
                <tr
                  key={r.id}
                  className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30"
                >
                  <td className="px-4 py-2">
                    <StatusBadge status={r.status} />
                  </td>
                  <td className="px-4 py-2">
                    {r.passRate === null ? (
                      <span className="text-xs text-zinc-400">—</span>
                    ) : (
                      <span className={`text-xs font-semibold tabular-nums ${rateColor(r.passRate)}`}>
                        {Math.round(r.passRate * 100)}%
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-700 dark:text-zinc-300">
                    <span className="text-emerald-600 dark:text-emerald-400">{r.passedItems} ✓</span>
                    {' · '}
                    <span className="text-red-600 dark:text-red-400">{r.failedItems} ✗</span>
                    {' / '}
                    {r.totalItems}
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-500">
                    {new Date(r.startedAt).toLocaleString()}
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

function rateColor(rate: number): string {
  const pct = rate * 100;
  if (pct === 100) return 'text-emerald-600 dark:text-emerald-400';
  if (pct >= 80) return 'text-amber-600 dark:text-amber-400';
  return 'text-red-600 dark:text-red-400';
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
