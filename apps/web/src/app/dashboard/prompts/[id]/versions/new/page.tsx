'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { dashboardAPI, APIError, type Prompt } from '@/lib/api';
import { useProject } from '@/lib/project-context';

export default function NewPromptVersionPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const promptId = params?.id ?? '';

  const [pr, setPrompt] = useState<Prompt | null>(null);
  const [versionLabel, setVersionLabel] = useState('');
  const [content, setContent] = useState('');
  const [changelog, setChangelog] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!currentProject || !promptId) return;
    dashboardAPI
      .getPrompt(currentProject.id, promptId)
      .then(setPrompt)
      .catch((err) =>
        setError(err instanceof Error ? err.message : 'Failed to load prompt'),
      );
  }, [currentProject, promptId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentProject) return;
    if (versionLabel.trim() === '') {
      setError('Version label is required');
      return;
    }
    if (content.trim() === '') {
      setError('Content is required');
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const created = await dashboardAPI.createPromptVersion(currentProject.id, promptId, {
        version: versionLabel.trim(),
        content,
        changelog: changelog.trim() || undefined,
      });
      router.push(`/dashboard/prompts/${promptId}/versions/${created.id}`);
    } catch (err) {
      // 409 from server when the label is already taken on this prompt —
      // surface a specific message instead of a generic error.
      if (err instanceof APIError && err.status === 409) {
        setError(`Version label "${versionLabel.trim()}" already exists on this prompt. Try another.`);
      } else {
        setError(err instanceof Error ? err.message : 'Failed to create version');
      }
      setSaving(false);
    }
  };

  if (!currentProject) {
    return <p className="text-sm text-muted-foreground">Select a project first.</p>;
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <Link
          href={`/dashboard/prompts/${promptId}`}
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Back to prompt
        </Link>
        <h1 className="mt-1 text-3xl font-bold tracking-tight flex items-center gap-2">
          <span aria-hidden>✨</span> New version
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {pr ? <>For prompt <strong>{pr.name}</strong>.</> : null} Versions are
          immutable — once created, they stay. If you need to change content,
          bump the version label.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Version</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <label htmlFor="version-label" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Version label
              </label>
              <Input
                id="version-label"
                autoFocus
                required
                value={versionLabel}
                onChange={(e) => setVersionLabel(e.target.value)}
                placeholder="v1, v2-rc, 2026-05-15-experiment, …"
                maxLength={100}
              />
              <p className="text-[11px] text-zinc-500 mt-1">
                Unique per prompt. The platform rejects duplicates with a 409.
              </p>
            </div>
            <div>
              <label htmlFor="version-content" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Content
              </label>
              <textarea
                id="version-content"
                required
                value={content}
                onChange={(e) => setContent(e.target.value)}
                rows={16}
                placeholder={'You are a helpful agent. Use the tools when…'}
                className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm font-mono text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-amber-500/40 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
              />
              <p className="text-[11px] text-zinc-500 mt-1">
                Opaque text. Use templates like <code>{`{{var}}`}</code> if your
                runtime supports them; for chat-message arrays, paste the JSON
                — your code decodes when reading.
              </p>
            </div>
            <div>
              <label htmlFor="version-changelog" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Changelog <span className="text-zinc-400">(optional)</span>
              </label>
              <Input
                id="version-changelog"
                value={changelog}
                onChange={(e) => setChangelog(e.target.value)}
                placeholder="Tightened response style after VEN-321"
                maxLength={4000}
              />
            </div>
          </CardContent>
        </Card>

        {error && (
          <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-2">
          <Link href={`/dashboard/prompts/${promptId}`}>
            <Button type="button" variant="ghost" disabled={saving}>
              Cancel
            </Button>
          </Link>
          <Button type="submit" disabled={saving}>
            {saving ? 'Creating…' : 'Create version'}
          </Button>
        </div>
      </form>
    </div>
  );
}
