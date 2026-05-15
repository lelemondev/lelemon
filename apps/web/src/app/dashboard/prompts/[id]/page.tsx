'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { dashboardAPI, type Prompt, type PromptVersion } from '@/lib/api';
import { useProject } from '@/lib/project-context';

export default function PromptDetailPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const promptId = params?.id ?? '';

  const [pr, setPrompt] = useState<Prompt | null>(null);
  const [versions, setVersions] = useState<PromptVersion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!currentProject || !promptId) return;
    setLoading(true);
    try {
      // Promise.allSettled — one failed call should not blank the page (rule
      // from .claude/rules/anti-patterns.md).
      const results = await Promise.allSettled([
        dashboardAPI.getPrompt(currentProject.id, promptId),
        dashboardAPI.listPromptVersions(currentProject.id, promptId, { limit: 200 }),
      ]);
      if (results[0].status === 'fulfilled') setPrompt(results[0].value);
      else throw results[0].reason;
      setVersions(results[1].status === 'fulfilled' ? results[1].value.data : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load prompt');
    } finally {
      setLoading(false);
    }
  }, [currentProject, promptId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleDelete = async () => {
    if (!currentProject || !pr) return;
    if (!window.confirm(`Delete prompt "${pr.name}" and all its versions?`)) return;
    try {
      await dashboardAPI.deletePrompt(currentProject.id, pr.id);
      router.push('/dashboard/prompts');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete prompt');
    }
  };

  if (!currentProject) {
    return <p className="text-sm text-muted-foreground">Select a project first.</p>;
  }

  if (loading && !pr) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error && !pr) {
    return (
      <div className="space-y-4">
        <Link href="/dashboard/prompts" className="text-sm text-amber-600 hover:underline">
          ← Prompts
        </Link>
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error}
        </div>
      </div>
    );
  }

  if (!pr) return null;

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/dashboard/prompts"
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Back to prompts
        </Link>
        <div className="mt-1 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">{pr.name}</h1>
            {pr.description && (
              <p className="mt-1 text-sm text-muted-foreground">{pr.description}</p>
            )}
            <div className="mt-2 flex items-center gap-2 text-xs text-zinc-500">
              <Badge variant="outline">{versions.length} {versions.length === 1 ? 'version' : 'versions'}</Badge>
              <span>· Updated {new Date(pr.updatedAt).toLocaleString()}</span>
            </div>
          </div>
          <div className="flex gap-2">
            <Link href={`/dashboard/prompts/${pr.id}/versions/new`}>
              <Button variant="outline">+ New version</Button>
            </Link>
            <Button variant="ghost" className="text-red-600 hover:text-red-700" onClick={handleDelete}>
              Delete
            </Button>
          </div>
        </div>
      </div>

      {error && (
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error}
        </div>
      )}

      <VersionsTable promptId={pr.id} versions={versions} />

      <HowToAttachCard />
    </div>
  );
}

// ----- versions table ----------------------------------------------------

function VersionsTable({ promptId, versions }: { promptId: string; versions: PromptVersion[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Versions</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        {versions.length === 0 ? (
          <div className="px-6 py-8 text-center text-sm text-muted-foreground">
            No versions yet. Create one to start versioning this prompt.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-y border-zinc-200 dark:border-zinc-700 text-left text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Version</th>
                <th className="px-4 py-2 font-medium">Changelog</th>
                <th className="px-4 py-2 font-medium">Created by</th>
                <th className="px-4 py-2 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {versions.map((v) => (
                <tr
                  key={v.id}
                  className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30"
                >
                  <td className="px-4 py-2">
                    <Link
                      href={`/dashboard/prompts/${promptId}/versions/${v.id}`}
                      className="font-mono text-xs font-semibold text-amber-600 hover:underline dark:text-amber-400"
                    >
                      {v.version}
                    </Link>
                  </td>
                  <td className="px-4 py-2 max-w-md">
                    {v.changelog ? (
                      <span className="text-xs text-zinc-700 dark:text-zinc-300 line-clamp-1" title={v.changelog}>
                        {v.changelog}
                      </span>
                    ) : (
                      <span className="text-xs text-zinc-400 italic">—</span>
                    )}
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-500">
                    {v.createdBy ?? <span className="italic">automation</span>}
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-500">
                    {new Date(v.createdAt).toLocaleString()}
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

// ----- how to attach -----------------------------------------------------

function HowToAttachCard() {
  const snippet = `// In your SDK / instrumented code, attach the version id to every trace
// that ran this prompt. The platform uses it for the payoff view: which
// runs and which traces correspond to which prompt version.

lelemon.trace({
  name: 'agent-step',
  metadata: { prompt_version_id: '<paste the version id from the dashboard>' },
});

// For eval runs, pass it on the runs/eval-runs start call:
//   POST /api/v1/eval-runs  { evalId, promptVersionId }`;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <span aria-hidden>🧷</span> Attach a version
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground mb-2">
          The version id is the bridge between this prompt and your traces /
          eval runs. Attach it once in your code, then every trace and every
          eval run carries the link back to the version that produced it.
        </p>
        <pre className="text-[11px] bg-zinc-50 dark:bg-zinc-800 rounded-md p-3 overflow-auto font-mono text-zinc-700 dark:text-zinc-300">
          {snippet}
        </pre>
      </CardContent>
    </Card>
  );
}
