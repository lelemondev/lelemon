'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useState } from 'react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  dashboardAPI,
  type Scorer,
  type ScorerType,
  type CreateEvalInput,
} from '@/lib/api';
import { useProject } from '@/lib/project-context';

// Pre-baked config templates per scorer type. Showing one in the textarea
// when the user picks a type is faster than reading docs.
const CONFIG_TEMPLATES: Record<ScorerType, string> = {
  exact_match: '',
  contains: '{\n  "value": "civic"\n}',
  json_path: '{\n  "path": "results.0.id",\n  "op": "eq",\n  "value": "v1"\n}',
  regex: '{\n  "pattern": "^honda \\\\w+$"\n}',
  client_reported: '',
};

const SCORER_DESCRIPTIONS: Record<ScorerType, string> = {
  exact_match: "Deep-equals actual against the dataset item's expected. No config.",
  contains: 'Substring (string actual) or membership (array/map actual).',
  json_path: 'Extract a dotted path and compare with op (eq, ne, gt, gte, lt, lte).',
  regex: 'Test a regular expression against a string actual.',
  client_reported:
    'Verdict supplied by the SDK/CI alongside each result — useful for LLM-as-judge against your own provider key, or domain-specific checks. The platform stores it verbatim and ANDs it with built-in scorers.',
};

interface ScorerDraft {
  id: string;
  name: string;
  type: ScorerType;
  configText: string;
}

function newDraft(idx: number): ScorerDraft {
  return {
    id: `scorer-${idx + 1}`,
    name: '',
    type: 'exact_match',
    configText: '',
  };
}

export default function NewEvalPage() {
  const { currentProject } = useProject();
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const datasetId = params?.id ?? '';

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [scorers, setScorers] = useState<ScorerDraft[]>([newDraft(0)]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateScorer = (i: number, patch: Partial<ScorerDraft>) => {
    setScorers((prev) =>
      prev.map((sc, idx) => {
        if (idx !== i) return sc;
        const next = { ...sc, ...patch };
        // Swap to a fresh template when the type changes and the current
        // config is empty or still equals the previous template.
        if (patch.type && patch.type !== sc.type && (sc.configText === '' || sc.configText === CONFIG_TEMPLATES[sc.type])) {
          next.configText = CONFIG_TEMPLATES[patch.type];
        }
        return next;
      }),
    );
  };

  const addScorer = () => setScorers((prev) => [...prev, newDraft(prev.length)]);
  const removeScorer = (i: number) =>
    setScorers((prev) => (prev.length > 1 ? prev.filter((_, idx) => idx !== i) : prev));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentProject) return;
    if (name.trim() === '') {
      setError('Name is required');
      return;
    }
    if (scorers.length === 0) {
      setError('Add at least one scorer');
      return;
    }

    // Parse each scorer's config; bail on the first JSON error with a clear
    // pointer to which scorer is broken.
    const parsedScorers: Scorer[] = [];
    for (let i = 0; i < scorers.length; i++) {
      const sc = scorers[i];
      let config: Record<string, unknown> | undefined;
      const text = sc.configText.trim();
      if (text !== '') {
        try {
          const parsed = JSON.parse(text);
          if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
            setError(`Scorer ${i + 1}: config must be a JSON object`);
            return;
          }
          config = parsed as Record<string, unknown>;
        } catch {
          setError(`Scorer ${i + 1}: config is not valid JSON`);
          return;
        }
      }
      parsedScorers.push({
        id: sc.id,
        name: sc.name.trim() || undefined,
        type: sc.type,
        config,
      });
    }

    const input: CreateEvalInput = {
      datasetId,
      name: name.trim(),
      description: description.trim() || undefined,
      scorers: parsedScorers,
    };

    setSaving(true);
    setError(null);
    try {
      const created = await dashboardAPI.createEval(currentProject.id, input);
      router.push(`/dashboard/datasets/${datasetId}/evals/${created.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create eval');
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
          href={`/dashboard/datasets/${datasetId}`}
          className="text-xs text-zinc-500 hover:text-amber-600 dark:hover:text-amber-400"
        >
          ← Back to dataset
        </Link>
        <h1 className="mt-1 text-3xl font-bold tracking-tight flex items-center gap-2">
          <span aria-hidden>🍋</span> New eval
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Pick scorers; the platform applies them server-side every time the
          SDK posts an item result.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Definition</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <label htmlFor="eval-name" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Name
              </label>
              <Input
                id="eval-name"
                required
                autoFocus
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="vehicle-search-exact"
                maxLength={200}
              />
            </div>
            <div>
              <label htmlFor="eval-desc" className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Description <span className="text-zinc-400">(optional)</span>
              </label>
              <Input
                id="eval-desc"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Exact-match baseline for the vehicle search regression set"
                maxLength={2000}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between gap-2 space-y-0">
            <CardTitle className="text-base">Scorers</CardTitle>
            <Button type="button" variant="outline" size="sm" onClick={addScorer}>
              + Add scorer
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            {scorers.map((sc, i) => (
              <div
                key={sc.id}
                className="border border-zinc-200 dark:border-zinc-700 rounded-md p-3 space-y-2"
              >
                <div className="flex items-start gap-2">
                  <Badge variant="outline" className="font-mono text-[10px] mt-1">
                    {sc.id}
                  </Badge>
                  <div className="flex-1 grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[11px] text-zinc-500 dark:text-zinc-400">Name (optional)</label>
                      <Input
                        value={sc.name}
                        onChange={(e) => updateScorer(i, { name: e.target.value })}
                        placeholder="e.g. exact-id-match"
                        className="h-8 text-xs"
                      />
                    </div>
                    <div>
                      <label className="text-[11px] text-zinc-500 dark:text-zinc-400">Type</label>
                      <Select
                        value={sc.type}
                        onValueChange={(v) => updateScorer(i, { type: v as ScorerType })}
                      >
                        <SelectTrigger className="h-8 text-xs">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="exact_match">exact_match</SelectItem>
                          <SelectItem value="contains">contains</SelectItem>
                          <SelectItem value="json_path">json_path</SelectItem>
                          <SelectItem value="regex">regex</SelectItem>
                          <SelectItem value="client_reported">client_reported</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  {scorers.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeScorer(i)}
                      className="text-[11px] text-red-600 hover:text-red-700 dark:text-red-400 mt-1"
                    >
                      Remove
                    </button>
                  )}
                </div>
                <p className="text-[11px] text-zinc-500 dark:text-zinc-400 italic">
                  {SCORER_DESCRIPTIONS[sc.type]}
                </p>
                {/* Configless scorers (empty template) don't need a config
                    textarea. exact_match drives off dataset_item.expected;
                    client_reported takes its verdict from the SDK at post-time. */}
                {CONFIG_TEMPLATES[sc.type] !== '' && (
                  <div>
                    <label className="text-[11px] text-zinc-500 dark:text-zinc-400">
                      Config (JSON)
                    </label>
                    <textarea
                      value={sc.configText}
                      onChange={(e) => updateScorer(i, { configText: e.target.value })}
                      rows={5}
                      placeholder={CONFIG_TEMPLATES[sc.type]}
                      className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-xs font-mono text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-amber-500/40 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
                    />
                  </div>
                )}
              </div>
            ))}
          </CardContent>
        </Card>

        {error && (
          <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-2">
          <Link href={`/dashboard/datasets/${datasetId}`}>
            <Button type="button" variant="ghost" disabled={saving}>
              Cancel
            </Button>
          </Link>
          <Button type="submit" disabled={saving}>
            {saving ? 'Creating…' : 'Create eval'}
          </Button>
        </div>
      </form>
    </div>
  );
}
