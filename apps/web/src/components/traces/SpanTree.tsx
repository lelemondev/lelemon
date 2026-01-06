'use client';

import { SpanTreeNode } from './SpanTreeNode';
import type { SpanTreeProps } from './types';

export function SpanTree({
  nodes,
  selectedSpanId,
  expandedNodes,
  onSelectSpan,
  onToggleExpand,
}: SpanTreeProps) {
  if (nodes.length === 0) {
    return (
      <div className="p-8 text-center text-zinc-500 dark:text-zinc-400">
        No spans recorded
      </div>
    );
  }

  return (
    <div className="py-2">
      {nodes.map(node => (
        <SpanTreeNode
          key={node.span.id}
          node={node}
          isSelected={selectedSpanId === node.span.id}
          isExpanded={expandedNodes.has(node.span.id)}
          selectedSpanId={selectedSpanId}
          expandedNodes={expandedNodes}
          onSelect={() => onSelectSpan(node.span.id)}
          onToggle={() => onToggleExpand(node.span.id)}
          onSelectSpan={onSelectSpan}
          onToggleExpand={onToggleExpand}
        />
      ))}
    </div>
  );
}
