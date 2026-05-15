'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { dashboardAPI, type Prompt } from '@/lib/api';
import { useProject } from '@/lib/project-context';

export default function PromptsPage() {
  const { currentProject } = useProject();

  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showCreateForm, setShowCreateForm] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [creating, setCreating] = useState(false);

  const refresh = useCallback(async () => {
    if (!currentProject) return;
    setLoading(true);
    try {
      const page = await dashboardAPI.listPrompts(currentProject.id, { limit: 200 });
      setPrompts(page.data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load prompts');
    } finally {
      setLoading(false);
    }
  }, [currentProject]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentProject || name.trim() === '') return;
    setCreating(true);
    try {
      await dashboardAPI.createPrompt(currentProject.id, {
        name: name.trim(),
        description: description.trim() || undefined,
      });
      setName('');
      setDescription('');
      setShowCreateForm(false);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create prompt');
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string, promptName: string) => {
    if (!currentProject) return;
    if (!window.confirm(`Delete prompt "${promptName}" and all its versions?`)) return;
    try {
      await dashboardAPI.deletePrompt(currentProject.id, id);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete prompt');
    }
  };

  if (!currentProject) {
    return (
      <div className="space-y-6">
        <h1 className="text-3xl font-bold tracking-tight">Prompts</h1>
        <p className="text-sm text-muted-foreground">Select a project to manage its prompts.</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <span aria-hidden>✨</span> Prompts
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Version your prompts here. Tie traces and eval runs to versions to
            answer the only question that matters: <em>did the new prompt help?</em>
          </p>
        </div>
        <Button onClick={() => setShowCreateForm((v) => !v)}>
          {showCreateForm ? 'Cancel' : '+ New prompt'}
        </Button>
      </div>

      {showCreateForm && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Create prompt</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreate} className="space-y-3">
              <div>
                <label htmlFor="prompt-name" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                  Name
                </label>
                <Input
                  id="prompt-name"
                  autoFocus
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="agent-system"
                  maxLength={200}
                />
              </div>
              <div>
                <label htmlFor="prompt-desc" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                  Description <span className="text-zinc-400">(optional)</span>
                </label>
                <Input
                  id="prompt-desc"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="The system prompt for the WhatsApp sales agent"
                  maxLength={2000}
                />
              </div>
              <div className="flex justify-end gap-2 pt-1">
                <Button type="button" variant="ghost" onClick={() => setShowCreateForm(false)} disabled={creating}>
                  Cancel
                </Button>
                <Button type="submit" disabled={creating || name.trim() === ''}>
                  {creating ? 'Creating…' : 'Create prompt'}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {error && (
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error}
        </div>
      )}

      {loading ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 rounded-xl" />
          ))}
        </div>
      ) : prompts.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <div className="text-4xl mb-2" aria-hidden>✨</div>
            <h3 className="font-semibold text-zinc-900 dark:text-white">No prompts yet</h3>
            <p className="mt-1 text-sm text-muted-foreground max-w-md mx-auto">
              A prompt is a series of versions. Each version is an immutable
              snapshot — bump a version every time you edit, attach the version
              id to your traces, and the loop closes itself.
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {prompts.map((p) => (
            <Card key={p.id} className="group transition-shadow hover:shadow-md">
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-2">
                  <CardTitle className="text-base">
                    <Link
                      href={`/dashboard/prompts/${p.id}`}
                      className="hover:text-amber-600 dark:hover:text-amber-400 transition-colors"
                    >
                      {p.name}
                    </Link>
                  </CardTitle>
                  <button
                    type="button"
                    onClick={() => handleDelete(p.id, p.name)}
                    className="opacity-0 group-hover:opacity-100 transition-opacity text-xs text-red-600 hover:text-red-700 dark:text-red-400"
                    aria-label={`Delete ${p.name}`}
                  >
                    Delete
                  </button>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground line-clamp-2 min-h-[2.5rem]">
                  {p.description || <span className="italic text-zinc-400">No description</span>}
                </p>
                <p className="mt-3 text-[11px] text-zinc-400 dark:text-zinc-500">
                  Created {new Date(p.createdAt).toLocaleDateString()}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
