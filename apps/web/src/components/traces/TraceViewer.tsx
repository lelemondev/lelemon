'use client';

import { useState, useMemo, useCallback } from 'react';
import { SpanTree } from './SpanTree';
import { SpanDetail } from './SpanDetail';
import { flattenSpanTree, getAncestorIds } from './utils';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import type { TraceDetailResponse } from '@/lib/api';

interface TraceViewerProps {
  trace: TraceDetailResponse;
}

export function TraceViewer({ trace }: TraceViewerProps) {
  const [selectedSpanId, setSelectedSpanId] = useState<string | null>(null);

  // El árbol ya viene pre-procesado del backend
  const treeNodes = trace.spanTree ?? [];

  // Spans aplanados (para búsquedas y selección)
  const processedSpans = useMemo(() => flattenSpanTree(treeNodes), [treeNodes]);

  // Por defecto expandir todos los spans
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(() => {
    const allIds = processedSpans.map(s => s.id);
    return new Set(allIds);
  });

  // Span seleccionado
  const selectedSpan = useMemo(
    () => processedSpans.find(s => s.id === selectedSpanId) ?? null,
    [processedSpans, selectedSpanId]
  );

  const handleSelectSpan = useCallback(
    (spanId: string) => {
      setSelectedSpanId(spanId);

      // Auto-expandir ancestros para mostrar el span seleccionado
      const ancestors = getAncestorIds(processedSpans, spanId);
      if (ancestors.length > 0) {
        setExpandedNodes(prev => {
          const next = new Set(prev);
          ancestors.forEach(id => next.add(id));
          return next;
        });
      }
    },
    [processedSpans]
  );

  const handleToggleExpand = useCallback((spanId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(spanId)) {
        next.delete(spanId);
      } else {
        next.add(spanId);
      }
      return next;
    });
  }, []);

  const handleExpandAll = useCallback(() => {
    const allIds = processedSpans.map(s => s.id);
    setExpandedNodes(new Set(allIds));
  }, [processedSpans]);

  const handleCollapseAll = useCallback(() => {
    setExpandedNodes(new Set());
  }, []);

  return (
    <>
      {/* Desktop: 2 columnas */}
      <div className="hidden lg:grid lg:grid-cols-[1fr_1fr] lg:gap-4 h-full">
        {/* Columna izquierda: Árbol */}
        <div className="border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden flex flex-col">
          {/* Header */}
          <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50 flex-shrink-0">
            <h3 className="font-medium text-sm text-zinc-700 dark:text-zinc-300">
              Spans ({processedSpans.length})
            </h3>
            <div className="flex gap-1">
              <button
                onClick={handleExpandAll}
                className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 px-2 py-0.5 rounded hover:bg-zinc-200 dark:hover:bg-zinc-700"
              >
                Expand
              </button>
              <button
                onClick={handleCollapseAll}
                className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 px-2 py-0.5 rounded hover:bg-zinc-200 dark:hover:bg-zinc-700"
              >
                Collapse
              </button>
            </div>
          </div>
          {/* Tree */}
          <div className="flex-1 overflow-auto min-h-0">
            <SpanTree
              nodes={treeNodes}
              selectedSpanId={selectedSpanId}
              expandedNodes={expandedNodes}
              onSelectSpan={handleSelectSpan}
              onToggleExpand={handleToggleExpand}
            />
          </div>
        </div>

        {/* Columna derecha: Detalle */}
        <div className="border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden flex flex-col">
          <SpanDetail span={selectedSpan} allSpans={processedSpans} onClose={() => setSelectedSpanId(null)} />
        </div>
      </div>

      {/* Mobile: Tabs */}
      <div className="lg:hidden">
        <Tabs defaultValue="tree">
          <TabsList className="w-full">
            <TabsTrigger value="tree" className="flex-1">
              Span Tree
            </TabsTrigger>
            <TabsTrigger value="detail" className="flex-1" disabled={!selectedSpan}>
              Detail {selectedSpan ? `(${selectedSpan.name.slice(0, 15)}...)` : ''}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="tree" className="mt-4">
            <div className="border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden">
              <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-200 dark:border-zinc-700">
                <h3 className="font-semibold text-sm text-zinc-900 dark:text-white">Spans</h3>
                <div className="flex gap-1">
                  <button
                    onClick={handleExpandAll}
                    className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 px-2 py-1"
                  >
                    Expand
                  </button>
                  <button
                    onClick={handleCollapseAll}
                    className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 px-2 py-1"
                  >
                    Collapse
                  </button>
                </div>
              </div>
              <div className="max-h-[60vh] overflow-auto">
                <SpanTree
                  nodes={treeNodes}
                  selectedSpanId={selectedSpanId}
                  expandedNodes={expandedNodes}
                  onSelectSpan={handleSelectSpan}
                  onToggleExpand={handleToggleExpand}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="detail" className="mt-4">
            <div className="border border-zinc-200 dark:border-zinc-700 rounded-xl bg-card overflow-hidden">
              <SpanDetail span={selectedSpan} allSpans={processedSpans} />
            </div>
          </TabsContent>
        </Tabs>
      </div>
    </>
  );
}
