'use client';

import { ExternalLink } from 'lucide-react';

interface TraceInfoProps {
  traceId: string;
}

export function TraceInfo({ traceId }: TraceInfoProps) {
  const dashboardUrl = process.env.NEXT_PUBLIC_DASHBOARD_URL || 'http://localhost:3000';
  const traceUrl = `${dashboardUrl}/dashboard/traces/${traceId}`;

  return (
    <a
      href={traceUrl}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-1 px-2 py-0.5 bg-purple-500/10 text-purple-400 rounded hover:bg-purple-500/20 transition-colors"
    >
      <span className="font-mono">{traceId.slice(0, 8)}...</span>
      <ExternalLink className="w-3 h-3" />
    </a>
  );
}
