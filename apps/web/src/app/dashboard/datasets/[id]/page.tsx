'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCallback, useEffect, useMemo, useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { dashboardAPI, type Dataset, type DatasetItem, type Eval } from '@/lib/api';
import { useProject } from '@/lib/project-context';

type AddMode = 'closed' | 'manual' | 'import';

export default function DatasetDetailPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const datasetId = params?.id ?? '';

  const [dataset, setDataset] = useState<Dataset | null>(null);
  const [items, setItems] = useState<DatasetItem[]>([]);
  const [evals, setEvals] = useState<Eval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [addMode, setAddMode] = useState<AddMode>('closed');

  const refresh = useCallback(async () => {
    if (!currentProject || !datasetId) return;
    setLoading(true);
    try {
      // Fan out — three independent reads. Promise.allSettled so one slow or
      // failing call doesn't block the rest (anti-patterns rule on multi-fetch).
      const results = await Promise.allSettled([
        dashboardAPI.getDataset(currentProject.id, datasetId),
        dashboardAPI.listDatasetItems(currentProject.id, datasetId, { limit: 200 }),
        dashboardAPI.listEvals(currentProject.id, { datasetId, limit: 100 }),
      ]);
      if (results[0].status === 'fulfilled') setDataset(results[0].value);
      else throw results[0].reason;
      setItems(results[1].status === 'fulfilled' ? results[1].value.data : []);
      setEvals(results[2].status === 'fulfilled' ? results[2].value.data : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dataset');
    } finally {
      setLoading(false);
    }
  }, [currentProject, datasetId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleDeleteItem = useCallback(
    async (itemId: string) => {
      if (!currentProject) return;
      if (!window.confirm('Delete this eval case?')) return;
      try {
        await dashboardAPI.deleteDatasetItem(currentProject.id, datasetId, itemId);
        await refresh();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete item');
      }
    },
    [currentProject, datasetId, refresh],
  );

  const handleDeleteDataset = useCallback(async () => {
    if (!currentProject || !dataset) return;
    if (!window.confirm(`Delete dataset "${dataset.name}" and all its items?`)) return;
    try {
      await dashboardAPI.deleteDataset(currentProject.id, dataset.id);
      router.push('/dashboard/datasets');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete dataset');
    }
  }, [currentProject, dataset, router]);

  if (!currentProject) {
    return <p className="text-sm text-muted-foreground">Select a project first.</p>;
  }

  if (loading && !dataset) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error && !dataset) {
    return (
      <div className="space-y-4">
        <Link href="/dashboard/datasets" className="text-sm text-amber-600 hover:underline">
          ← Datasets
        </Link>
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
          {error}
        </div>
      </div>
    );
  }

  if (!dataset) return null;

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/dashboard/datasets"
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Datasets
        </Link>
        <div className="mt-1 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
              {dataset.name}
            </h1>
            {dataset.description && (
              <p className="mt-1 text-sm text-muted-foreground">{dataset.description}</p>
            )}
            <div className="mt-2 flex items-center gap-2 text-xs text-zinc-500">
              <Badge variant="outline">{items.length} {items.length === 1 ? 'item' : 'items'}</Badge>
              <span>· Updated {new Date(dataset.updatedAt).toLocaleString()}</span>
            </div>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => setAddMode((m) => (m === 'manual' ? 'closed' : 'manual'))}>
              + Add item
            </Button>
            <Button variant="outline" onClick={() => setAddMode((m) => (m === 'import' ? 'closed' : 'import'))}>
              Import JSON
            </Button>
            <Button variant="ghost" className="text-red-600 hover:text-red-700" onClick={handleDeleteDataset}>
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

      {addMode === 'manual' && (
        <ManualAddForm
          projectId={currentProject.id}
          datasetId={dataset.id}
          onSaved={() => {
            setError(null);
            setAddMode('closed');
            void refresh();
          }}
          onCancel={() => setAddMode('closed')}
          onError={setError}
        />
      )}

      {addMode === 'import' && (
        <ImportForm
          projectId={currentProject.id}
          datasetId={dataset.id}
          onSaved={() => {
            setError(null);
            setAddMode('closed');
            void refresh();
          }}
          onCancel={() => setAddMode('closed')}
          onError={setError}
        />
      )}

      <ItemsTable items={items} onDelete={handleDeleteItem} />

      <EvalsSection datasetId={dataset.id} evals={evals} />
    </div>
  );
}

