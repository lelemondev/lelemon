// Re-export types from api.ts - backend now handles all processing
export type {
  ProcessedSpan,
  SpanNode,
  ToolUse,
  TimelineContext
} from '@/lib/api';

// Alias for backward compatibility
export type ExtendedSpan = import('@/lib/api').ProcessedSpan;

/**
 * Props del árbol de spans
 */
export interface SpanTreeProps {
  nodes: import('@/lib/api').SpanNode[];
  selectedSpanId: string | null;
  expandedNodes: Set<string>;
  onSelectSpan: (spanId: string) => void;
  onToggleExpand: (spanId: string) => void;
}

/**
 * Props de un nodo individual
 */
export interface SpanTreeNodeProps {
  node: import('@/lib/api').SpanNode;
  isSelected: boolean;
  isExpanded: boolean;
  selectedSpanId: string | null;
  expandedNodes: Set<string>;
  onSelect: () => void;
  onToggle: () => void;
  onSelectSpan: (id: string) => void;
  onToggleExpand: (id: string) => void;
}

/**
 * Props del panel de detalle
 */
export interface SpanDetailProps {
  span: import('@/lib/api').ProcessedSpan | null;
  allSpans?: import('@/lib/api').ProcessedSpan[];
  onClose?: () => void;
}

/**
 * Props del timeline
 */
export interface SpanTimelineProps {
  start: number;
  width: number;
  type: string;
  status: string;
}

/**
 * Configuración visual por tipo de span
 */
export interface SpanTypeConfig {
  icon: string;
  color: string;
  bgColor: string;
  label: string;
}
