'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useProject } from '@/lib/project-context';
import {
  type Dataset,
  type DatasetItem,
  type ProcessedSpan,
  dashboardAPI,
} from '@/lib/api';

/**
 * AddToDatasetDialog turns a real production span into a curated eval case.
 *
 * Highest-leverage Phase 1 action: the user lands on a failing trace, opens
 * the offending span, clicks "Add to dataset" — input is pre-filled from the
 * span, they fill in what *should* have happened, hit save. Dataset is either
 * picked from the project's existing list or created inline. No leaving the
 * dashboard.
 *
 * We deliberately do NOT pre-fill `expected` from `span.output` — what
 * happened ≠ what should have happened, and seeding it would bake every
 * buggy output into a "gold" expectation. The backend mirrors this rule.
 */
const NEW_DATASET_VALUE = '__new__';

interface AddToDatasetDialogProps {
  span: ProcessedSpan | null;
  open: boolean;
  onClose: () => void;
  onSuccess?: (item: DatasetItem) => void;
}

export function AddToDatasetDialog({ span, open, onClose, onSuccess }: AddToDatasetDialogProps) {
  const { currentProject } = useProject();

  const [datasets, setDatasets] = useState<Dataset[]>([]);
  const [datasetsLoading, setDatasetsLoading] = useState(false);
  const [datasetId, setDatasetId] = useState<string>('');
  const [newDatasetName, setNewDatasetName] = useState('');
  const [expectedText, setExpectedText] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Reset state on every open so the dialog is clean each time.
  useEffect(() => {
    if (!open) return;
    setDatasetId('');
    setNewDatasetName('');
    setExpectedText('');
    setError(null);
    setSubmitting(false);

    if (!currentProject) return;
    setDatasetsLoading(true);
    dashboardAPI
      .listDatasets(currentProject.id, { limit: 200 })
      .then((page) => setDatasets(page.data))
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load datasets');
      })
      .finally(() => setDatasetsLoading(false));
  }, [open, currentProject]);

  // ESC closes the dialog (matches the rest of the trace viewer's UX).
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open, onClose]);

  const isNewDataset = datasetId === NEW_DATASET_VALUE;

  const inputPreview = useMemo(() => {
    if (!span) return '';
    try {
      return JSON.stringify(span.input, null, 2);
    } catch {
      return String(span.input);
    }
  }, [span]);

  // Expected is optional. Empty string → no expectation. Non-empty must parse
  // as JSON (a bare string like `hello` should be written as `"hello"`).
  const parseExpected = useCallback((): { ok: true; value: unknown } | { ok: false; err: string } => {
    const trimmed = expectedText.trim();
    if (trimmed === '') return { ok: true, value: undefined };
    try {
      return { ok: true, value: JSON.parse(trimmed) };
    } catch {
      return { ok: false, err: 'Expected must be valid JSON (e.g. "hello", {"k": 1}, or 42)' };
    }
  }, [expectedText]);

  const submitDisabled =
    !span ||
    !currentProject ||
    submitting ||
    datasetId === '' ||
    (isNewDataset && newDatasetName.trim() === '');

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!span || !currentProject) return;

      const expected = parseExpected();
      if (!expected.ok) {
        setError(expected.err);
        return;
      }

      setSubmitting(true);
      setError(null);
      try {
        // Resolve the dataset (existing pick OR inline create).
        let targetDatasetId = datasetId;
        if (isNewDataset) {
          const created = await dashboardAPI.createDataset(currentProject.id, {
            name: newDatasetName.trim(),
          });
          targetDatasetId = created.id;
        }

        const item = await dashboardAPI.addDatasetItemFromTrace(
          currentProject.id,
          targetDatasetId,
          {
            traceId: span.traceId,
            spanId: span.id,
            expected: expected.value,
          },
        );

        onSuccess?.(item);
        onClose();
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to add to dataset');
      } finally {
        setSubmitting(false);
      }
    },
    [
      span,
      currentProject,
      datasetId,
      isNewDataset,
      newDatasetName,
      parseExpected,
      onSuccess,
      onClose,
    ],
  );

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-to-dataset-title"
      className="fixed inset-0 z-50 flex items-center justify-center"
    >
      {/* Backdrop */}
      <button
        type="button"
        aria-label="Close dialog"
        onClick={onClose}
        className="absolute inset-0 bg-zinc-950/70 backdrop-blur-sm"
      />

      {/* Card */}
      <div className="relative w-full max-w-2xl mx-4 max-h-[90vh] overflow-auto rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-700 dark:bg-zinc-900">
        <form onSubmit={handleSubmit}>
          {/* Header */}
          <div className="flex items-start justify-between gap-4 px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <div>
              <h2
                id="add-to-dataset-title"
                className="text-lg font-semibold text-zinc-900 dark:text-white flex items-center gap-2"
              >
                <span aria-hidden>🍋</span> Add to dataset
              </h2>
              <p className="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
                Save this span as an eval case. Input is copied from the span;
                you provide what <em>should</em> have happened.
              </p>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="p-1 rounded hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-500"
              aria-label="Close"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="px-6 py-4 space-y-4">
            {/* Dataset picker */}
            <div className="space-y-1.5">
              <label
                htmlFor="dataset-select"
                className="text-xs font-medium text-zinc-700 dark:text-zinc-300"
              >
                Dataset
              </label>
              <Select value={datasetId} onValueChange={setDatasetId} disabled={datasetsLoading}>
                <SelectTrigger id="dataset-select">
                  <SelectValue
                    placeholder={datasetsLoading ? 'Loading datasets…' : 'Pick a dataset'}
                  />
                </SelectTrigger>
                <SelectContent>
                  {datasets.map((d) => (
                    <SelectItem key={d.id} value={d.id}>
                      {d.name}
                    </SelectItem>
                  ))}
                  <SelectItem value={NEW_DATASET_VALUE}>＋ Create new dataset…</SelectItem>
                </SelectContent>
              </Select>

              {isNewDataset && (
                <Input
                  autoFocus
                  placeholder="e.g. vehicle-search-regressions"
                  value={newDatasetName}
                  onChange={(e) => setNewDatasetName(e.target.value)}
                  className="mt-2"
                  maxLength={200}
                />
              )}
            </div>

            {/* Input preview (read-only) */}
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                Input (from span)
              </label>
              <pre className="max-h-32 overflow-auto rounded-md border border-zinc-200 bg-zinc-50 px-3 py-2 text-xs text-zinc-700 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                {inputPreview || <span className="text-zinc-400">(empty)</span>}
              </pre>
            </div>

            {/* Expected */}
            <div className="space-y-1.5">
              <label
                htmlFor="expected-text"
                className="text-xs font-medium text-zinc-700 dark:text-zinc-300"
              >
                Expected (JSON, optional)
              </label>
              <textarea
                id="expected-text"
                value={expectedText}
                onChange={(e) => setExpectedText(e.target.value)}
                rows={4}
                placeholder={'e.g. {"minResults": 1}'}
                className="w-full rounded-md border border-zinc-200 bg-white px-3 py-2 text-sm font-mono text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-amber-500/40 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-100"
              />
              <p className="text-[11px] text-zinc-500 dark:text-zinc-400">
                Leave empty when you&apos;ll use an LLM-as-judge or only care
                about structural assertions.
              </p>
            </div>

            {error && (
              <div
                role="alert"
                className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400"
              >
                {error}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-2 px-6 py-3 border-t border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/40">
            <Button type="button" variant="ghost" onClick={onClose} disabled={submitting}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitDisabled}>
              {submitting ? 'Saving…' : 'Add to dataset'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