// ----- evals section ------------------------------------------------------

function EvalsSection({ datasetId, evals }: { datasetId: string; evals: Eval[] }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-2 space-y-0">
        <CardTitle className="text-base flex items-center gap-2">
          <span aria-hidden>🍋</span> Evals
        </CardTitle>
        <Link
          href={`/dashboard/datasets/${datasetId}/evals/new`}
          className="inline-flex items-center justify-center rounded-md text-xs font-medium border border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-800 px-3 py-1.5 transition-colors"
        >
          + New eval
        </Link>
      </CardHeader>
      <CardContent className="p-0">
        {evals.length === 0 ? (
          <div className="px-6 py-8 text-center text-sm text-muted-foreground">
            No evals yet. Define one to score how your agent does on these
            cases over time — pick scorers, give it a name, then drive runs
            from your CI via the SDK.
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="border-y border-zinc-200 dark:border-zinc-700 text-left text-xs uppercase text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Name</th>
                <th className="px-4 py-2 font-medium">Scorers</th>
                <th className="px-4 py-2 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {evals.map((e) => (
                <tr
                  key={e.id}
                  className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30"
                >
                  <td className="px-4 py-2">
                    <Link
                      href={`/dashboard/datasets/${datasetId}/evals/${e.id}`}
                      className="font-medium text-zinc-900 dark:text-white hover:text-amber-600 dark:hover:text-amber-400"
                    >
                      {e.name}
                    </Link>
                    {e.description && (
                      <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">
                        {e.description}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    <div className="flex flex-wrap gap-1">
                      {e.scorers.map((s) => (
                        <Badge key={s.id} variant="outline" className="text-[10px]">
                          {s.type}
                        </Badge>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-2 text-xs text-zinc-500">
                    {new Date(e.createdAt).toLocaleDateString()}
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

// ----- items table -------------------------------------------------------

function ItemsTable({
  items,
  onDelete,
}: {
  items: DatasetItem[];
  onDelete: (itemId: string) => void;
}) {
  if (items.length === 0) {
    return (
      <Card>
        <CardContent className="py-10 text-center">
          <div className="text-3xl mb-2" aria-hidden>📝</div>
          <p className="text-sm text-muted-foreground max-w-md mx-auto">
            No items yet. Open a trace, find a span worth evaluating, and click
            <Badge variant="outline" className="mx-1">🍋 Add to dataset</Badge>
            to seed cases from real production behaviour.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Items</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <table className="w-full text-sm">
          <thead className="border-b border-zinc-200 dark:border-zinc-700 text-left text-xs uppercase text-zinc-500">
            <tr>
              <th className="px-4 py-2 font-medium">Input</th>
              <th className="px-4 py-2 font-medium">Expected</th>
              <th className="px-4 py-2 font-medium">Source</th>
              <th className="px-4 py-2 font-medium">Created</th>
              <th className="px-4 py-2 w-0" />
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <ItemRow key={it.id} item={it} onDelete={onDelete} />
            ))}
          </tbody>
        </table>
      </CardContent>
    </Card>
  );
}

function ItemRow({ item, onDelete }: { item: DatasetItem; onDelete: (id: string) => void }) {
  const inputPreview = useMemo(() => previewJSON(item.input), [item.input]);
  const expectedPreview = useMemo(() => previewJSON(item.expected), [item.expected]);

  return (
    <tr className="border-b border-zinc-100 dark:border-zinc-800 last:border-0 hover:bg-zinc-50 dark:hover:bg-zinc-800/30">
      <td className="px-4 py-2 max-w-xs">
        <span className="block truncate font-mono text-xs text-zinc-700 dark:text-zinc-300" title={inputPreview}>
          {inputPreview}
        </span>
      </td>
      <td className="px-4 py-2 max-w-xs">
        {item.expected === null || item.expected === undefined ? (
          <span className="text-zinc-400 text-xs italic">none</span>
        ) : (
          <span className="block truncate font-mono text-xs text-zinc-700 dark:text-zinc-300" title={expectedPreview}>
            {expectedPreview}
          </span>
        )}
      </td>
      <td className="px-4 py-2">
        {item.sourceTraceId ? (
          <Link
            href={`/dashboard/traces/${item.sourceTraceId}`}
            className="text-xs text-amber-600 hover:underline dark:text-amber-400"
            title={`From trace ${item.sourceTraceId}`}
          >
            trace
          </Link>
        ) : (
          <span className="text-xs text-zinc-400">manual</span>
        )}
      </td>
      <td className="px-4 py-2 text-xs text-zinc-500">
        {new Date(item.createdAt).toLocaleDateString()}
      </td>
      <td className="px-4 py-2 text-right">
        <button
          type="button"
          onClick={() => onDelete(item.id)}
          className="text-xs text-red-600 hover:text-red-700 dark:text-red-400"
        >
          Delete
        </button>
      </td>
    </tr>
  );
}

function previewJSON(v: unknown): string {
  if (v === null || v === undefined) return '';
  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}

// ----- add forms ---------------------------------------------------------

function ManualAddForm({
  projectId,
  datasetId,
  onSaved,
  onCancel,
  onError,
}: {
  projectId: string;
  datasetId: string;
  onSaved: () => void;
  onCancel: () => void;
  onError: (msg: string) => void;
}) {
  const [inputText, setInputText] = useState('');
  const [expectedText, setExpectedText] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (inputText.trim() === '') {
      onError('Input is required');
      return;
    }
    let inputValue: unknown;
    try {
      inputValue = JSON.parse(inputText);
    } catch {
      onError('Input must be valid JSON (wrap plain strings in double quotes)');
      return;
    }
    let expectedValue: unknown;
    if (expectedText.trim() !== '') {
      try {
        expectedValue = JSON.parse(expectedText);
      } catch {
        onError('Expected must be valid JSON');
        return;
      }
    }
    setSaving(true);
    try {
      await dashboardAPI.createDatasetItem(projectId, datasetId, {
        input: inputValue,
        expected: expectedValue,
      });
      onSaved();
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Failed to save item');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Add item manually</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-3">
          <JsonField label="Input (JSON)" required value={inputText} onChange={setInputText} placeholder={'"what is the cheapest car?"'} />
          <JsonField label="Expected (JSON, optional)" value={expectedText} onChange={setExpectedText} placeholder={'{"minResults": 1}'} />
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={onCancel} disabled={saving}>
              Cancel
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? 'Saving…' : 'Add item'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function ImportForm({
  projectId,
  datasetId,
  onSaved,
  onCancel,
  onError,
}: {
  projectId: string;
  datasetId: string;
  onSaved: () => void;
  onCancel: () => void;
  onError: (msg: string) => void;
}) {
  const [text, setText] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch {
      onError('Body must be a JSON array of items');
      return;
    }
    if (!Array.isArray(parsed)) {
      onError('Body must be a JSON array (e.g. [{ "input": "…" }])');
      return;
    }
    setSaving(true);
    try {
      const items = parsed.map((entry) => {
        if (typeof entry !== 'object' || entry === null || !('input' in entry)) {
          throw new Error('Each item must be an object with an "input" field');
        }
        const e = entry as Record<string, unknown>;
        return {
          input: e.input,
          expected: e.expected,
          metadata: (e.metadata as Record<string, unknown> | undefined) ?? undefined,
        };
      });
      await dashboardAPI.importDatasetItems(projectId, datasetId, items);
      // Success — parent handles refresh + error reset via onSaved.
      onSaved();
    } catch (err) {
      onError(err instanceof Error ? err.message : 'Failed to import items');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Import items (JSON array)</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-3">
          <JsonField
            label="JSON array"
            required
            rows={8}
            value={text}
            onChange={setText}
            placeholder={'[\n  {"input": "honda civic", "expected": {"minResults": 1}},\n  {"input": "lambo", "expected": {"minResults": 0}}\n]'}
          />
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={onCancel} disabled={saving}>
              Cancel
            </Button>
            <Button type="submit" disabled={saving || text.trim() === ''}>
              {saving ? 'Importing…' : 'Import'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

function JsonField({
  label,
  value,
  onChange,
  placeholder,
  rows = 4,
  required = false,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  rows?: number;
  required?: boolean;
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
        {label}
        {required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={rows}
        placeholder={placeholder}
        className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm font-mono text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-amber-500/40 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
      />
    </div>
  );
}
