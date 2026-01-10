'use client';

import { SpanTypeIcon } from './SpanTypeIcon';
import { formatDuration, formatCost } from './utils';
import { cn } from '@/lib/utils';
import type { SpanTreeNodeProps } from './types';

export function SpanTreeNode({
  node,
  isSelected,
  isExpanded,
  selectedSpanId,
  expandedNodes,
  onSelect,
  onToggle,
  onSelectSpan,
  onToggleExpand,
}: SpanTreeNodeProps) {
  const { span, children, depth } = node;
  const hasChildren = children.length > 0;
  const indent = depth * 20;

  // Determine display info based on span type
  const isToolUse = span.isToolUse;
  const isLlm = span.type === 'llm';
  const isAgent = span.type === 'agent';

  // Display name based on type
  const getDisplayName = () => {
    if (isLlm) {
      // For LLM spans, show descriptive name based on subType
      if (span.subType === 'planning') return 'Planning';
      if (span.subType === 'response') return 'Generation';
      // Fallback: if name is same as model, show generic name
      if (span.name === span.model || span.name?.includes('anthropic') || span.name?.includes('gpt')) {
        return 'Generation';
      }
    }
    if (isToolUse) {
      return 'Tool Usage';
    }
    return span.name;
  };
  const displayName = getDisplayName();

  // Total tokens for LLM spans
  const totalTokens = (span.inputTokens || 0) + (span.outputTokens || 0);

  // Color classes based on type and status
  const getNameColorClass = () => {
    // Errors are always red
    if (span.status === 'error') return 'text-red-500';

    // Synthetic tool use nodes
    if (isToolUse) return 'text-cyan-400';

    // By span type
    switch (span.type) {
      case 'llm':
        // Planning = purple, Response = green
        return span.subType === 'planning' ? 'text-purple-400' : 'text-emerald-400';
      case 'agent':
        return 'text-white';
      case 'tool':
        return 'text-cyan-400';
      case 'retrieval':
        return 'text-blue-400';
      case 'embedding':
        return 'text-indigo-400';
      case 'guardrail':
        return 'text-amber-400';
      case 'rerank':
        return 'text-orange-400';
      case 'custom':
      default:
        return 'text-zinc-400';
    }
  };

  return (
    <div>
      <div
        className={cn(
          'px-2 py-2 cursor-pointer border-l-2 transition-colors',
          isSelected
            ? 'bg-amber-50 dark:bg-amber-900/20 border-amber-500'
            : 'border-transparent hover:bg-zinc-50 dark:hover:bg-zinc-800/50'
        )}
        style={{ paddingLeft: `${8 + indent}px` }}
        onClick={onSelect}
      >
        <div className="flex items-center gap-2">
          {/* Expand/Collapse button */}
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggle();
            }}
            className={cn(
              'w-4 h-4 flex items-center justify-center rounded flex-shrink-0',
              hasChildren ? 'hover:bg-zinc-200 dark:hover:bg-zinc-700' : 'invisible'
            )}
          >
            <svg
              className={cn(
                'w-3 h-3 text-zinc-400 transition-transform',
                isExpanded && 'rotate-90'
              )}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M8.25 4.5l7.5 7.5-7.5 7.5" />
            </svg>
          </button>

          {/* Type Icon */}
          <SpanTypeIcon type={span.type} size="sm" />

          {/* Main content */}
          <div className="flex-1 min-w-0">
            {/* Row 1: Name + Metrics */}
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <span className={cn('font-medium text-sm truncate', getNameColorClass())}>
                  {displayName}
                </span>
              </div>

              {/* Right side metrics */}
              <div className="flex items-center gap-3 text-xs flex-shrink-0">
                {/* Tokens (only for LLM) */}
                {isLlm && totalTokens > 0 && (
                  <span className="text-zinc-400 tabular-nums">
                    # {totalTokens.toLocaleString()}
                  </span>
                )}

                {/* Duration */}
                <span className="text-zinc-400 tabular-nums flex items-center gap-1">
                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {formatDuration(span.durationMs)}
                </span>

                {/* Cost (only for LLM) */}
                {isLlm && span.costUsd !== null && span.costUsd > 0 && (
                  <span className="text-amber-500 font-mono tabular-nums">
                    $ {formatCost(span.costUsd)}
                  </span>
                )}

                {/* Status indicator */}
                <div className={cn(
                  'w-2 h-2 rounded-full flex-shrink-0',
                  span.status === 'success' && 'bg-emerald-500',
                  span.status === 'error' && 'bg-red-500',
                  span.status === 'pending' && 'bg-amber-500 animate-pulse'
                )} />
              </div>
            </div>

            {/* Row 2: Subtitle (model for LLM, tool name for tool use) */}
            {isLlm && span.model && (
              <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate mt-0.5 ml-0">
                {span.model}
              </div>
            )}
            {isToolUse && (
              <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate mt-0.5 ml-0">
                {span.name}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Children (recursive) */}
      {isExpanded && hasChildren && (
        <div>
          {children.map(child => (
            <SpanTreeNode
              key={child.span.id}
              node={child}
              isSelected={selectedSpanId === child.span.id}
              isExpanded={expandedNodes.has(child.span.id)}
              selectedSpanId={selectedSpanId}
              expandedNodes={expandedNodes}
              onSelect={() => onSelectSpan(child.span.id)}
              onToggle={() => onToggleExpand(child.span.id)}
              onSelectSpan={onSelectSpan}
              onToggleExpand={onToggleExpand}
            />
          ))}
        </div>
      )}
    </div>
  );
}
