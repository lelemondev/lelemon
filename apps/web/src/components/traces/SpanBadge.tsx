import { SPAN_TYPE_CONFIG } from './utils';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { Span } from '@/lib/api';

interface SpanBadgeProps {
  type: Span['type'];
  className?: string;
}

export function SpanBadge({ type, className }: SpanBadgeProps) {
  const config = SPAN_TYPE_CONFIG[type];

  return (
    <Badge
      variant="outline"
      className={cn('text-xs uppercase', config.bgColor, config.color, className)}
    >
      {config.label}
    </Badge>
  );
}
