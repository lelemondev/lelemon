import { SPAN_TYPE_CONFIG } from './utils';
import { cn } from '@/lib/utils';
import type { Span } from '@/lib/api';

interface SpanTypeIconProps {
  type: Span['type'];
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const SIZE_CLASSES = {
  sm: 'w-4 h-4',
  md: 'w-5 h-5',
  lg: 'w-6 h-6',
};

export function SpanTypeIcon({ type, size = 'md', className }: SpanTypeIconProps) {
  const config = SPAN_TYPE_CONFIG[type];

  return (
    <svg
      className={cn(SIZE_CLASSES[size], config.color, className)}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={1.5}
    >
      <path strokeLinecap="round" strokeLinejoin="round" d={config.icon} />
    </svg>
  );
}
