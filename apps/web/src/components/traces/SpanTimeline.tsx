import { cn } from '@/lib/utils';
import type { SpanTimelineProps } from './types';

const TYPE_COLORS: Record<string, string> = {
  llm: 'bg-purple-500',
  tool: 'bg-blue-500',
  retrieval: 'bg-green-500',
  custom: 'bg-zinc-500',
};

export function SpanTimeline({ start, width, type, status }: SpanTimelineProps) {
  const barColor = status === 'error'
    ? 'bg-red-500'
    : TYPE_COLORS[type] || 'bg-zinc-500';

  return (
    <div className="relative h-1.5 w-full bg-zinc-100 dark:bg-zinc-800 rounded-full overflow-hidden">
      <div
        className={cn(
          'absolute h-full rounded-full transition-all',
          barColor,
          status === 'pending' && 'animate-pulse'
        )}
        style={{
          left: `${start}%`,
          width: `${Math.max(width, 2)}%`,
        }}
      />
    </div>
  );
}
